package mailer

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestMIME(t *testing.T) {
	b, e := Encode("notes@example.com", Message{To: []string{"to@example.com"}, CC: []string{"cc@example.com"}, Subject: "งานวันนี้", HTML: []byte("<p>งาน</p>"), ID: "abc", Files: []File{{Name: "test.txt", MIME: "text/plain", Data: []byte("hello")}}})
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Contains(b, []byte("Cc: cc@example.com")) || !bytes.Contains(b, []byte(base64.StdEncoding.EncodeToString([]byte("hello")))) || !bytes.Contains(b, []byte("filename=test.txt")) {
		t.Fatalf("invalid MIME: %s", b)
	}
}
