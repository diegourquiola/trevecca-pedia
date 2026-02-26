package utils

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aymanbagabas/go-udiff"
	"github.com/google/uuid"
)

func CreateNewPage(ctx context.Context, db *sql.DB, dataDir string, req NewPageRequest) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// TODO: authenticate request and author

	// create page db entry
	var pageId uuid.UUID
	err = tx.QueryRowContext(ctx, `
		INSERT INTO pages (slug, name, archive_date)
		VALUES ($1, $2, $3)
		RETURNING uuid;
	`, req.Slug, req.Name, req.ArchiveDate).Scan(&pageId)
	if err != nil {
		return err
	}

	// create revision db entry
	var revId uuid.UUID
	err = tx.QueryRowContext(ctx, `
		INSERT INTO revisions (page_id, author, slug, name, archive_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING uuid;
	`, pageId, req.Author, req.Slug, req.Name, req.ArchiveDate).Scan(&revId)
	if err != nil {
		return err
	}

	// create snapshot db entry
	var snapId uuid.UUID
	err = tx.QueryRowContext(ctx, `
		INSERT INTO snapshots (page, revision)
		VALUES ($1, $2)
		RETURNING uuid;
	`, pageId, revId).Scan(&snapId)
	if err != nil {
		return err
	}

	// FILE STUFF
	pagePath := filepath.Join(dataDir, "pages", fmt.Sprintf("%s.md", req.Slug))
	revPath := filepath.Join(dataDir, "revisions", fmt.Sprintf("%s_%s.txt", req.Slug, revId))
	snapPath := filepath.Join(dataDir, "snapshots", fmt.Sprintf("%s_%s.md", req.Slug, snapId))

	err = os.MkdirAll(filepath.Join(dataDir, "pages"), 0755)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(dataDir, "revisions"), 0755)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(dataDir, "snapshots"), 0755)
	if err != nil {
		return err
	}

	diff := udiff.Unified(pagePath, pagePath, "", req.Content)

	err = os.WriteFile(pagePath, []byte(req.Content), 0644)
	if err != nil {
		return err
	}
	err = os.WriteFile(snapPath, []byte(req.Content), 0644)
	if err != nil {
		return err
	}
	err = os.WriteFile(revPath, []byte(diff), 0644)
	if err != nil {
		return err
	}


	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}
