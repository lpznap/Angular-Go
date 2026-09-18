package httpapi

import (
	"dailyworknotes/internal/service"
	"errors"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
	"net/mail"
	"strings"
	"time"
)

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *API) register(c fiber.Ctx) error {
	var v credentials
	if e := c.Bind().JSON(&v); e != nil {
		return fiber.NewError(400, "Invalid JSON")
	}
	v.Email = strings.ToLower(strings.TrimSpace(v.Email))
	addr, e := mail.ParseAddress(v.Email)
	if e != nil || addr.Address != v.Email || len(v.Email) > 254 || len(v.Password) < 12 || len(v.Password) > 72 {
		return fiber.NewError(400, "Use a valid email and a password of 12–72 bytes")
	}
	p, e := bcrypt.GenerateFromPassword([]byte(v.Password), 12)
	if e != nil {
		return e
	}
	id := service.ID()
	_, e = a.Repo.Pool.Exec(c.Context(), "INSERT INTO users(id,email,password_hash) VALUES($1,$2,$3)", id, v.Email, string(p))
	if e != nil {
		return e
	}
	return a.newSession(c, id)
}

var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), 12)

func (a *API) login(c fiber.Ctx) error {
	var v credentials
	if e := c.Bind().JSON(&v); e != nil {
		return fiber.NewError(400, "Invalid JSON")
	}
	var id, p string
	e := a.Repo.Pool.QueryRow(c.Context(), "SELECT id,password_hash FROM users WHERE email=$1", strings.ToLower(strings.TrimSpace(v.Email))).Scan(&id, &p)
	if e != nil {
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(v.Password))
		return fiber.NewError(401, "Incorrect email or password")
	}
	if bcrypt.CompareHashAndPassword([]byte(p), []byte(v.Password)) != nil {
		return fiber.NewError(401, "Incorrect email or password")
	}
	return a.newSession(c, id)
}
func (a *API) newSession(c fiber.Ctx, id string) error {
	token, csrf := service.ID()+service.ID(), service.ID()
	expires := time.Now().Add(7 * 24 * time.Hour)
	_, e := a.Repo.Pool.Exec(c.Context(), "INSERT INTO sessions(token_hash,user_id,csrf,expires_at) VALUES($1,$2,$3,$4)", hash(token), id, csrf, expires)
	if e != nil {
		return e
	}
	c.Cookie(&fiber.Cookie{Name: "daily_session", Value: token, HTTPOnly: true, Secure: a.Config.Secure, SameSite: "Strict", Path: "/", Expires: expires})
	c.Locals("user", id)
	c.Locals("csrf", csrf)
	return a.me(c)
}
func (a *API) me(c fiber.Ctx) error {
	var email, tz string
	if e := a.Repo.Pool.QueryRow(c.Context(), "SELECT email,timezone FROM users WHERE id=$1", user(c)).Scan(&email, &tz); e != nil {
		return e
	}
	return c.JSON(fiber.Map{"id": user(c), "email": email, "timezone": tz, "csrf": c.Locals("csrf"), "maxFileBytes": a.Config.MaxFile})
}
func (a *API) logout(c fiber.Ctx) error {
	_, e := a.Repo.Pool.Exec(c.Context(), "DELETE FROM sessions WHERE token_hash=$1", hash(c.Cookies("daily_session")))
	if e != nil {
		return e
	}
	c.Cookie(&fiber.Cookie{Name: "daily_session", Value: "", Path: "/", HTTPOnly: true, Secure: a.Config.Secure, SameSite: "Strict", Expires: time.Now().Add(-time.Hour)})
	return c.SendStatus(204)
}
func (a *API) settings(c fiber.Ctx) error {
	var v struct {
		Timezone string `json:"timezone"`
	}
	if e := c.Bind().JSON(&v); e != nil {
		return fiber.NewError(400, "Invalid JSON")
	}
	if _, e := time.LoadLocation(v.Timezone); e != nil {
		return fiber.NewError(400, "Unknown IANA timezone")
	}
	_, e := a.Repo.Pool.Exec(c.Context(), "UPDATE users SET timezone=$1 WHERE id=$2", v.Timezone, user(c))
	if e != nil {
		return e
	}
	return a.me(c)
}
func (a *API) projects(c fiber.Ctx) error {
	rows, e := a.Repo.Pool.Query(c.Context(), "SELECT id,name,color FROM projects WHERE user_id=$1 ORDER BY name", user(c))
	if e != nil {
		return e
	}
	defer rows.Close()
	out := []fiber.Map{}
	for rows.Next() {
		var id, name, color string
		if e = rows.Scan(&id, &name, &color); e != nil {
			return e
		}
		out = append(out, fiber.Map{"id": id, "name": name, "color": color})
	}
	if e = rows.Err(); e != nil {
		return e
	}
	return c.JSON(out)
}
func (a *API) createProject(c fiber.Ctx) error {
	var v struct {
		Name string `json:"name"`
	}
	if e := c.Bind().JSON(&v); e != nil {
		return fiber.NewError(400, "Invalid JSON")
	}
	v.Name = strings.TrimSpace(v.Name)
	if len(v.Name) < 1 || len(v.Name) > 200 {
		return fiber.NewError(400, "Project name must be 1–200 bytes")
	}
	_, e := a.Repo.Pool.Exec(c.Context(), "INSERT INTO projects(id,user_id,name) VALUES($1,$2,$3)", service.ID(), user(c), v.Name)
	if e != nil {
		return e
	}
	return a.projects(c)
}
func (a *API) deleteProject(c fiber.Ctx) error {
	_, e := a.Repo.Pool.Exec(c.Context(), "DELETE FROM projects WHERE id=$1 AND user_id=$2", c.Params("id"), user(c))
	if e != nil {
		return e
	}
	return c.SendStatus(204)
}
