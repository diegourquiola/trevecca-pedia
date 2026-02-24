package filesystem

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"wiki/database"

	"github.com/google/uuid"
)

func init() {
	fmt.Printf("[filesystem/filenames.go] module loaded\n")
}

func GetPageFilename(ctx context.Context, db *sql.DB, dataDir string, id string) (string, error) {
	uuid, err := database.GetUUID(ctx, db, id)
	if err != nil {
		return "", err
	}
	var slug string
	err = db.QueryRowContext(ctx, `
		SELECT slug
		FROM pages
		WHERE uuid=$1;
	`, uuid).Scan(&slug)
	if err != nil {
		return "", err
	}
	filename := filepath.Join(dataDir, "pages", fmt.Sprintf("%s.md", slug))
	return filename, nil
}

func GetRevisionFilename(ctx context.Context, db *sql.DB, dataDir string, uuid uuid.UUID) (string, error) {
	var slug string
	err := db.QueryRowContext(ctx, `
		SELECT slug
		FROM revisions
		WHERE uuid=$1
		LIMIT 1;
	`, uuid).Scan(&slug)
	if err != nil {
		fmt.Printf("[GetRevisionFilename] Query failed for rev %s: %v\n", uuid, err)
		return "", err
	}
	filename := filepath.Join(dataDir, "revisions", fmt.Sprintf("%s_%s.txt", slug, uuid))
	fmt.Printf("[GetRevisionFilename] Built filename: %s (slug from db: %s)\n", filename, slug)
	return filename, nil
}

func GetSnapshotFilename(ctx context.Context, db *sql.DB, dataDir string, uuid uuid.UUID) (string, error) {
	var slug string
	err := db.QueryRowContext(ctx, `
		SELECT revisions.slug
		FROM snapshots JOIN revisions ON snapshots.revision = revisions.uuid
		WHERE snapshots.uuid=$1
		LIMIT 1;
	`, uuid).Scan(&slug)
	if err != nil {
		fmt.Printf("[GetSnapshotFilename] Query failed for snap %s: %v\n", uuid, err)
		return "", err
	}
	filename := filepath.Join(dataDir, "snapshots", fmt.Sprintf("%s_%s.md", slug, uuid))
	fmt.Printf("[GetSnapshotFilename] Built filename: %s (slug from db: %s)\n", filename, slug)
	return filename, nil
}
