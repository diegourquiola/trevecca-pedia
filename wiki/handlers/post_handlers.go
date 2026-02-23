package handlers

import (
	"context"
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

	var revReq utils.RevisionRequest
	err = c.Request.ParseMultipartForm(32 << 20)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "bad request format",
		})
		return
	}
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

	pageId, err := database.GetUUID(ctx, db, revReq.PageId)
	if err != nil {
		werr := wikierrors.DatabaseError(err)
		c.AbortWithStatusJSON(werr.Code, gin.H{
			"error": werr.Details,
		})
		return
	}
	pageInfo, err := database.GetPageInfo(ctx, db, pageId)
	if err != nil {
		werr := wikierrors.DatabaseError(err)
		c.AbortWithStatusJSON(werr.Code, gin.H{
			"error": werr.Details,
		})
		return
	}

	revReq.Slug = c.PostForm("slug")
	if revReq.Slug == "" {
		revReq.Slug = pageInfo.Slug
	}
	revReq.Name = c.PostForm("name")
	if revReq.Name == "" {
		revReq.Name = pageInfo.Name
	}

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

	err = requests.PostRevision(ctx, db, dataDir, revReq)
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
