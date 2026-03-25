# Reesource Tracker

Reesource Tracker is a full-stack application for tracking samples, products, and locations. It uses Go for the backend, Bun (with Svelte) for the frontend, and SQLC for type-safe database access.

## Development Guide

### Prerequisites

- [Go](https://golang.org/doc/install) (v1.18+ recommended)
- [Bun](https://bun.sh/) (v1+ recommended)
- [Docker](https://docs.docker.com/engine/install/ubuntu/) (for unit testing, installed in your WSL distribution)

### Backend Setup (Go)

1. **Install Go dependencies:**

   ```powershell

   go mod tidy
   ```

2. **Database setup:**
   - Set the `DATABASE_URL` environment variable:

     ```bash

     # Format: postgresql://USER:PASS@HOST:PORT/DATABASE?sslmode=disable
     export DATABASE_URL="postgresql://postgres:YOUR_PASSWORD@localhost:5432/reesource_tracker?sslmode=disable"
     ```

   - Migrations run automatically on startup.

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

   - With both the frontend and backend running, the app will be available at [http://localhost](http://localhost)

### Unit Testing

The project uses Testcontainers for integration testing with PostgreSQL.

#### Setup

1. **Configure Docker daemon to expose TCP socket:**

   In your WSL distribution, edit or create `/etc/docker/daemon.json`:

   ```json
   {
     "hosts": ["unix:///var/run/docker.sock", "tcp://127.0.0.1:2375"]
   }
   ```

   Restart the Docker service:

   ```bash
   sudo systemctl restart docker
   ```

2. **Set DOCKER_HOST environment variable:**

   In your PowerShell terminal (or add to your profile):

   ```powershell
   $env:DOCKER_HOST = "tcp://127.0.0.1:2375"
   ```

3. **Configure Testcontainers properties:**

   Ensure the following `.testcontainers.properties` file exists in your home directory (`C:\Users\<YourUsername>\.testcontainers.properties`):

   ```properties
   docker.client.strategy=org.testcontainers.dockerclient.NpipeSocketClientProviderStrategy
   docker.host=tcp://localhost:2375
   ```

4. **Run tests:**

   ```powershell
   go test ./... -v
   ```

## Usage of SQLC

- SQLC reads SQL queries from `database/query.sql` and generates Go code for type-safe database access.
- Configuration is in `sqlc.yaml`.
- After editing SQL files, always run `sqlc generate` to update Go code.

## Project Structure

- `main.go` - Entry point for the Go backend
- `api/` - API routes and handlers
- `lib/database/` - Database models, query code, and wrappers
- `client/` - Frontend (Svelte + Bun)
  - `src/` - Main source code for the frontend
    - `lib/` - Shared frontend utilities and components
      - `components/` - Reusable Svelte components (UI, forms, etc.)
  - `public/` - Static assets served by the frontend
- `database/` - Database migrations (PostgreSQL) and data storage
- `build/` - Compiled binaries and static build outputs
