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
	var err error

	pageId, err := database.GetUUID(ctx, db, revReq.PageId)
	if err != nil {
		return wikierrors.DatabaseError(err)
	}
	var pageSlug string
	err = db.QueryRowContext(ctx, `
		SELECT slug FROM pages WHERE uuid=$1;
	`, pageId).Scan(&pageSlug)
	if err != nil {
		return wikierrors.DatabaseError(err)
	}
	originalSlug := pageSlug
	pageContent, err := filesystem.GetPageContent(ctx, db, dataDir, pageId)
	if err != nil {
		return wikierrors.FilesystemError(err)
	}

	revisionTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer revisionTx.Rollback()
	revId, err := utils.CreateRevision(ctx, db, revisionTx, dataDir, revReq)
	if err != nil {
		return wikierrors.DatabaseFilesystemError(err)
	}
	err = revisionTx.Commit()
	if err != nil {
		os.Remove(filepath.Join(dataDir, "revisions", fmt.Sprintf("%s_%s.txt", originalSlug, revId)))
		revisionTx.Rollback()
		return err
	}

	// cleanupSlug is the slug the page will have after a successful UpdatePage.
	// Set it before the transaction commits so snapshot cleanup uses the right name.
	cleanupSlug := revReq.Slug

	pageTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return wikierrors.DatabaseError(err)
	}
	defer pageTx.Rollback()
	err = utils.UpdatePage(ctx, db, pageTx, dataDir, revId)
	if err != nil {
		// UpdatePage may have partially renamed revision/snapshot files before
		// failing; attempt to reverse any renames that already happened.
		revertRenames(ctx, db, dataDir, pageId, originalSlug, revReq.Slug)
		os.Remove(filepath.Join(dataDir, "revisions", fmt.Sprintf("%s_%s.txt", originalSlug, revId)))
		os.WriteFile(filepath.Join(dataDir, "pages", fmt.Sprintf("%s.md", originalSlug)), []byte(pageContent), 0644)
		return wikierrors.DatabaseFilesystemError(err)
	}
	err = pageTx.Commit()
	if err != nil {
		// UpdatePage succeeded (all renames completed) but the DB commit failed;
		// reverse all renames and restore the page file.
		revertRenames(ctx, db, dataDir, pageId, originalSlug, revReq.Slug)
		os.Remove(filepath.Join(dataDir, "revisions", fmt.Sprintf("%s_%s.txt", originalSlug, revId)))
		os.WriteFile(filepath.Join(dataDir, "pages", fmt.Sprintf("%s.md", originalSlug)), []byte(pageContent), 0644)
		return wikierrors.DatabaseFilesystemError(err)
	}

	missingRevs, err := database.GetMissingRevisions(ctx, db, revId)
	if err != nil {
		return wikierrors.DatabaseError(err)
	}
	if len(missingRevs) >= 10 {
		snapTx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return wikierrors.DatabaseError(err)
		}
		defer snapTx.Rollback()
		snapId, err := utils.CreateSnapshot(ctx, db, snapTx, dataDir, pageId, revId)
		if err != nil {
			os.Remove(filepath.Join(dataDir, "snapshots", fmt.Sprintf("%s_%s.md", cleanupSlug, snapId)))
			return wikierrors.DatabaseFilesystemError(err)
		}
		err = snapTx.Commit()
		if err != nil {
			os.Remove(filepath.Join(dataDir, "snapshots", fmt.Sprintf("%s_%s.md", cleanupSlug, snapId)))
			return wikierrors.DatabaseFilesystemError(err)
		}
	}

	return nil
}
