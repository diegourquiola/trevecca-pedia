package utils

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"wiki/database"
	wikierrors "wiki/errors"
	"wiki/filesystem"

	"github.com/aymanbagabas/go-udiff"
	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/google/uuid"
)

func CreateRevision(ctx context.Context, db *sql.DB, tx *sql.Tx, dataDir string, revReq RevisionRequest) (uuid.UUID, error) {
	fmt.Printf("[CreateRevision] Starting with dataDir: %s\n", dataDir)
	fmt.Printf("[CreateRevision] revReq.PageId: %s, revReq.Slug: %s\n", revReq.PageId, revReq.Slug)
	pageUUID, err := database.GetUUID(ctx, db, revReq.PageId)
	if err != nil {
		fmt.Printf("[CreateRevision] GetUUID failed: %v\n", err)
		return uuid.UUID{}, err
	}
	fmt.Printf("[CreateRevision] Got pageUUID: %s\n", pageUUID)
	var revUUID uuid.UUID
	err = tx.QueryRowContext(ctx, `
			INSERT INTO revisions (page_id, author, slug, name, archive_date, deleted_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING uuid;
			`, pageUUID, revReq.Author, revReq.Slug, revReq.Name, revReq.ArchiveDate, revReq.DeletedAt).Scan(&revUUID)
	if err != nil {
		fmt.Printf("[CreateRevision] INSERT revisions failed: %s\n", err)
		return uuid.UUID{}, err
	}
	fmt.Printf("[CreateRevision] Created revision with UUID: %s\n", revUUID)

	// create the diff and make the revision
	pageContent, err := filesystem.GetPageContent(ctx, db, dataDir, pageUUID)
	if err != nil {
		fmt.Printf("[CreateRevision] GetPageContent failed: %v\n", err)
		return uuid.UUID{}, err
	}
	fmt.Printf("[CreateRevision] Got pageContent length: %d\n", len(pageContent))
	pageFilename := filepath.Join(dataDir, "pages", fmt.Sprintf("%s.md", revReq.Slug))
	fmt.Printf("[CreateRevision] pageFilename: %s\n", pageFilename)
	diff := udiff.Unified(pageFilename, pageFilename, pageContent, revReq.NewContent)

	filename := fmt.Sprintf("%s_%s.txt", revReq.Slug, revUUID)
	fmt.Printf("[CreateRevision] Writing revision file: %s\n", filename)
	err = os.MkdirAll(filepath.Join(dataDir, "revisions"), 0755)
	if err != nil {
		fmt.Printf("[CreateRevision] MkdirAll failed: %v\n", err)
		return uuid.UUID{}, fmt.Errorf("creating revisions directory: %w", err)
	}
	err = os.WriteFile(filepath.Join(dataDir, "revisions", filename), []byte(diff), 0644)
	if err != nil {
		fmt.Printf("[CreateRevision] WriteFile failed: %v\n", err)
		return uuid.UUID{}, err
	}
	fmt.Printf("[CreateRevision] Successfully wrote revision file\n")

	return revUUID, nil
}

func UpdatePage(ctx context.Context, db *sql.DB, tx *sql.Tx, dataDir string, revId uuid.UUID) error {
	fmt.Printf("[UpdatePage] Starting with dataDir: %s, revId: %s\n", dataDir, revId)
	var revInfo database.RevInfo
	err := tx.QueryRowContext(ctx, `
		SELECT uuid, page_id, date_time, author, slug, name, archive_date, deleted_at
		FROM revisions
		WHERE uuid=$1;
	`, revId).Scan(&revInfo.UUID, &revInfo.PageId, &revInfo.DateTime, &revInfo.Author,
		&revInfo.Slug, &revInfo.Name, &revInfo.ArchiveDate, &revInfo.DeletedAt)
	if err != nil {
		fmt.Printf("[UpdatePage] SELECT revisions failed: %v\n", err)
		return err
	}
	fmt.Printf("[UpdatePage] Got revInfo - slug: %s, pageId: %s\n", revInfo.Slug, revInfo.PageId)
	var currSlug string
	err = tx.QueryRowContext(ctx, `
		SELECT slug FROM pages WHERE uuid=$1;
	`, revInfo.PageId).Scan(&currSlug)
	if err != nil {
		fmt.Printf("[UpdatePage] SELECT pages slug failed: %v\n", err)
		return err
	}
	fmt.Printf("[UpdatePage] Got currSlug: %s, revInfo.Slug: %s\n", currSlug, revInfo.Slug)
	contentAtRev, err := GetContentAtRevision(ctx, db, dataDir, *revInfo.PageId, revId)
	if err != nil {
		fmt.Printf("[UpdatePage] GetContentAtRevision failed: %v\n", err)
		return wikierrors.DatabaseFilesystemError(err)
	}
	fmt.Printf("[UpdatePage] Got contentAtRev length: %d\n", len(contentAtRev))

	_, err = tx.ExecContext(ctx, `
		UPDATE pages
		SET slug=$1, name=$2, archive_date=$3, deleted_at=$4
		WHERE uuid=$5;
	`, revInfo.Slug, revInfo.Name, revInfo.ArchiveDate, revInfo.DeletedAt, revInfo.PageId)
	if err != nil {
		fmt.Printf("[UpdatePage] UPDATE pages failed: %v\n", err)
		return wikierrors.DatabaseError(err)
	}
	fmt.Printf("[UpdatePage] Updated pages table\n")

	if revInfo.Slug != currSlug {
		fmt.Printf("[UpdatePage] Slug changed from %s to %s\n", currSlug, revInfo.Slug)
		err = os.Rename(filepath.Join(dataDir, "pages", fmt.Sprintf("%s.md", currSlug)),
			filepath.Join(dataDir, "pages", fmt.Sprintf("%s.md", revInfo.Slug)))
		if err != nil {
			fmt.Printf("[UpdatePage] Rename page file failed: %v\n", err)
			return err
		}
		fmt.Printf("[UpdatePage] Renamed page file\n")
		revs, err := tx.QueryContext(ctx, `
			SELECT uuid FROM revisions WHERE page_id=$1;
		`, revInfo.PageId)
		if err != nil {
			fmt.Printf("[UpdatePage] SELECT revisions failed: %v\n", err)
			return err
		}
		defer revs.Close()
		for revs.Next() {
			var currRevId uuid.UUID
			err = revs.Scan(&currRevId)
			if err != nil {
				fmt.Printf("[UpdatePage] Scan revision UUID failed: %v\n", err)
				return err
			}
			if currRevId == *revInfo.UUID {
				continue
			}
			err = os.Rename(filepath.Join(dataDir, "revisions", fmt.Sprintf("%s_%s.txt", currSlug, currRevId)),
				filepath.Join(dataDir, "revisions", fmt.Sprintf("%s_%s.txt", revInfo.Slug, currRevId)))
			if err != nil {
				fmt.Printf("[UpdatePage] Rename revision file failed: %v\n", err)
				return err
			}
		}
		revs.Close()
		fmt.Printf("[UpdatePage] Renamed revision files\n")
		snaps, err := tx.QueryContext(ctx, `
			SELECT uuid FROM snapshots WHERE page=$1;
		`, revInfo.PageId)
		if err != nil {
			fmt.Printf("[UpdatePage] SELECT snapshots failed: %v\n", err)
			return err
		}
		defer snaps.Close()
		for snaps.Next() {
			var currSnapId uuid.UUID
			err = snaps.Scan(&currSnapId)
			if err != nil {
				fmt.Printf("[UpdatePage] Scan snapshot UUID failed: %v\n", err)
				return err
			}
			err = os.Rename(filepath.Join(dataDir, "snapshots", fmt.Sprintf("%s_%s.md", currSlug, currSnapId)),
				filepath.Join(dataDir, "snapshots", fmt.Sprintf("%s_%s.md", revInfo.Slug, currSnapId)))
			if err != nil {
				fmt.Printf("[UpdatePage] Rename snapshot file failed: %v\n", err)
				return err
			}
		}
		snaps.Close()
		fmt.Printf("[UpdatePage] Renamed snapshot files\n")
	}

	pageFilename := fmt.Sprintf("%s.md", revInfo.Slug)
	pageFilepath := filepath.Join(dataDir, "pages", pageFilename)
	err = os.WriteFile(pageFilepath, []byte(contentAtRev), 0644)
	if err != nil {
		fmt.Printf("[UpdatePage] WriteFile page failed: %v\n", err)
		return wikierrors.FilesystemError(err)
	}
	fmt.Printf("[UpdatePage] Wrote page file\n")

	return nil
}

func CreateSnapshot(ctx context.Context, db *sql.DB, tx *sql.Tx, dataDir string, pageId uuid.UUID, revId uuid.UUID) (uuid.UUID, error) {
	var snapUUID uuid.UUID
	err := tx.QueryRowContext(ctx, `
			INSERT INTO snapshots (page, revision)
			VALUES ($1, $2)
			RETURNING uuid;
			`, pageId, revId).Scan(&snapUUID)
	if err != nil {
		return uuid.UUID{}, err
	}

	snapContent, err := GetContentAtRevision(ctx, db, dataDir, pageId, revId)
	if err != nil {
		return uuid.UUID{}, err
	}

	var pageSlug string
	err = tx.QueryRowContext(ctx, `
		SELECT slug FROM pages WHERE uuid=$1;
	`, pageId).Scan(&pageSlug)
	if err != nil {
		return uuid.UUID{}, err
	}

	filename := fmt.Sprintf("%s_%s.md", pageSlug, snapUUID)
	err = os.MkdirAll(filepath.Join(dataDir, "snapshots"), 0755)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("creating snapshots directory: %w", err)
	}
	err = os.WriteFile(filepath.Join(dataDir, "snapshots", filename), []byte(snapContent), 0644)
	if err != nil {
		return uuid.UUID{}, err
	}

	return snapUUID, nil
}

func GetContentAtRevision(ctx context.Context, db *sql.DB, dataDir string, pageId uuid.UUID, revId uuid.UUID) (string, error) {
	fmt.Printf("[GetContentAtRevision] Starting - pageId: %s, revId: %s\n", pageId, revId)
	lastSnap, err := database.GetMostRecentSnapshot(ctx, db, revId)
	if err == sql.ErrNoRows {
		fmt.Printf("[GetContentAtRevision] No snapshot found (ErrNoRows)\n")
		return "", wikierrors.RevisionNotFound()
	}
	if err != nil {
		fmt.Printf("[GetContentAtRevision] GetMostRecentSnapshot failed: %v\n", err)
		return "", wikierrors.DatabaseError(err)
	}
	fmt.Printf("[GetContentAtRevision] Got lastSnap.UUID: %s\n", lastSnap.UUID)
	missingRevs, err := database.GetMissingRevisions(ctx, db, revId)
	if err != nil {
		fmt.Printf("[GetContentAtRevision] GetMissingRevisions failed: %v\n", err)
		return "", wikierrors.DatabaseError(err)
	}
	fmt.Printf("[GetContentAtRevision] Got %d missing revisions\n", len(missingRevs))
	fmt.Printf("[GetContentAtRevision] Calling GetSnapshotContent with snapUUID: %s\n", lastSnap.UUID)
	revContent, err := filesystem.GetSnapshotContent(ctx, db, dataDir, lastSnap.UUID)
	if err != nil {
		fmt.Printf("[GetContentAtRevision] GetSnapshotContent failed: %v\n", err)
		return "", wikierrors.FilesystemError(err)
	}
	fmt.Printf("[GetContentAtRevision] Got revContent length: %d\n", len(revContent))

	// i hope and pray that this works
	// update: it worked. most errors were elsewhere :)
	for i, r := range missingRevs {
		fmt.Printf("[GetContentAtRevision] Processing missing revision %d/%d: %s\n", i+1, len(missingRevs), *r.UUID)
		revDiff, err := filesystem.GetRevisionContent(ctx, db, dataDir, *r.UUID)
		if err != nil {
			fmt.Printf("[GetContentAtRevision] GetRevisionContent failed for rev %s: %v\n", *r.UUID, err)
			return "", wikierrors.FilesystemError(err)
		}
		fmt.Printf("[GetContentAtRevision] Got revDiff length: %d\n", len(revDiff))
		files, _, err := gitdiff.Parse(bytes.NewReader([]byte(revDiff)))
		if err != nil {
			return "", fmt.Errorf("couldn't parse revision: %w", err)
		}
		if len(files) == 0 {
			continue
		}
		src := bytes.NewReader([]byte(revContent))
		var dst bytes.Buffer

		err = gitdiff.Apply(&dst, src, files[0])
		if err != nil {
			if errors.Is(err, &gitdiff.Conflict{}) {
				return "", fmt.Errorf("conflict while applying revision: %w", err)
			}
			return "", fmt.Errorf("applying revision: %w", err)
		}
		revContent = dst.String()
	}
	return revContent, nil
}

func GetPageInfoPreview(ctx context.Context, db *sql.DB, dataDir string, pageId uuid.UUID) (*PageInfoPrev, error) {
	pageInfo, err := database.GetPageInfo(ctx, db, pageId)
	if err != nil {
		return nil, err
	}
	preview, err := filesystem.GetPagePreview(ctx, db, dataDir, pageId, 250)
	if err != nil {
		return nil, err
	}
	var lastEditTime time.Time
	if pageInfo.LastRevisionId == nil {
		pageInfo.LastRevisionId = &uuid.Nil
		lastEditTime = time.Time{}
	} else {
		err := db.QueryRowContext(ctx, `
			SELECT date_time FROM revisions WHERE uuid=$1;
		`, pageInfo.LastRevisionId).Scan(&lastEditTime)
		if err != nil {
			return nil, err
		}
	}
	if pageInfo.ArchiveDate == nil {
		pageInfo.ArchiveDate = &time.Time{}
	}
	return &PageInfoPrev{
		UUID:         pageInfo.UUID,
		Slug:         pageInfo.Slug,
		Name:         pageInfo.Name,
		LastEditTime: lastEditTime,
		ArchiveDate:  *pageInfo.ArchiveDate,
		Preview:      preview,
	}, nil
}

func GetIndexInfo(ctx context.Context, db *sql.DB, dataDir string, pageId string) (*IndexInfo, error) {
	pageUUID, err := database.GetUUID(ctx, db, pageId)
	if err != nil {
		return nil, err
	}
	var indexInfo IndexInfo
	var lastRev uuid.UUID
	var archiveDate *time.Time
	err = db.QueryRowContext(ctx, `
		SELECT slug, name, last_revision_id, archive_date
		FROM pages WHERE uuid=$1;
	`, pageUUID).Scan(&indexInfo.Slug, &indexInfo.Name, &lastRev, &archiveDate)
	if err != nil {
		return nil, err
	}
	if lastRev != uuid.Nil {
		err = db.QueryRowContext(ctx, `
		SELECT date_time FROM revisions WHERE uuid=$1; 
		`, lastRev).Scan(&indexInfo.LastModified)
		if err != nil {
			return nil, err
		}
	} else {
		indexInfo.LastModified = time.Time{}
	}
	if archiveDate != nil {
		indexInfo.ArchiveDate = *archiveDate
	} else {
		indexInfo.ArchiveDate = time.Time{}
	}
	indexInfo.Content, err = filesystem.GetPageContent(ctx, db, dataDir, pageUUID)
	if err != nil {
		return nil, err
	}
	return &indexInfo, nil
}
