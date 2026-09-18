package httpapi

import (
	"context"
	"dailyworknotes/internal/domain"
	"dailyworknotes/internal/service"
	"dailyworknotes/internal/storage"
	"github.com/gofiber/fiber/v3"
	"mime"
	"time"
)

func (a *API) attachments(c fiber.Ctx) error {
	if _, e := a.Repo.Get(c.Context(), user(c), c.Params("id")); e != nil {
		return e
	}
	v, e := a.Repo.Attachments(c.Context(), user(c), c.Params("id"))
	if e != nil {
		return e
	}
	return c.JSON(v)
}
func (a *API) upload(c fiber.Ctx) error {
	if _, e := a.Repo.Get(c.Context(), user(c), c.Params("id")); e != nil {
		return e
	}
	f, e := c.FormFile("file")
	if e != nil {
		return fiber.NewError(400, "Select a file")
	}
	if f.Size > a.Config.MaxFile {
		return fiber.NewError(413, "File is too large")
	}
	r, e := f.Open()
	if e != nil {
		return e
	}
	defer r.Close()
	b, e := storage.ReadLimited(r, a.Config.MaxFile)
	if e != nil {
		return e
	}
	typ, e := storage.Validate(f.Filename, b, a.Config.MaxFile)
	if e != nil {
		return e
	}
	id := service.ID()
	if e = a.Store.Put(id, b); e != nil {
		return e
	}
	_, e = a.Repo.Pool.Exec(c.Context(), "INSERT INTO attachments(id,note_id,user_id,name,mime,size,storage_key) VALUES($1,$2,$3,$4,$5,$6,$1)", id, c.Params("id"), user(c), f.Filename, typ, len(b))
	if e != nil {
		_ = a.Store.Remove(id)
		return e
	}
	return c.Status(201).JSON(domain.Attachment{ID: id, NoteID: c.Params("id"), Name: f.Filename, MIME: typ, Size: int64(len(b))})
}
func (a *API) download(c fiber.Ctx) error {
	var key, name, typ string
	e := a.Repo.Pool.QueryRow(c.Context(), "SELECT storage_key,name,mime FROM attachments WHERE id=$1 AND user_id=$2", c.Params("id"), user(c)).Scan(&key, &name, &typ)
	if e != nil {
		return e
	}
	disposition := "attachment"
	if c.Query("preview") == "1" && (typ == "application/pdf" || len(typ) > 6 && typ[:6] == "image/") {
		disposition = "inline"
	}
	c.Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": name}))
	c.Set("Content-Type", typ)
	c.Set("Content-Security-Policy", "sandbox")
	c.Set("Cache-Control", "private, no-store")
	b, e := a.Store.Read(key)
	if e != nil {
		return e
	}
	return c.Send(b)
}
func (a *API) deleteAttachment(c fiber.Ctx) error {
	tx, e := a.Repo.Pool.Begin(c.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(c.Context())
	var key string
	e = tx.QueryRow(c.Context(), "DELETE FROM attachments WHERE id=$1 AND user_id=$2 RETURNING storage_key", c.Params("id"), user(c)).Scan(&key)
	if e != nil {
		return e
	}
	if _, e = tx.Exec(c.Context(), "INSERT INTO file_deletions(storage_key) VALUES($1) ON CONFLICT DO NOTHING", key); e != nil {
		return e
	}
	if e = tx.Commit(c.Context()); e != nil {
		return e
	}
	return c.SendStatus(204)
}
func (a *API) Cleanup(ctx context.Context) {
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			rows, e := a.Repo.Pool.Query(ctx, "SELECT storage_key FROM file_deletions LIMIT 100")
			if e != nil {
				continue
			}
			keys := []string{}
			for rows.Next() {
				var key string
				if rows.Scan(&key) == nil {
					keys = append(keys, key)
				}
			}
			rows.Close()
			for _, key := range keys {
				if a.Store.Remove(key) == nil {
					_, _ = a.Repo.Pool.Exec(ctx, "DELETE FROM file_deletions WHERE storage_key=$1", key)
				}
			}
			_, _ = a.Repo.Pool.Exec(ctx, "DELETE FROM sessions WHERE expires_at<now()")
		}
	}
}
