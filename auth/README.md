# Authentication Service

JWT-based authentication service with role-based access control for Trevecca-Pedia.

## Prerequisites (Nothing new here, just a reminder)

- **Go 1.25+** — install via Homebrew: `brew install go`
- **Docker** — install [Docker Desktop](https://www.docker.com/products/docker-desktop/)
- **air** (hot reload) — install after Go is set up:

```bash
go install github.com/air-verse/air@latest
```

Then make sure Go's bin directory is in your PATH (add this to your `~/.zshrc` or `~/.bashrc`):

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

Reload your shell after: `source ~/.zshrc`

## Setup & Running

### 1. Clone the Repo

```bash
git clone <repo-url>
cd trevecca-pedia/auth
```

### 2. Install Dependencies

From the `auth` directory:

```bash
go mod download
```

### 3. Start the Database

From the `auth-db` directory, set up and start the auth database:

```bash
cd ../auth-db
cp .env.example .env
```

Edit `auth-db/.env` and set a real password for `POSTGRES_PASSWORD` (the compose file will refuse to start without it). Then:

```bash
docker compose up -d
```

This starts `auth-db` (PostgreSQL) on port `5433`. Wait for it to be healthy:

```bash
docker compose ps
```

Then go back to the `auth` directory:

```bash
cd ../auth
```

### 4. Run the Auth Service

The easiest way is to create a `.env` file in the `auth` directory so you don't have to type the variables every time. Copy the example:

```bash
cp .env.example .env
```

The example has the right values for local development. If you want the dev user (`dev@trevecca.edu / devpass`) seeded on startup, set `DEV_SEED=true` in your `.env`. Then run:

```bash
air .
```

**Alternative** — if you prefer not to use a `.env` file, you can pass the variables inline. Replace `<your-db-password>` with the value you set for `POSTGRES_PASSWORD` in `auth-db/.env`:

```bash
PORT=8083 \
DATABASE_URL="postgres://auth_user:<your-db-password>@localhost:5433/auth?sslmode=disable" \
JWT_SECRET="dev-secret-key-change-in-production-please" \
JWT_EXP_HOURS=24 \
CORS_ORIGINS="http://localhost:3000,http://localhost:5173,http://localhost:8080" \
DEV_SEED=true \
air .
```

The service starts on port `8083`. With `DEV_SEED=true` it automatically creates a dev user on startup:

- **Email:** `dev@trevecca.edu`
- **Password:** `devpass`
- **Role:** `contributor`

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `8083` | HTTP server port |
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `JWT_SECRET` | Yes | — | Secret key for signing JWTs |
| `JWT_EXP_HOURS` | No | `24` | Token expiration in hours |
| `CORS_ORIGINS` | No | `localhost:3000,5173,8080` | Comma-separated allowed origins |
| `DEV_SEED` | No | `false` | Create dev user on startup (dev only) |

## API Endpoints

> **Note:** In the full stack, the frontend does not call the auth service directly. Requests flow through the web service → API layer → auth service. The endpoints below are the auth service's own routes, useful for testing it in isolation.
### Health Check

```
GET /healthz
```

```json
{ "status": "ok" }
```

### Register

```
POST /auth/register
Content-Type: application/json

{ "email": "student@trevecca.edu", "password": "mypassword123" }
```

- Email must be `@trevecca.edu`
- Email must be in the `allowed_emails` whitelist
- Password must be at least 8 characters
- New users are assigned the `contributor` role automatically

**201 Created:**
```json
{
  "accessToken": "eyJhbGci...",
  "user": { "id": "uuid", "email": "student@trevecca.edu", "roles": ["contributor"] }
}
```

### Login

```
POST /auth/login
Content-Type: application/json

{ "email": "dev@trevecca.edu", "password": "devpass" }
```

**200 OK:**
```json
{
  "accessToken": "eyJhbGci...",
  "user": { "id": "uuid", "email": "dev@trevecca.edu", "roles": ["contributor"] }
}
```

### Get Current User

```
GET /auth/me
Authorization: Bearer <token>
```

**200 OK:**
```json
{ "id": "uuid", "email": "dev@trevecca.edu", "roles": ["contributor"] }
```

## JWT Contract

Tokens are signed with **HS256** and contain:

| Claim | Value |
|-------|-------|
| `sub` | User UUID |
| `email` | User email |
| `roles` | Array of role names |
| `iss` | `trevecca-pedia-auth` |
| `aud` | `trevecca-pedia` |
| `exp` | Now + `JWT_EXP_HOURS` |

The `JWT_SECRET` must be shared with any other service that validates tokens locally (e.g. the API layer).

## Granting Access (Email Whitelist)

Registration is restricted to emails that have been manually approved. Users whose email is not in the whitelist will get a `403 Forbidden` error when trying to register.

To grant someone access, connect to the auth database and run:

```sql
INSERT INTO allowed_emails (email) VALUES ('student@trevecca.edu');
```

To connect to the database:

```bash
docker exec -it trevecca-auth-db psql -U auth_user -d auth
```

To see who is currently whitelisted:

```sql
SELECT * FROM allowed_emails;
```

To revoke access (only works if the user hasn't registered yet):

```sql
DELETE FROM allowed_emails WHERE email = 'student@trevecca.edu';
```

> **Note:** The `dev@trevecca.edu` dev user is created directly by `DEV_SEED` and does not need to be in the whitelist.

## User Roles

| Role | Description |
|------|-------------|
| `reader` | Can view wiki pages |
| `contributor` | Can create and edit wiki pages |
| `admin` | Reserved for future use |

## Testing

```bash
# Health check
curl http://localhost:8083/healthz

# Login
curl -X POST http://localhost:8083/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"dev@trevecca.edu","password":"devpass"}'

# Get current user (replace TOKEN)
curl http://localhost:8083/auth/me \
  -H "Authorization: Bearer TOKEN"
```

Or run the automated smoke test script:

```bash
./test-auth.sh
```

## Troubleshooting

**Database connection fails**
- Check the container is running: `docker compose ps` (from `auth-db/`)
- Check port 5433 is not in use: `lsof -i :5433`

**Port 8083 already in use**
- Change `PORT` in the run command or check what's using it: `lsof -i :8083`

**Invalid token errors in other services**
- Ensure `JWT_SECRET` is identical in the auth service and the API layer
- Check the token hasn't expired (default 24h)
- Verify format is `Authorization: Bearer <token>`

**Dev user not created**
- Confirm `DEV_SEED=true` is set
- Check the service logs for errors on startup
