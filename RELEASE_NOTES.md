# CalendarAdvanced v0.1.0

Initial release candidate for self-hosted Docker Compose installations.

## Highlights

- Local-first calendar, event, task and contact management.
- DAV synchronization for events, tasks and contacts with conflict handling and sync history.
- Dashboard with DAV status, current workload and quick navigation.
- Import/export for ICS, Excel, CSV/XLSX and full JSON backup/restore.
- First-run admin setup, local authentication, 2FA, audit log and hardened Docker runtime.

## Installation

```bash
cp .env.example .env
docker compose up -d --build
```

Open `http://localhost:8090` and create the first admin account.

## Notes

- Use `CALENDAR_COOKIE_SECURE=true` behind HTTPS.
- Back up the Docker volume `calendaradvanced_data`.
- For stable encrypted DAV and 2FA secret storage, configure `CALENDAR_TOKEN_ENCRYPTION_KEY` before production use.
