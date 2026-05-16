# Windows Build and Install

## Prerequisites

- Git
- Go 1.26+
- PostgreSQL 16+ or 18+
- Optional: Inno Setup 6 if you want a `.exe` installer

## Clone and Build

From PowerShell:

```powershell
git clone <your-repo-url>
cd cims-go
powershell -ExecutionPolicy Bypass -File .\scripts\build-windows.ps1
```

Output:

- ZIP bundle: `dist\windows\CIMS-Windows-dev.zip`
- Optional installer: `dist\windows\CIMS-Setup-dev.exe`

## What the Bundle Contains

- `cims.exe`
- `templates\`
- `static\`
- `db\migrations\`
- `.env`
- `.env.example`
- `start-cims.bat`

## Configure

Edit `.env` in the installed folder or extracted bundle:

```env
ADDR=:8090
DATABASE_URL=postgres://cims:cims@localhost:5432/cims?sslmode=disable
ADMIN_USERNAME=admin
ADMIN_PASSWORD=admin123
SESSION_HASH_KEY=change-this-session-hash-key-32-bytes
SESSION_BLOCK_KEY=0123456789abcdef
```

## Run

Double-click:

```text
start-cims.bat
```

or from terminal:

```powershell
.\cims.exe
```

The app auto-loads `.env` from the app folder on startup.

## Notes

- The application is still a Go web server. Windows users open it in the browser after starting it.
- PostgreSQL must already exist and match `DATABASE_URL`.
- The app runs migrations on startup.
