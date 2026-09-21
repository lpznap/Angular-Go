# Verified versions and dependency purposes

Version checks performed September 18, 2026 using official documentation and package
registries. Direct dependencies are exact in package.json/go.mod; package-lock.json
and go.sum capture resolved dependencies. Container images have exact release tags.

| Component | Selected version | Purpose |
|---|---|---|
| Node.js | 24.21.0 LTS | Angular build and test runtime |
| Angular | 22.1.7 | Standalone, zoneless application; router, typed forms, HttpClient, Signals |
| Angular CLI / build | 22.1.8 | Application compiler, production bundler, development proxy |
| Material / CDK | 22.1.7 | Accessible controls and interaction primitives |
| TypeScript | 6.0.2 | Strict compile-time models and template checking |
| RxJS | 7.8.2 | Debounce, cancellation, asynchronous requests |
| tslib | 2.8.1 | TypeScript runtime helpers |
| Vitest | 4.1.11 | Frontend unit tests; compatible with Angular build's `^4.0.8` peer requirement |
| Playwright | 1.63.0 | Chromium browser journeys and artifacts |
| Prettier | 3.6.2 | Deterministic frontend formatting checks |
| Go | 1.27.1 | Backend runtime, testing, vet, slog |
| Fiber | 3.5.0 | All inbound HTTP routes, middleware, handlers, errors, and application tests |
| pgx | 5.11.0 | PostgreSQL pool, parameterized queries, and transactions |
| sqlc | 1.31.1 | Generates typed Go note queries from SQL |
| golang-migrate | 4.19.1 | Versioned schema migration runner |
| bluemonday | 1.0.27 | Server-side HTML allowlist sanitation |
| x/crypto | 0.57.0 | bcrypt password hashing |
| PostgreSQL | 17.11 | Persistent relational storage; follows the workspace's PostgreSQL 17 deployment choice |
| Gotenberg | 8.30.1 | Chromium HTML-to-PDF, complex-script shaping, Noto Thai fonts |
| Mailpit | 1.27.4 | Local SMTP receiver and email inspection |
| nginx | 1.28.2 | Static frontend and same-origin API proxy |

Angular's published 22.1.7 peer metadata requires TypeScript `>=6.0 <6.1`, and its
Node engine accepts `^24.15.0`. Node 23 installed on this workstation is not used.
Vitest 5 is newer, but conflicts with Angular build's supported peer range; 4.1.11
is selected deliberately. No `--force` or `--legacy-peer-deps` workaround is used.

Official references:

- [Angular compatibility](https://angular.dev/reference/versions)
- [Angular releases](https://angular.dev/reference/releases)
- [Node release lifecycle](https://nodejs.org/en/about/previous-releases)
- [Go downloads](https://go.dev/dl/)
- [Fiber releases](https://github.com/gofiber/fiber/releases)
- [pgx releases](https://github.com/jackc/pgx/releases)
- [sqlc releases](https://github.com/sqlc-dev/sqlc/releases)
- [golang-migrate releases](https://github.com/golang-migrate/migrate/releases)
- [PostgreSQL 18.6 release](https://www.postgresql.org/docs/release/18.6/)
- [Gotenberg fonts and configuration](https://gotenberg.dev/docs/configuration)
- [Mailpit Docker releases](https://mailpit.axllent.org/docs/install/docker/)

Go's standard `net/http` is used only for outgoing calls to the PDF service and test
request construction. Incoming HTTP is exclusively Fiber. SMTP is implemented using
the standard SMTP client with deadlines, STARTTLS verification, MIME attachments,
and explicit acceptance semantics.

For native development, `scripts/pdf-service.mjs` uses the pinned Playwright
Chromium build through the same multipart endpoint. PDF HTML includes the checked-in
static [Noto Sans Thai fonts](https://github.com/notofonts/noto-fonts/tree/main/hinted/ttf/NotoSansThai)
under the SIL Open Font License. Static fonts are used instead of variable fonts
to preserve embedded TrueType font subsets and improve PDF text extraction.
