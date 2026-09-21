package httpapi

import (
	"dailyworknotes/internal/domain"
	"dailyworknotes/internal/report"
	"dailyworknotes/internal/repository"
	"dailyworknotes/internal/repository/db"
	"dailyworknotes/internal/service"
	"dailyworknotes/internal/storage"
	"encoding/json"
	"github.com/gofiber/fiber/v3"
	"strings"
	"time"
)

type selection struct {
	IDs           []string `json:"ids"`
	From          string   `json:"from"`
	To            string   `json:"to"`
	AttachmentIDs []string `json:"attachmentIds"`
}

func (a *API) selected(c fiber.Ctx, s selection) ([]domain.Note, error) {
	if len(s.IDs) > 500 {
		return nil, fiber.NewError(400, "Select at most 500 notes")
	}
	out := []domain.Note{}
	if len(s.IDs) > 0 {
		seen := map[string]bool{}
		for _, id := range s.IDs {
			if seen[id] {
				continue
			}
			seen[id] = true
			n, e := a.Repo.Get(c.Context(), user(c), id)
			if e != nil {
				return nil, e
			}
			out = append(out, n)
		}
		return out, nil
	}
	for _, d := range []string{s.From, s.To} {
		if d != "" {
			if _, e := time.Parse("2006-01-02", d); e != nil {
				return nil, fiber.NewError(400, "Invalid date")
			}
		}
	}
	if s.From != "" && s.To != "" && s.From > s.To {
		return nil, fiber.NewError(400, "Invalid date range")
	}
	out, e := a.Repo.List(c.Context(), user(c), domain.Filter{From: s.From, To: s.To, Page: 1, Size: 501})
	if len(out) > 500 {
		return nil, fiber.NewError(400, "Narrow the range to 500 notes or fewer")
	}
	return out, e
}
func (a *API) selectedFiles(c fiber.Ctx, notes []domain.Note, ids []string) ([]domain.Attachment, error) {
	all, e := a.Repo.Attachments(c.Context(), user(c), "")
	if e != nil {
		return nil, e
	}
	allowed := map[string]bool{}
	for _, n := range notes {
		allowed[n.ID] = true
	}
	wanted := map[string]bool{}
	for _, id := range ids {
		wanted[id] = true
	}
	out := []domain.Attachment{}
	var total int64
	for _, f := range all {
		if wanted[f.ID] && allowed[f.NoteID] {
			out = append(out, f)
			delete(wanted, f.ID)
			total += f.Size
		}
	}
	if len(wanted) > 0 {
		return nil, domain.ErrNotFound
	}
	if total > 50<<20 {
		return nil, fiber.NewError(400, "Selected attachments exceed 50 MB")
	}
	return out, nil
}
func (a *API) export(c fiber.Ctx) error {
	var s selection
	if e := c.Bind().JSON(&s); e != nil {
		return fiber.NewError(400, "Invalid JSON")
	}
	notes, e := a.selected(c, s)
	if e != nil {
		return e
	}
	var b []byte
	format := c.Params("format")
	mime := "application/json"
	switch format {
	case "json":
		b, e = json.MarshalIndent(service.Backup{SchemaVersion: 1, Notes: notes, Attachments: []domain.Attachment{}}, "", "  ")
	case "csv":
		mime = "text/csv; charset=utf-8"
		b, e = report.CSV(notes)
	case "pdf":
		mime = "application/pdf"
		b, e = report.PDF(c.Context(), a.Config.PDFURL, notes)
	case "zip":
		mime = "application/zip"
		files, err := a.selectedFiles(c, notes, s.AttachmentIDs)
		if err != nil {
			return err
		}
		contents := map[string][]byte{}
		for i := range files {
			files[i].ArchivePath = "attachments/" + files[i].ID
			data, err := a.Store.Read(files[i].StorageKey)
			if err != nil {
				return err
			}
			contents[files[i].ArchivePath] = data
		}
		pdf, err := report.PDF(c.Context(), a.Config.PDFURL, notes)
		if err != nil {
			return err
		}
		b, e = service.ZIP(service.Backup{SchemaVersion: 1, Notes: notes, Attachments: files}, pdf, contents)
	default:
		return fiber.NewError(400, "Use pdf, csv, json, or zip")
	}
	if e != nil {
		return e
	}
	c.Set("Content-Type", mime)
	c.Set("Content-Disposition", "attachment; filename=daily-work-notes."+format)
	return c.Send(b)
}
func (a *API) parseImport(c fiber.Ctx) (service.ParsedBackup, error) {
	f, e := c.FormFile("file")
	if e != nil {
		return service.ParsedBackup{}, fiber.NewError(400, "Select a backup file")
	}
	r, e := f.Open()
	if e != nil {
		return service.ParsedBackup{}, e
	}
	defer r.Close()
	b, e := storage.ReadLimited(r, 100<<20)
	if e != nil {
		return service.ParsedBackup{}, e
	}
	return service.ParseBackup(b, strings.HasSuffix(strings.ToLower(f.Filename), ".zip"), a.Config.MaxFile)
}
func (a *API) previewImport(c fiber.Ctx) error {
	p, e := a.parseImport(c)
	if e != nil {
		return e
	}
	duplicates := []string{}
	for _, n := range p.Backup.Notes {
		var owner string
		e = a.Repo.Pool.QueryRow(c.Context(), "SELECT user_id FROM notes WHERE id=$1", n.ID).Scan(&owner)
		if e == nil && owner == user(c) {
			duplicates = append(duplicates, n.ID)
		}
	}
	return c.JSON(fiber.Map{"notes": len(p.Backup.Notes), "attachments": len(p.Backup.Attachments), "duplicates": duplicates})
}
func (a *API) importBackup(c fiber.Ctx) error {
	p, e := a.parseImport(c)
	if e != nil {
		return e
	}
	mode := c.FormValue("mode")
	if mode != "skip" && mode != "replace" {
		return fiber.NewError(400, "Choose skip or replace")
	}
	tx, e := a.Repo.Pool.Begin(c.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(c.Context())
	q := db.New(tx)
	// Serialize restores for this owner; note writes still use optimistic versions.
	if _, e = tx.Exec(c.Context(), "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", user(c)); e != nil {
		return e
	}
	written := []string{}
	committed := false
	defer func() {
		if !committed {
			for _, key := range written {
				_ = a.Store.Remove(key)
			}
		}
	}()
	mapping := map[string]string{}
	successful, skipped := 0, 0
	for _, n := range p.Backup.Notes {
		var owner string
		var version int32
		e = tx.QueryRow(c.Context(), "SELECT user_id,version FROM notes WHERE id=$1 FOR UPDATE", n.ID).Scan(&owner, &version)
		original := n.ID
		if e == nil && owner == user(c) {
			if mode == "skip" {
				skipped++
				continue
			}
			n.Version = version
			if _, e = repository.Update(c.Context(), q, user(c), n); e != nil {
				return e
			}
			if _, e = tx.Exec(c.Context(), "INSERT INTO file_deletions(storage_key) SELECT storage_key FROM attachments WHERE note_id=$1 AND user_id=$2 ON CONFLICT DO NOTHING", n.ID, user(c)); e != nil {
				return e
			}
			if _, e = tx.Exec(c.Context(), "DELETE FROM attachments WHERE note_id=$1 AND user_id=$2", n.ID, user(c)); e != nil {
				return e
			}
		} else {
			if e == nil {
				n.ID = service.ID()
			}
			if _, e = repository.Create(c.Context(), q, user(c), n); e != nil {
				return e
			}
		}
		mapping[original] = n.ID
		successful++
	}
	attached := 0
	for _, f := range p.Backup.Attachments {
		note, ok := mapping[f.NoteID]
		if !ok {
			continue
		}
		id := service.ID()
		if e = a.Store.Put(id, p.Files[f.ArchivePath]); e != nil {
			return e
		}
		written = append(written, id)
		_, e = tx.Exec(c.Context(), "INSERT INTO attachments(id,note_id,user_id,name,mime,size,storage_key) VALUES($1,$2,$3,$4,$5,$6,$1)", id, note, user(c), f.Name, f.MIME, f.Size)
		if e != nil {
			return e
		}
		attached++
	}
	// Commit errors can mean the response was lost after a successful commit.
	// Preserve staged files until reconciliation rather than breaking references.
	committed = true
	if e = tx.Commit(c.Context()); e != nil {
		return fiber.NewError(503, "Restore outcome could not be confirmed. Refresh your notes before retrying.")
	}
	return c.JSON(fiber.Map{"successful": successful, "skipped": skipped, "failed": 0, "attachments": attached})
}
