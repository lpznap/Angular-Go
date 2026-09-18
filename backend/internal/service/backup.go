package service

import (
	"archive/zip"
	"bytes"
	"dailyworknotes/internal/domain"
	"dailyworknotes/internal/storage"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
)

const BackupVersion = 1

type Backup struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Notes         []domain.Note       `json:"notes"`
	Attachments   []domain.Attachment `json:"attachments"`
}
type ParsedBackup struct {
	Backup Backup
	Files  map[string][]byte
}

func ParseBackup(raw []byte, zipped bool, limit int64) (ParsedBackup, error) {
	out := ParsedBackup{Files: map[string][]byte{}}
	payload := raw
	bad := func(s string) (ParsedBackup, error) { return out, domain.ValidationError{Message: s} }
	if zipped {
		z, e := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
		if e != nil {
			return bad("Invalid ZIP")
		}
		if len(z.File) > 1001 {
			return bad("Archive contains too many files")
		}
		var total uint64
		for _, f := range z.File {
			if f.Name == "" || path.Clean(f.Name) != f.Name || strings.ContainsAny(f.Name, "\\:\x00") || strings.HasPrefix(f.Name, "/") || strings.HasPrefix(f.Name, "../") || !f.Mode().IsRegular() {
				return bad("Unsafe archive path or entry")
			}
			if _, ok := out.Files[f.Name]; ok {
				return bad("Duplicate archive path")
			}
			total += f.UncompressedSize64
			if total > 100<<20 || f.UncompressedSize64 > 50<<20 {
				return bad("Archive exceeds decompression limits")
			}
			r, e := f.Open()
			if e != nil {
				return out, e
			}
			b, e := storage.ReadLimited(r, 50<<20)
			_ = r.Close()
			if e != nil {
				return out, e
			}
			out.Files[f.Name] = b
		}
		var ok bool
		payload, ok = out.Files["backup.json"]
		if !ok {
			return bad("ZIP must contain backup.json")
		}
	}
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	if e := dec.Decode(&out.Backup); e != nil {
		return bad("Invalid backup JSON: " + e.Error())
	}
	if e := dec.Decode(new(any)); e != io.EOF {
		return bad("Trailing JSON data")
	}
	if out.Backup.SchemaVersion != BackupVersion || len(out.Backup.Notes) > 500 || len(out.Backup.Attachments) > 500 {
		return bad("Unsupported schema or too many items")
	}
	ids := map[string]bool{}
	for i := range out.Backup.Notes {
		n := &out.Backup.Notes[i]
		if len(n.ID) != 32 || strings.Trim(n.ID, "0123456789abcdef") != "" || ids[n.ID] {
			return bad("Invalid or duplicate note ID")
		}
		ids[n.ID] = true
		if e := n.Validate(); e != nil {
			return out, e
		}
	}
	paths := map[string]bool{}
	aids := map[string]bool{}
	for _, a := range out.Backup.Attachments {
		if !ids[a.NoteID] || len(a.ID) != 32 || strings.Trim(a.ID, "0123456789abcdef") != "" || aids[a.ID] || a.ArchivePath != "attachments/"+a.ID || paths[a.ArchivePath] {
			return bad("Invalid attachment reference")
		}
		paths[a.ArchivePath] = true
		aids[a.ID] = true
		b, ok := out.Files[a.ArchivePath]
		if !ok {
			return bad("Referenced attachment missing; use ZIP to restore files")
		}
		typ, e := storage.Validate(a.Name, b, limit)
		if e != nil {
			return out, e
		}
		if typ != a.MIME || int64(len(b)) != a.Size {
			return bad("Attachment metadata does not match contents")
		}
	}
	return out, nil
}
func ZIP(b Backup, pdf []byte, files map[string][]byte) ([]byte, error) {
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	j, e := json.MarshalIndent(b, "", "  ")
	if e != nil {
		return nil, e
	}
	files["backup.json"] = j
	files["report.pdf"] = pdf
	for name, data := range files {
		f, e := w.Create(name)
		if e != nil {
			return nil, e
		}
		if _, e = f.Write(data); e != nil {
			return nil, e
		}
	}
	if e = w.Close(); e != nil {
		return nil, e
	}
	return out.Bytes(), nil
}
func BackupSummary(b Backup) string {
	return fmt.Sprintf("%d notes, %d attachments", len(b.Notes), len(b.Attachments))
}
