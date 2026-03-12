package database

import (
	"context"
	"database/sql"
	"strings"
	wikierrors "wiki/errors"
)

type Category struct {
	ID       int        `db:"id" json:"id"`
	Slug     string     `db:"slug" json:"slug"`
	Name     string     `db:"name" json:"name"`
	ParentID *int       `db:"parent_id" json:"parent_id,omitempty"`
	Path     string     `db:"path" json:"-"`        // Internal ltree format
	FullSlug string     `json:"full_slug"`          // Computed: "people/faculty"
	Children []Category `json:"children,omitempty"` // For tree view only
}

func ListCategories(ctx context.Context, db *sql.DB) ([]Category, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, slug, name, parent_id, path
		FROM categories
		ORDER BY path;
	`)
	if err != nil {
		return nil, wikierrors.DatabaseError(err)
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var cat Category
		err := rows.Scan(&cat.ID, &cat.Slug, &cat.Name, &cat.ParentID, &cat.Path)
		if err != nil {
			return nil, wikierrors.DatabaseError(err)
		}
		cat.FullSlug = computeFullSlug(cat.Path)
		categories = append(categories, cat)
	}
	if err := rows.Err(); err != nil {
		return nil, wikierrors.DatabaseError(err)
	}
	return categories, nil
}

func GetRootCategories(ctx context.Context, db *sql.DB) ([]Category, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, slug, name, parent_id, path
		FROM categories
		WHERE parent_id IS NULL
		ORDER BY path;
	`)
	if err != nil {
		return nil, wikierrors.DatabaseError(err)
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var cat Category
		err := rows.Scan(&cat.ID, &cat.Slug, &cat.Name, &cat.ParentID, &cat.Path)
		if err != nil {
			return nil, wikierrors.DatabaseError(err)
		}
		cat.FullSlug = computeFullSlug(cat.Path)
		categories = append(categories, cat)
	}
	if err := rows.Err(); err != nil {
		return nil, wikierrors.DatabaseError(err)
	}
	return categories, nil
}

func GetCategoryTree(ctx context.Context, db *sql.DB) ([]Category, error) {
	categories, err := ListCategories(ctx, db)
	if err != nil {
		return nil, err
	}

	childrenMap := make(map[int][]Category)
	rootCategories := []Category{}

	for _, cat := range categories {
		if cat.ParentID == nil {
			rootCategories = append(rootCategories, cat)
		} else {
			childrenMap[*cat.ParentID] = append(childrenMap[*cat.ParentID], cat)
		}
	}

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

func GetCategoryBySlugPath(ctx context.Context, db *sql.DB, slugPath string) (*Category, error) {
	if !isValidSlugPath(slugPath) {
		return nil, wikierrors.InvalidCatSlug()
	}

	pathParts := strings.Split(slugPath, "/")
	ltreePath := "root." + strings.Join(pathParts, ".")

	var cat Category
	err := db.QueryRowContext(ctx, `
        SELECT id, slug, name, parent_id, path 
        FROM categories 
        WHERE path = $1::ltree
    `, ltreePath).Scan(&cat.ID, &cat.Slug, &cat.Name, &cat.ParentID, &cat.Path)
	if err == sql.ErrNoRows {
		return nil, wikierrors.CategoryNotFound()
	}
	if err != nil {
		return nil, wikierrors.DatabaseError(err)
	}

	cat.FullSlug = computeFullSlug(cat.Path)
	return &cat, nil
}

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
		return nil, wikierrors.DatabaseError(err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, wikierrors.DatabaseError(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, wikierrors.DatabaseError(err)
	}
	return ids, nil
}

func GetPageCategories(ctx context.Context, db *sql.DB, pageId string) ([]Category, error) {
	pageUUID, err := GetUUID(ctx, db, pageId)
	if err != nil {
		return nil, wikierrors.PageNotFound()
	}
	rows, err := db.QueryContext(ctx, `
		SELECT c.id, c.slug, c.name, c.parent_id, c.path
		FROM categories c
		JOIN page_categories pc ON c.id = pc.category
		WHERE pc.page_id = $1
		ORDER BY c.path
	`, pageUUID)
	if err != nil {
		return nil, wikierrors.DatabaseError(err)
	}
	defer rows.Close()

	var cats []Category
	for rows.Next() {
		var cat Category
		err := rows.Scan(&cat.ID, &cat.Slug, &cat.Name, &cat.ParentID, &cat.Path)
		if err != nil {
			return nil, wikierrors.DatabaseError(err)
		}
		cat.FullSlug = computeFullSlug(cat.Path)
		cats = append(cats, cat)
	}
	return cats, nil
}

func SetPageCategories(ctx context.Context, db *sql.DB, pageId string, categorySlugs []string) error {
	pageUUID, err := GetUUID(ctx, db, pageId)
	if err != nil {
		return wikierrors.PageNotFound()
	}

	var catIDs []int
	seen := make(map[int]struct{}, len(categorySlugs))
	for _, slug := range categorySlugs {
		cat, err := GetCategoryBySlugPath(ctx, db, slug)
		if err != nil {
			return err
		}
		if _, ok := seen[cat.ID]; ok {
			continue
		}
		seen[cat.ID] = struct{}{}
		catIDs = append(catIDs, cat.ID)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return wikierrors.DatabaseError(err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		DELETE FROM page_categories
		WHERE page_id = $1;
	`, pageUUID)
	if err != nil {
		return wikierrors.DatabaseError(err)
	}

	for _, id := range catIDs {
		_, err = tx.ExecContext(ctx, `
		INSERT INTO page_categories (page_id, category)
		VALUES ($1, $2);
		`, pageUUID, id)
		if err != nil {
			return wikierrors.DatabaseError(err)
		}
	}

	return tx.Commit()
}

func isValidSlugPath(path string) bool {
	parts := strings.SplitSeq(path, "/")
	for part := range parts {
		if !isValidSlug(part) {
			return false
		}
	}
	return true
}

func isValidSlug(slug string) bool {
	if slug == "" || strings.HasPrefix(slug, "-") || strings.HasSuffix(slug, "-") || strings.Contains(slug, "--") {
		return false
	}
	for _, r := range slug {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return false
		}
	}
	return true
}

func computeFullSlug(path string) string {
	parts := strings.Split(path, ".")
	if len(parts) > 1 && parts[0] == "root" {
		parts = parts[1:]
	}
	return strings.Join(parts, "/")
}
