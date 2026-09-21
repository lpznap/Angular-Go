# Verification record

Implementation checks were run on Windows on September 18, 2026. Final source and
documentation review continued on September 21. These are executed results, not
claims inferred from the presence of tests.

## Passed

- Angular production build with Node 24.21.0 and strict TypeScript/template checks.
- Four Vitest tests for timezone day boundaries, inclusive week boundaries,
  date-only draft defaults, and duration formatting.
- Go build and `go vet ./...`.
- OpenAPI schema/reference validation and YAML parsing for Compose and CI.
- Frontend Prettier formatting and Git whitespace checks.
- Go unit tests for note validation, HTML sanitization, file contents and size
  validation, storage persistence, MIME construction, CSV injection protection,
  English/Thai report content, backup validation, duplicate IDs, missing files,
  traversal, symlink entries, and excessive declared decompression sizes.
- Fiber application tests against real PostgreSQL 18.6, using isolated schemas:
  registration, sessions, CSRF, note creation/editing, stale-version rejection,
  search and inclusive date boundaries, pagination, cross-account ownership,
  uploads and authorized downloads, JSON/ZIP restore, skip/replace, restored file
  contents, SMTP failure history, duplicate-send suppression, logout, deletion
  queues, failed-restore transaction rollback, and crash-orphan reconciliation.
- Two Playwright Chromium journeys against the real Angular app, Go API,
  PostgreSQL, filesystem, Mailpit, and native Chromium PDF service. The final run
  passed both tests in approximately 18 seconds, with PDF checks enabled:
  - Account creation, logout/login, autosave, rich-text formatting, time entry,
    tasks, persisted attachments after reload, SMTP email with a PDF and selected
    attachment verified in Mailpit, JSON export/restore preview and skip, PDF
    download, ZIP export and replace restoration with file contents, and mobile
    layout overflow check.
  - Concurrent edits in two tabs return a conflict and preserve the stale local
    draft across reload.
- Desktop dashboard and mobile editor screenshots visually inspected.
- Exported A4 PDF rendered with Poppler and visually inspected for Thai text,
  line spacing, task content, and clipping. Noto Sans Thai Regular/Bold and Latin
  font subsets were verified as embedded TrueType fonts. Thai body text was
  extracted successfully. Some extractors may omit a combined Thai tone mark in
  a heading even though it renders correctly; use JSON/CSV for lossless text data.
- npm dependency audit reported zero vulnerabilities during installation.

## Not executed here

- Docker image builds, `docker compose up --build -d --wait`, and the actual
  Gotenberg container: Docker is not available on this workstation. The tested
  native Chromium renderer implements the same multipart request contract, but
  it is not a substitute for validating the Gotenberg image and Compose lifecycle.
- PostgreSQL 17.11 container tests: the workspace deployment was changed to the
  PostgreSQL 17 series after the native 18.6 integration run. CI now targets
  17.11; that matrix result is not claimed as passed locally.
- Linux race-detector checks and GitHub Actions execution. The workflow includes
  them, but no remote CI run was initiated.
- A real external SMTP provider or confirmed recipient delivery. Mailpit proves
  SMTP acceptance and captured message content only.

## Reproduce

Use the README's exact setup, migration, build, unit-test, and Playwright commands.
Run the full Compose journey before relying on container deployment. No real
credentials are included. Existing local test accounts and files are stored only
in ignored workspace tool/data directories; portable tools are not source dependencies.

## Operational limits

- Calendar pages show the current result page, with pagination below the month
  grid. They do not silently claim to show every note in a large month.
- Fiber/fasthttp does not expose every client disconnect as context cancellation.
  Explicit request, database, SMTP, and renderer deadlines bound work; browser
  search cancellation stops obsolete client subscriptions.
- SMTP acceptance is not confirmed delivery. A failed network response can leave
  an uncertain send outcome; intentional retries use a new key after review.
- Database and filesystem commits are separate. Compensating cleanup, a durable
  deletion queue, and one-hour orphan reconciliation handle ordinary failures and
  process crashes. Infrastructure backups must include both persistent volumes.
