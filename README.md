# CIMS Go/Postgres

Customized Information Management System for utility files, inventory movements, purchases, sales, AR/AP, rebates, other income, checks, and expenses.

## Stack

- Go 1.26
- `chi` HTTP router
- `html/template` + HTMX
- Postgres through `pgxpool`
- `goose` SQL migrations

## First Run

Local machine needs Docker Desktop or local Postgres.

```bash
docker compose up --build
```

Open `http://localhost:8080`.

Default admin:

- Username: `admin`
- Password: `admin123`

Change `ADMIN_USERNAME`, `ADMIN_PASSWORD`, `SESSION_HASH_KEY`, and `SESSION_BLOCK_KEY` before real office/LAN use.

## Local Go Run

```bash
cp .env.example .env
export $(grep -v '^#' .env | xargs)
go run ./cmd/cims
```

`DATABASE_URL` must point to a running Postgres database. The app runs migrations on startup.

The app also auto-loads `.env` from the working directory or executable directory if present.

## Windows Package

From Windows PowerShell after clone:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build-windows.ps1
```

This produces:

- `dist\windows\CIMS-Windows-dev.zip`
- optional `dist\windows\CIMS-Setup-dev.exe` if Inno Setup is installed

See [docs/windows.md](docs/windows.md).

## Restore a Postgres SQL Backup

Use `cmd/restore-sql` to empty the current public tables, then load a plain PostgreSQL `.sql` backup.

```powershell
$env:DATABASE_URL = "postgres://cims:cims@localhost:5432/cims?sslmode=disable"
go run .\cmd\restore-sql -yes -file "C:\path\to\cims_postgres_backup.sql" -psql "C:\Program Files\PostgreSQL\16\bin\psql.exe"
```

If `psql.exe` is already in `PATH`, the `-psql` option can be omitted.

## Forms Included

Utility files:

- Stock Categories
- Branches
- Suppliers
- Customers
- Expenses Chart
- Stocks
- Other Income Chart

Transaction files:

- Stock In
- Stock Out
- Checks In
- Other Income
- Purchases
- Sales
- Stock Transactions
- AP Credit/Debit
- AR Credit/Debit
- Rebates
- Expenses

## Verification

```bash
go fmt ./...
go test ./...
go vet ./...
go build ./cmd/cims
```

Database integration verification needs Postgres available.
