---
# Web Auth (Browser-Facing)

This app uses a server-managed auth model:

- The browser logs in via the web service (`/auth/login`).
- On success, the web service sets a JWT in an `HttpOnly` cookie named `auth_token`.
- JavaScript cannot read `HttpOnly` cookies. This is intentional.
- Use `/auth/me` to learn who is logged in (web service reads the cookie and verifies upstream).

Because the token is not accessible to JavaScript, browser code should not try to add an `Authorization` header.

## Routes

### `POST /auth/login`

Logs the user in and sets the `auth_token` cookie.

Request:
```json
{
  "email": "user@trevecca.edu",
  "password": "password"
}
```

Success response:
```json
{
  "user": {
    "id": "00000000-0000-0000-0000-000000000000",
    "email": "user@trevecca.edu",
    "roles": ["contributor"]
  }
}
```

Notes:
- The upstream API layer returns `accessToken`, but the web service strips it and only returns `user`.

### `POST /auth/register`

Registers a new user and sets the `auth_token` cookie.

Request:
```json
{
  "email": "user@trevecca.edu",
  "password": "password"
}
```

Success response:
```json
{
  "user": {
    "id": "00000000-0000-0000-0000-000000000000",
    "email": "user@trevecca.edu",
    "roles": ["contributor"]
  }
}
```

### `GET /auth/me`

Returns the current logged-in user based on the `auth_token` cookie.

Success response:
```json
{
  "id": "00000000-0000-0000-0000-000000000000",
  "email": "user@trevecca.edu",
  "roles": ["contributor"]
}
```

Failure response:
- `401 {"error":"not authenticated"}` (no cookie / empty cookie)

### `POST /auth/logout`

Clears the `auth_token` cookie.

Success response:
```json
{
  "message": "logged out"
}
```

## Frontend Usage

From browser JS:
- `fetch('/auth/login', ...)` logs in and sets cookie.
- `fetch('/auth/me')` returns user if cookie is valid.
- `fetch('/auth/logout', { method: 'POST' })` logs out.

See `web/static/js/auth.js` for the intended usage pattern.
