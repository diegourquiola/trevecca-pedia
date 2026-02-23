package filesystem

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"wiki/database"

	"github.com/google/uuid"
)

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
		SELECT pages.slug
		FROM revisions JOIN pages ON revisions.page_id = pages.uuid
		WHERE revisions.uuid=$1
		LIMIT 1;
	`, uuid).Scan(&slug)
	if err != nil {
		return "", err
	}
	filename := filepath.Join(dataDir, "revisions", fmt.Sprintf("%s_%s.txt", slug, uuid))
	return filename, nil
}

func GetSnapshotFilename(ctx context.Context, db *sql.DB, dataDir string, uuid uuid.UUID) (string, error) {
	var slug string
	err := db.QueryRowContext(ctx, `
		SELECT pages.slug
		FROM snapshots JOIN pages ON snapshots.page = pages.uuid
		WHERE snapshots.uuid=$1
		LIMIT 1;
	`, uuid).Scan(&slug)
	if err != nil {
		return "", err
	}
	filename := filepath.Join(dataDir, "snapshots", fmt.Sprintf("%s_%s.md", slug, uuid))
	return filename, nil
}
