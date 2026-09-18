# Daily Work Notes

A private workspace for daily notes, tasks, projects, attachments, email summaries,
reports, and backups. Angular + Material/CDK frontend, Go Fiber API, PostgreSQL,
filesystem attachments, Mailpit SMTP, and Gotenberg Chromium PDF rendering.

## Start the complete environment

Install Docker Desktop with Linux containers on Windows/macOS, or Docker Engine
with the Compose plugin on Linux. No host Node, Go, or PostgreSQL installation is
needed for the container setup.

Windows PowerShell:

```powershell
Copy-Item .env.example .env
docker compose up --build -d --wait
```

macOS / Linux:

```sh
cp .env.example .env
docker compose up --build -d --wait
```

| Service | Address |
|---|---|
| Frontend | http://localhost:4200 |
| Backend liveness | http://localhost:8080/health |
| Backend readiness | http://localhost:8080/ready |
| Interactive OpenAPI | http://localhost:8080/docs |
| OpenAPI source | http://localhost:8080/openapi.yaml |
| Mailpit inbox | http://localhost:8025 |
| Local SMTP | localhost:1025 |

Open the frontend, choose **New here? Create an account**, and use an email address
and a password of at least 12 characters (maximum 72 UTF-8 bytes). There is no
default account or password. Subsequent visits use Sign in. Each account owns a
separate workspace. Registration is open in this local application.

The migration service runs before the backend. PostgreSQL and attachments use
named volumes and survive container recreation. All published ports bind to
loopback. The frontend proxies API calls so session cookies remain same-origin.
Always use `localhost:4200`, matching `APP_ORIGIN`, rather than mixing localhost
and 127.0.0.1 in the browser.

```sh
docker compose logs -f backend
docker compose stop
docker compose start
docker compose down
```

`down` preserves data. **`docker compose down -v` permanently deletes the database
and attachment volumes.** Export a ZIP backup before intentionally resetting data.

## Walkthrough

1. **Overview:** See today's notes, completed tasks, time, and an inclusive
   Monday–Sunday weekly summary using the account's display timezone.
2. **My notes:** Create a note with date, project/customer, status, priority, tags,
   hours/minutes, rich description, tasks, blockers, and next steps. The visual rich
   editor provides bold, italic, headings, lists, and quotes. Save manually or leave
   autosave enabled. A one-second debounce persists edits and shows save status.
3. **Drafts and conflicts:** Unsaved content is stored in this browser under your
   account. Failed saves keep the draft. A stale version stops autosave and offers
   server reload or a draft download. Review a recovered draft before saving.
4. **Attachments:** Save the note first. Drop one file or use the picker. Upload
   progress, validation errors, previews, downloads, and removal are supported.
   Allowed: PNG/JPEG/GIF/WebP, PDF, UTF-8 TXT, DOCX, XLSX; default 10 MiB each.
5. **Find work:** Search title, description, project, and tags. Combine inclusive
   date filters with project, status, priority, and tag filters. Sort and page
   results. Select notes for bulk email/export. Calendar uses a Monday-first month
   grid with month navigation and paginated results.
6. **Projects:** Create reusable project suggestions. Removing a suggestion keeps
   historical project text on notes. Duplicating a note copies its content; files
   remain attached to the original.
7. **Email summaries:** Start from one/selected notes or choose today/this week.
   Enter To and optional CC, edit the subject, refresh the HTML preview, and choose
   attachments/PDF. Send through SMTP and inspect the actual message in Mailpit.
   History records recipients, subject, time, and outcome. **Accepted** means the
   SMTP server accepted the message; it does not mean delivery is confirmed.
   Repeated requests with the same key are suppressed. Use the explicit new-send
   action to retry after checking the recorded outcome.
8. **Reports & backups:** Export one/selected notes or a date range. PDF includes
   formatted details, tasks and total time. CSV uses UTF-8 BOM, standard escaping,
   and formula-injection protection. JSON backs up notes only. ZIP includes JSON,
   PDF, and the attachments you select.
9. **Restore:** Choose JSON/ZIP, review counts and duplicates, then Skip or Replace.
   Replace also replaces a duplicate note's attachments; a notes-only JSON removes
   its existing files. The UI confirms replacement. Invalid batches roll back.
10. **Settings:** Choose an IANA display timezone, toggle theme in the top bar, or
    clear account-scoped local drafts. Work dates remain date-only; stored
    timestamps are PostgreSQL timestamptz and API UTC timestamps.

Import [the sample backup](docs/sample-backup.json) to try English and Thai content.
It uses fixed example dates; adjust the date filter or note dates as desired.

## Configuration

Copy `.env.example`, then configure:

- `POSTGRES_*`: local database credentials. Use a URL-safe password or percent-encode
  it in a custom database URL if adapting the Compose configuration.
- `APP_ORIGIN`: exact browser origin, used for mutation origin checks.
- `COOKIE_SECURE`: `true` for HTTPS; local HTTP defaults to `false`.
- `MAX_FILE_BYTES`: per-file size, default 10485760, maximum 52428800.
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`,
  `SMTP_REQUIRE_TLS`: server-side mail settings only.
- Native backend only: `DATABASE_URL`, `ADDRESS`, `STORAGE_PATH`, and `PDF_URL`.

For a real provider, use its STARTTLS host/port (usually 587), set credentials and
`SMTP_REQUIRE_TLS=true`, and use a verified From address. Credentials are never
sent to Angular or written to request logs. Implicit-TLS port 465 is not supported;
use STARTTLS. Mailpit is a local catcher, not an external delivery provider.

For network deployment, configure HTTPS at a reverse proxy, set the exact origin
and secure cookies, restrict registration as appropriate, and keep database,
Mailpit, and Gotenberg private. The default Compose file is a local environment.

## Native development

Use Node **24.21.0 LTS**, Go **1.27.1**, and Docker for backing services. See
[version decisions](docs/versions.md) for exact libraries and official references.

```sh
docker compose up -d db mailpit pdf
docker compose run --rm migrate
cd backend
go run ./cmd/server
```

Default native backend uses local PostgreSQL on 5432 and local SMTP/PDF services.
The API documentation files should be available in the process working directory:
run a built backend from the repository root for `/docs`:

```sh
cd backend
go build -o ../daily-server ./cmd/server
cd ..
./daily-server
```

On Windows use `go build -o ../daily-server.exe ./cmd/server`, then `cd ..` and
`.\daily-server.exe` from the repository root. In another terminal:

```sh
cd frontend
npm ci
npm start
```

If Docker is unavailable, the PDF service has a native development alternative.
After installing frontend dependencies and Playwright Chromium, run from the root:

```sh
node scripts/pdf-service.mjs
```

It listens only on localhost:3000, implements the same HTML multipart endpoint used
by the Go report client, disables page scripts and remote requests, and renders real
PDFs using Playwright Chromium. The backend embeds static Noto Sans Thai Regular
and Bold fonts in PDF HTML, so Thai shaping does not depend on system fonts.
The checked-in font license is in `backend/internal/report/fonts/OFL.txt`.
PostgreSQL and SMTP must still be available; install them natively or use Compose.
The native renderer is for local development; Compose uses Gotenberg.

PowerShell environment example: `$env:DATABASE_URL='postgres://...'`.
POSIX example: `export DATABASE_URL='postgres://...'`.

## Migrations and generated queries

```sh
docker compose run --rm migrate
docker compose run --rm migrate -path=/migrations -database='postgres://daily:daily_local@db:5432/daily?sslmode=disable' version
```

Destructive rollback, only for an intentionally disposable database:

```sh
docker compose run --rm migrate -path=/migrations -database='postgres://daily:daily_local@db:5432/daily?sslmode=disable' down 1
```

Install sqlc 1.31.1, then regenerate checked-in Go queries:

```sh
cd backend
sqlc generate
```

The SQL files are the source of truth. Do not edit generated files manually.

## Checks

Backend unit tests and real PostgreSQL integration tests:

```sh
cd backend
gofmt -w cmd internal
go vet ./...
go test ./...
go build ./cmd/server
```

Set `TEST_DATABASE_URL` to a **dedicated test database** to enable integration
tests. They create an isolated random schema and remove only that schema. The
database role needs schema creation permission.

```powershell
$env:TEST_DATABASE_URL='postgres://daily:daily_local@localhost:5432/daily?sslmode=disable'
go test -count=1 -v ./...
```

```sh
TEST_DATABASE_URL='postgres://daily:daily_local@localhost:5432/daily?sslmode=disable' go test -race -count=1 ./...
```

Frontend:

```sh
cd frontend
npm ci
npm test
npm run lint
npm run build
npx playwright install chromium
npm run e2e
```

The Compose stack must be running for Playwright. It creates unique test accounts,
verifies saved notes, drafts/conflicts, attachments, real Mailpit messages, exports,
and restore. Artifacts include screenshots, traces on failure, and a Thai PDF for
visual review. `APP_URL` and `MAILPIT_URL` override service locations.
`SKIP_PDF=1` runs the core browser journey without the PDF/ZIP portion when the
renderer is unavailable; this is **not** a full end-to-end PDF verification.

GitHub Actions runs formatting, static analysis, database tests, frontend tests and
builds, generated-query drift checking, and the full Compose browser journey.
See [verification results](docs/verification.md) for what actually ran locally.

## Architecture and boundaries

`frontend/src/app/features` holds lazy standalone feature components. Signals own
UI state and computed summaries; HttpClient/RxJS handle async calls, debounced
search cancellation and editor subscriptions. Typed reactive forms drive input.
Angular sanitizes rendered HTML and the server applies a stricter allowlist.

`backend/internal/domain` contains models and validation; `service` contains note
rules and backup parsing; `repository` wraps pgx/sqlc; `storage`, `mailer`, and
`report` isolate integrations. `httpapi` holds Fiber middleware, DTO binding, and
transaction orchestration. `cmd/server` configures the pool, logging, cleanup,
timeouts, and graceful shutdown. Sessions use random hashed tokens and bcrypt
passwords, HttpOnly/SameSite cookies, origin checks and CSRF tokens.

Filesystem and PostgreSQL cannot participate in one atomic commit. Files are
written before metadata commits, compensated on known failures, and old files use
a durable deletion queue. A reconciler removes unreferenced files older than one
hour after crashes or uncertain database commit responses. Keep both volumes together when making infrastructure
backups. See [backup schema and restore semantics](docs/backup-schema.md).

Current deliberate bounds: 500 notes per export/import, 50 MiB selected files per
export, 100 MiB decompressed backup limit, 100 recent email-history items, one
uploaded file per picker/drop operation. Work on multiple notes uses independent
version checks. SMTP cannot guarantee exactly-once external delivery after a
network failure; uncertain outcomes require human review.
