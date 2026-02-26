package utils

import (
	"time"

	"github.com/google/uuid"
)

type Revision struct {
	UUID			uuid.UUID	`json:"uuid"`
	PageId			uuid.UUID	`json:"page_id"`
	RevDateTime		time.Time	`json:"rev_date_time"`
	Author			string		`json:"author"`
	Slug			string		`json:"slug"`
	Name			string		`json:"name"`
	ArchiveDate		*time.Time	`json:"archive_date"`
	DeletedAt		*time.Time	`json:"deleted_at"`
	Content			string		`json:"content"`
}

type RevisionRequest struct {
	PageId			string		`json:"page_id"`
	Author			string		`json:"author"`
	Slug			string		`json:"slug"`
	Name			string		`json:"name"`
	ArchiveDate		*time.Time	`json:"archive_date"`
	DeletedAt		*time.Time	`json:"deleted_at"`
	NewContent		string		`json:"new_content"`
}

