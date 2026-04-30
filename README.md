# Reesource Tracker

Reesource Tracker is a full-stack application for tracking samples, products, and locations. It uses Go for the backend, Bun (with Svelte) for the frontend, and SQLC for type-safe database access.

## Deployment Guide

The easiest way to deploy is by using the provided docker image. The CI workflow publishes the production image to GitHub Container Registry as `ghcr.io/apollo-fire/reesource-tracker/reesource-tracker:latest`.

### Deployment Prerequisites

- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/)

### Docker Compose Deployment

You can use this image by creating a `compose.yml` file on the deployment host. For example:

```yaml
services:
  postgres:
    image: postgres:17
    restart: unless-stopped
    environment:
      POSTGRES_DB: reesource_tracker
      POSTGRES_USER: reesource_tracker
      POSTGRES_PASSWORD: change-me
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test:
        ["CMD-SHELL", "pg_isready -U reesource_tracker -d reesource_tracker"]
      interval: 10s
      timeout: 5s
      retries: 5

  app:
    image: ghcr.io/apollo-fire/reesource-tracker/reesource-tracker:latest
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      DATABASE_URL: postgresql://reesource_tracker:change-me@postgres:5432/reesource_tracker?sslmode=disable
    ports:
      - "80:80"

volumes:
  postgres_data:
```

Start the stack:

```bash
docker compose up -d
```

The application will be available on port `80`. Database migrations run automatically when the application container starts.

### Updating

Pull the latest published image and recreate the app container:

```bash
docker compose pull app
docker compose up -d app
```

### Notes

- Change the PostgreSQL password before deploying.
- Do not set `DEV=true` in production.

## Development Guide

For development, build and run the backend and frontend locally from source.

### Local Prerequisites

- [Go](https://golang.org/doc/install) (v1.18+ recommended)
- [Bun](https://bun.sh/) (v1+ recommended)
- [Docker](https://docs.docker.com/engine/install/ubuntu/) (for unit testing, installed in your WSL distribution)

### Backend Setup (Go)

1. **Install Go dependencies:**

   ```powershell

   go mod tidy
   ```

2. **Environment variables:**

   Create a `.env` file in the project root with the following variables:

   ```bash
   DEV=true
   DATABASE_URL=postgresql://USER:PASS@HOST:PORT/DATABASE?sslmode=disable
   ```

   - `DEV=true`: Enables development mode, which proxies frontend requests to the Vite dev server (running on port 5173). In production mode (when `DEV` is not set), the backend serves static files from the `client` directory.

   - `DATABASE_URL`: Connection string for the PostgreSQL database.

   - `MAGIC_LINK_WEBHOOK_URL` _(optional)_: Enables the email (magic link) authentication method. When set, the server `POST`s a JSON payload to this URL whenever a sign-in link is requested:

     ```json
     {
       "email": "user@example.com",
       "name": "User Name",
       "login_url": "https://your-app/app?magic_token=<token>"
     }
     ```

     If this variable is not set, the email auth method is disabled and the `/api/auth/email/*` endpoints return an error.

   - `SESSION_SECRET` _(optional)_: Secret used to sign session cookies. A random value is generated on startup if not provided — set this explicitly in production to preserve sessions across restarts.

3. **Database setup:**
   - Configure the `DATABASE_URL` in the `.env` file as shown above.

   - Migrations run automatically on startup.

4. **SQLC code generation:**
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

5. **Run the backend:**

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

#### Setup (Local Windows Development)

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

## Authentication

The app supports two sign-in methods, which can be used simultaneously.

### Passkeys (WebAuthn)

The default method. A user account is created via an **assignment link** — a short-lived, single-use URL that ties a device's passkey to a specific user. Assignment links can be created by an admin or by the user themselves from the account page.

### Email (Magic Links)

Enabled by setting `MAGIC_LINK_WEBHOOK_URL`. When a user requests a sign-in link, the server dispatches a `POST` to the webhook with the login URL; no email server is required — use any HTTP endpoint (e.g. a transactional email provider's API or an automation webhook).

Flow:
1. `POST /api/auth/email/login/request` — client submits `{"email": "..."}`. Always returns `200`; does not reveal whether the address is registered.
2. The webhook receives `{"email", "name", "login_url"}` and sends the link to the user.
3. User clicks the link; the frontend extracts `?magic_token=` from the URL and calls `POST /api/auth/email/login/consume` with `{"token": "..."}` to establish a session.

Tokens expire after **10 minutes** and are single-use. A **30-second cooldown** per user prevents request flooding.

#### Email registration via assignment link

An assignment link can also be used to register an email address instead of a passkey, allowing users without a WebAuthn-capable device to register:

`POST /api/auth/email/register` — body: `{"assignment_token": "...", "email": "..."}`.

### Auth API reference

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/auth/session` | — | Current session info |
| `POST` | `/api/auth/logout` | — | Clear session cookie |
| `GET` | `/api/auth/features` | — | Feature flags (e.g. `magic_links_enabled`) |
| `POST` | `/api/auth/register/begin` | — | Begin passkey registration (requires assignment token) |
| `POST` | `/api/auth/register/finish` | — | Complete passkey registration |
| `POST` | `/api/auth/login/begin` | — | Begin passkey login |
| `POST` | `/api/auth/login/finish` | — | Complete passkey login |
| `POST` | `/api/auth/email/register` | — | Register email via assignment link |
| `POST` | `/api/auth/email/login/request` | — | Request magic link |
| `POST` | `/api/auth/email/login/consume` | — | Consume magic link token, establish session |
| `GET` | `/api/auth/self/assignment-link` | ✓ | Get current user's active assignment link |
| `POST` | `/api/auth/self/assignment-link` | ✓ | Create assignment link for current user |
| `DELETE` | `/api/auth/self/assignment-link` | ✓ | Revoke current user's assignment link |
| `GET` | `/api/auth/self/passkeys` | ✓ | List current user's passkeys |
| `POST` | `/api/auth/self/passkeys/:credential_id/revoke` | ✓ | Revoke a passkey |
| `GET` | `/api/auth/self/emails` | ✓ | List current user's registered emails |
| `POST` | `/api/auth/self/emails` | ✓ | Add an email |
| `POST` | `/api/auth/self/emails/:id/remove` | ✓ | Remove an email |

Admin endpoints follow the same pattern under `/api/auth/admin/users/:user_id/`.

## Usage of SQLC

- SQLC reads SQL queries from `database/query.sql` and generates Go code for type-safe database access.
- Configuration is in `sqlc.yaml`.
- After editing SQL files, always run `sqlc generate` to update Go code.

## Project Structure

- `main.go` — Entry point for the Go backend
- `api/` — API routes and handlers
  - `auth/` — Authentication sub-packages (passkeys, emails, assignmentlinks, bootstrap)
  - `middleware/` — Session parsing and role enforcement
- `lib/` — Business logic and shared helpers (no HTTP dependencies)
  - `auth/` — Session, passkey, magic link, and response-mapping logic
  - `database/` — sqlc-generated models and query wrappers
- `client/` — Frontend (Svelte 5 + Bun)
  - `src/lib/` — Shared utilities and components
  - `src/views/` — Page-level Svelte views
- `database/` — SQL migrations and query definitions
- `build/` — Compiled binaries and static build outputs
