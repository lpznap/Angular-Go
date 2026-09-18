package httpapi

import (
	"bytes"
	"context"
	"dailyworknotes/internal/config"
	"dailyworknotes/internal/domain"
	"dailyworknotes/internal/repository"
	"dailyworknotes/internal/service"
	"dailyworknotes/internal/storage"
	"encoding/json"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type client struct {
	t            *testing.T
	app          *fiber.App
	cookie, csrf string
}

func (c *client) request(method, path string, body any) (int, []byte) {
	c.t.Helper()
	var b io.Reader
	if body != nil {
		raw, e := json.Marshal(body)
		if e != nil {
			c.t.Fatal(e)
		}
		b = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, b)
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}
func (c *client) do(req *http.Request) (int, []byte) {
	c.t.Helper()
	req.Header.Set("Origin", "http://localhost:4200")
	req.Header.Set("X-CSRF-Token", c.csrf)
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}
	resp, e := c.app.Test(req, fiber.TestConfig{Timeout: 10000})
	if e != nil {
		c.t.Fatal(e)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	for _, v := range resp.Cookies() {
		if v.Name == "daily_session" {
			c.cookie = v.Name + "=" + v.Value
		}
	}
	return resp.StatusCode, b
}
func (c *client) register(email string) {
	code, b := c.request("POST", "/api/auth/register", map[string]string{"email": email, "password": "test-password-12345"})
	if code != 200 {
		c.t.Fatalf("register %d %s", code, b)
	}
	var u struct {
		CSRF string `json:"csrf"`
	}
	_ = json.Unmarshal(b, &u)
	c.csrf = u.CSRF
}
func (c *client) upload(path, name string, data []byte, mode string) (int, []byte) {
	c.t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	f, _ := w.CreateFormFile("file", name)
	_, _ = f.Write(data)
	if mode != "" {
		_ = w.WriteField("mode", mode)
	}
	_ = w.Close()
	req := httptest.NewRequest("POST", path, &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return c.do(req)
}
func testAPI(t *testing.T) *API {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set: real PostgreSQL integration test")
	}
	ctx := context.Background()
	admin, e := pgxpool.New(ctx, url)
	if e != nil {
		t.Fatal(e)
	}
	schema := "test_" + service.ID()
	if _, e = admin.Exec(ctx, "CREATE SCHEMA "+schema); e != nil {
		t.Fatal(e)
	}
	pc, e := pgxpool.ParseConfig(url)
	if e != nil {
		t.Fatal(e)
	}
	pc.ConnConfig.RuntimeParams["search_path"] = schema
	pool, e := pgxpool.NewWithConfig(ctx, pc)
	if e != nil {
		t.Fatal(e)
	}
	sql, e := os.ReadFile(filepath.Join("..", "..", "db", "migrations", "000001_initial.up.sql"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = pool.Exec(ctx, string(sql)); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { pool.Close(); _, _ = admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	r := repository.New(pool)
	cfg := config.Load()
	cfg.Origin = "http://localhost:4200"
	cfg.SMTPHost = "127.0.0.1"
	cfg.SMTPPort = "1"
	return &API{Config: cfg, Repo: r, Notes: &service.Notes{Repo: r}, Store: storage.Store{Root: t.TempDir(), Limit: 10485760}}
}
func TestAuthenticatedJourney(t *testing.T) {
	a := testAPI(t)
	app := a.App()
	c := &client{t: t, app: app}
	code, _ := c.request("GET", "/api/notes", nil)
	if code != 401 {
		t.Fatal("anonymous read allowed")
	}
	c.register("one@example.com")
	n := domain.Note{WorkDate: "2026-09-18", Title: "Design งานวันนี้", Description: "<p>Progress</p>", Project: "Website", Tags: []string{"design"}, Status: "progress", Priority: "high", Minutes: 65, Tasks: []domain.Task{{Text: "Review", Done: true}}}
	inputJSON, _ := json.Marshal(n)
	var input map[string]any
	_ = json.Unmarshal(inputJSON, &input)
	input["createdAt"], input["updatedAt"] = "", ""
	code, b := c.request("POST", "/api/notes", input)
	if code != 201 {
		t.Fatalf("create %d %s", code, b)
	}
	_ = json.Unmarshal(b, &n)
	old := n
	n.Title = "Updated"
	code, b = c.request("PUT", "/api/notes/"+n.ID, n)
	if code != 200 {
		t.Fatalf("update %d %s", code, b)
	}
	_ = json.Unmarshal(b, &n)
	code, _ = c.request("PUT", "/api/notes/"+n.ID, old)
	if code != 409 {
		t.Fatal("stale write did not conflict")
	}
	code, b = c.request("GET", "/api/notes?from=2026-09-18&to=2026-09-18&q=design&size=1", nil)
	if code != 200 || !bytes.Contains(b, []byte(n.ID)) {
		t.Fatalf("search %d %s", code, b)
	}
	_, b = c.request("GET", "/api/notes?from=2026-09-19", nil)
	if bytes.Contains(b, []byte(n.ID)) {
		t.Fatal("date boundary incorrect")
	}
	_, b = c.request("GET", "/api/notes?size=1&page=2", nil)
	if bytes.Contains(b, []byte(n.ID)) {
		t.Fatal("pagination incorrect")
	}
	other := &client{t: t, app: app}
	other.register("two@example.com")
	code, _ = other.request("GET", "/api/notes/"+n.ID, nil)
	if code != 404 {
		t.Fatal("cross-user read allowed")
	}
	code, _ = other.request("PUT", "/api/notes/"+n.ID, n)
	if code != 404 {
		t.Fatal("cross-user write allowed")
	}
	code, b = c.upload("/api/notes/"+n.ID+"/attachments", "งาน.txt", []byte("hello งาน"), "")
	if code != 201 {
		t.Fatalf("upload %d %s", code, b)
	}
	var attachment domain.Attachment
	_ = json.Unmarshal(b, &attachment)
	code, b = c.request("GET", "/api/attachments/"+attachment.ID, nil)
	if code != 200 || string(b) != "hello งาน" {
		t.Fatalf("download %d %s", code, b)
	}
	code, _ = other.request("GET", "/api/attachments/"+attachment.ID, nil)
	if code != 404 {
		t.Fatal("cross-user attachment read allowed")
	}
	code, _ = c.upload("/api/notes/"+n.ID+"/attachments", "fake.png", []byte("not an image"), "")
	if code != 400 {
		t.Fatal("invalid file allowed")
	}
	savedCSRF := c.csrf
	c.csrf = "wrong"
	code, _ = c.request("DELETE", "/api/notes/"+n.ID+"?version=2", nil)
	if code != 403 {
		t.Fatal("CSRF bypass")
	}
	c.csrf = savedCSRF
	code, backup := c.request("POST", "/api/reports/json", selection{IDs: []string{n.ID}})
	if code != 200 {
		t.Fatalf("export %d %s", code, backup)
	}
	code, b = c.upload("/api/imports/preview", "backup.json", backup, "")
	if code != 200 || !bytes.Contains(b, []byte(n.ID)) {
		t.Fatalf("preview %d %s", code, b)
	}
	code, b = c.upload("/api/imports", "backup.json", backup, "skip")
	if code != 200 || !bytes.Contains(b, []byte(`"skipped":1`)) {
		t.Fatalf("skip %d %s", code, b)
	}
	// Exported attachment backup is constructed without requiring a PDF service in this transaction test.
	var pack service.Backup
	_ = json.Unmarshal(backup, &pack)
	attachment.ArchivePath = "attachments/" + attachment.ID
	pack.Attachments = []domain.Attachment{attachment}
	archive, e := service.ZIP(pack, []byte("pdf fixture"), map[string][]byte{attachment.ArchivePath: []byte("hello งาน")})
	if e != nil {
		t.Fatal(e)
	}
	code, b = c.upload("/api/imports", "backup.zip", archive, "replace")
	if code != 200 || !bytes.Contains(b, []byte(`"attachments":1`)) {
		t.Fatalf("replace %d %s", code, b)
	}
	_, b = c.request("GET", "/api/notes/"+n.ID+"/attachments", nil)
	var attachments []domain.Attachment
	_ = json.Unmarshal(b, &attachments)
	if len(attachments) != 1 {
		t.Fatal("restore attachments missing")
	}
	_, b = c.request("GET", "/api/attachments/"+attachments[0].ID, nil)
	if string(b) != "hello งาน" {
		t.Fatal("restored content mismatch")
	}
	send := emailRequest{selection: selection{IDs: []string{n.ID}}, To: []string{"recipient@example.com"}, Subject: "Daily update", Key: service.ID()}
	code, b = c.request("POST", "/api/email/send", send)
	if code != 200 || !bytes.Contains(b, []byte(`"outcome":"failed"`)) {
		t.Fatalf("SMTP failure %d %s", code, b)
	}
	_, b = c.request("POST", "/api/email/send", send)
	if !bytes.Contains(b, []byte(`"duplicate":true`)) {
		t.Fatal("duplicate send not suppressed")
	}
	send.Subject = "Changed"
	code, _ = c.request("POST", "/api/email/send", send)
	if code != 409 {
		t.Fatal("idempotency payload mismatch allowed")
	}
	code, b = c.upload("/api/imports", "unsafe.json", []byte(`{"schemaVersion":2}`), "replace")
	if code != 400 {
		t.Fatalf("unsafe restore %d %s", code, b)
	}
	_, b = c.request("GET", "/api/notes/"+n.ID, nil)
	_ = json.Unmarshal(b, &n)
	code, b = c.request("DELETE", "/api/notes/"+n.ID+"?version="+jsonNumber(n.Version), nil)
	if code != 204 {
		t.Fatalf("delete %d %s", code, b)
	}
	var queued int
	_ = a.Repo.Pool.QueryRow(context.Background(), "SELECT count(*) FROM file_deletions").Scan(&queued)
	if queued < 1 {
		t.Fatal("file cleanup not queued")
	}
	code, _ = c.request("POST", "/api/auth/logout", map[string]string{})
	if code != 204 {
		t.Fatal("logout failed")
	}
	code, _ = c.request("GET", "/api/me", nil)
	if code != 401 {
		t.Fatal("session not revoked")
	}
}
func TestRestoreRollbackAndCrashFileReconciliation(t *testing.T) {
	a := testAPI(t)
	c := &client{t: t, app: a.App()}
	c.register("restore@example.com")
	n := domain.Note{ID: service.ID(), WorkDate: "2026-09-18", Title: "Restored", Status: "todo", Priority: "low", Tasks: []domain.Task{}, Tags: []string{}}
	attachment := domain.Attachment{ID: service.ID(), NoteID: n.ID, Name: "work.txt", MIME: "text/plain; charset=utf-8", Size: 4}
	attachment.ArchivePath = "attachments/" + attachment.ID
	backup, err := service.ZIP(service.Backup{SchemaVersion: 1, Notes: []domain.Note{n}, Attachments: []domain.Attachment{attachment}}, nil, map[string][]byte{attachment.ArchivePath: []byte("work")})
	if err != nil {
		t.Fatal(err)
	}
	root := a.Store.Root
	a.Store.Root = filepath.Join(root, "missing-directory")
	code, _ := c.upload("/api/imports", "backup.zip", backup, "replace")
	if code != 500 {
		t.Fatalf("expected storage failure, got %d", code)
	}
	var count int
	if err = a.Repo.Pool.QueryRow(context.Background(), "SELECT count(*) FROM notes").Scan(&count); err != nil || count != 0 {
		t.Fatalf("failed restore left notes: %d %v", count, err)
	}
	a.Store.Root = root
	oldKey, freshKey := service.ID(), service.ID()
	if err = a.Store.Put(oldKey, []byte("old orphan")); err != nil {
		t.Fatal(err)
	}
	if err = a.Store.Put(freshKey, []byte("in flight")); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	if err = os.Chtimes(a.Store.Path(oldKey), old, old); err != nil {
		t.Fatal(err)
	}
	a.ReconcileFiles(context.Background())
	if _, err = os.Stat(a.Store.Path(oldKey)); !os.IsNotExist(err) {
		t.Fatal("old orphan was not removed")
	}
	if _, err = os.Stat(a.Store.Path(freshKey)); err != nil {
		t.Fatal("in-flight file removed")
	}
}
func jsonNumber(v int32) string { b, _ := json.Marshal(v); return string(b) }
func TestFiberHealthAndErrors(t *testing.T) {
	a := &API{Config: config.Config{Origin: "http://localhost:4200"}}
	app := a.App()
	c := &client{t: t, app: app}
	code, b := c.request("GET", "/health", nil)
	if code != 200 || !strings.Contains(string(b), "ok") {
		t.Fatal("health failed")
	}
	code, b = c.request("POST", "/api/auth/login", map[string]string{})
	if code != 500 {
		t.Logf("nil repository recovered response: %d %s", code, b)
	}
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{}`))
	resp, e := app.Test(req)
	if e != nil {
		t.Fatal(e)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Fatal("missing origin accepted")
	}
}
