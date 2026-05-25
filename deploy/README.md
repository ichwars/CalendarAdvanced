# CalendarAdvanced deployment notes

The default deployment target is Docker Compose with a persistent `/data` volume.

```bash
cp .env.example .env
docker compose up -d --build
```

The runtime container is intentionally hardened:

- read-only root filesystem
- `cap_drop: ALL`
- `no-new-privileges`
- tmpfs `/tmp`
- persistent named volume for `/data`
- healthcheck through `/health`

For reverse proxy deployments, terminate TLS before CalendarAdvanced and set `CALENDAR_COOKIE_SECURE=true` when HTTPS is used. CalendarAdvanced uses the direct peer address for audit/rate-limit logging; keep external IP allow/block rules and internet-facing rate limits at the reverse proxy.
