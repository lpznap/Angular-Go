package storage

import (
	"archive/zip"
	"bytes"
	"dailyworknotes/internal/domain"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type Store struct {
	Root  string
	Limit int64
}

func (s Store) Path(key string) string {
	if len(key) != 32 || strings.ContainsAny(key, "/\\.") {
		panic("invalid internal storage key")
	}
	return filepath.Join(s.Root, key)
}
func (s Store) Put(key string, b []byte) error {
	f, e := os.OpenFile(s.Path(key), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	if _, e = f.Write(b); e == nil {
		e = f.Sync()
	}
	closeErr := f.Close()
	if e == nil {
		e = closeErr
	}
	if e != nil {
		_ = s.Remove(key)
	}
	return e
}
func (s Store) Read(key string) ([]byte, error) { return os.ReadFile(s.Path(key)) }
func (s Store) Remove(key string) error {
	e := os.Remove(s.Path(key))
	if errors.Is(e, os.ErrNotExist) {
		return nil
	}
	return e
}
func Validate(name string, b []byte, limit int64) (string, error) {
	bad := domain.ValidationError{Message: "Unsupported file contents, filename, or size"}
	if len(b) == 0 || int64(len(b)) > limit || len(name) > 240 || strings.ContainsAny(name, "/\\\r\n\x00") {
		return "", bad
	}
	ext := strings.ToLower(filepath.Ext(name))
	detected := http.DetectContentType(b)
	switch ext {
	case ".png":
		if detected == "image/png" {
			return detected, nil
		}
	case ".jpg", ".jpeg":
		if detected == "image/jpeg" {
			return detected, nil
		}
	case ".gif":
		if detected == "image/gif" {
			return detected, nil
		}
	case ".webp":
		if detected == "image/webp" {
			return detected, nil
		}
	case ".pdf":
		if bytes.HasPrefix(b, []byte("%PDF-")) {
			return "application/pdf", nil
		}
	case ".txt":
		if utf8.Valid(b) && !bytes.ContainsRune(b, 0) {
			return "text/plain; charset=utf-8", nil
		}
	case ".docx", ".xlsx":
		z, e := zip.NewReader(bytes.NewReader(b), int64(len(b)))
		if e != nil {
			return "", bad
		}
		var total uint64
		found, types := false, false
		for _, f := range z.File {
			total += f.UncompressedSize64
			if total > 100<<20 {
				return "", bad
			}
			if f.Name == "[Content_Types].xml" {
				types = true
			}
			if (ext == ".docx" && f.Name == "word/document.xml") || (ext == ".xlsx" && f.Name == "xl/workbook.xml") {
				found = true
			}
		}
		if found && types {
			if ext == ".docx" {
				return "application/vnd.openxmlformats-officedocument.wordprocessingml.document", nil
			}
			return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", nil
		}
	}
	return "", bad
}
func ReadLimited(r io.Reader, n int64) ([]byte, error) {
	b, e := io.ReadAll(io.LimitReader(r, n+1))
	if int64(len(b)) > n {
		return nil, domain.ValidationError{Message: "File exceeds size limit"}
	}
	return b, e
}
