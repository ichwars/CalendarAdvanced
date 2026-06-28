# CalendarAdvanced deployment notes

The default deployment target is Docker Compose with a persistent `/data` volume.

```bash
docker compose up -d --build
```

The `.env` file is optional. Create one from `.env.example` when you want to override production settings such as `CALENDAR_PUBLIC_URL`, `CALENDAR_COOKIE_SECURE`, SMTP or a stable `CALENDAR_TOKEN_ENCRYPTION_KEY`.

For source deployments, build the image from the Dockerfile:

```bash
git pull --ff-only
docker compose build --pull
docker compose up -d --remove-orphans
```

The runtime container is intentionally hardened:

- read-only root filesystem
- `cap_drop: ALL`
- `no-new-privileges`
- tmpfs `/tmp`
- persistent named volume for `/data`
- healthcheck through `/health`

For reverse proxy deployments, terminate TLS before CalendarAdvanced and set `CALENDAR_COOKIE_SECURE=true` when HTTPS is used. CalendarAdvanced uses the direct peer address for audit/rate-limit logging; keep external IP allow/block rules and internet-facing rate limits at the reverse proxy.
