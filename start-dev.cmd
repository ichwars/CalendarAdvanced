@echo off
setlocal
cd /d "%~dp0"

if not exist ".env" (
  copy ".env.example" ".env" >nul
)

docker compose up -d --build
if errorlevel 1 (
  echo.
  echo CalendarAdvanced backend could not be started. Check Docker Desktop and port 8090.
  exit /b 1
)

echo.
echo CalendarAdvanced backend: http://127.0.0.1:8090
echo CalendarAdvanced UI:      http://127.0.0.1:5173
echo.

cd /d "%~dp0frontend"
"C:\Program Files\nodejs\node.exe" dev-server.mjs
