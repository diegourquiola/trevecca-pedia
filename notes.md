# Authentication Layer — Reviewer Notes

Branch: `authentication-layer`

This document explains the logic behind the non-obvious decisions in this PR for anyone reviewing the code.

---

## Architecture Overview

Authentication is implemented as a **dedicated service** (`auth/`) with its own PostgreSQL database (`auth-db/`). No auth logic lives in the wiki service or the API layer — the API layer only validates tokens locally.

```
Browser → Web (8080) → API Layer (2745) → Auth Service (8083) → auth-db (5433)
                                        → Wiki Service (9454) → wiki-db (5432)
```

---

## 1. JWT Validation is Done Locally in the API Layer

**File:** `api-layer/middleware/auth.go`

The API layer validates JWT tokens **itself** using the shared `JWT_SECRET` instead of making a network call to the auth service on every protected request. This is intentional.

Why: calling the auth service to validate a token on every wiki write would add latency and create a hard dependency — if auth goes down, writes would fail even though the token is perfectly valid. HMAC-SHA256 tokens are stateless and self-contained, so local validation is safe.

The tradeoff: tokens cannot be revoked server-side before they expire (24h default). This is an acceptable risk for an MVP.

The API layer verifies:
- Signature (using the shared secret)
- Signing method must be HMAC (rejects RS256 confusion attacks)
- Issuer must be `trevecca-pedia-auth`
- Audience must be `trevecca-pedia`
- Expiration timestamp

---

## 2. Two-Layer Registration Guard

**File:** `auth/internal/http/handlers_auth.go` — `Register()`

Registration has two independent checks before creating a user:

1. **Email domain check** — must end in `@trevecca.edu`. This is a fast string check done in-process.
2. **Whitelist check** — email must exist in the `allowed_emails` table in the auth DB.

Both must pass. The domain check comes first so we don't hit the database for obviously invalid emails. The whitelist means an admin must manually insert a row into `allowed_emails` before anyone can register, even with a valid Trevecca email. This prevents open self-registration.

---

## 3. The `allowed_emails` Table

**File:** `auth-db/init/0002_whitelist.sql`

```sql
CREATE TABLE IF NOT EXISTS allowed_emails (
    email TEXT PRIMARY KEY
);
```

This is a simple allowlist — no foreign key to `users`. An admin adds an email here before the user registers. After registration, the row in `allowed_emails` is not removed, so re-registration after account deletion would be allowed without admin re-approval. This is intentional for simplicity.

To whitelist a user in production:
```sql
INSERT INTO allowed_emails (email) VALUES ('student@trevecca.edu');
```

---

## 4. Roles are Stored in the Database, Embedded in the JWT

**Files:** `auth/internal/store/postgres.go`, `auth/internal/auth/jwt.go`

On login/register, the auth service fetches the user's roles from the `user_roles` junction table and embeds them in the JWT payload. The API layer reads roles directly from the token claims — no DB lookup.

All new users are automatically assigned the `contributor` role on registration (`Register()` handler, line ~145). The three available roles seeded at DB init are: `reader`, `contributor`, `admin`.

Role enforcement in the API layer uses `middleware.RequireRole(role)` chained after `AuthMiddleware()`.

---

## 5. Request Flow for Protected Wiki Routes

**File:** `api-layer/middleware/auth.go`, `api-layer/cmd/main.go`

Wiki **read** routes (`GET /v1/wiki/pages`, `GET /v1/wiki/pages/:id`) are **public** — no token required.

Wiki **write** routes (`POST /v1/wiki/pages/new`, `POST /v1/wiki/pages/:id/revisions`, `POST /v1/wiki/pages/:id/delete`) require a valid JWT with the `contributor` role.

The middleware stores parsed claims in the Gin context so downstream handlers can access `userID`, `email`, and `roles` without re-parsing the token.

---

## 6. Token Storage on the Frontend

**File:** `web/static/js/auth.js`

JWT tokens are stored in `localStorage` under the key `auth_token`. The user object is stored separately under `auth_user`. The nav bar reads `auth_user` synchronously on page load to show/hide the login link and user dropdown — no network call is made just to render the nav.

`/auth/me` is only called on the profile page to verify the token is still valid against the auth service. If it returns non-200, `clearAuth()` is called and the user is shown as logged out.

Known limitation: tokens cannot be invalidated server-side before expiry. A forced logout only clears localStorage on that browser.

---

## 7. The Web Service Proxies Auth, Not the API Layer Directly

**Files:** `web/auth/proxy.go`, `web/auth/auth_handlers.go`

The browser talks to the **web service** (`localhost:8080/auth/*`), which proxies to the **API layer** (`localhost:2745/v1/auth/*`), which proxies to the **auth service** (`localhost:8083/auth/*`).

This keeps the browser from needing to know internal service URLs and allows the web service to add request transformation or middleware in the future without changing the frontend JS.

---

## 8. Dev Seed User

**File:** `auth/internal/http/handlers_auth.go` (seeding logic in `auth/cmd/auth/main.go`)

When `DEV_SEED=true` in the auth service `.env`, a development user is created at startup:
- Email: `dev@trevecca.edu`
- Password: `devpass`
- Role: `contributor`

This is controlled by an env flag and should never be enabled in production. The `.env.example` ships with `DEV_SEED=false`.
