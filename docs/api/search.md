# Search Service

These routes are served by the API layer under `/v1/search`.

For web frontend work (browser code): you typically do not call these directly.
- The web server already proxies search for the UI by calling the API layer from server-side Go.
- If you do call the API layer from the browser, these endpoints are public and do not require auth.

## Prefix

All routes to the API Layer begin with a version and the service.  
<br>
So, calls to the `search` service begin with: `/v1/search`  

---

## Routes

### HTTP `GET` Requests

| Type      | Route                                     | Arguments                 | Description       |
| ---       | ---                                       | ---                       | ---               |
| `GET`     | `/search{?q=query}`                       | `q`                       | Returns a list of search results matching the query. |
| `GET`     | `/health`                                 | N/A                       | Returns the health status of the service. |

Note: the API layer currently exposes `GET /v1/search/search`. It does not expose a `/health` route for search.

#### Arguments
`q`: the search query string  
`query`: any string to search for

---

#### `/search`
**Description:** Returns a list of page slugs that match the search query.  
The search is performed across page slugs, names (titles), and content with different weighting factors:  
- Name (title): boost of 5.0 (highest priority)  
- Slug: boost of 2.0 (medium priority)  
- Content: boost of 1.0 (lowest priority)  

**Type:** `GET`
**Arguments:**
`q`: the search query string (required)

**Response Format:**
```json
{
  "total": 42,
  "results": ["page-slug-1", "page-slug-2", "page-slug-3"]
}
```

**Fields:**
`total`: the total number of matching results  
`results`: an array of page slugs that match the search query

---

#### `/health`
**Description:** Returns the health status of the search service.  
**Type:** `GET`
**Arguments:** None

**Response Format:**
```json
{
  "status": "healthy"
}
```

---

### HTTP `POST` Requests

| Type      | Route                                     | Arguments             | Description       |
| ---       | ---                                       | ---                   | ---               |
| `POST`    | `/reindex`                                | N/A                   | Triggers a full reindex of all pages from the wiki service. |

Note: the API layer currently does not expose `POST /v1/search/reindex`.

#### `/reindex`
**Description:** Triggers a complete reindex of all pages from the wiki service.  
This fetches all indexable pages from the wiki service's `/indexable-pages` endpoint and rebuilds the search index.  
**Type:** `POST`
**Arguments:** None

**Response Format:**
```json
{
  "message": "reindex completed successfully"
}
```

---

## Service Architecture

The search service uses [Bleve](https://blevesearch.com/) as its full-text search engine. It maintains a local index that is populated by fetching data from the wiki service.

### Data Flow
1. The search service fetches pages from the wiki service's `/indexable-pages` endpoint
2. Pages are indexed with their slug as the document ID
3. Search queries are executed against the local Bleve index
4. Results return matching page slugs

### Indexed Fields
- `slug`: keyword-analyzed field (exact match, boost 2.0)
- `name`: text field with English analyzer (boost 5.0)
- `content`: text field with English analyzer (boost 1.0)
- `last_modified`: datetime field
- `archive_date`: datetime field

### Startup Behavior
On startup, the service automatically attempts to perform a full index if the index doesn't exist or is empty.

---

## Configuration

The search service requires the following configuration:

- `INDEX_DIR`: Path to the directory where the search index is stored
- `WIKI_URL`: Base URL of the wiki service (e.g., `http://wiki:8080/v1/wiki`) for fetching indexable pages

See `.env.example` for all configuration options.
