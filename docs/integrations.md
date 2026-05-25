# Integrations

CalendarAdvanced treats integrations as adapters. Local SQLite remains the canonical store.

## CalDAV

CalDAV is exposed under `/.well-known/caldav` and `/dav/`. Users create separate CalDAV tokens in the integrations screen. Normal login passwords are not accepted for CalDAV.

## Local fallback

CalendarAdvanced must remain useful without external services. Events, calendars, CSV and XLSX exports work locally.
