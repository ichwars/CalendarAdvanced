# CalendarAdvanced

CalendarAdvanced is a self-hosted, local-first calendar web application for private servers. It follows the same practical architecture style as RailKeeper2: one Go service, a built React frontend, SQLite as the default database, Docker Compose deployment, first-run setup, local authentication, hardened containers, audit logs and an explicit update-check flow.

## Highlights

- Go backend as a modular monolith with `net/http` and clear internal package boundaries.
- React + TypeScript frontend built with Vite and served by the production Go binary.
- SQLite primary storage with migrations in `backend/migrations`.
- No default credentials. The first user is created through the setup screen.
- Argon2id password hashing through the system Argon2 library.
- HTTP-only sessions, CSRF tokens, role checks, setup/login/password-reset rate limits and audit logging.
- Calendars, events, recurring events, attendees, reminders, invitations, local exports and free/busy/conflict data.
- CalDAV/DAVx⁵ adapter under `/dav/` and `/.well-known/caldav` with separate revocable CalDAV tokens.
- Dark theme by default with German and English UI translations.

## Quick start

```bash
cp .env.example .env
docker compose up -d --build
```

Open `http://localhost:8090`. On first start, CalendarAdvanced opens the setup screen for the first admin account. No default user exists.

## Docker Compose installation

For a small private server, clone the repository, copy `.env.example` to `.env`, adjust the public URL and start the service:

```bash
git clone https://github.com/ichwars/CalendarAdvanced.git
cd CalendarAdvanced
cp .env.example .env
docker compose up -d --build
```

Important production settings:

- Set `CALENDAR_PUBLIC_URL` to the URL users open in the browser.
- Set `CALENDAR_COOKIE_SECURE=true` when serving through HTTPS.
- Set `CALENDAR_TOKEN_ENCRYPTION_KEY` to a stable 32-byte base64 value before first production use, or keep the generated `/data/calendaradvanced.key` backed up.
- Keep the `calendaradvanced_data` volume in your backup plan.

Check the running container:

```bash
docker compose ps
docker compose logs -f calendaradvanced
```

## Local development

Start backend and UI together:

```bash
start-dev.cmd
```

Backend:

```bash
cd backend
go test ./...
go run ./cmd/calendaradvanced
```

Frontend:

```bash
cd frontend
npm ci
npm run dev
```

Production frontend build:

```bash
cd frontend
npm ci
npm run build
```

Useful development defaults:

```bash
CALENDAR_ADDR=:8080
CALENDAR_HOST_PORT=8090
CALENDAR_DATA_DIR=./data
CALENDAR_MIGRATIONS_DIR=./migrations
CALENDAR_SEEDS_DIR=./seeds
CALENDAR_STATIC_DIR=../frontend/dist
CALENDAR_PUBLIC_URL=http://localhost:8090
CALENDAR_COOKIE_SECURE=false
```

## Update

CalendarAdvanced does not self-modify containers. The admin update screen checks a GitHub-Releases-compatible endpoint and shows the current version, available version, release notes and release URL. Update with:

```bash
docker compose pull
docker compose up -d
```

Database migrations run on application start.

## Security posture

CalendarAdvanced is designed for trusted self-hosted installations, but avoids the dangerous defaults: no default admin, no clear-text tokens in the database, role checks server-side, strict security headers and audit logs for security-relevant activity. Read `docs/security.md` before exposing the app outside a private network.
