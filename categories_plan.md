## Final Implementation Plan: Hierarchical Categories

### Phase 1: Database Migration (002_hierarchical_categories.sql)

**1.1 Enable ltree and update schema:**
```sql
CREATE EXTENSION IF NOT EXISTS ltree;

-- Modify categories table
ALTER TABLE categories 
ADD COLUMN parent_id INTEGER REFERENCES categories(id) ON DELETE CASCADE,
ADD COLUMN path LTREE NOT NULL DEFAULT 'root';

-- Add indexes
CREATE INDEX idx_categories_path ON categories USING GIST(path);
CREATE INDEX idx_categories_parent ON categories(parent_id);

-- Update existing data
UPDATE categories SET path = 'root.' || slug WHERE parent_id IS NULL;

-- Constraint: lowercase letters, numbers, hyphens only
ALTER TABLE categories 
ADD CONSTRAINT chk_slug_format CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$');
```

**1.2 Circular reference prevention trigger:**
```sql
CREATE OR REPLACE FUNCTION check_category_circular_reference()
RETURNS TRIGGER AS $$
BEGIN
    -- Prevent a category from being its own parent
    IF NEW.parent_id = NEW.id THEN
        RAISE EXCEPTION 'Category cannot be its own parent';
    END IF;
    
    -- Prevent circular references via path check
    IF NEW.parent_id IS NOT NULL THEN
        IF EXISTS (
            SELECT 1 FROM categories 
            WHERE id = NEW.parent_id 
            AND path @> NEW.path
        ) THEN
            RAISE EXCEPTION 'Circular reference detected: parent is already a descendant';
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_check_category_circular
BEFORE INSERT OR UPDATE ON categories
FOR EACH ROW EXECUTE FUNCTION check_category_circular_reference();
```

**Error codes to handle in Go:**
- `P0001` with message 'Category cannot be its own parent'
- `P0001` with message 'Circular reference detected'

**1.3 Helper view:**
```sql
CREATE VIEW page_category_full AS
SELECT 
    pc.page_id,
    c.id AS category_id,
    c.slug AS category_slug,
    c.name AS category_name,
    c.path,
    c.parent_id,
    CASE 
        WHEN c.parent_id IS NULL THEN c.slug
        ELSE (SELECT slug FROM categories WHERE id = c.parent_id) || '/' || c.slug
    END AS full_slug
FROM page_categories pc
JOIN categories c ON pc.category = c.id;
```

---

### Phase 2: Database Functions (wiki/database/categories.go)

```go
package database

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "strings"
    
    "github.com/lib/pq"
)

type Category struct {
    ID       int      `db:"id" json:"id"`
    Slug     string   `db:"slug" json:"slug"`
    Name     string   `db:"name" json:"name"`
    ParentID *int     `db:"parent_id" json:"parent_id,omitempty"`
    Path     string   `db:"path" json:"-"`
    FullSlug string   `json:"full_slug"` // Computed: "people/faculty"
}

// Circular reference error types
var (
    ErrSelfParent       = errors.New("category cannot be its own parent")
    ErrCircularRef      = errors.New("circular reference detected")
    ErrInvalidSlug      = errors.New("invalid slug format (lowercase, hyphens only)")
    ErrCategoryNotFound = errors.New("category not found")
)

// ListCategories returns flat list (default) or tree (if tree=true)
func ListCategories(ctx context.Context, db *sql.DB, tree bool) ([]Category, error) {
    if tree {
        return getCategoryTree(ctx, db)
    }
    
    // Flat list ordered by path
    rows, err := db.QueryContext(ctx, `
        SELECT id, slug, name, parent_id, path 
        FROM categories 
        ORDER BY path
    `)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var categories []Category
    for rows.Next() {
        var cat Category
        err := rows.Scan(&cat.ID, &cat.Slug, &cat.Name, &cat.ParentID, &cat.Path)
        if err != nil {
            return nil, err
        }
        cat.FullSlug = computeFullSlug(cat.Path)
        categories = append(categories, cat)
    }
    return categories, nil
}

// GetRootCategories returns only top-level categories (no parent)
func GetRootCategories(ctx context.Context, db *sql.DB) ([]Category, error) {
    rows, err := db.QueryContext(ctx, `
        SELECT id, slug, name, parent_id, path 
        FROM categories 
        WHERE parent_id IS NULL
        ORDER BY slug
    `)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var categories []Category
    for rows.Next() {
        var cat Category
        err := rows.Scan(&cat.ID, &cat.Slug, &cat.Name, &cat.ParentID, &cat.Path)
        if err != nil {
            return nil, err
        }
        cat.FullSlug = cat.Slug // Root categories have same full_slug as slug
        categories = append(categories, cat)
    }
    return categories, nil
}

// GetCategoryBySlugPath looks up by full path like "people/faculty"
func GetCategoryBySlugPath(ctx context.Context, db *sql.DB, slugPath string) (*Category, error) {
    // Validate format
    if !isValidSlugPath(slugPath) {
        return nil, ErrInvalidSlug
    }
    
    // Convert "people/faculty" to ltree path "root.people.faculty"
    pathParts := strings.Split(slugPath, "/")
    ltreePath := "root." + strings.Join(pathParts, ".")
    
    var cat Category
    err := db.QueryRowContext(ctx, `
        SELECT id, slug, name, parent_id, path 
        FROM categories 
        WHERE path = $1::ltree
    `, ltreePath).Scan(&cat.ID, &cat.Slug, &cat.Name, &cat.ParentID, &cat.Path)
    
    if err == sql.ErrNoRows {
        return nil, ErrCategoryNotFound
    }
    if err != nil {
        return nil, err
    }
    
    cat.FullSlug = computeFullSlug(cat.Path)
    return &cat, nil
}

// GetDescendantCategoryIDs returns category ID + all descendants
// For "people", returns [people_id, faculty_id, staff_id, students_id]
func GetDescendantCategoryIDs(ctx context.Context, db *sql.DB, slugPath string) ([]int, error) {
    cat, err := GetCategoryBySlugPath(ctx, db, slugPath)
    if err != nil {
        return nil, err
    }
    
    rows, err := db.QueryContext(ctx, `
        SELECT id FROM categories 
        WHERE path <@ $1::ltree
    `, cat.Path)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var ids []int
    for rows.Next() {
        var id int
        if err := rows.Scan(&id); err != nil {
            return nil, err
        }
        ids = append(ids, id)
    }
    return ids, nil
}

// GetPageCategories returns all categories for a page
func GetPageCategories(ctx context.Context, db *sql.DB, pageID string) ([]Category, error) {
    rows, err := db.QueryContext(ctx, `
        SELECT c.id, c.slug, c.name, c.parent_id, c.path 
        FROM categories c
        JOIN page_categories pc ON c.id = pc.category
        WHERE pc.page_id = $1
        ORDER BY c.path
    `, pageID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var categories []Category
    for rows.Next() {
        var cat Category
        err := rows.Scan(&cat.ID, &cat.Slug, &cat.Name, &cat.ParentID, &cat.Path)
        if err != nil {
            return nil, err
        }
        cat.FullSlug = computeFullSlug(cat.Path)
        categories = append(categories, cat)
    }
    return categories, nil
}

// SetPageCategories replaces all categories for a page
// categories: ["people/faculty", "departments"]
func SetPageCategories(ctx context.Context, db *sql.DB, pageID string, categorySlugs []string) error {
    // Validate all slugs first
    for _, slug := range categorySlugs {
        if !isValidSlugPath(slug) {
            return ErrInvalidSlug
        }
    }
    
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Delete existing
    _, err = tx.ExecContext(ctx, `
        DELETE FROM page_categories WHERE page_id = $1
    `, pageID)
    if err != nil {
        return err
    }
    
    // Insert new (if any provided)
    if len(categorySlugs) > 0 {
        // Build value list with resolved IDs
        valueList := make([]string, 0, len(categorySlugs))
        args := []interface{}{pageID}
        argCount := 1
        
        for _, slug := range categorySlugs {
            cat, err := GetCategoryBySlugPath(ctx, db, slug)
            if err != nil {
                return fmt.Errorf("invalid category %q: %w", slug, err)
            }
            argCount++
            valueList = append(valueList, fmt.Sprintf("($1, $%d)", argCount))
            args = append(args, cat.ID)
        }
        
        query := fmt.Sprintf(`
            INSERT INTO page_categories (page_id, category) 
            VALUES %s
        `, strings.Join(valueList, ", "))
        
        _, err = tx.ExecContext(ctx, query, args...)
        if err != nil {
            return err
        }
    }
    
    return tx.Commit()
}

// Helper: Handle PostgreSQL errors for circular references
func IsCircularReferenceError(err error) bool {
    if pgErr, ok := err.(*pq.Error); ok {
        if pgErr.Code == "P0001" {
            msg := pgErr.Message
            return strings.Contains(msg, "own parent") || 
                   strings.Contains(msg, "Circular reference")
        }
    }
    return false
}

// isValidSlugPath validates "people/faculty" format
func isValidSlugPath(path string) bool {
    parts := strings.Split(path, "/")
    for _, part := range parts {
        if !isValidSlug(part) {
            return false
        }
    }
    return true
}

// isValidSlug validates single slug segment
func isValidSlug(slug string) bool {
    // Must match: lowercase, numbers, hyphens only
    // Cannot start/end with hyphen, no consecutive hyphens
    if slug == "" || strings.HasPrefix(slug, "-") || strings.HasSuffix(slug, "-") {
        return false
    }
    for _, r := range slug {
        if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
            return false
        }
    }
    return !strings.Contains(slug, "--")
}

// computeFullSlug converts "root.people.faculty" to "people/faculty"
func computeFullSlug(ltreePath string) string {
    parts := strings.Split(ltreePath, ".")
    if len(parts) > 1 && parts[0] == "root" {
        parts = parts[1:]
    }
    return strings.Join(parts, "/")
}

// getCategoryTree builds nested structure for tree view
func getCategoryTree(ctx context.Context, db *sql.DB) ([]Category, error) {
    categories, err := ListCategories(ctx, db, false)
    if err != nil {
        return nil, err
    }
    
    // Build parent -> children map
    childrenMap := make(map[int][]Category)
    rootCategories := []Category{}
    
    for _, cat := range categories {
        if cat.ParentID == nil {
            rootCategories = append(rootCategories, cat)
        } else {
            childrenMap[*cat.ParentID] = append(childrenMap[*cat.ParentID], cat)
        }
    }
    
    // Recursively build tree
    var buildTree func(cat Category) Category
    buildTree = func(cat Category) Category {
        if children, ok := childrenMap[cat.ID]; ok {
            cat.Children = make([]Category, len(children))
            for i, child := range children {
                cat.Children[i] = buildTree(child)
            }
        }
        return cat
    }
    
    result := make([]Category, len(rootCategories))
    for i, root := range rootCategories {
        result[i] = buildTree(root)
    }
    
    return result, nil
}
```

---

### Phase 3: Updated Page Queries (wiki/requests/get.go)

```go
// GetPagesCategory with tree support (Option C)
func GetPagesCategory(ctx context.Context, db *sql.DB, dataDir string, 
    catSlug string, ind int, count int, exact bool) ([]utils.PageInfoPrev, error) {
    
    var categoryIDs []int
    var err error
    
    if exact {
        // Exact match only - no descendants
        cat, err := database.GetCategoryBySlugPath(ctx, db, catSlug)
        if err != nil {
            return nil, err
        }
        categoryIDs = []int{cat.ID}
    } else {
        // Option C: Get category and all descendants
        categoryIDs, err = database.GetDescendantCategoryIDs(ctx, db, catSlug)
        if err != nil {
            return nil, err
        }
    }
    
    // Query using IN clause with category IDs
    query := `
        SELECT COUNT(DISTINCT p.uuid) FROM pages p
        JOIN page_categories pc ON p.uuid = pc.page_id
        WHERE pc.category = ANY($1) AND p.deleted_at IS NULL
    `
    
    var pagesCount int
    err = db.QueryRowContext(ctx, query, pq.Array(categoryIDs)).Scan(&pagesCount)
    if err != nil {
        return nil, wikierrors.DatabaseError(err)
    }
    
    pagesCount -= ind
    if pagesCount <= 0 {
        return []utils.PageInfoPrev{}, nil
    }
    
    uuids, err := db.QueryContext(ctx, `
        SELECT DISTINCT p.uuid FROM pages p
        JOIN page_categories pc ON p.uuid = pc.page_id
        WHERE pc.category = ANY($1) AND p.deleted_at IS NULL
        ORDER BY p.slug
        LIMIT $2 OFFSET $3
    `, pq.Array(categoryIDs), count, ind)
    
    if err != nil {
        return nil, wikierrors.DatabaseError(err)
    }
    defer uuids.Close()
    
    var pages []utils.PageInfoPrev
    for uuids.Next() {
        var id uuid.UUID
        uuids.Scan(&id)
        pageInfo, err := utils.GetPageInfoPreview(ctx, db, dataDir, id)
        if err != nil {
            return nil, wikierrors.DatabaseError(err)
        }
        if pageInfo != nil {
            pages = append(pages, *pageInfo)
        }
    }
    
    return pages, nil
}
```

---

### Phase 4: API Handlers (wiki/handlers/)

**New: category_handlers.go**
```go
package handlers

func CategoriesHandler(c *gin.Context) {
    ctx := context.Background()
    db, err := utils.GetDatabase()
    if err != nil {
        // error handling...
        return
    }
    defer db.Close()
    
    tree := c.DefaultQuery("tree", "false") == "true"
    rootOnly := c.DefaultQuery("root", "false") == "true"
    
    var categories interface{}
    if rootOnly {
        categories, err = database.GetRootCategories(ctx, db)
    } else {
        categories, err = database.ListCategories(ctx, db, tree)
    }
    
    if err != nil {
        // error handling...
        return
    }
    
    c.JSON(http.StatusOK, categories)
}
```

**New: page_category_handlers.go**
```go
func GetPageCategoriesHandler(c *gin.Context) {
    ctx := context.Background()
    db, err := utils.GetDatabase()
    if err != nil {
        // error handling...
        return
    }
    defer db.Close()
    
    pageID := c.Param("id")
    
    // Resolve UUID from slug if needed
    pageUUID, err := database.GetUUID(ctx, db, pageID)
    if err != nil {
        // PageNotFound error...
        return
    }
    
    categories, err := database.GetPageCategories(ctx, db, pageUUID.String())
    if err != nil {
        // error handling...
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"categories": categories})
}

func SetPageCategoriesHandler(c *gin.Context) {
    ctx := context.Background()
    db, err := utils.GetDatabase()
    if err != nil {
        // error handling...
        return
    }
    defer db.Close()
    
    pageID := c.Param("id")
    
    // Resolve UUID from slug
    pageUUID, err := database.GetUUID(ctx, db, pageID)
    if err != nil {
        // PageNotFound error...
        return
    }
    
    var req struct {
        Categories []string `json:"categories" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
            "error": "categories array required",
        })
        return
    }
    
    err = database.SetPageCategories(ctx, db, pageUUID.String(), req.Categories)
    if err != nil {
        // Handle specific errors
        if err == database.ErrInvalidSlug {
            c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
                "error": "Invalid category slug format",
            })
            return
        }
        if err == database.ErrCategoryNotFound {
            c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
                "error": "One or more categories not found",
            })
            return
        }
        // Other errors...
        return
    }
    
    c.Status(http.StatusOK)
}
```

**Modified: PagesHandler in get_handlers.go**
```go
func PagesHandler(c *gin.Context) {
    // ... existing setup ...
    
    catQuery := c.DefaultQuery("category", "")
    exact := c.DefaultQuery("exact", "false") == "true"
    
    if catQuery != "" {
        pages, err = requests.GetPagesCategory(ctx, db, dataDir, catQuery, ind, count, exact)
        // ... error handling ...
    }
    
    // ... rest of handler ...
}
```

---

### Phase 5: API Layer Updates (api-layer/handlers/wiki/)

```go
// categories.go
func GetCategories(c *gin.Context) {
    tree := c.DefaultQuery("tree", "false")
    root := c.DefaultQuery("root", "false")
    res, err := http.Get(fmt.Sprintf("%s/categories?tree=%s&root=%s", 
        config.WikiServiceURL, tree, root))
    // ... proxy logic ...
}

// Modify existing GetPages to pass through exact param
func GetPages(c *gin.Context) {
    catQuery := c.DefaultQuery("category", "")
    slugsQuery := c.DefaultQuery("slugs", "")
    exact := c.DefaultQuery("exact", "false")
    // ... existing params ...
    
    res, err := http.Get(fmt.Sprintf("%s/pages?category=%s&slugs=%s&exact=%s&index=%d&count=%d",
        config.WikiServiceURL, catQuery, slugsQuery, exact, ind, count))
    // ... proxy logic ...
}
```

---

### Phase 6: Routing Updates

**wiki/cmd/main.go:**
```go
func main() {
    r := gin.Default()
    
    // Existing routes...
    r.GET("/pages", handlers.PagesHandler)
    r.GET("/pages/:id", handlers.PageHandler)
    
    // New category routes
    r.GET("/categories", handlers.CategoriesHandler)
    r.GET("/pages/:id/categories", handlers.GetPageCategoriesHandler)
    r.POST("/pages/:id/categories", handlers.SetPageCategoriesHandler) // Requires auth
    
    // ... rest ...
}
```

**api-layer:**
```go
// Add category proxy
v1.GET("/wiki/categories", wiki.GetCategories)
// Page routes already exist, just need to update GetPages
```

---

### Phase 7: Frontend (Web Layer)

**Templates to create:**
1. `category_selector.templ` - Tree/multi-select component
2. `category_pills.templ` - Display assigned categories

**Templates to modify:**
1. `wiki_create_content.templ` - Add category selector (optional)
2. `wiki_edit_content.templ` - Add category selector + current pills
3. `wiki_entry_content.templ` - Show category pills below title

**JavaScript needed:**
- Category tree toggle
- Multi-select handling
- Form submission with categories array

---

### Phase 8: Seed Data Update

**Updated categories in init/03-schema.sql:**
```sql
-- Root categories
INSERT INTO categories (slug, name, path) VALUES 
    ('people', 'People', 'root.people'),
    ('buildings', 'Buildings', 'root.buildings'),
    ('departments', 'Departments', 'root.departments'),
    ('student-life', 'Student Life', 'root.student_life'),
    ('miscellaneous', 'Miscellaneous', 'root.miscellaneous');

-- People subcategories
INSERT INTO categories (slug, name, parent_id, path) VALUES 
    ('faculty', 'Faculty', (SELECT id FROM categories WHERE slug='people'), 'root.people.faculty'),
    ('staff', 'Staff', (SELECT id FROM categories WHERE slug='people'), 'root.people.staff'),
    ('students', 'Students', (SELECT id FROM categories WHERE slug='people'), 'root.people.students');

-- Buildings subcategories  
INSERT INTO categories (slug, name, parent_id, path) VALUES 
    ('academic', 'Academic', (SELECT id FROM categories WHERE slug='buildings'), 'root.buildings.academic'),
    ('residential', 'Residential', (SELECT id FROM categories WHERE slug='buildings'), 'root.buildings.residential');

-- Update page assignments to use full paths
-- dan-boone → people/faculty
UPDATE page_categories 
SET category = (SELECT id FROM categories WHERE path = 'root.people.faculty')
WHERE page_id = '07918316-875e-4581-87ab-5b8d1d8bdd3a';

-- newsies → student-life (no change needed, it's a root)
-- (existing assignments stay the same)

-- Add more assignments
INSERT INTO page_categories VALUES ('dcf043c9-b897-4444-9252-fcfc996b0db8', (SELECT id FROM categories WHERE path = 'root.miscellaneous')); -- about
```

---

### Implementation Order (Revised)

1. **Migration 002** - Schema + trigger + circular ref prevention
2. **Database functions** - categories.go with tree support
3. **Category handlers** - GET /categories, GET/POST /pages/:id/categories
4. **Update page queries** - GetPagesCategory with exact/tree param
5. **API layer** - Proxy updates + routing
6. **Frontend templates** - Selector + display components
7. **Page create/edit integration** - Wire up category selection
8. **Seed data** - Hierarchical categories + updated assignments
9. **Testing** - Verify circular ref prevention, tree queries, edge cases

---

### Testing Checklist

- [ ] GET /categories returns flat list
- [ ] GET /categories?tree=true returns nested structure
- [ ] GET /categories?root=true returns only top-level categories
- [ ] GET /pages?category=people returns all in People tree (Option C)
- [ ] GET /pages?category=people&exact=true returns only direct assignments
- [ ] POST /pages/:id/categories with circular ref fails gracefully
- [ ] Invalid slug format rejected (uppercase, special chars)
- [ ] Page creation without categories works (optional)
- [ ] Categories displayed on page view
- [ ] Subcategory assignment inherits to parent queries

---

