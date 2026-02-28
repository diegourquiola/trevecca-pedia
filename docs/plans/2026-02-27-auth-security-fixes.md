# Auth Security Fixes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix all 5 critical and 9 important security issues found in the authentication layer PR review.

**Architecture:** Fixes span four layers — `auth/` service, `api-layer/` middleware, `web/` proxy, and `auth-db/` infrastructure. The largest change replaces `localStorage` JWT storage with `HttpOnly` cookies to eliminate XSS token theft risk. All other fixes are surgical, touching 1-3 lines each.

**Tech Stack:** Go 1.22, gin-gonic, golang-jwt/v5, PostgreSQL, vanilla JS

---

## Task 1: Fix hardcoded JWT secret fallback (CRIT-1)

**Files:**
- Modify: `api-layer/config/config.go:31-37`
- Modify: `api-layer/cmd/main.go` (add startup validation)

**Problem:** `GetJWTSecret()` silently falls back to a public placeholder string when `JWT_SECRET` is unset. Any attacker can forge valid JWTs.

**Step 1: Replace the fallback with a fatal error**

In `api-layer/config/config.go`, replace the entire `GetJWTSecret()` function:

```go
func GetJWTSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		panic("JWT_SECRET environment variable is required but not set")
	}
	return secret
}
```

**Step 2: Validate at startup, not per-request**

In `api-layer/cmd/main.go`, add a startup check at the top of `main()` before any route registration:

```go
func main() {
	// Validate required secrets at startup — fail fast before accepting traffic
	_ = config.GetJWTSecret() // panics if JWT_SECRET unset

	r := gin.Default()
	// ... rest unchanged
```

**Step 3: Commit**
```bash
git add api-layer/config/config.go api-layer/cmd/main.go
git commit -m "fix(api-layer): panic on missing JWT_SECRET instead of using hardcoded fallback"
```

---

## Task 2: Fix faulty expiry check — tokens with no exp claim accepted forever (CRIT-2)

**Files:**
- Modify: `auth/internal/auth/jwt.go:66-99`
- Modify: `api-layer/middleware/auth.go:88-122`

**Problem:** Both JWT validators contain `if claims.ExpiresAt != nil && ...` which silently accepts tokens with no `exp` claim. The `golang-jwt/v5` library already validates expiration — the manual check is both redundant and wrong.

**Step 1: Fix `auth/internal/auth/jwt.go`**

In `ValidateToken`, add `jwt.WithExpirationRequired()` to `ParseWithClaims` and delete the manual expiry block (lines 93-96):

```go
func (j *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secret, nil
	}, jwt.WithExpirationRequired()) // require exp claim; reject tokens without it

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	if claims.Issuer != j.issuer {
		return nil, fmt.Errorf("invalid issuer")
	}

	if len(claims.Audience) == 0 || claims.Audience[0] != j.audience {
		return nil, fmt.Errorf("invalid audience")
	}

	// Expiration is validated by jwt.ParseWithClaims + WithExpirationRequired above.
	// Do NOT add a manual check here — if claims.ExpiresAt != nil is wrong
	// because it silently accepts tokens with no exp claim.

	return claims, nil
}
```

**Step 2: Fix `api-layer/middleware/auth.go`**

Same fix in `validateToken`: add the option, remove the manual block (lines 116-119):

```go
func validateToken(tokenString string) (*Claims, error) {
	secret := config.GetJWTSecret()

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	}, jwt.WithExpirationRequired())

	if err != nil {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	if claims.Issuer != "trevecca-pedia-auth" {
		return nil, fmt.Errorf("invalid issuer")
	}

	if len(claims.Audience) == 0 || claims.Audience[0] != "trevecca-pedia" {
		return nil, fmt.Errorf("invalid audience")
	}

	// Expiration validated by ParseWithClaims + WithExpirationRequired.
	return claims, nil
}
```

Also remove the `"time"` import from `api-layer/middleware/auth.go` since it's no longer used.

**Step 3: Commit**
```bash
git add auth/internal/auth/jwt.go api-layer/middleware/auth.go
git commit -m "fix(auth): require exp claim in JWT; remove faulty manual expiry check that accepted exp-less tokens"
```

---

## Task 3: Fix Config struct leaking secrets via %v (CRIT-4)

**Files:**
- Modify: `auth/internal/config/config.go`

**Problem:** `Config` is a plain struct with exported `DatabaseURL` and `JWTSecret` fields. Any `log.Printf("%+v", cfg)` call emits the raw signing key and DB credentials.

**Step 1: Add a `String()` method that redacts secrets**

Append to `auth/internal/config/config.go` after the `Load()` function:

```go
// String redacts sensitive fields so Config can be safely logged.
func (c Config) String() string {
	return fmt.Sprintf(
		"{Port:%s DatabaseURL:[REDACTED] JWTSecret:[REDACTED] JWTExpHours:%d CORSOrigins:%v DevSeed:%t}",
		c.Port, c.JWTExpHours, c.CORSOrigins, c.DevSeed,
	)
}
```

**Step 2: Commit**
```bash
git add auth/internal/config/config.go
git commit -m "fix(auth): redact DatabaseURL and JWTSecret from Config String() to prevent secret leakage in logs"
```

---

## Task 4: Fix discarded error in duplicate-user check (CRIT-5)

**Files:**
- Modify: `auth/internal/http/handlers_auth.go:127`

**Problem:** `existingUser, _ := h.store.GetUserByEmail(...)` silently discards the error. A DB failure here produces `existingUser == nil`, so execution continues to `CreateUser` with an unverified assumption that the user doesn't exist.

**Step 1: Assign the error and check it correctly**

Change line 127 from:
```go
existingUser, _ := h.store.GetUserByEmail(c.Request.Context(), req.Email)
if existingUser != nil {
```

To:
```go
existingUser, err := h.store.GetUserByEmail(c.Request.Context(), req.Email)
if err == nil && existingUser != nil {
	// err == nil means the user was found (no error); it's a duplicate
```

The full block becomes:
```go
// Check if user already exists
existingUser, err := h.store.GetUserByEmail(c.Request.Context(), req.Email)
if err == nil && existingUser != nil {
	c.JSON(http.StatusConflict, gin.H{"error": "user with this email already exists"})
	return
}
// err != nil here means "user not found" (expected) or a transient DB error;
// the subsequent CreateUser call will surface any real DB failure.
```

**Step 2: Commit**
```bash
git add auth/internal/http/handlers_auth.go
git commit -m "fix(auth): stop discarding error from GetUserByEmail in duplicate-user check"
```

---

## Task 5: Fix misleading log message on login failure (IMP-6)

**Files:**
- Modify: `auth/internal/http/handlers_auth.go:59`

**Problem:** The log always says "user not found" regardless of whether the real cause was a DB error. This masks failures during incident investigation.

**Step 1: Log the actual error**

Change line 59 from:
```go
log.Printf("Login failed for %s: user not found", req.Email)
```
To:
```go
log.Printf("Login failed for %s: %v", req.Email, err)
```

**Step 2: Commit**
```bash
git add auth/internal/http/handlers_auth.go
git commit -m "fix(auth): log actual error on login failure instead of always saying 'user not found'"
```

---

## Task 6: Normalize email to lowercase before whitelist lookup (IMP-7)

**Files:**
- Modify: `auth/internal/http/handlers_auth.go:108-124`

**Problem:** The domain suffix check lowercases the email, but `IsEmailWhitelisted` and `GetUserByEmail` receive the original mixed-case email. A user registering as `User@Trevecca.Edu` passes the domain check but fails the case-sensitive DB lookup.

**Step 1: Normalize at the start of Register handler**

Add a normalization line after the `ShouldBindJSON` block succeeds. Insert after line 105 (`return` of the bind error check):

```go
// Normalize email to lowercase for consistent DB lookups
req.Email = strings.ToLower(req.Email)
```

This must come before any domain check or DB call.

**Step 2: Commit**
```bash
git add auth/internal/http/handlers_auth.go
git commit -m "fix(auth): normalize email to lowercase before whitelist and DB lookups in Register"
```

---

## Task 7: Fix Bearer scheme case inconsistency (IMP-5)

**Files:**
- Modify: `auth/internal/http/middleware.go:24`

**Problem:** The auth service middleware compares `parts[0] != "Bearer"` (case-sensitive) while the API layer middleware uses `strings.ToLower(parts[0]) != "bearer"` (case-insensitive). RFC 7235 says the scheme is case-insensitive. The auth service is the stricter one — make it consistent.

**Step 1: Make auth service middleware case-insensitive**

Change line 24 from:
```go
if len(parts) != 2 || parts[0] != "Bearer" {
```
To:
```go
if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
```

The `strings` package is already imported in this file.

**Step 2: Commit**
```bash
git add auth/internal/http/middleware.go
git commit -m "fix(auth): make Bearer scheme comparison case-insensitive to match RFC 7235 and api-layer behavior"
```

---

## Task 8: Fix JWT error details leaked to API middleware clients (IMP-4)

**Files:**
- Modify: `api-layer/middleware/auth.go:43-47`

**Problem:** The middleware returns `err.Error()` directly, which exposes internal details like `"invalid issuer"` and `"invalid audience"` to HTTP clients — information an attacker can use to craft better tokens.

**Step 1: Return a generic message**

Change lines 43-47 from:
```go
if err != nil {
	c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	c.Abort()
	return
}
```
To:
```go
if err != nil {
	c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
	c.Abort()
	return
}
```

**Step 2: Commit**
```bash
git add api-layer/middleware/auth.go
git commit -m "fix(api-layer): return generic 401 message instead of leaking JWT validation details to clients"
```

---

## Task 9: Fix Render() errors silently dropped (IMP-8)

**Files:**
- Modify: `web/auth/auth_handlers.go`

**Problem:** `page.Render(...)` returns an error that is ignored. A template failure returns 200 with broken HTML and no log.

**Step 1: Check and log the error**

Replace the entire file content:

```go
package auth

import (
	"log"
	"web/templates/auth"
	"web/templates/components"

	"github.com/gin-gonic/gin"
)

func GetLoginPage(c *gin.Context) {
	content := auth.AuthPage()
	page := components.Page("Log In", content)
	if err := page.Render(c.Request.Context(), c.Writer); err != nil {
		log.Printf("error rendering login page: %v", err)
	}
}

func GetProfilePage(c *gin.Context) {
	content := auth.ProfilePage()
	page := components.Page("Profile", content)
	if err := page.Render(c.Request.Context(), c.Writer); err != nil {
		log.Printf("error rendering profile page: %v", err)
	}
}
```

**Step 2: Commit**
```bash
git add web/auth/auth_handlers.go
git commit -m "fix(web): log errors from template Render() instead of silently dropping them"
```

---

## Task 10: Fix hardcoded DB password in auth-db docker-compose (IMP-9)

**Files:**
- Modify: `auth-db/docker-compose.yml`
- Create: `auth-db/.env.example`
- Create: `auth-db/.gitignore`

**Problem:** `POSTGRES_PASSWORD: authpass` is committed to version control.

**Step 1: Update docker-compose.yml to use env var**

Replace the `environment:` block:
```yaml
    environment:
      POSTGRES_USER: ${POSTGRES_USER:-auth_user}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}
      POSTGRES_DB: ${POSTGRES_DB:-auth}
```

**Step 2: Create `auth-db/.env.example`**
```
POSTGRES_USER=auth_user
POSTGRES_PASSWORD=change_me_in_production
POSTGRES_DB=auth
```

**Step 3: Create `auth-db/.gitignore`**
```
.env
```

**Step 4: Commit**
```bash
git add auth-db/docker-compose.yml auth-db/.env.example auth-db/.gitignore
git commit -m "fix(auth-db): remove hardcoded DB password; require POSTGRES_PASSWORD env var"
```

---

## Task 11: Add HTTP client timeouts to proxy handlers (IMP-2)

**Files:**
- Modify: `api-layer/handlers/auth/auth.go`
- Modify: `web/auth/proxy.go`

**Problem:** Both files use `http.Post(...)` and `http.DefaultClient.Do(...)` which have no timeout. A hanging auth service will block proxy goroutines forever.

**Step 1: Add a package-level client with timeout in `api-layer/handlers/auth/auth.go`**

Add after the imports, before `PostLogin`:
```go
// proxyClient is used for all calls to the auth service.
// A 10-second timeout prevents goroutine leaks if the auth service hangs.
var proxyClient = &http.Client{Timeout: 10 * time.Second}
```

Add `"time"` to the imports.

Replace all uses of `http.Post(...)` with the client:

For `PostLogin` (line 20):
```go
res, err := proxyClient.Post(config.AuthServiceURL+"/auth/login", "application/json", bytes.NewReader(body))
```

For `PostRegister` (line 44):
```go
res, err := proxyClient.Post(config.AuthServiceURL+"/auth/register", "application/json", bytes.NewReader(body))
```

For `GetMe` (line 73):
```go
res, err := proxyClient.Do(req)
```

Also pass the request context to the outbound request so client cancellations propagate:
```go
req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, config.AuthServiceURL+"/auth/me", nil)
```

**Step 2: Same changes in `web/auth/proxy.go`**

Add the same `proxyClient` var (with `"time"` import). Replace `http.Post(...)` and `http.DefaultClient.Do(req)` identically.

Also update `GetMe`'s request creation:
```go
req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, config.AuthURL+"/me", nil)
```

**Step 3: Commit**
```bash
git add api-layer/handlers/auth/auth.go web/auth/proxy.go
git commit -m "fix: add 10s timeout to all proxy HTTP clients; propagate request context to upstream calls"
```

---

## Task 12: Make registration atomic with a DB transaction (IMP-3)

**Files:**
- Modify: `auth/internal/store/postgres.go`
- Modify: `auth/internal/http/handlers_auth.go:141-188`

**Problem:** `Register` does three separate DB calls — `CreateUser`, `GetRoleByName`, `AddUserRole` — with no transaction. A failure mid-way leaves a user with no role.

**Step 1: Add `RegisterUser` transactional method to the store**

Append to `auth/internal/store/postgres.go`:

```go
// RegisterUser atomically creates a user and assigns the named role in one transaction.
// If any step fails, the entire operation is rolled back.
func (s *Store) RegisterUser(ctx context.Context, email, passwordHash, roleName string) (*User, []string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() // no-op if Commit() succeeds

	// Create user
	var user User
	err = tx.QueryRowContext(ctx, queryCreateUser, email, passwordHash).Scan(
		&user.ID,
		&user.Email,
		&user.CreatedAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create user: %w", err)
	}

	// Get role by name
	var role Role
	err = tx.QueryRowContext(ctx, queryGetRoleByName, roleName).Scan(&role.ID, &role.Name)
	if err == sql.ErrNoRows {
		return nil, nil, fmt.Errorf("role not found: %s", roleName)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("get role: %w", err)
	}

	// Assign role to user
	_, err = tx.ExecContext(ctx, queryAddUserRole, user.ID, role.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("add user role: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &user, []string{role.Name}, nil
}
```

**Step 2: Replace the 3-step sequence in `handlers_auth.go`**

Replace lines 141-168 (from `// Create user` through the `GetUserRoles` call) with:

```go
// Create user, assign contributor role — atomically in one transaction
user, roles, err := h.store.RegisterUser(c.Request.Context(), req.Email, hashedPassword, "contributor")
if err != nil {
	log.Printf("Error registering user %s: %v", req.Email, err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	return
}
```

Then continue with the JWT generation (was line 172):
```go
// Generate JWT token so user is logged in immediately after registration
token, err := h.jwtService.GenerateToken(user.ID, user.Email, roles)
```

**Step 3: Commit**
```bash
git add auth/internal/store/postgres.go auth/internal/http/handlers_auth.go
git commit -m "fix(auth): wrap user creation + role assignment in a DB transaction to prevent partial registrations"
```

---

## Task 13: Replace localStorage JWT storage with HttpOnly cookies (CRIT-3)

**Files:**
- Modify: `web/auth/proxy.go` (cookie setter + logout handler)
- Modify: `web/cmd/main.go` (add logout route)
- Modify: `web/static/js/auth.js` (remove token from localStorage/sessionStorage)

**Problem:** JWTs in `localStorage` are accessible to any JavaScript on the page. A single XSS in user-generated wiki content can exfiltrate the token. `HttpOnly` cookies are invisible to JavaScript.

**Architecture of the fix:**
- The **web proxy layer** (`web/auth/proxy.go`) receives the JWT in the auth service's JSON response, sets it as an `HttpOnly` cookie, and strips it from the JSON before returning to the browser.
- `GetMe` reads the cookie and adds the `Authorization: Bearer` header when forwarding to the API layer.
- A new `PostLogout` handler clears the cookie server-side.
- JavaScript stores **only non-sensitive user info** (email, roles) in `sessionStorage` for UI purposes.

### Sub-task 13a: Rewrite `web/auth/proxy.go`

Replace the entire file:

```go
package auth

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
	"web/config"

	"github.com/gin-gonic/gin"
)

const authCookieName = "auth_token"

// proxyClient is used for all calls to the upstream API layer.
var proxyClient = &http.Client{Timeout: 10 * time.Second}

// authServiceResponse is the shape of a successful login/register response from the API layer.
type authServiceResponse struct {
	AccessToken string          `json:"accessToken"`
	User        json.RawMessage `json:"user"`
}

// setAuthCookie writes the JWT as an HttpOnly cookie on the response.
// HttpOnly prevents JavaScript from reading the token, blocking XSS-based token theft.
func setAuthCookie(c *gin.Context, token string) {
	// Secure should be true in production (HTTPS). Set COOKIE_SECURE=true in prod env.
	secure := config.GetEnv("COOKIE_SECURE", "false") == "true"
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
	})
}

// clearAuthCookie expires the auth cookie, logging the user out.
func clearAuthCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

// handleAuthPost proxies a POST to the API layer, intercepts the token from the
// response body, sets it as an HttpOnly cookie, and returns only {user: ...} to the browser.
func handleAuthPost(c *gin.Context, upstreamPath string) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to read request"})
		return
	}

	res, err := proxyClient.Post(config.AuthURL+upstreamPath, "application/json", bytes.NewReader(body))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "auth service unavailable"})
		return
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}

	// On success, extract the token, set it as HttpOnly cookie, return only user data.
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusCreated {
		var parsed authServiceResponse
		if err := json.Unmarshal(respBody, &parsed); err == nil && parsed.AccessToken != "" {
			setAuthCookie(c, parsed.AccessToken)
			c.Data(res.StatusCode, "application/json", mustMarshal(map[string]json.RawMessage{
				"user": parsed.User,
			}))
			return
		}
		// Parsing failed — fall through and pass the raw response (handles unexpected shapes)
		log.Printf("warn: could not extract token from auth response at %s", upstreamPath)
	}

	// Error response — pass through as-is (400, 401, 403, 409, 500, etc.)
	c.Data(res.StatusCode, res.Header.Get("Content-Type"), respBody)
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// PostLogin proxies POST /auth/login → API layer POST /v1/auth/login
func PostLogin(c *gin.Context) {
	handleAuthPost(c, "/login")
}

// PostRegister proxies POST /auth/register → API layer POST /v1/auth/register
func PostRegister(c *gin.Context) {
	handleAuthPost(c, "/register")
}

// GetMe proxies GET /auth/me → API layer GET /v1/auth/me
// Reads the HttpOnly cookie and converts it to an Authorization header for the API layer.
func GetMe(c *gin.Context) {
	tokenCookie, err := c.Cookie(authCookieName)
	if err != nil || tokenCookie == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, config.AuthURL+"/me", nil)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	req.Header.Set("Authorization", "Bearer "+tokenCookie)

	res, err := proxyClient.Do(req)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "auth service unavailable"})
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}

	c.Data(res.StatusCode, res.Header.Get("Content-Type"), body)
}

// PostLogout clears the auth cookie, logging the user out server-side.
// Since the token is HttpOnly, only the server can remove it.
func PostLogout(c *gin.Context) {
	clearAuthCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
```

### Sub-task 13b: Add logout route in `web/cmd/main.go`

Add `r.POST("/auth/logout", auth.PostLogout)` to the auth proxy routes block:

```go
// Auth API proxy routes
r.POST("/auth/login", auth.PostLogin)
r.POST("/auth/register", auth.PostRegister)
r.GET("/auth/me", auth.GetMe)
r.POST("/auth/logout", auth.PostLogout)
```

### Sub-task 13c: Update `web/static/js/auth.js`

The JavaScript no longer stores or reads the JWT (it's in an HttpOnly cookie). User info (email, roles) uses `sessionStorage` for UI rendering only.

**Changes:**

1. `getToken()` — **remove entirely** (tokens are in HttpOnly cookie, not accessible to JS)

2. `getUser()` — change `localStorage` → `sessionStorage`:
```js
function getUser() {
    var user = sessionStorage.getItem('auth_user')
    if (!user) return null
    try {
        return JSON.parse(user)
    } catch {
        clearAuth()
        return null
    }
}
```

3. `saveAuth(token, user)` — ignore token parameter, store user to `sessionStorage`:
```js
function saveAuth(token, user) {
    // token is now an HttpOnly cookie set by the server — JS cannot read it
    sessionStorage.setItem('auth_user', JSON.stringify(user))
}
```

4. `clearAuth()` — change `localStorage` → `sessionStorage`:
```js
function clearAuth() {
    sessionStorage.removeItem('auth_user')
}
```

5. `login()` — response now has `{user: {...}}` (token stripped by proxy); update `saveAuth` call:
```js
    saveAuth('', data.user)
```

6. `register()` — same:
```js
    saveAuth('', data.user)
```

7. `fetchProfile()` — remove manual Authorization header (cookie is sent automatically by browser):
```js
async function fetchProfile() {
    try {
        const resp = await fetch('/auth/me')  // cookie sent automatically
        if (!resp.ok) {
            clearAuth()
            showProfileState('unauthenticated')
            return
        }
        const user = await resp.json()
        saveAuth('', user)
        populateProfile(user)
        showProfileState('content')
    } catch {
        clearAuth()
        showProfileState('unauthenticated')
    }
}
```

8. `logout()` — POST to server to clear HttpOnly cookie, then redirect:
```js
async function logout() {
    try {
        await fetch('/auth/logout', { method: 'POST' })
    } catch {}
    clearAuth()
    window.location.href = '/'
}
```

9. `updateNavAuth()` — make async, fall back to `/auth/me` when sessionStorage is empty (e.g., after a page refresh in a new tab):
```js
async function updateNavAuth() {
    var user = getUser()
    if (user) {
        _applyNavUser(user)
        return
    }
    // sessionStorage is empty (new tab / page refresh) but cookie may still be valid
    try {
        const resp = await fetch('/auth/me')
        if (resp.ok) {
            user = await resp.json()
            saveAuth('', user)
            _applyNavUser(user)
            return
        }
    } catch {}
    _applyNavGuest()
}

function _applyNavUser(user) {
    const loginLink = document.getElementById('nav-login-link')
    const userMenu = document.getElementById('nav-user-menu')
    const userEmail = document.getElementById('nav-user-email')
    if (!loginLink || !userMenu) return
    loginLink.classList.add('hidden')
    userMenu.classList.remove('hidden')
    if (userEmail) userEmail.textContent = user.email || ''
}

function _applyNavGuest() {
    const loginLink = document.getElementById('nav-login-link')
    const userMenu = document.getElementById('nav-user-menu')
    if (!loginLink || !userMenu) return
    loginLink.classList.remove('hidden')
    userMenu.classList.add('hidden')
}
```

Remove the old `updateNavAuth()` function body entirely and replace with the above. Remove `getToken()` from the file.

**Step 4: Commit**
```bash
git add web/auth/proxy.go web/cmd/main.go web/static/js/auth.js
git commit -m "fix(web): store JWT in HttpOnly cookie instead of localStorage to prevent XSS token theft"
```

---

## Summary of All Changes

| Task | Files | Issue Fixed |
|------|-------|-------------|
| 1 | `api-layer/config/config.go`, `api-layer/cmd/main.go` | CRIT-1: hardcoded JWT secret fallback |
| 2 | `auth/internal/auth/jwt.go`, `api-layer/middleware/auth.go` | CRIT-2: tokens with no exp claim accepted |
| 3 | `auth/internal/config/config.go` | CRIT-4: Config struct leaks secrets |
| 4 | `auth/internal/http/handlers_auth.go` | CRIT-5: discarded error in duplicate-user check |
| 5 | `auth/internal/http/handlers_auth.go` | IMP-6: misleading log on login failure |
| 6 | `auth/internal/http/handlers_auth.go` | IMP-7: email not normalized before whitelist lookup |
| 7 | `auth/internal/http/middleware.go` | IMP-5: Bearer scheme case inconsistency |
| 8 | `api-layer/middleware/auth.go` | IMP-4: JWT error details leaked to clients |
| 9 | `web/auth/auth_handlers.go` | IMP-8: Render() errors dropped |
| 10 | `auth-db/docker-compose.yml`, new `.env.example`, `.gitignore` | IMP-9: hardcoded DB password |
| 11 | `api-layer/handlers/auth/auth.go`, `web/auth/proxy.go` | IMP-2: no HTTP client timeouts |
| 12 | `auth/internal/store/postgres.go`, `auth/internal/http/handlers_auth.go` | IMP-3: non-transactional registration |
| 13 | `web/auth/proxy.go`, `web/cmd/main.go`, `web/static/js/auth.js` | CRIT-3: JWT in localStorage |
