package service

import (
	"archive/zip"
	"bytes"
	"dailyworknotes/internal/domain"
	"encoding/json"
	"testing"
)

func backupFixture() Backup {
	return Backup{SchemaVersion: 1, Notes: []domain.Note{{ID: "0123456789abcdef0123456789abcdef", Title: "งาน", WorkDate: "2026-09-18", Status: "todo", Priority: "low"}}, Attachments: []domain.Attachment{}}
}
func TestBackupRoundTrip(t *testing.T) {
	b := backupFixture()
	raw, _ := json.Marshal(b)
	p, e := ParseBackup(raw, false, 100)
	if e != nil || p.Backup.Notes[0].Title != "งาน" {
		t.Fatalf("%v", e)
	}
	b.SchemaVersion = 2
	raw, _ = json.Marshal(b)
	if _, e = ParseBackup(raw, false, 100); e == nil {
		t.Fatal("accepted unknown schema")
	}
}
func TestRejectArchiveTraversal(t *testing.T) {
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	f, _ := w.Create("../escape.txt")
	_, _ = f.Write([]byte("evil"))
	_ = w.Close()
	if _, e := ParseBackup(b.Bytes(), true, 100); e == nil {
		t.Fatal("accepted traversal")
	}
}
func TestRejectMissingFileAndDuplicateNotes(t *testing.T) {
	b := backupFixture()
	b.Notes = append(b.Notes, b.Notes[0])
	raw, _ := json.Marshal(b)
	if _, e := ParseBackup(raw, false, 100); e == nil {
		t.Fatal("accepted duplicate IDs")
	}
	b = backupFixture()
	b.Attachments = []domain.Attachment{{ID: "1123456789abcdef0123456789abcdef", NoteID: b.Notes[0].ID, ArchivePath: "attachments/1123456789abcdef0123456789abcdef", Name: "a.txt", MIME: "text/plain; charset=utf-8", Size: 3}}
	raw, _ = json.Marshal(b)
	if _, e := ParseBackup(raw, false, 100); e == nil {
		t.Fatal("accepted missing attachment")
	}
}
