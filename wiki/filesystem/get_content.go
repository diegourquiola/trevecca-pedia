package filesystem

import (
	"context"
	"database/sql"
	"os"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

func GetPageContent(ctx context.Context, db *sql.DB, dataDir string, pageId uuid.UUID) (string, error) {
	filename, err := GetPageFilename(ctx, db, dataDir, pageId.String())
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func GetRevisionContent(ctx context.Context, db *sql.DB, dataDir string, revId uuid.UUID) (string, error) {
	if revId == uuid.Nil {
		return "", nil
	}
	filename, err := GetRevisionFilename(ctx, db, dataDir, revId)
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func GetSnapshotContent(ctx context.Context, db *sql.DB, dataDir string, snapId uuid.UUID) (string, error) {
	filename, err := GetSnapshotFilename(ctx, db, dataDir, snapId)
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func GetPagePreview(ctx context.Context, db *sql.DB, dataDir string, pageId uuid.UUID, length int) (string, error) {
	content, err := GetPageContent(ctx, db, dataDir, pageId)
	if err != nil {
		return "", err
	}

	// Remove horizontal bars (---, ***, etc.)
	content = regexp.MustCompile(`(?m)^[-*]{3,}\s*$`).ReplaceAllString(content, "")

	// Convert headings (# Heading) to bold (**Heading**)
	content = regexp.MustCompile(`(?m)^#{1,6}\s+(.+?)\s*$`).ReplaceAllString(content, "**$1**")


	// Remove newlines
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.ReplaceAll(content, "\r", " ")

	// Get first length characters
	runes := []rune(content)
	if len(runes) > length {
		content = string(runes[:length])
	}

	return strings.TrimSpace(content), nil
}
