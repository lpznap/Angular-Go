package httpapi

import (
	"context"
	"dailyworknotes/internal/mailer"
	"dailyworknotes/internal/report"
	"dailyworknotes/internal/service"
	"encoding/json"
	"github.com/gofiber/fiber/v3"
	"net/mail"
	"strings"
	"time"
)

type emailRequest struct {
	selection
	DateTo  string   `json:"dateTo"`
	To      []string `json:"to"`
	CC      []string `json:"cc"`
	Subject string   `json:"subject"`
	PDF     bool     `json:"pdf"`
	Key     string   `json:"key"`
}

func (a *API) emailPreview(c fiber.Ctx) error {
	var v selection
	if e := c.Bind().JSON(&v); e != nil {
		return fiber.NewError(400, "Invalid JSON")
	}
	notes, e := a.selected(c, v)
	if e != nil {
		return e
	}
	b, e := report.HTML(notes)
	if e != nil {
		return e
	}
	return c.JSON(fiber.Map{"html": string(b)})
}
func (a *API) emailSend(c fiber.Ctx) error {
	var v emailRequest
	if e := c.Bind().JSON(&v); e != nil {
		return fiber.NewError(400, "Invalid JSON")
	}
	v.selection.To = v.DateTo
	if len(v.To) == 0 || len(v.To)+len(v.CC) > 20 || len(v.Subject) == 0 || len(v.Subject) > 300 || strings.ContainsAny(v.Subject, "\r\n") || len(v.Key) < 16 || len(v.Key) > 100 {
		return fiber.NewError(400, "Provide recipients, subject, and an idempotency key")
	}
	for _, s := range append(append([]string{}, v.To...), v.CC...) {
		address, e := mail.ParseAddress(s)
		if e != nil || address.Address != s || strings.ContainsAny(s, "\r\n") {
			return fiber.NewError(400, "Invalid recipient address")
		}
	}
	notes, e := a.selected(c, v.selection)
	if e != nil {
		return e
	}
	files, e := a.selectedFiles(c, notes, v.AttachmentIDs)
	if e != nil {
		return e
	}
	html, e := report.HTML(notes)
	if e != nil {
		return e
	}
	message := mailer.Message{To: v.To, CC: v.CC, Subject: v.Subject, HTML: html, ID: service.ID()}
	for _, f := range files {
		b, e := a.Store.Read(f.StorageKey)
		if e != nil {
			return e
		}
		message.Files = append(message.Files, mailer.File{Name: f.Name, MIME: f.MIME, Data: b})
	}
	if v.PDF {
		b, e := report.PDF(c.Context(), a.Config.PDFURL, notes)
		if e != nil {
			return e
		}
		message.Files = append(message.Files, mailer.File{Name: "report.pdf", MIME: "application/pdf", Data: b})
	}
	raw, _ := json.Marshal(v)
	fingerprint := hash(string(raw))
	recipients, _ := json.Marshal(fiber.Map{"to": v.To, "cc": v.CC})
	tag, e := a.Repo.Pool.Exec(c.Context(), "INSERT INTO email_attempts(id,user_id,idempotency_key,fingerprint,recipients,subject,outcome) VALUES($1,$2,$3,$4,$5,$6,'sending') ON CONFLICT(user_id,idempotency_key) DO NOTHING", message.ID, user(c), v.Key, fingerprint, recipients, v.Subject)
	if e != nil {
		return e
	}
	if tag.RowsAffected() == 0 {
		var old, outcome string
		e = a.Repo.Pool.QueryRow(c.Context(), "SELECT fingerprint,outcome FROM email_attempts WHERE user_id=$1 AND idempotency_key=$2", user(c), v.Key).Scan(&old, &outcome)
		if e != nil {
			return e
		}
		if old != fingerprint {
			return fiber.NewError(409, "Idempotency key was used for different content")
		}
		return c.JSON(fiber.Map{"outcome": outcome, "duplicate": true})
	}
	outcome, detail := "accepted", "SMTP accepted the message. Delivery is not confirmed."
	if e = mailer.Send(c.Context(), a.Config, message); e != nil {
		outcome = "failed"
		detail = "SMTP did not confirm acceptance. Check the provider before intentionally retrying; a network failure can leave delivery uncertain."
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := a.Repo.Pool.Exec(ctx, "UPDATE email_attempts SET outcome=$1,detail=$2 WHERE id=$3", outcome, detail, message.ID)
	if err != nil {
		return fiber.NewError(503, "Send outcome could not be recorded. Do not automatically retry.")
	}
	return c.JSON(fiber.Map{"outcome": outcome, "detail": detail, "duplicate": false})
}
func (a *API) emailHistory(c fiber.Ctx) error {
	rows, e := a.Repo.Pool.Query(c.Context(), "SELECT id,recipients,subject,outcome,detail,created_at FROM email_attempts WHERE user_id=$1 ORDER BY created_at DESC LIMIT 100", user(c))
	if e != nil {
		return e
	}
	defer rows.Close()
	out := []fiber.Map{}
	for rows.Next() {
		var id, subject, outcome, detail string
		var recipients json.RawMessage
		var at time.Time
		if e = rows.Scan(&id, &recipients, &subject, &outcome, &detail, &at); e != nil {
			return e
		}
		out = append(out, fiber.Map{"id": id, "recipients": recipients, "subject": subject, "outcome": outcome, "detail": detail, "createdAt": at})
	}
	if e = rows.Err(); e != nil {
		return e
	}
	return c.JSON(out)
}
