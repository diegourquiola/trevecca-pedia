package database

import (
	"time"

	"github.com/google/uuid"
)

type PageInfo struct {
	UUID			uuid.UUID	`db:"uuid" json:"uuid"`
	Slug			string		`db:"slug" json:"slug"`
	Name			string		`db:"name" json:"name"`
	LastRevisionId	*uuid.UUID	`db:"last_revision_id" json:"last_revision_id"`
	ArchiveDate		*time.Time	`db:"archive_date" json:"archive_date"`
}

type RevInfo struct {
	UUID		*uuid.UUID	`db:"uuid" json:"uuid"`
	PageId		*uuid.UUID	`db:"page_id" json:"page_id"`
	DateTime	*time.Time	`db:"date_time" json:"date_time"`
	Author		*string		`db:"author" json:"author"`
	Slug		string		`db:"slug" json:"slug"`
	Name		string		`db:"name" json:"name"`
	ArchiveDate	*time.Time	`db:"archive_date" json:"archive_date"`
	DeletedAt	*time.Time	`db:"deleted_at" json:"deleted_at"`
}

type SnapInfo struct {
	UUID		uuid.UUID	`db:"uuid" json:"uuid"`
	Page		uuid.UUID	`db:"page" json:"page"`
	Revision	*uuid.UUID	`db:"revision" json:"revision"`
}
