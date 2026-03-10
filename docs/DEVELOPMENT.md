# Development Setup

## Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Docker](https://docs.docker.com/get-docker/) and Docker Compose
- [air](https://github.com/air-verse/air) (Go live reload)
- [templ](https://templ.guide/) (only needed if working on the web service locally)
- [Bun](https://bun.sh/) or Node.js (only needed if working on the web service locally)

## Architecture Overview

TreveccaPedia is composed of the following services:

| Service     | Port  | Description                          |
|-------------|-------|--------------------------------------|
| **web**     | 8080  | Go web frontend (templ + Tailwind)   |
| **api-layer** | 2745 | API gateway / reverse proxy         |
| **wiki**    | 9454  | Wiki CRUD service                    |
| **search**  | 7724  | Full-text search (Bleve)             |
| **auth**    | 8083  | Authentication (JWT)                 |
| **wiki-db** | 5432  | PostgreSQL database for wiki         |
| **auth-db** | 5433  | PostgreSQL database for auth         |

Request flow:

```
Browser -> web (:8080) -> api-layer (:2745) -> wiki (:9454)    -> wiki-db (:5432)
                                             -> search (:7724)
                                             -> auth (:8083)    -> auth-db (:5433)
```

## Quick Start

### 1. Set up environment files

Each service has a `.env.example` file. Copy it to `.env` in each service directory:

```bash
for dir in wiki search auth api-layer web wiki-db auth-db; do
    cp "$dir/.env.example" "$dir/.env"
done
```

Review and update passwords/secrets as needed. The defaults work for local development.

### 2. Start the environment

Use `dev.sh` to start the development environment. Pass the names of the services you want to work on locally -- everything else runs in Docker.

```bash
# Work on the wiki service locally, everything else in Docker
./dev.sh wiki

# Work on wiki and web locally
./dev.sh wiki web

# Work on auth locally
./dev.sh auth

# Run everything in Docker (no local services)
./dev.sh
```

The script will:

1. Start both databases (wiki-db, auth-db) in Docker
2. Wait for the databases to be healthy
3. Build and start the remaining services in Docker
4. Print a summary showing what's running where

(Note: This will likely take a while the first time, but should be faster on subsequent starts.)

### 3. Start your local services

After `dev.sh` finishes, start the services you're working on using air:

```bash
cd wiki && air
```

In a separate terminal for each additional local service:

```bash
cd web && air
```

> **Note:** The web service requires `templ` and `bun` to be installed for its pre-build step (`templ generate && bun build:css`), which air runs automatically.

## dev.sh Reference

```
Usage: ./dev.sh [services...]

Commands:
  ./dev.sh [services...]    Start environment (listed services run locally)
  ./dev.sh stop             Stop all Docker services
  ./dev.sh status           Show Docker service status
  ./dev.sh logs [service]   Tail logs (optionally for a specific service)
  ./dev.sh help             Show help message

Available services: wiki search auth api-layer web
```

### Examples

```bash
# Typical: working on one service
./dev.sh wiki

# Working on frontend and backend together
./dev.sh wiki web

# Check what's running
./dev.sh status

# View logs for a specific Docker service
./dev.sh logs search

# View all Docker service logs
./dev.sh logs

# Shut everything down
./dev.sh stop
```

## How It Works

The root `docker-compose.yml` defines all services using Docker Compose [profiles](https://docs.docker.com/compose/how-tos/profiles/). Each Go service is assigned to its own profile (e.g., the wiki service is in the `wiki` profile). The two databases have no profile, so they always start.

When you run `./dev.sh wiki`, the script:

- Starts wiki-db and auth-db (no profile, always started)
- Activates the profiles for search, auth, api-layer, and web
- Skips the wiki profile, leaving port 9454 free for your local air instance

All services use `network_mode: host`, so every service -- whether in Docker or running locally -- is accessible on `localhost` at its configured port. No special networking or URL configuration is needed.

## Running Everything Manually

If you prefer not to use `dev.sh`, you can start services individually.

### Databases

```bash
# Start both databases
docker compose up -d wiki-db auth-db

# Or start them from their own directories (original setup)
cd wiki-db && docker compose up -d
cd auth-db && docker compose up -d
```

### Individual services in Docker

```bash
# Start a specific service (e.g., search)
docker compose --profile search up -d search
```

### Individual services with air

```bash
cd wiki && air
cd search && air
cd auth && air
cd api-layer && air
cd web && air
```

## Service Configuration

Each service reads its configuration from a `.env` file in its directory. Key environment variables:

| Variable | Service | Default | Description |
|----------|---------|---------|-------------|
| `WIKI_DB_HOST` | wiki | `localhost` | Database host |
| `WIKI_DB_PORT` | wiki | `5432` | Database port |
| `WIKI_DATA_DIR` | wiki | `../wiki-fs` | Filesystem storage path |
| `INDEX_DIR` | search | `../wiki-fs/index` | Search index path |
| `API_LAYER_URL` | search, web | `http://127.0.0.1:2745/v1` | API layer URL |
| `AUTH_DB_HOST` | auth | `localhost` | Auth database host |
| `AUTH_DB_PORT` | auth | `5433` | Auth database port |
| `JWT_SECRET` | auth, api-layer | (see .env.example) | Must match between services |
| `WIKI_SERVICE_URL` | api-layer | `http://127.0.0.1:9454` | Wiki service URL |
| `SEARCH_SERVICE_URL` | api-layer | `http://127.0.0.1:7724` | Search service URL |
| `AUTH_SERVICE_URL` | api-layer | `http://127.0.0.1:8083` | Auth service URL |

## Troubleshooting

### Port already in use

If a service fails to start because its port is in use, check that you haven't started the same service both in Docker and locally:

```bash
./dev.sh status          # check Docker services
lsof -i :9454           # check what's using a specific port
```

### Database connection refused

Ensure the databases are healthy before starting services:

```bash
./dev.sh status
# Look for "healthy" status on wiki-db and auth-db
```

### Rebuilding Docker images

If you've changed a service's code and need to rebuild its Docker image:

```bash
docker compose --profile wiki up -d --build wiki
```

Or stop and restart the full environment:

```bash
./dev.sh stop
./dev.sh wiki    # rebuilds automatically with --build
```
