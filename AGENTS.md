# TreveccaPedia AGENTS.md

Guidelines for agentic coding assistants working on the TreveccaPedia codebase.

## Project Overview

TreveccaPedia is a multi-service Go-based wiki platform. It uses microservices architecture with the following services:
- **web** (port 8080): Go frontend using templ + Tailwind CSS
- **api-layer** (port 2745): API gateway / reverse proxy
- **wiki** (port 9454): Wiki CRUD service
- **search** (port 7724): Full-text search (Bleve)
- **auth** (port 8083): Authentication (JWT)
- **moderation**: Content moderation service
- **wiki-db** (port 5432), **auth-db** (port 5433), **mod-db** (port 5434): PostgreSQL databases

## Build/Lint/Test Commands

### Development Environment (dev.sh)

The primary way to manage the development environment:

```bash
# Start development environment
./dev.sh wiki web    # Work on wiki and web locally, rest in Docker
./dev.sh auth        # Work on auth locally
./dev.sh             # Run everything in Docker
./dev.sh stop        # Stop all Docker services
./dev.sh status      # Check service status
./dev.sh logs [svc]  # View logs (optionally for specific service)
```

### Go Services (run from service directory)

```bash
cd <service>  # e.g., auth, wiki, search, api-layer, moderation, web

# Development (hot reload)
air                     # Requires: go install github.com/air-verse/air@latest

# Build
go build -o ./tmp/main ./cmd/main.go

# Run (requires .env file)
go run ./cmd/main.go

```

### Auth Service Only (has Makefile)

```bash
cd auth
make build              # Build the auth service binary
make run                # Run the auth service
make dev                # Run with hot reload (air)
make test               # Run tests
make test-coverage      # Run tests with coverage
make fmt                # Format code
make lint               # Run linter
make clean              # Clean build artifacts
```

### Web Service (CSS/Templ)

```bash
cd web

# Install dependencies (first time)
bun install             # or npm install

# CSS (Tailwind v4)
bun build:css           # Build Tailwind CSS (production)
bun watch:css           # Watch and rebuild CSS (development)

# Templates
templ generate          # Generate Go templates from .templ files

# Note: air.toml already runs: templ generate && bun build:css
```

### Docker Commands

```bash
# Build and run a specific service
docker compose --profile wiki up -d --build wiki
docker compose --profile auth up -d --build auth

# Run databases only
docker compose up -d wiki-db auth-db

# View logs
docker compose logs -f wiki
docker compose logs -f

# Rebuild after code changes
docker compose --profile <service> up -d --build <service>
```

## Code Style Guidelines

### Imports

- Group imports: stdlib, blank imports, third-party, internal
- Use blank import for side effects (e.g., database drivers): `_ "github.com/lib/pq"`
- Alias packages to avoid conflicts (e.g., `httphandler "auth/internal/http"`)

```go
import (
    "context"
    "database/sql"
    
    _ "github.com/lib/pq"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    
    "auth/internal/config"
)
```

### Formatting

- Use `go fmt` standard formatting
- Tabs for indentation (Go standard)
- Line length: practical, no hard limit
- No trailing whitespace

### Naming Conventions

- **Packages**: lowercase, single word (e.g., `auth`, `handlers`)
- **Files**: snake_case (e.g., `get_handlers.go`, `jwt_test.go`)
- **Types**: PascalCase, exported (e.g., `JWTService`, `Claims`)
- **Interfaces**: `-er` suffix (e.g., `Handler`, `Reader`)
- **Functions/Methods**: PascalCase for exported, camelCase for unexported
- **Variables**: camelCase (e.g., `userID`, `dataStore`)
- **Constants**: PascalCase for exported, camelCase for unexported
- **Error variables**: start with `Err` (e.g., `ErrNotFound`)
- **Test functions**: `Test` + function name (e.g., `TestGenerateToken`)
- **Table-driven tests**: `tt` for test case variable, `t.Run(tt.name, ...)`

### Types

- Use struct tags for JSON: `` `json:"field_name"` ``
- Use UUIDs for IDs (github.com/google/uuid)
- Define custom error types with fields: `Code`, `Type`, `Details`

### Error Handling

- Wrap errors with context: `fmt.Errorf("failed to X: %w", err)`
- Use custom error types for domain errors (see `wiki/errors/`)
- Check errors immediately after function calls
- Log fatal errors in main: `log.Fatalf("Failed to X: %v", err)`
- Return errors; don't panic in library code

```go
// Custom error pattern (wiki/errors/)
type WikiError struct {
    Code    int
    Type    string
    Details string
    err     error
}

func (w WikiError) Error() string { return w.Details }
func (w WikiError) Unwrap() error { return w.err }
```

### Project Structure

```
<service>/
├── cmd/
│   └── main.go           # Entry point
├── internal/
│   ├── auth/             # Auth logic
│   ├── config/           # Configuration
│   ├── http/             # HTTP handlers & middleware
│   └── store/            # Database layer
├── handlers/             # HTTP handlers (alternative pattern)
├── utils/                # Utilities
├── go.mod
├── .air.toml             # Air hot-reload config
├── .env.example          # Environment template
├── Dockerfile
└── README.md
```

### Configuration

- Load from environment variables via `.env` file
- Use `github.com/joho/godotenv` for loading
- Provide defaults for development
- Redact sensitive fields in `String()` method

### Database

- Use `database/sql` with `lib/pq` driver
- Use connection strings: `host=... port=... dbname=... user=... password=... sslmode=disable`
- Always close connections: `defer db.Close()`
- Use `context.WithTimeout()` for operations

### HTTP Handlers (Gin)

- Return JSON errors: `c.AbortWithStatusJSON(code, gin.H{"error": "..."})`
- Parse query params: `c.DefaultQuery("key", "default")`
- Parse path params: `c.Param("id")`
- Set trusted proxies: `r.SetTrustedProxies(nil)`

### Testing

- Use table-driven tests with `tests := []struct{ ... }`
- Use `t.Run(tt.name, func(t *testing.T) {...})` for subtests
- Test both success and error cases
- Use `t.Fatalf` for setup failures, `t.Errorf` for assertion failures
- Mock external dependencies

```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid", "input", "output", false},
        {"invalid", "bad", "", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Dependencies

- **Go**: 1.25+
- **Web Framework**: Gin (github.com/gin-gonic/gin)
- **JWT**: golang-jwt/jwt/v5
- **UUID**: google/uuid
- **Database**: lib/pq (PostgreSQL driver)
- **Password Hashing**: golang.org/x/crypto/bcrypt
- **Templating**: a-h/templ (web service)
- **CSS**: Tailwind CSS v4

## Environment Variables

Each service has `.env` file. Copy from `.env.example`:

```bash
for dir in wiki search auth api-layer web wiki-db auth-db moderation; do
    cp "$dir/.env.example" "$dir/.env"
done
```

Key variables: `PORT`, `DATABASE_URL` (or `*_DB_HOST/PORT/NAME/USER/PASSWORD`), `JWT_SECRET`

## Running a Single Test

```bash
# From service directory:
go test ./... -run TestFunctionName -v
go test ./internal/auth -run TestGenerateToken -v
go test -run "TestJWT" -v  # Run all tests matching pattern
```
