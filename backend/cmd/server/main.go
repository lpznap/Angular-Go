package main

import (
	"context"
	"dailyworknotes/internal/config"
	"dailyworknotes/internal/httpapi"
	"dailyworknotes/internal/repository"
	"dailyworknotes/internal/service"
	"dailyworknotes/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pc, e := pgxpool.ParseConfig(cfg.DatabaseURL)
	if e != nil {
		slog.Error("invalid database configuration")
		os.Exit(1)
	}
	pc.MaxConns = 10
	pc.ConnConfig.ConnectTimeout = 5 * time.Second
	pc.ConnConfig.RuntimeParams["statement_timeout"] = "15000"
	pool, e := pgxpool.NewWithConfig(ctx, pc)
	if e != nil {
		slog.Error("database pool failed")
		os.Exit(1)
	}
	defer pool.Close()
	if e = pool.Ping(ctx); e != nil {
		slog.Error("database unavailable")
		os.Exit(1)
	}
	if e = os.MkdirAll(cfg.Storage, 0700); e != nil {
		slog.Error("storage unavailable")
		os.Exit(1)
	}
	repo := repository.New(pool)
	api := &httpapi.API{Config: cfg, Repo: repo, Notes: &service.Notes{Repo: repo}, Store: storage.Store{Root: cfg.Storage, Limit: cfg.MaxFile}}
	// A previous process may have exited between SMTP transmission and recording its result.
	_, _ = pool.Exec(ctx, "UPDATE email_attempts SET outcome='unknown',detail='Process stopped before recording SMTP outcome. Verify before retrying.' WHERE outcome='sending'")
	app := api.App()
	go api.Cleanup(ctx)
	go func() {
		<-ctx.Done()
		if e := app.ShutdownWithTimeout(15 * time.Second); e != nil {
			slog.Error("shutdown timed out")
		}
	}()
	if e = app.Listen(cfg.Address); e != nil && ctx.Err() == nil {
		slog.Error("server stopped", "error", e.Error())
		os.Exit(1)
	}
}
