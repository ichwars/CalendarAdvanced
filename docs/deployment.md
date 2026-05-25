# Deployment

## Docker Compose

```bash
cp .env.example .env
# edit .env, especially CALENDAR_COOKIE_SECURE and CALENDAR_TOKEN_ENCRYPTION_KEY
docker compose up -d --build
```

Open `http://localhost:8090` and complete first-run setup.

## Production checklist

- Put CalendarAdvanced behind HTTPS.
- Set `CALENDAR_PUBLIC_URL` to the external URL.
- Set `CALENDAR_COOKIE_SECURE=true`.
- Set a 32-byte `CALENDAR_TOKEN_ENCRYPTION_KEY` for stable encrypted local secrets.
- Keep `/data` on persistent storage with backups.
- Keep the container read-only and use the provided `/tmp` tmpfs.
- Do not expose SQLite or `/data` directly.

## Healthcheck

`/health` returns a JSON status object. The container healthcheck calls the built-in `healthcheck` command.

## Updates

CalendarAdvanced does not modify its own container. Update with:

```bash
docker compose pull
docker compose up -d
```

If building from source:

```bash
docker compose up -d --build
```

Migrations run on startup. Keep a backup before updating.
