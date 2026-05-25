# CalDAV / DAVx⁵

## URLs

- Discovery: `https://your-host/.well-known/caldav`
- Base URL: `https://your-host/dav/`
- Calendar home: `/dav/calendars/{email}/`
- Calendar collection: `/dav/calendars/{email}/{calendarId}/`

## Authentication

Use HTTP Basic with the CalendarAdvanced user e-mail as username and a generated CalDAV token as password. The normal login password is intentionally rejected. Tokens are stored as hashes and can be revoked.

## Supported adapter behavior

- `PROPFIND /dav/` returns current-user-principal and calendar-home-set.
- `PROPFIND /dav/calendars/{email}/` returns calendar collections.
- `REPORT /dav/calendars/{email}/{calendarId}/` returns VEVENT calendar data and ETags.
- `GET /dav/calendars/{email}/{calendarId}/...ics` returns calendar data.

## Sync state

The schema contains `caldav_sync_state` with resource path, ETag and sync token. Event ETags are updated when local events change. PUT/DELETE expansion should update the sync state idempotently.

## DAVx⁵ setup

1. Create a CalDAV token in CalendarAdvanced.
2. In DAVx⁵, add a CalDAV account using URL/login.
3. Use the base URL `/dav/`.
4. Use e-mail as username and the generated token as password.
5. Select discovered calendars.
