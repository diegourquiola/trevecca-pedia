package requests

import (
	"context"
	"database/sql"
	"wiki/database"
	wikierrors "wiki/errors"
	"wiki/filesystem"
	"wiki/utils"

	"github.com/google/uuid"
)

func GetPage(ctx context.Context, db *sql.DB, dataDir string, id string) (utils.Page, error) {
	var page utils.Page
	var info *database.PageInfo
	var content string
	var lastRev *database.RevInfo
	var pageId uuid.UUID
	var err error

	pageId, err = database.GetUUID(ctx, db, id)
	if err == sql.ErrNoRows {
		return utils.Page{}, wikierrors.PageNotFound()
	}
	if err != nil {
		return utils.Page{}, wikierrors.DatabaseError(err)
	}

	info, err = database.GetPageInfo(ctx, db, pageId)
	if err != nil {
		return utils.Page{}, wikierrors.DatabaseError(err)
	}

	pageDeleted, err := database.GetPageDeleted(ctx, db, info.UUID)
	if err != nil {
		return utils.Page{}, wikierrors.DatabaseError(err)
	}
	if pageDeleted {
		return utils.Page{}, wikierrors.PageDeleted()
	}

	content, err = filesystem.GetPageContent(ctx, db, dataDir, pageId)
	if err != nil {
		return utils.Page{}, wikierrors.FilesystemError(err)
	}

	if info.LastRevisionId != nil {
		lastRev, err = database.GetRevisionInfo(ctx, db, *info.LastRevisionId)
		if err == sql.ErrNoRows {
			return utils.Page{}, wikierrors.RevisionNotFound()
		}
		if err != nil {
			return utils.Page{}, wikierrors.DatabaseError(err)
		}
	} else {
		lastRev = &database.RevInfo{}
	}

	page = utils.Page{UUID: info.UUID, Slug: info.Slug, Name: info.Name, ArchiveDate: info.ArchiveDate,
		LastEdit: lastRev.UUID, LastEditTime: lastRev.DateTime,
		Content: content}

	return page, nil
}

func GetPages(ctx context.Context, db *sql.DB, dataDir string, ind int, count int) ([]utils.PageInfoPrev, error) {
	var pagesCount int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pages WHERE deleted_at IS NULL").Scan(&pagesCount)
	if err != nil {
		return nil, wikierrors.DatabaseError(err)
	}
	pagesCount -= ind
	if pagesCount <= 0 {
		return []utils.PageInfoPrev{}, nil
	}

	uuids, err := db.QueryContext(ctx,
		"SELECT uuid FROM pages WHERE deleted_at IS NULL ORDER BY slug LIMIT $1 OFFSET $2",
		count, ind)
	if err != nil {
		return nil, wikierrors.DatabaseError(err)
	}
	defer uuids.Close()

	var pages []utils.PageInfoPrev

	// i can almost guarantee this disgusting loop
	// could be done better and be much less tragic
	for uuids.Next() {
		var id uuid.UUID
		uuids.Scan(&id)
		pageInfo, err := utils.GetPageInfoPreview(ctx, db, dataDir, id)
		if err != nil {
			return nil, wikierrors.DatabaseFilesystemError(err)
		}
		if pageInfo == nil {
			continue
		}
		pages = append(pages, *pageInfo)
	}

	return pages, nil
}

func GetPagesBySlugs(ctx context.Context, db *sql.DB, dataDir string, slugList []string) []utils.PageInfoPrev {
	var pages []utils.PageInfoPrev
	for _, slug := range slugList {
		uuid, err := database.GetUUID(ctx, db, slug)
		if err != nil {
			continue
		}
		pageInfoPrev, err := utils.GetPageInfoPreview(ctx, db, dataDir, uuid)
		if err != nil {
			continue
		}
		if pageInfoPrev != nil {
			pages = append(pages, *pageInfoPrev)
		}
	}
	return pages
}

func GetPagesCategory(ctx context.Context, db *sql.DB, dataDir string, cat int, ind int, count int) ([]utils.PageInfoPrev, error) {
	var pagesCount int
	err := db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM pages
		JOIN page_categories ON pages.uuid = page_categories.page_id
		WHERE page_categories.category=$1 AND pages.deleted_at IS NULL`,
		cat).Scan(&pagesCount)
	if pagesCount != 0 && err != nil {
		return nil, wikierrors.DatabaseError(err)
	}
	pagesCount -= ind
	if pagesCount <= 0 {
		return []utils.PageInfoPrev{}, nil
	}

	uuids, err := db.QueryContext(
		ctx,
		`SELECT uuid FROM pages
		JOIN page_categories ON pages.uuid = page_categories.page_id
		WHERE page_categories.category=$1 AND pages.deleted_at IS NULL
		LIMIT $2 OFFSET $3`,
		cat, count, ind)
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
		if pageInfo == nil {
			continue
		}
		pages = append(pages, *pageInfo)
	}

	return pages, nil
}

func GetRevision(ctx context.Context, db *sql.DB, dataDir string, revId string) (utils.Revision, error) {
	var err error
	var rev = utils.Revision{}

	if err := uuid.Validate(revId); err != nil {
		return utils.Revision{}, wikierrors.InvalidID(err)
	}
	rev.UUID, err = uuid.Parse(revId)
	if err != nil {
		return utils.Revision{}, wikierrors.InvalidID(err)
	}

	revInfo, err := database.GetRevisionInfo(ctx, db, rev.UUID)
	if err == sql.ErrNoRows {
		return utils.Revision{}, wikierrors.RevisionNotFound()
	}
	if err != nil {
		return utils.Revision{}, wikierrors.DatabaseError(err)
	}
	if revInfo == nil || revInfo.PageId == nil {
		return utils.Revision{}, wikierrors.RevisionNotFound()
	}
	pageInfo, err := database.GetPageInfo(ctx, db, *revInfo.PageId)
	if err == sql.ErrNoRows {
		return utils.Revision{}, wikierrors.PageNotFound()
	}
	if err != nil {
		return utils.Revision{}, wikierrors.DatabaseError(err)
	}
	pageDeleted, err := database.GetPageDeleted(ctx, db, pageInfo.UUID)
	if err != nil {
		return utils.Revision{}, wikierrors.DatabaseError(err)
	}
	if pageDeleted {
		return utils.Revision{}, wikierrors.PageDeleted()
	}
	rev.PageId = *revInfo.PageId
	rev.RevDateTime = *revInfo.DateTime
	rev.Author = *revInfo.Author
	rev.Slug = revInfo.Slug
	rev.Name = revInfo.Name
	rev.ArchiveDate = revInfo.ArchiveDate
	rev.DeletedAt = revInfo.DeletedAt


	rev.Content, err = utils.GetContentAtRevision(ctx, db, dataDir, rev.PageId, rev.UUID)
	if err != nil {
		// GetContentAtRevision returns wikierror
		return utils.Revision{}, err
	}

	return rev, nil
}

func GetRevisions(ctx context.Context, db *sql.DB, pageId string, ind int, count int) ([]database.RevInfo, error) {

	pageUUID, err := database.GetUUID(ctx, db, pageId)
	if err == sql.ErrNoRows {
		return nil, wikierrors.PageNotFound()
	}
	if err != nil {
		return nil, wikierrors.DatabaseError(err)
	}

	pageDeleted, err := database.GetPageDeleted(ctx, db, pageUUID)
	if err != nil {
		return nil, wikierrors.DatabaseError(err)
	}
	if pageDeleted {
		return nil, wikierrors.PageDeleted()
	}

	var revCount int
	err = db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM revisions WHERE page_id=$1",
		pageUUID).Scan(&revCount)
	if err != nil {
		return nil, wikierrors.DatabaseError(err)
	}
	revCount -= ind
	if revCount <= 0 {
		return []database.RevInfo{}, nil
	}

	uuids, err := db.QueryContext(
		ctx,
		"SELECT uuid FROM revisions WHERE page_id=$1 LIMIT $2 OFFSET $3",
		pageUUID, count, ind)
	if err != nil {
		return nil, wikierrors.DatabaseError(err)
	}
	defer uuids.Close()

	var revs []database.RevInfo

	for uuids.Next() {
		var id uuid.UUID
		uuids.Scan(&id)
		revInfo, err := database.GetRevisionInfo(ctx, db, id)
		if err != nil {
			return nil, wikierrors.DatabaseError(err)
		}
		if revInfo == nil {
			continue
		}
		revs = append(revs, *revInfo)
	}

	return revs, nil
}
