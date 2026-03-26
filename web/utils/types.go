package utils

import (
	"time"

	"github.com/google/uuid"
)

type Page struct {
	UUID         uuid.UUID  `json:"uuid"`
	Slug         string     `json:"slug"`
	Name         string     `json:"name"`
	ArchiveDate  *time.Time `json:"archive_date"`
	LastEditUUID *uuid.UUID `json:"last_edit"`
	LastEditTime time.Time  `json:"last_edit_time"`
	Content      string     `json:"content"`
	Categories   []Category `json:"categories"`
}

type PageInfoPrev struct {
	UUID         uuid.UUID  `json:"uuid"`
	Slug         string     `json:"slug"`
	Name         string     `json:"name"`
	LastEditTime time.Time  `json:"last_edit_time"`
	ArchiveDate  *time.Time `json:"archive_date"`
	Preview      string     `json:"preview"`
}

type Category struct {
	ID       int        `json:"id"`
	Slug     string     `json:"slug"`
	Name     string     `json:"name"`
	FullSlug string     `json:"full_slug"`
	Children []Category `json:"children,omitempty"`
}

type CategoryFlat struct {
	ID          int    `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	FullSlug    string `json:"full_slug"`
	Depth       int    `json:"depth"`
	DisplayName string `json:"display_name"`
}

// RevisionList represents a revision from the list endpoint (/pages/{id}/revisions or /revisions)
// Uses "date_time" field name and nullable fields as returned by the list API
type RevisionList struct {
	UUID        *uuid.UUID `json:"uuid"`
	PageId      *uuid.UUID `json:"page_id"`
	DateTime    *time.Time `json:"date_time"`
	Author      *string    `json:"author"`
	Slug        string     `json:"slug"`
	Name        string     `json:"name"`
	ArchiveDate *time.Time `json:"archive_date"`
	DeletedAt   *time.Time `json:"deleted_at"`
}

// RevisionDetail represents a revision from the detail endpoint (/pages/{id}/revisions/{revId})
// Uses "rev_date_time" field name and non-nullable fields, includes content
type RevisionDetail struct {
	UUID        uuid.UUID  `json:"uuid"`
	PageId      uuid.UUID  `json:"page_id"`
	RevDateTime time.Time  `json:"rev_date_time"`
	Author      string     `json:"author"`
	Slug        string     `json:"slug"`
	Name        string     `json:"name"`
	ArchiveDate *time.Time `json:"archive_date"`
	DeletedAt   *time.Time `json:"deleted_at"`
	Content     string     `json:"content"`
}

// Revision represents a page revision from the API
// Deprecated: Use RevisionList for list endpoints or RevisionDetail for detail endpoints
// Note: The API uses different field names for list vs detail endpoints
type Revision struct {
	UUID        uuid.UUID  `json:"uuid"`
	PageId      uuid.UUID  `json:"page_id"`
	RevDateTime time.Time  `json:"rev_date_time"`
	Author      string     `json:"author"`
	Slug        string     `json:"slug"`
	Name        string     `json:"name"`
	ArchiveDate *time.Time `json:"archive_date"`
	DeletedAt   *time.Time `json:"deleted_at"`
	Content     string     `json:"content"`
}

// ToRevision converts a RevisionList to the deprecated Revision type for backward compatibility
// Returns zero values for fields that are nil in the source
func (rl RevisionList) ToRevision() Revision {
	var revUUID uuid.UUID
	if rl.UUID != nil {
		revUUID = *rl.UUID
	}

	var pageId uuid.UUID
	if rl.PageId != nil {
		pageId = *rl.PageId
	}

	var dateTime time.Time
	if rl.DateTime != nil {
		dateTime = *rl.DateTime
	}

	var author string
	if rl.Author != nil {
		author = *rl.Author
	}

	return Revision{
		UUID:        revUUID,
		PageId:      pageId,
		RevDateTime: dateTime,
		Author:      author,
		Slug:        rl.Slug,
		Name:        rl.Name,
		ArchiveDate: rl.ArchiveDate,
		DeletedAt:   rl.DeletedAt,
		Content:     "",
	}
}

// ToRevision converts a RevisionDetail to the deprecated Revision type for backward compatibility
func (rd RevisionDetail) ToRevision() Revision {
	return Revision{
		UUID:        rd.UUID,
		PageId:      rd.PageId,
		RevDateTime: rd.RevDateTime,
		Author:      rd.Author,
		Slug:        rd.Slug,
		Name:        rd.Name,
		ArchiveDate: rd.ArchiveDate,
		DeletedAt:   rd.DeletedAt,
		Content:     rd.Content,
	}
}

// RevisionListToRevisions converts a slice of RevisionList to a slice of deprecated Revision
func RevisionListToRevisions(list []RevisionList) []Revision {
	result := make([]Revision, len(list))
	for i, rl := range list {
		result[i] = rl.ToRevision()
	}
	return result
}
