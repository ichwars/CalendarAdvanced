# Security

CalendarAdvanced uses OWASP ASVS as the baseline for security decisions. This document records the relevant implementation choices and operational limits.

## Identity and sessions

- There is no default user. First-run setup is required before login.
- Passwords are hashed with Argon2id through the system Argon2 library.
- Password policy requires at least 12 characters with lower-case, upper-case, digit and symbol classes.
- Session tokens are generated with `crypto/rand` and only SHA-256 token hashes are stored in SQLite.
- Session cookies are HTTP-only and SameSite Strict.
- `CALENDAR_COOKIE_SECURE=true` should be used behind HTTPS.
- CSRF tokens are random and compared against the session-bound hash for write requests.
- Password changes, password resets, 2FA changes and reset completion revoke active sessions.

## Two-factor authentication

- TOTP uses HMAC-SHA1 with a 30-second step and ±1 step drift allowance.
- Backup codes are random one-time codes and only token hashes are stored.
- Backup-code use is audit logged.
- Disabling 2FA requires the current password.

## Authorization

Server-side role checks are required. The frontend only hides actions for usability.

- Admin: system, users, integrations and global settings.
- Editor: calendars and events.
- Viewer: read-only access.

## Rate limits

The application enforces in-process rate limits for setup, login and password reset. The schema also contains `rate_limits` for persistent or distributed deployments. The app intentionally uses the direct peer address instead of trusting arbitrary `X-Forwarded-For` headers. For internet-facing use, keep reverse-proxy rate limits enabled too.

## Audit logging

Audit events are written for setup, login, logout, password changes, password resets, 2FA changes, user management, calendar changes, event changes, integrations, exports and backups. Secrets are never logged.

## Security headers

The API middleware sends:

- Content-Security-Policy
- X-Content-Type-Options: nosniff
- Referrer-Policy: no-referrer
- X-Frame-Options: DENY
- Permissions-Policy

## Backups

JSON backups include app data such as calendars and events. They intentionally exclude password hashes, sessions, reset tokens, TOTP secrets and backup codes.

## Known limits for this MVP

- CalDAV write operations are intentionally narrow. The adapter implements discovery and read/query resources and should be expanded with full PUT/DELETE scheduling tests before public multi-client write use.
- Native libraries `libsqlite3` and `libargon2` are part of the runtime trust base.
