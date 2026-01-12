# Reesource Tracker

Reesource Tracker is a full-stack application for tracking samples, products, and locations. It uses Go for the backend, Bun (with Svelte) for the frontend, and SQLC for type-safe database access.

## Development Guide

### Prerequisites

- [Go](https://golang.org/doc/install) (v1.18+ recommended)
- [Bun](https://bun.sh/) (v1+ recommended)

### Backend Setup (Go)

1. **Install Go dependencies:**
   ```powershell
   go mod tidy
   ```
2. **Database setup:**
   - **SQLite (default):** The database file is at `database/db.sqlite`. No additional setup required.
   - **PostgreSQL:** Set environment variables to use PostgreSQL instead:
     ```bash
     export DB_DRIVER=postgres
     export POSTGRES_HOST=localhost
     export POSTGRES_PORT=5432
     export POSTGRES_USER=postgres
     export POSTGRES_PASSWORD=your_password
     export POSTGRES_DB=reesource_tracker
     export POSTGRES_SSLMODE=disable  # or 'require' for SSL
     ```
   - Migrations are database-agnostic and will run automatically on startup.
3. **SQLC code generation:**
   - Install SQLC directly via Go:
     ```powershell
     go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
     ```
   - After installation, restart your terminal to ensure the Go bin directory is in your PATH.
   - Generate Go code from SQL queries:
     ```powershell
     sqlc generate
     ```
   - This will generate type-safe Go code for database access in `lib/database/query.sql.go`.
4. **Run the backend:**
   ```powershell
   go run main.go
   ```
   Or build and run:
   ```powershell
   go build -o build/reesource-tracker.exe main.go
   ./build/reesource-tracker.exe
   ```

### Frontend Setup (Bun + Svelte)

1. **Install Bun:**
   - [Install Bun](https://bun.sh/docs/installation)
2. **Install frontend dependencies:**
   ```powershell
   cd client
   bun install
   ```
3. **Run the frontend dev server:**
   ```powershell
   bun run --bun dev
   ```
   - The app will be available at [http://localhost](http://localhost)

### Usage of SQLC

- SQLC reads SQL queries from `database/query.sql` and generates Go code for type-safe database access.
- Configuration is in `sqlc.yaml`.
- After editing SQL files, always run `sqlc generate` to update Go code.

### Docker (Optional)

- Docker is not required for development. Deployment is handled by GitHub Actions.

## Project Structure

- `main.go` - Entry point for the Go backend
- `api/` - API routes and handlers
- `lib/database/` - Database models, query code, and wrappers
- `client/` - Frontend (Svelte + Bun)
  - `src/` - Main source code for the frontend
    - `lib/` - Shared frontend utilities and components
      - `components/` - Reusable Svelte components (UI, forms, etc.)
  - `public/` - Static assets served by the frontend
- `database/` - Database migrations and SQL files (SQLite/PostgreSQL compatible)
- `build/` - Compiled binaries and static build outputs

## Useful Commands

- **Go:** `go mod tidy`, `go run .`, `go build`
- **Bun:** `bun install`, `bun run --bun dev`, `bun run --bun build`
- **SQLC:** `sqlc generate`
