package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func GetPageInfo(ctx context.Context, db *sql.DB, uuid uuid.UUID) (*PageInfo, error) {
	var p PageInfo
	err := db.QueryRowContext(
		ctx,
		"SELECT uuid, slug, name, last_revision_id, archive_date FROM pages WHERE uuid=$1", uuid.String()).
		Scan(&p.UUID, &p.Slug, &p.Name, &p.LastRevisionId, &p.ArchiveDate)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func GetPageDeleted(ctx context.Context, db *sql.DB, pageUUID uuid.UUID) (bool, error) {
	var pageDeleted *time.Time
	err := db.QueryRowContext(ctx, `
		SELECT deleted_at FROM pages WHERE uuid=$1;
	`, pageUUID).Scan(&pageDeleted)
	if err != nil {
		return false, err
	}
	if pageDeleted != nil {
		return true, nil
	}
	return false, nil
}

func GetPageRevisionsInfo(ctx context.Context, db *sql.DB, pageId uuid.UUID) ([]RevInfo, error) {
	var revs []RevInfo
	rows, err := db.QueryContext(
		ctx,
		`SELECT uuid, date_time, author, slug, name, archive_date, deleted_at 
				FROM revisions WHERE page_id=$1 ORDER BY date_time`,
		pageId.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var row RevInfo
		err := rows.Scan(&row.UUID, &row.DateTime, &row.Author, &row.Slug, &row.Name, &row.ArchiveDate, &row.DeletedAt)
		if err != nil {
			return nil, err
		}
		revs = append(revs, row)
	}

	return revs, nil
}

func GetRevisionInfo(ctx context.Context, db *sql.DB, revId uuid.UUID) (*RevInfo, error) {
	var rev RevInfo
	err := db.QueryRowContext(
		ctx,
		`SELECT uuid, page_id, date_time, author, slug, name, archive_date, deleted_at 
				FROM revisions WHERE uuid=$1`,
		revId).Scan(&rev.UUID, &rev.PageId, &rev.DateTime, &rev.Author, &rev.Slug, &rev.Name, &rev.ArchiveDate, &rev.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &rev, nil
}

func GetMostRecentSnapshot(ctx context.Context, db *sql.DB, revId uuid.UUID) (*SnapInfo, error) {
	revInfo, err := GetRevisionInfo(ctx, db, revId)
	if err != nil {
		return nil, err
	}
	pageId := revInfo.PageId
	var snapId uuid.UUID
	var snap SnapInfo
	var snapCount int
	err = db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM snapshots
		WHERE page=$1`,
		pageId).
		Scan(&snapCount)
	if err != nil {
		return nil, err
	}
	switch snapCount {
	case 1:
		err = db.QueryRowContext(
			ctx,
			`SELECT uuid FROM snapshots
			WHERE page=$1`,
			pageId).
			Scan(&snapId)
		if err != nil {
			return nil, err
		}
	case 0:
		return nil, sql.ErrNoRows
	default:
		err = db.QueryRowContext(
			ctx,
			`SELECT snapshots.uuid FROM snapshots
			JOIN revisions ON snapshots.revision = revisions.uuid
			WHERE snapshots.page=$1
			ORDER BY revisions.date_time
			DESC`,
			pageId).
			Scan(&snapId)
		if err != nil {
			return nil, err
		}
	}
	err = db.QueryRowContext(
		ctx,
		"SELECT * FROM snapshots WHERE uuid=$1",
		snapId).
		Scan(&snap.UUID, &snap.Page, &snap.Revision)
	if err != nil {
		return nil, err
	}
	return &snap, nil
}

func GetMissingRevisions(ctx context.Context, db *sql.DB, revId uuid.UUID) ([]RevInfo, error) {
	var revs []RevInfo
	var count int
	var snapRevTime time.Time
	revInfo, err := GetRevisionInfo(ctx, db, revId)
	if err != nil {
		return nil, err
	}
	if revInfo.PageId == nil {
		return nil, sql.ErrNoRows
	}
	pageId := *revInfo.PageId

	snap, err := GetMostRecentSnapshot(ctx, db, revId)
	if err != nil {
		return nil, err
	}
	// shouldn't ever be nil, but I'll leave this here I guess
	if snap.Revision == nil {
		snapRevTime = time.Time{}
	} else if *snap.Revision == revId {
		return nil, nil
	} else {
		snap_rev, err := GetRevisionInfo(ctx, db, *snap.Revision)
		if err != nil {
			return nil, err
		}
		snapRevTime = *snap_rev.DateTime
	}
	err = db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM revisions WHERE page_id=$1 AND date_time > $2",
		pageId, snapRevTime).
		Scan(&count)
	if err != nil {
		return nil, err
	}
	revs = make([]RevInfo, count)

	revIds, err := db.QueryContext(
		ctx,
		"SELECT uuid FROM revisions WHERE page_id=$1 AND date_time > $2 ORDER BY date_time ASC",
		pageId, snapRevTime)
	if err != nil {
		return nil, err
	}
	defer revIds.Close()

	for i := 0; revIds.Next(); i++ {
		var id uuid.UUID
		revIds.Scan(&id)
		rev, err := GetRevisionInfo(ctx, db, id)
		if err != nil {
			return nil, err
		}
		revs[i] = *rev
	}

	return revs, nil
}
