package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
	"wiki/database"
	wikierrors "wiki/errors"
	"wiki/requests"
	"wiki/utils"

	"github.com/gin-gonic/gin"
)

func NewPageHandler(c *gin.Context) {
	ctx := context.Background()
	db, err := utils.GetDatabase()
	if err != nil {
		werr, is := wikierrors.AsWikiError(err)
		if !is {
			werr = wikierrors.InternalError(err)
		}
		c.AbortWithStatusJSON(werr.Code, gin.H{
			"error": werr.Details,
		})
		return
	}
	defer db.Close()
	dataDir := utils.GetDataDir()

	var newPageReq utils.NewPageRequest
	err = c.Request.ParseMultipartForm(32 << 20)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "bad request format",
		})
		return
	}
	file, err := c.FormFile("new_page")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "bad request format",
		})
		return
	}
	f, err := file.Open()
	if err != nil {
		werr := wikierrors.FilesystemError(err)
		c.AbortWithStatusJSON(werr.Code, gin.H{
			"error": werr.Details,
		})
		return
	}
	defer f.Close()
	newPageBytes, err := io.ReadAll(f)
	if err != nil {
		werr := wikierrors.InternalError(err)
		c.AbortWithStatusJSON(werr.Code, gin.H{
			"error": werr.Details,
		})
		return
	}
	newPageReq.Author = c.PostForm("author")

	newPageReq.Slug = c.PostForm("slug")
	if newPageReq.Slug == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "bad request format",
		})
		return
	}
	newPageReq.Name = c.PostForm("name")
	if newPageReq.Name == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "bad request format",
		})
		return
	}

	// Handle optional archive_date
	archiveDateStr := c.PostForm("archive_date")
	if archiveDateStr != "" {
		archiveDate, err := time.Parse("2006-01-02", archiveDateStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "bad request format",
			})
			return
		}
		newPageReq.ArchiveDate = &archiveDate
	}

	newPageReq.Content = string(newPageBytes)

	err = utils.CreateNewPage(ctx, db, dataDir, newPageReq)
	if err != nil {
		werr, is := wikierrors.AsWikiError(err)
		if !is {
			werr = wikierrors.InternalError(err)
		}
		c.AbortWithStatusJSON(werr.Code, gin.H{
			"error": werr.Details,
		})
		return
	}
	c.Status(http.StatusOK)
}

func DeletePageHandler(c *gin.Context) {
	ctx := context.Background()
	db, err := utils.GetDatabase()
	if err != nil {
		werr, is := wikierrors.AsWikiError(err)
		if !is {
			werr = wikierrors.InternalError(err)
		}
		c.AbortWithStatusJSON(werr.Code, gin.H{
			"error": werr.Details,
		})
		return
	}
	defer db.Close()
	dataDir := utils.GetDataDir()

	var delReq utils.DeletePageRequest
	err = c.Request.ParseForm()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "bad request format",
		})
		return
	}

	delReq.Slug = c.PostForm("slug")
	delReq.User = c.PostForm("user")

	err = requests.DeletePage(ctx, db, dataDir, delReq)
	if err != nil {
		werr, is := wikierrors.AsWikiError(err)
		if !is {
			werr = wikierrors.InternalError(err)
		}
		c.AbortWithStatusJSON(werr.Code, gin.H{
			"error": werr.Details,
		})
		return
	}

	c.Status(http.StatusOK)

}

func NewRevisionHandler(c *gin.Context) {
	fmt.Printf("[NewRevisionHandler] Starting request\n")
	ctx := context.Background()
	db, err := utils.GetDatabase()
	if err != nil {
		fmt.Printf("[NewRevisionHandler] Database connection failed: %v\n", err)
		werr, is := wikierrors.AsWikiError(err)
		if !is {
			werr = wikierrors.InternalError(err)
		}
		c.AbortWithStatusJSON(werr.Code, gin.H{
			"error": werr.Details,
		})
		return
	}
	defer db.Close()
	dataDir := utils.GetDataDir()
	fmt.Printf("[NewRevisionHandler] dataDir: %s\n", dataDir)

	var revReq utils.RevisionRequest
	err = c.Request.ParseMultipartForm(32 << 20)
	if err != nil {
		fmt.Printf("[NewRevisionHandler] ParseMultipartForm failed: %v\n", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "bad request format",
		})
		return
	}
	fmt.Printf("[NewRevisionHandler] Parsed multipart form\n")
	file, err := c.FormFile("new_content")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "bad request format",
		})
		return
	}
	f, err := file.Open()
	if err != nil {
		werr := wikierrors.FilesystemError(err)
		c.AbortWithStatusJSON(werr.Code, gin.H{
			"error": werr.Details,
		})
		return
	}
	defer f.Close()
	newPageBytes, err := io.ReadAll(f)
	if err != nil {
		werr := wikierrors.InternalError(err)
		c.AbortWithStatusJSON(werr.Code, gin.H{
			"error": werr.Details,
		})
		return
	}
	revReq.PageId = c.PostForm("page_id")
	revReq.Author = c.PostForm("author")
	fmt.Printf("[NewRevisionHandler] page_id: %s, author: %s\n", revReq.PageId, revReq.Author)

	pageId, err := database.GetUUID(ctx, db, revReq.PageId)
	if err != nil {
		fmt.Printf("[NewRevisionHandler] GetUUID failed: %v\n", err)
		werr := wikierrors.DatabaseError(err)
		c.AbortWithStatusJSON(werr.Code, gin.H{
			"error": werr.Details,
		})
		return
	}
	fmt.Printf("[NewRevisionHandler] Got page UUID: %s\n", pageId)
	pageInfo, err := database.GetPageInfo(ctx, db, pageId)
	if err != nil {
		fmt.Printf("[NewRevisionHandler] GetPageInfo failed: %v\n", err)
		werr := wikierrors.DatabaseError(err)
		c.AbortWithStatusJSON(werr.Code, gin.H{
			"error": werr.Details,
		})
		return
	}
	fmt.Printf("[NewRevisionHandler] Got page info - slug: %s, name: %s\n", pageInfo.Slug, pageInfo.Name)

	revReq.Slug = c.PostForm("slug")
	if revReq.Slug == "" {
		revReq.Slug = pageInfo.Slug
	}
	revReq.Name = c.PostForm("name")
	if revReq.Name == "" {
		revReq.Name = pageInfo.Name
	}
	fmt.Printf("[NewRevisionHandler] Request slug: %s, name: %s\n", revReq.Slug, revReq.Name)

	archiveDateStr := c.PostForm("archive_date")
	if archiveDateStr != "" {
		archiveDate, err := time.Parse("2006-01-02", archiveDateStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "bad request format",
			})
			return
		}
		revReq.ArchiveDate = &archiveDate
	} else {
		revReq.ArchiveDate = pageInfo.ArchiveDate
	}

	var deletedAt *time.Time
	err = db.QueryRowContext(ctx, `
		SELECT deleted_at FROM pages WHERE uuid=$1;
	`, pageId).Scan(&deletedAt)
	if err != nil {
		werr := wikierrors.DatabaseError(err)
		c.AbortWithStatusJSON(werr.Code, gin.H{
			"error": werr.Details,
		})
		return
	}
	if deletedAt != nil {
		werr := wikierrors.PageDeleted()
		c.AbortWithStatusJSON(werr.Code, gin.H{
			"error": werr.Details,
		})
		return
	}
	revReq.DeletedAt = deletedAt

	revReq.NewContent = string(newPageBytes)

	fmt.Printf("[NewRevisionHandler] Calling PostRevision\n")
	err = requests.PostRevision(ctx, db, dataDir, revReq)
	if err != nil {
		fmt.Printf("[NewRevisionHandler] PostRevision failed: %v\n", err)
		werr, is := wikierrors.AsWikiError(err)
		if !is {
			werr = wikierrors.InternalError(err)
		}
		c.AbortWithStatusJSON(werr.Code, gin.H{
			"error": werr.Details,
		})
		return
	}

	fmt.Printf("[NewRevisionHandler] Success\n")
	c.Status(http.StatusOK)
}
