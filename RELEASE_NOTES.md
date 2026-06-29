# CalendarAdvanced v0.2.0

Second release for self-hosted Docker Compose installations.

## Highlights

- Adds PROJALLG support to the existing Excel import for `Montageplanung KWXX.xlsx` files.
- Keeps ManPower planning imports in the same import card.
- Imports multiple projects on the same day as separate appointments.
- Improves short appointment rendering in the day and week calendar views.
- Hardens the Android app startup path so an unreachable server no longer leaves a black screen.

## Installation

```bash
docker compose up -d --build
```

Open `http://localhost:8090` or your configured reverse-proxy URL.

## Notes

- The `.env` file is optional, but recommended for production settings.
- Use `CALENDAR_COOKIE_SECURE=true` behind HTTPS.
- Back up the Docker volume `calendaradvanced_data`.
- For stable encrypted DAV and 2FA secret storage, configure `CALENDAR_TOKEN_ENCRYPTION_KEY` before production use.
