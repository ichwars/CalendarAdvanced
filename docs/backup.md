# Backup and restore

## Local data

SQLite database and local runtime files live under `/data` in the container volume.

## JSON backup

The admin endpoint `/api/v1/system/backup` exports app data in JSON. It excludes:

- password hashes
- sessions
- password reset tokens
- TOTP secrets
- 2FA backup codes

## Local calendar exports

Use `/api/v1/exports/csv` and `/api/v1/exports/xlsx` for local event exports.

## Restore rules

Restore validation checks app name and backup schema version. Restore must not overwrite authentication tables. A full restore workflow should be run in maintenance mode so clients cannot write while data is being imported.
