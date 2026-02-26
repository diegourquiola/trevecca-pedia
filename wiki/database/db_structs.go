package database

import (
	"time"

	"github.com/google/uuid"
)

type PageInfo struct {
	UUID			uuid.UUID	`db:"uuid"`
	Slug			string		`db:"slug"`
	Name			string		`db:"name"`
	LastRevisionId	*uuid.UUID	`db:"last_revision_id"`
	ArchiveDate		*time.Time	`db:"archive_date"`
}

type RevInfo struct {
	UUID		*uuid.UUID	`db:"uuid"`
	PageId		*uuid.UUID	`db:"page_id"`
	DateTime	*time.Time	`db:"date_time"`
	Author		*string		`db:"author"`
	Slug		string		`db:"slug"`
	Name		string		`db:"name"`
	ArchiveDate	*time.Time	`db:"archive_date"`
	DeletedAt	*time.Time	`db:"deleted_at"`
}

type SnapInfo struct {
	UUID		uuid.UUID	`db:"uuid"`
	Page		uuid.UUID	`db:"page"`
	Revision	*uuid.UUID	`db:"revision"`
}
