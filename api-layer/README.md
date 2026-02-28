# API Layer Service

## Usage

Set up go:
```
go get api-layer/cmd
```

Make sure to set up environment variables (in `api-layer` directory):
```
cp .env.example ./.env
source .env
```

Using air in the `api-layer` directory:
```
air .
```

## Info

This service starts an HTTP server on port `:2745`

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JWT_SECRET` | Yes | (dev default) | Must match auth service JWT secret |

## Endpoints

### Public Endpoints (No Auth Required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/wiki/pages` | List wiki pages |
| GET | `/v1/wiki/pages/:id` | Get a wiki page |
| GET | `/v1/wiki/pages/:id/revisions` | List page revisions |
| GET | `/v1/wiki/pages/:id/revisions/:rev` | Get a specific revision |

### Protected Endpoints (Auth Required)

These endpoints require:
- Valid JWT token in `Authorization: Bearer <token>` header
- User must have the `contributor` role

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/wiki/pages/new` | Create a new wiki page |
| POST | `/v1/wiki/pages/:id/revisions` | Create a new page revision |

### Authentication Errors

| Status | Error | Description |
|--------|-------|-------------|
| 401 | `authorization header required` | No token provided |
| 401 | `invalid authorization format` | Token format incorrect |
| 401 | `invalid token` | Token validation failed |
| 403 | `insufficient permissions` | User lacks required role |

### Example: Creating a Page with Auth

```bash
# Get token from auth service
TOKEN=$(curl -s -X POST http://localhost:8083/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"dev@trevecca.edu","password":"devpass"}' | jq -r '.accessToken')

# Create page with token
curl -X POST http://localhost:2745/v1/wiki/pages/new \
  -H "Authorization: Bearer $TOKEN" \
  -F "slug=my-page" \
  -F "name=My Page" \
  -F "author=dev@trevecca.edu" \
  -F "new_page=@content.md"
```

