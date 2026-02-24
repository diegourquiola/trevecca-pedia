package requests

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"wiki/database"
	wikierrors "wiki/errors"
	"wiki/filesystem"
	"wiki/utils"

	"github.com/aymanbagabas/go-udiff"
	"github.com/google/uuid"
)

func init() {
	fmt.Printf("[requests/post.go] module loaded\n")
}

// revertRenames attempts to rename revision and snapshot files back from newSlug
// to oldSlug for every revision/snapshot belonging to pageId. Errors from
// individual renames are silently ignored because a file may not yet have been
// renamed (i.e. the failure occurred before that point in UpdatePage).
func revertRenames(ctx context.Context, db *sql.DB, dataDir string, pageId uuid.UUID, oldSlug, newSlug string) {
	if oldSlug == newSlug {
		return
	}

	revRows, err := db.QueryContext(ctx, `SELECT uuid FROM revisions WHERE page_id=$1;`, pageId)
	if err == nil {
		defer revRows.Close()
		for revRows.Next() {
			var revId uuid.UUID
			if revRows.Scan(&revId) == nil {
				os.Rename(
					filepath.Join(dataDir, "revisions", fmt.Sprintf("%s_%s.txt", newSlug, revId)),
					filepath.Join(dataDir, "revisions", fmt.Sprintf("%s_%s.txt", oldSlug, revId)),
				)
			}
		}
	}

	snapRows, err := db.QueryContext(ctx, `SELECT uuid FROM snapshots WHERE page=$1;`, pageId)
	if err == nil {
		defer snapRows.Close()
		for snapRows.Next() {
			var snapId uuid.UUID
			if snapRows.Scan(&snapId) == nil {
				os.Rename(
					filepath.Join(dataDir, "snapshots", fmt.Sprintf("%s_%s.md", newSlug, snapId)),
					filepath.Join(dataDir, "snapshots", fmt.Sprintf("%s_%s.md", oldSlug, snapId)),
				)
			}
		}
	}
}

func cleanupRevisionFailure(ctx context.Context, db *sql.DB, dataDir string, pageId uuid.UUID, revId uuid.UUID, originalSlug, newSlug, pageContent string, prevLastRevision *uuid.UUID) {
	revertRenames(ctx, db, dataDir, pageId, originalSlug, newSlug)

	revNewPath := filepath.Join(dataDir, "revisions", fmt.Sprintf("%s_%s.txt", newSlug, revId))
	revOldPath := filepath.Join(dataDir, "revisions", fmt.Sprintf("%s_%s.txt", originalSlug, revId))
	_ = os.Remove(revNewPath)
	if originalSlug != newSlug {
		_ = os.Remove(revOldPath)
	}

	originalPagePath := filepath.Join(dataDir, "pages", fmt.Sprintf("%s.md", originalSlug))
	newPagePath := filepath.Join(dataDir, "pages", fmt.Sprintf("%s.md", newSlug))
	if originalSlug != newSlug {
		_ = os.Rename(newPagePath, originalPagePath)
		_ = os.Remove(newPagePath)
	}
	_ = os.WriteFile(originalPagePath, []byte(pageContent), 0644)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	_, _ = tx.ExecContext(ctx, `
		UPDATE pages
		SET last_revision_id=$1
		WHERE uuid=$2;
	`, prevLastRevision, pageId)
	_, _ = tx.ExecContext(ctx, `
		DELETE FROM revisions
		WHERE uuid=$1;
	`, revId)
	_ = tx.Commit()
}

func DeletePage(ctx context.Context, db *sql.DB, dataDir string, delReq utils.DeletePageRequest) error {
	pageUUID, err := database.GetUUID(ctx, db, delReq.Slug)
	if err != nil {
		return wikierrors.DatabaseError(err)
	}
	pageInfo, err := database.GetPageInfo(ctx, db, pageUUID)
	if err != nil {
		return wikierrors.DatabaseError(err)
	}

	pageDeleted, err := database.GetPageDeleted(ctx, db, pageInfo.UUID)
	if err != nil {
		return wikierrors.DatabaseError(err)
	}
	if pageDeleted {
		return wikierrors.PageDeleted()
	}

	// remove from database
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return wikierrors.InternalError(err)
	}

	var deletedAt *time.Time
	err = tx.QueryRowContext(ctx, `
		UPDATE pages
		SET deleted_at=NOW()
		WHERE uuid=$1
		RETURNING deleted_at;
	`, pageInfo.UUID).Scan(&deletedAt)
	if err != nil {
		tx.Rollback()
		return wikierrors.DatabaseError(err)
	}

	var revId uuid.UUID
	err = tx.QueryRowContext(ctx, `
			INSERT INTO revisions (page_id, author, slug, name, archive_date, deleted_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
			RETURNING uuid;
			`, pageInfo.UUID, delReq.User, pageInfo.Slug, pageInfo.Name, pageInfo.ArchiveDate, deletedAt).
		Scan(&revId)
	if err != nil {
		fmt.Printf("Error writing to db: %s\n", err)
		return wikierrors.DatabaseError(err)
	}

	// Create diff with empty change
	pageContent, err := filesystem.GetPageContent(ctx, db, dataDir, pageUUID)
	if err != nil {
		tx.Rollback()
		return wikierrors.FilesystemError(err)
	}
	// Create a diff showing no changes (old == new)
	pageFilename := fmt.Sprintf("%s.md", pageInfo.Slug)
	diff := udiff.Unified(pageFilename, pageFilename, pageContent, pageContent)
	// Write the revision file
	filename := fmt.Sprintf("%s_%s.txt", pageInfo.Slug, revId)
	err = os.MkdirAll(filepath.Join(dataDir, "revisions"), 0755)
	if err != nil {
		tx.Rollback()
		return wikierrors.FilesystemError(err)
	}
	err = os.WriteFile(filepath.Join(dataDir, "revisions", filename), []byte(diff), 0644)
	if err != nil {
		tx.Rollback()
		return wikierrors.FilesystemError(err)
	}

	err = tx.Commit()
	if err != nil {
		os.Remove(filepath.Join(dataDir, "revisions", fmt.Sprintf("%s_%s.txt", pageInfo.Slug, revId)))
		return wikierrors.DatabaseError(err)
	}
	return nil
}

func PostRevision(ctx context.Context, db *sql.DB, dataDir string, revReq utils.RevisionRequest) error {
	fmt.Printf("[PostRevision] Starting with dataDir: %s\n", dataDir)
	fmt.Printf("[PostRevision] revReq.PageId: %s, revReq.Author: %s\n", revReq.PageId, revReq.Author)
	fmt.Printf("[PostRevision] revReq.Slug: %s, revReq.Name: %s\n", revReq.Slug, revReq.Name)
	var err error

	pageId, err := database.GetUUID(ctx, db, revReq.PageId)
	if err != nil {
		fmt.Printf("[PostRevision] GetUUID failed: %v\n", err)
		return wikierrors.DatabaseError(err)
	}
	fmt.Printf("[PostRevision] Got pageId: %s\n", pageId)
	var pageSlug string
	var prevLastRevision *uuid.UUID
	err = db.QueryRowContext(ctx, `
		SELECT slug, last_revision_id FROM pages WHERE uuid=$1;
	`, pageId).Scan(&pageSlug, &prevLastRevision)
	if err != nil {
		fmt.Printf("[PostRevision] SELECT pages failed: %v\n", err)
		return wikierrors.DatabaseError(err)
	}
	fmt.Printf("[PostRevision] Got pageSlug: %s, prevLastRevision: %v\n", pageSlug, prevLastRevision)
	originalSlug := pageSlug
	pageContent, err := filesystem.GetPageContent(ctx, db, dataDir, pageId)
	if err != nil {
		fmt.Printf("[PostRevision] GetPageContent failed: %v\n", err)
		return wikierrors.FilesystemError(err)
	}
	fmt.Printf("[PostRevision] Got pageContent length: %d\n", len(pageContent))

	revisionTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		fmt.Printf("[PostRevision] BeginTx (revisionTx) failed: %v\n", err)
		return err
	}
	defer revisionTx.Rollback()
	fmt.Printf("[PostRevision] Started revision transaction\n")
	revId, err := utils.CreateRevision(ctx, db, revisionTx, dataDir, revReq)
	if err != nil {
		fmt.Printf("[PostRevision] CreateRevision failed: %v\n", err)
		return wikierrors.DatabaseFilesystemError(err)
	}
	fmt.Printf("[PostRevision] Created revision with ID: %s\n", revId)
	err = revisionTx.Commit()
	if err != nil {
		fmt.Printf("[PostRevision] revisionTx.Commit() failed: %v\n", err)
		os.Remove(filepath.Join(dataDir, "revisions", fmt.Sprintf("%s_%s.txt", revReq.Slug, revId)))
		revisionTx.Rollback()
		return err
	}
	fmt.Printf("[PostRevision] Committed revision transaction\n")

	// cleanupSlug is the slug the page will have after a successful UpdatePage.
	// Set it before the transaction commits so snapshot cleanup uses the right name.
	cleanupSlug := revReq.Slug
	fmt.Printf("[PostRevision] cleanupSlug set to: %s\n", cleanupSlug)

	pageTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		fmt.Printf("[PostRevision] BeginTx (pageTx) failed: %v\n", err)
		return wikierrors.DatabaseError(err)
	}
	defer pageTx.Rollback()
	fmt.Printf("[PostRevision] Started page transaction\n")
	err = utils.UpdatePage(ctx, db, pageTx, dataDir, revId)
	if err != nil {
		fmt.Printf("[PostRevision] UpdatePage failed: %v\n", err)
		cleanupRevisionFailure(ctx, db, dataDir, pageId, revId, originalSlug, revReq.Slug, pageContent, prevLastRevision)
		return wikierrors.DatabaseFilesystemError(err)
	}
	fmt.Printf("[PostRevision] UpdatePage succeeded\n")
	err = pageTx.Commit()
	if err != nil {
		fmt.Printf("[PostRevision] pageTx.Commit() failed: %v\n", err)
		cleanupRevisionFailure(ctx, db, dataDir, pageId, revId, originalSlug, revReq.Slug, pageContent, prevLastRevision)
		return wikierrors.DatabaseFilesystemError(err)
	}
	fmt.Printf("[PostRevision] Committed page transaction\n")

	missingRevs, err := database.GetMissingRevisions(ctx, db, revId)
	if err != nil {
		fmt.Printf("[PostRevision] GetMissingRevisions failed: %v\n", err)
		return wikierrors.DatabaseError(err)
	}
	fmt.Printf("[PostRevision] Missing revisions count: %d\n", len(missingRevs))
	if len(missingRevs) >= 10 {
		snapTx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return wikierrors.DatabaseError(err)
		}
		defer snapTx.Rollback()
		snapId, err := utils.CreateSnapshot(ctx, db, snapTx, dataDir, pageId, revId)
		if err != nil {
			fmt.Printf("[PostRevision] CreateSnapshot failed: %v\n", err)
			os.Remove(filepath.Join(dataDir, "snapshots", fmt.Sprintf("%s_%s.md", cleanupSlug, snapId)))
			return wikierrors.DatabaseFilesystemError(err)
		}
		err = snapTx.Commit()
		if err != nil {
			fmt.Printf("[PostRevision] snapTx.Commit() failed: %v\n", err)
			os.Remove(filepath.Join(dataDir, "snapshots", fmt.Sprintf("%s_%s.md", cleanupSlug, snapId)))
			return wikierrors.DatabaseFilesystemError(err)
		}
		fmt.Printf("[PostRevision] Snapshot created successfully\n")
	}

	fmt.Printf("[PostRevision] Completed successfully\n")
	return nil
}
