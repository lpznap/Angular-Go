package mailer

import (
	"bytes"
	"context"
	"crypto/tls"
	"dailyworknotes/internal/config"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
)

type File struct {
	Name, MIME string
	Data       []byte
}
type Message struct {
	To, CC  []string
	Subject string
	HTML    []byte
	Files   []File
	ID      string
}

func Encode(from string, m Message) ([]byte, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fmt.Fprintf(&b, "From: %s\r\nTo: %s\r\n", from, strings.Join(m.To, ", "))
	if len(m.CC) > 0 {
		fmt.Fprintf(&b, "Cc: %s\r\n", strings.Join(m.CC, ", "))
	}
	fmt.Fprintf(&b, "Subject: %s\r\nDate: %s\r\nMessage-ID: <%s@daily-work-notes>\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=%q\r\n\r\n", mime.QEncoding.Encode("utf-8", m.Subject), time.Now().Format(time.RFC1123Z), m.ID, w.Boundary())
	parts := append([]File{{MIME: "text/html; charset=utf-8", Data: m.HTML}}, m.Files...)
	for _, f := range parts {
		h := textproto.MIMEHeader{}
		h.Set("Content-Type", f.MIME)
		h.Set("Content-Transfer-Encoding", "base64")
		if f.Name != "" {
			h.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": f.Name}))
		}
		p, e := w.CreatePart(h)
		if e != nil {
			return nil, e
		}
		encoded := base64.StdEncoding.EncodeToString(f.Data)
		for len(encoded) > 76 {
			fmt.Fprint(p, encoded[:76]+"\r\n")
			encoded = encoded[76:]
		}
		fmt.Fprint(p, encoded+"\r\n")
	}
	if e := w.Close(); e != nil {
		return nil, e
	}
	return b.Bytes(), nil
}
func Send(ctx context.Context, c config.Config, m Message) error {
	b, e := Encode(c.SMTPFrom, m)
	if e != nil {
		return e
	}
	conn, e := (&net.Dialer{Timeout: 15 * time.Second}).DialContext(ctx, "tcp", net.JoinHostPort(c.SMTPHost, c.SMTPPort))
	if e != nil {
		return e
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(45 * time.Second))
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	client, e := smtp.NewClient(conn, c.SMTPHost)
	if e != nil {
		return e
	}
	defer client.Close()
	hasTLS, _ := client.Extension("STARTTLS")
	if hasTLS {
		if e = client.StartTLS(&tls.Config{ServerName: c.SMTPHost, MinVersion: tls.VersionTLS12}); e != nil {
			return e
		}
	} else if c.SMTPTLS || c.SMTPUser != "" {
		return fmt.Errorf("SMTP server does not offer required TLS")
	}
	if c.SMTPUser != "" {
		if e = client.Auth(smtp.PlainAuth("", c.SMTPUser, c.SMTPPassword, c.SMTPHost)); e != nil {
			return e
		}
	}
	if e = client.Mail(c.SMTPFrom); e != nil {
		return e
	}
	for _, r := range append(append([]string{}, m.To...), m.CC...) {
		if e = client.Rcpt(r); e != nil {
			return e
		}
	}
	w, e := client.Data()
	if e != nil {
		return e
	}
	if _, e = w.Write(b); e != nil {
		return e
	}
	if e = w.Close(); e != nil {
		return e
	}
	_ = client.Quit()
	return nil
}
