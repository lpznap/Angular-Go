package config

import (
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL, Address, Origin, Storage, PDFURL, SMTPHost, SMTPPort, SMTPUser, SMTPPassword, SMTPFrom string
	Secure, SMTPTLS                                                                                     bool
	MaxFile                                                                                             int64
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func Load() Config {
	max, _ := strconv.ParseInt(env("MAX_FILE_BYTES", "10485760"), 10, 64)
	if max < 1 || max > 52428800 {
		max = 10485760
	}
	return Config{DatabaseURL: env("DATABASE_URL", "postgres://daily:daily_local@localhost:5432/daily?sslmode=disable"), Address: env("ADDRESS", ":8080"), Origin: env("APP_ORIGIN", "http://localhost:4200"), Storage: env("STORAGE_PATH", "./data"), PDFURL: env("PDF_URL", "http://localhost:3000"), SMTPHost: env("SMTP_HOST", "localhost"), SMTPPort: env("SMTP_PORT", "1025"), SMTPUser: os.Getenv("SMTP_USER"), SMTPPassword: os.Getenv("SMTP_PASSWORD"), SMTPFrom: env("SMTP_FROM", "notes@daily.local"), Secure: env("COOKIE_SECURE", "false") == "true", SMTPTLS: env("SMTP_REQUIRE_TLS", "false") == "true", MaxFile: max}
}
