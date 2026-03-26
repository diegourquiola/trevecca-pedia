# Wiki Service

These routes are served by the API layer under `/v1/wiki`.

For web frontend work (browser code): do not call these directly.
- The API layer requires a `Bearer` JWT for write routes.
- The JWT is stored as an `HttpOnly` cookie by the web service (`/auth/login`), so browser JavaScript cannot read it and therefore cannot set the `Authorization` header.
- If you need browser-driven wiki writes, add web-service proxy routes that read the cookie and forward `Authorization: Bearer <cookie>` upstream (same pattern as `GET /auth/me`).

## Prefix

All routes to the API Layer begin with a version and the service.  
<br>
So, calls to the `wiki` service begin with: `/v1/wiki`  

---

## Routes

## Authentication

- Read routes are public.
- Write routes require `Authorization: Bearer <jwt>` and the JWT must contain the `contributor` role (or higher privileged roles like `moderator` or `admin`).

### HTTP `GET` Requests

| Type      | Route                                     | Arguments                 | Description       |
| ---       | ---                                       | ---                       | ---               |
| `GET`     | `/pages{?index=ind&count=n&category=c&slugs=a,b,c&exact=bool}` | `index`, `count`, `category`, `slugs`, `exact` | Returns a list of page info and content.  |
| `GET`     | `/pages/:id`                              | `:id`                     | Returns the info and content for the specified page. |
| `GET`     | `/pages/:id/revisions{?index=ind&count=n}`| `:id`, `index`, `count`   | Returns a list of the revisions for the specified page. |
| `GET`     | `/pages/:id/revisions/:rev`               | `:id`, `:rev`             | Returns the info and content for the specified revision of the specified page. |
| `GET`     | `/indexable-pages{?index=ind&count=n}`    | `index`, `count`          | Returns a list of indexable pages for search indexing. |
| `GET`     | `/categories{?tree=bool&root=bool}`       | `tree`, `root`            | Returns all categories. |
| `GET`     | `/pages/:id/categories`                   | `:id`                     | Returns categories assigned to the specified page. |
| `GET`     | `/revisions{?author=email&index=ind&count=n}` | `author`, `index`, `count` | Returns revisions by author email, sorted by date (newest first). |

#### Arguments
`index`: the index to be the first item  
`count`: the count of entries to retrieve  
`tree`: if "true", returns categories in tree structure with parent-child relationships  
`root`: if "true", returns only root-level categories  
`category`: filter pages by category (category slug)  
`slugs`: comma-separated list of specific slugs to retrieve  
`exact`: if "true", enables exact matching for category/slug filters  
`:id`: the slug (or uuid) of the page  
`:rev`: the uuid of the page revision  
`{}`: content in curly braces is optional  
`ind`, `n`: any integer

---

#### `/indexable-pages`
**Description:** Returns a list of pages formatted for search indexing.
**Type:** `GET`
**Arguments:**
`index`: the index to be the first item
`count`: the count of entries to retrieve  

---

### HTTP `POST` Requests

| Type      | Route                                     | Arguments             | Description       |
| ---       | ---                                       | ---                   | ---               |
| `POST`    | `/pages/new`                              | N/A                   | Creates a new page entry with the submitted info.  |
| `POST`    | `/pages/:id/delete`                       | `:id`                 | Deletes the specified page.    |
| `POST`    | `/pages/:id/revisions`                    | `:id`                 | Creates a new revision of the specified page. |
| `POST`    | `/pages/:id/categories`                   | `:id`                 | Updates categories for the specified page. |

#### `/pages/new`
This is implemented using a multipart form, with the fields being passed in as form data.  This is useful because it allows the `new_page` file to be passed in as a file, rather than just a string.  

**Type:** `POST`
**Arguments:** None

**Fields:**
`slug`: the unique, human-readable identifier for the page  
    - all lowercase, kebab-case  
`name`: title of the page  
    - any string  
`author`: author identification  
    - not really implemented yet. using student id for now, but that definitely won't be the actual implementation.  
`archive_date`: date to mark page as archived/not relevant (if any)  
    - Date only: `YYYY-MM-DD`  
    - optional; can be blank;  
`new_page`: the markdown file with the page content  

#### `/pages/:id/delete`
Deletes the specified page.  

**Type:** `POST`
**Arguments:**
`:id`: the slug (or uuid) of the page to delete

**Fields:**
`slug`: the slug of the page to delete (should match `:id`)  
`user`: the user identification performing the deletion  

#### `/pages/:id/revisions`
Creates a new revision of the specified page with the difference between the current content and that specified in the `new_page` file.  
This is implemented using a multipart form, with the fields being passed in as form data.  This is useful because it allows the `new_page` file to be passed in as a file, rather than just a string.

**Type:** `POST`
**Arguments:**
`:id`: the slug (or uuid) of the page
    - this isn't really used as of right now, as the id of the page is also passed in as a form field (`page_id`)

**Fields:**
`page_id`: the slug (or uuid) of the page being revised  
`author`: author identification  
    - not really implemented yet. using student email username for now, but that definitely won't be the actual implementation.  
`new_content`: the markdown file with the new page content  

#### `/pages/:id/categories`
Updates categories assigned to the specified page. Accepts a JSON array of category IDs.

**Type:** `POST`
**Arguments:**
`:id`: the slug (or uuid) of the page

**Request Body:** JSON array of category IDs (UUIDs)
```json
["category-uuid-1", "category-uuid-2"]
```

---

## Example (Server-to-Server)

The API layer enforces auth on POST routes. If you're calling it from a service (or curl), include the bearer token.

Create a page:
```bash
curl -X POST "${API_LAYER_URL:-http://127.0.0.1:2745}/v1/wiki/pages/new" \
  -H "Authorization: Bearer $TOKEN" \
  -F "slug=example-page" \
  -F "name=Example Page" \
  -F "author=studentid" \
  -F "archive_date=2026-02-28" \
  -F "new_page=@./page.md"
```

Create a revision:
```bash
curl -X POST "${API_LAYER_URL:-http://127.0.0.1:2745}/v1/wiki/pages/example-page/revisions" \
  -H "Authorization: Bearer $TOKEN" \
  -F "page_id=example-page" \
  -F "author=username" \
  -F "new_content=@./page.md"
```

Get all categories:
```bash
curl -X GET "${API_LAYER_URL:-http://127.0.0.1:2745}/v1/wiki/categories"
```

Get categories in tree structure:
```bash
curl -X GET "${API_LAYER_URL:-http://127.0.0.1:2745}/v1/wiki/categories?tree=true"
```

Get categories for a specific page:
```bash
curl -X GET "${API_LAYER_URL:-http://127.0.0.1:2745}/v1/wiki/pages/example-page/categories"
```

Update categories for a page:
```bash
curl -X POST "${API_LAYER_URL:-http://127.0.0.1:2745}/v1/wiki/pages/example-page/categories" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '["category-uuid-1", "category-uuid-2"]'
```
