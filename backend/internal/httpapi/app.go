package httpapi

import (
	"context"
	"crypto/sha256"
	"dailyworknotes/internal/config"
	"dailyworknotes/internal/domain"
	"dailyworknotes/internal/repository"
	"dailyworknotes/internal/service"
	"dailyworknotes/internal/storage"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"log/slog"
	"os"
	"strconv"
	"time"
)

type API struct {
	Config config.Config
	Repo   *repository.Repository
	Notes  *service.Notes
	Store  storage.Store
}

func hash(s string) string    { v := sha256.Sum256([]byte(s)); return hex.EncodeToString(v[:]) }
func user(c fiber.Ctx) string { return c.Locals("user").(string) }
func (a *API) App() *fiber.App {
	app := fiber.New(fiber.Config{BodyLimit: 110 << 20, ReadTimeout: 30 * time.Second, WriteTimeout: 120 * time.Second, IdleTimeout: 60 * time.Second, ErrorHandler: func(c fiber.Ctx, e error) error {
		code, message := 500, "Unexpected server error"
		var v domain.ValidationError
		var f *fiber.Error
		var pg *pgconn.PgError
		switch {
		case errors.As(e, &v):
			code = 400
			message = e.Error()
		case errors.Is(e, domain.ErrNotFound), errors.Is(e, pgx.ErrNoRows):
			code = 404
			message = "Resource not found"
		case errors.Is(e, domain.ErrConflict):
			code = 409
			message = e.Error()
		case errors.As(e, &f):
			code = f.Code
			message = f.Message
		case errors.As(e, &pg) && pg.Code == "23505":
			code = 409
			message = "This item already exists"
		}
		if code == 500 {
			slog.Error("request failed", "requestId", requestid.FromContext(c), "errorType", strconv.Itoa(code))
		}
		return c.Status(code).JSON(fiber.Map{"error": fiber.Map{"message": message, "requestId": requestid.FromContext(c)}})
	}})
	app.Use(recover.New(), requestid.New())
	app.Use(func(c fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("Referrer-Policy", "same-origin")
		ctx, cancel := context.WithTimeout(c.Context(), 90*time.Second)
		defer cancel()
		c.SetContext(ctx)
		start := time.Now()
		e := c.Next()
		if e != nil {
			e = app.Config().ErrorHandler(c, e)
		}
		slog.Info("http", "method", c.Method(), "status", c.Response().StatusCode(), "durationMs", time.Since(start).Milliseconds(), "requestId", requestid.FromContext(c))
		return e
	})
	app.Get("/health", func(c fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })
	app.Get("/ready", func(c fiber.Ctx) error {
		if e := a.Repo.Pool.Ping(c.Context()); e != nil {
			return fiber.NewError(503, "Database unavailable")
		}
		return c.JSON(fiber.Map{"status": "ready"})
	})
	app.Get("/openapi.yaml", func(c fiber.Ctx) error {
		path := "./docs/openapi.yaml"
		if _, err := os.Stat(path); err != nil {
			path = "../docs/openapi.yaml"
		}
		return c.SendFile(path)
	})
	app.Get("/docs", func(c fiber.Ctx) error {
		c.Type("html")
		return c.SendString(`<!doctype html><html><head><title>Daily Work Notes API</title><link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.29.0/swagger-ui.css"></head><body><div id="swagger-ui"></div><script src="https://unpkg.com/swagger-ui-dist@5.29.0/swagger-ui-bundle.js"></script><script>SwaggerUIBundle({url:'/openapi.yaml',dom_id:'#swagger-ui'})</script></body></html>`)
	})
	auth := app.Group("/api/auth", limiter.New(limiter.Config{Max: 15, Expiration: time.Minute, LimitReached: func(c fiber.Ctx) error { return fiber.NewError(429, "Too many requests; please wait a minute") }}))
	auth.Post("/register", a.origin, a.register)
	auth.Post("/login", a.origin, a.login)
	api := app.Group("/api", a.session)
	api.Get("/me", a.me)
	api.Put("/settings", a.settings)
	api.Post("/auth/logout", a.logout)
	api.Get("/notes", a.list)
	api.Post("/notes", a.create)
	api.Get("/notes/:id", a.get)
	api.Put("/notes/:id", a.update)
	api.Delete("/notes/:id", a.remove)
	api.Post("/notes/:id/duplicate", a.duplicate)
	api.Get("/projects", a.projects)
	api.Post("/projects", a.createProject)
	api.Delete("/projects/:id", a.deleteProject)
	api.Get("/notes/:id/attachments", a.attachments)
	api.Post("/notes/:id/attachments", a.upload)
	api.Get("/attachments/:id", a.download)
	api.Delete("/attachments/:id", a.deleteAttachment)
	api.Post("/reports/:format", a.export)
	api.Post("/imports/preview", a.previewImport)
	api.Post("/imports", a.importBackup)
	api.Get("/email/history", a.emailHistory)
	api.Post("/email/preview", a.emailPreview)
	api.Post("/email/send", limiter.New(limiter.Config{Max: 10, Expiration: time.Minute, KeyGenerator: func(c fiber.Ctx) string { return user(c) }, LimitReached: func(c fiber.Ctx) error { return fiber.NewError(429, "Too many email requests; please wait a minute") }}), a.emailSend)
	return app
}
func (a *API) origin(c fiber.Ctx) error {
	if c.Get("Origin") != a.Config.Origin {
		return fiber.NewError(403, "Untrusted request origin")
	}
	return c.Next()
}
func (a *API) session(c fiber.Ctx) error {
	token := c.Cookies("daily_session")
	if token == "" {
		return fiber.NewError(401, "Please sign in")
	}
	var uid, csrf string
	e := a.Repo.Pool.QueryRow(c.Context(), "SELECT user_id,csrf FROM sessions WHERE token_hash=$1 AND expires_at>now()", hash(token)).Scan(&uid, &csrf)
	if e != nil {
		if errors.Is(e, pgx.ErrNoRows) {
			return fiber.NewError(401, "Session expired")
		}
		return e
	}
	if c.Method() != "GET" && c.Method() != "HEAD" {
		if c.Get("Origin") != a.Config.Origin || c.Get("X-CSRF-Token") != csrf {
			return fiber.NewError(403, "Invalid CSRF token")
		}
	}
	c.Locals("user", uid)
	c.Locals("csrf", csrf)
	return c.Next()
}

// Input ignores server-managed timestamps rather than parsing client values.
type noteInput struct {
	domain.Note
	CreatedAt json.RawMessage `json:"createdAt"`
	UpdatedAt json.RawMessage `json:"updatedAt"`
}

func (a *API) create(c fiber.Ctx) error {
	var input noteInput
	if e := c.Bind().JSON(&input); e != nil {
		return fiber.NewError(400, "Invalid JSON")
	}
	v, e := a.Notes.Save(c.Context(), user(c), input.Note, true)
	if e != nil {
		return e
	}
	return c.Status(201).JSON(v)
}
func (a *API) update(c fiber.Ctx) error {
	var input noteInput
	if e := c.Bind().JSON(&input); e != nil {
		return fiber.NewError(400, "Invalid JSON")
	}
	n := input.Note
	n.ID = c.Params("id")
	v, e := a.Notes.Save(c.Context(), user(c), n, false)
	if e != nil {
		return e
	}
	return c.JSON(v)
}
func (a *API) get(c fiber.Ctx) error {
	n, e := a.Repo.Get(c.Context(), user(c), c.Params("id"))
	if e != nil {
		return e
	}
	return c.JSON(n)
}
func (a *API) remove(c fiber.Ctx) error {
	v, _ := strconv.Atoi(c.Query("version"))
	if e := a.Notes.Delete(c.Context(), user(c), c.Params("id"), int32(v)); e != nil {
		return e
	}
	return c.SendStatus(204)
}
func (a *API) duplicate(c fiber.Ctx) error {
	n, e := a.Repo.Get(c.Context(), user(c), c.Params("id"))
	if e != nil {
		return e
	}
	n.Title += " (copy)"
	n, e = a.Notes.Save(c.Context(), user(c), n, true)
	if e != nil {
		return e
	}
	return c.Status(201).JSON(n)
}
func (a *API) list(c fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	size, _ := strconv.Atoi(c.Query("size", "20"))
	if page < 1 || page > 100000 || size < 1 || size > 100 {
		return fiber.NewError(400, "Invalid pagination")
	}
	f := domain.Filter{Search: c.Query("q"), From: c.Query("from"), To: c.Query("to"), Project: c.Query("project"), Status: c.Query("status"), Priority: c.Query("priority"), Tag: c.Query("tag"), Sort: c.Query("sort"), Page: 1, Size: int32(size + 1)}
	for _, d := range []string{f.From, f.To} {
		if d != "" {
			if _, e := time.Parse("2006-01-02", d); e != nil {
				return fiber.NewError(400, "Invalid date filter")
			}
		}
	}
	if f.From != "" && f.To != "" && f.From > f.To {
		return fiber.NewError(400, "Date range is reversed")
	}
	// Fetch one extra row with the exact requested offset.
	f.Offset = int32((page - 1) * size)
	rows, e := a.Repo.List(c.Context(), user(c), f)
	if e != nil {
		return e
	}
	hasMore := len(rows) > size
	if hasMore {
		rows = rows[:size]
	}
	return c.JSON(fiber.Map{"items": rows, "page": page, "size": size, "hasMore": hasMore})
}
