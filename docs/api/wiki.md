# Wiki Service

## Prefix

All routes to the API Layer begin with a version and the service.  
<br>
So, calls to the `wiki` service begin with: `/v1/wiki`  

---

## Routes

### HTTP `GET` Requests

| Type      | Route                                     | Arguments                 | Description       |
| ---       | ---                                       | ---                       | ---               |
| `GET`     | `/pages{?index=ind&count=n&slugs=a,b,c}`  | `index`, `count`, `slugs` | Returns a list of page info and content.  |
| `GET`     | `/pages/:id`                              | `:id`                     | Returns the info and content for the specified page. |
| `GET`     | `/pages/:id/revisions{?index=ind&count=n}`| `:id`, `index`, `count`   | Returns a list of the revisions for the specified page. |
| `GET`     | `/pages/:id/revisions/:rev`               | `:id`, `:rev`             | Returns the info and content for the specified revision of the specified page. |
| `GET`     | `/indexable-pages{?index=ind&count=n}`    | `index`, `count`          | Returns a list of indexable pages for search indexing. |

#### Arguments
`index`: the index to be the first item  
`count`: the count of entries to retrieve  
\[DON'T USE\] `category`: filter pages by category (category name or id)  
`slugs`: comma-separated list of specific slugs to retrieve  
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
    - UTC+0 Time: `YYYY-MM-DD HH:MM:SS`  
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
`new_page`: the markdown file with the new page content  

