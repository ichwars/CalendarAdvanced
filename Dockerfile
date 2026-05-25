# syntax=docker/dockerfile:1

FROM node:22-bookworm-slim AS frontend-build
WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.23-bookworm AS backend-build
RUN apt-get update && apt-get install -y --no-install-recommends libsqlite3-dev libargon2-dev ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /src/backend
COPY backend/go.mod ./
COPY backend/ ./
RUN CGO_ENABLED=1 go test ./...
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/calendaradvanced ./cmd/calendaradvanced

FROM debian:bookworm-slim AS runtime
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates libsqlite3-0 libargon2-1 && rm -rf /var/lib/apt/lists/* \
    && groupadd -r calendaradvanced && useradd -r -g calendaradvanced -d /nonexistent -s /usr/sbin/nologin calendaradvanced \
    && mkdir -p /app/web /app/migrations /app/seeds /data \
    && chown -R calendaradvanced:calendaradvanced /data
WORKDIR /app
COPY --from=backend-build /out/calendaradvanced /app/calendaradvanced
COPY backend/migrations /app/migrations
COPY backend/seeds /app/seeds
COPY --from=frontend-build /src/frontend/dist /app/web
USER calendaradvanced
EXPOSE 8080
ENV CALENDAR_ADDR=:8080 \
    CALENDAR_DATA_DIR=/data \
    CALENDAR_MIGRATIONS_DIR=/app/migrations \
    CALENDAR_SEEDS_DIR=/app/seeds \
    CALENDAR_STATIC_DIR=/app/web
ENTRYPOINT ["/app/calendaradvanced"]
