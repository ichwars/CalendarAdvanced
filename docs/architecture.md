# CalendarAdvanced architecture

CalendarAdvanced is a modular monolith. The production process is one Go binary that exposes the JSON API, CalDAV adapter and static frontend. SQLite remains the primary source of truth. Integrations can synchronize or export, but they are not the canonical data store unless the user explicitly configures synchronization.

## Backend boundaries

- `backend/cmd/calendaradvanced`: process entrypoint, configuration loading, logging, migrations and HTTP server lifecycle.
- `backend/internal/api`: route registration, middleware, cookies, request decoding and JSON response mapping.
- `backend/internal/application`: use cases, validation orchestration, audit calls, rate limits and integration workflows.
- `backend/internal/domain`: calendar, user, event, recurrence and audit rules that do not depend on HTTP or SQLite.
- `backend/internal/infrastructure`: native SQLite adapter, Argon2id, token encryption, CalDAV/iCalendar serialization, local Excel writer and static file serving.
- `backend/migrations`: normalized SQLite schema.
- `backend/seeds`: optional seed data; role seeds are also idempotently present in the first migration.

## Frontend boundaries

- `frontend/src/app`: shell, route state, layout and global styles.
- `frontend/src/features`: feature-level pages and workflows.
- `frontend/src/shared`: API adapter, i18n, theme, shared types, validation and small UI primitives.

## Runtime model

1. The container starts the Go binary.
2. The binary opens `/data/calendaradvanced.db` and applies migrations.
3. `/health` reports process health.
4. `/api/v1/*` serves JSON.
5. `/.well-known/caldav` and `/dav/*` serve CalDAV discovery and calendar resources.
6. All other paths serve the built frontend.

## Design rules

- No hidden default admin or seeded password.
- No cloud requirement for the core product.
- CalDAV code stays in integration boundaries.
- Security-sensitive tokens are random, hashed or encrypted depending on retrieval requirements.
- UI text is provided through translation keys, not inline component copy.
