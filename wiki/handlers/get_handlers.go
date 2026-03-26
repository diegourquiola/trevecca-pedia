package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"wiki/database"
	wikierrors "wiki/errors"
	"wiki/requests"
	"wiki/utils"

	"github.com/gin-gonic/gin"
)

func PagesHandler(c *gin.Context) {
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

	// URL Parameters
	catQuery := c.DefaultQuery("category", "")
	exact := c.DefaultQuery("exact", "false") == "true"

	ind, err := strconv.Atoi(c.DefaultQuery("index", "0"))
	if err != nil {
		ind = 0
	}

	count, err := strconv.Atoi(c.DefaultQuery("count", "10"))
	if err != nil {
		count = 10
	}

	slugs := c.DefaultQuery("slugs", "")

	var pages []utils.PageInfoPrev

	if catQuery != "" {
		pages, err = requests.GetPagesCategory(ctx, db, dataDir, catQuery, ind, count, exact)
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
	} else if slugs != "" {
		slugList := strings.Split(slugs, ",")
		pages = requests.GetPagesBySlugs(ctx, db, dataDir, slugList)
	} else {
		pages, err = requests.GetPages(ctx, db, dataDir, ind, count)
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
	}
	c.JSON(http.StatusOK, pages)
}

func PageHandler(c *gin.Context) {
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

	pageId := c.Param("id")
	page, err := requests.GetPage(ctx, db, dataDir, pageId)
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
	c.JSON(http.StatusOK, page)
}

func PageRevisionsHandler(c *gin.Context) {
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

	pageId := c.Param("id")
	ind, err := strconv.Atoi(c.DefaultQuery("index", "0"))
	if err != nil {
		ind = 0
	}
	count, err := strconv.Atoi(c.DefaultQuery("count", "10"))
	if err != nil {
		count = 10
	}
	var revisions []database.RevInfo
	if pageId != "" {
		revisions, err = requests.GetRevisions(ctx, db, pageId, ind, count)
	} else {
		author := c.DefaultQuery("author", "")
		revisions, err = requests.GetRevisionsByAuthor(ctx, db, author, ind, count)
	}
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
	c.JSON(http.StatusOK, revisions)
}

func PageRevisionHandler(c *gin.Context) {
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
	revId := c.Param("rev")
	revision, err := requests.GetRevision(ctx, db, dataDir, revId)
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
	c.JSON(http.StatusOK, revision)
}

func IndexablePagesHandler(c *gin.Context) {
	ind, err := strconv.Atoi(c.DefaultQuery("index", "0"))
	if err != nil {
		ind = 0
	}
	count, err := strconv.Atoi(c.DefaultQuery("count", "10"))
	if err != nil {
		count = 10
	}
	ctx := context.Background()
	db, err := utils.GetDatabase()
	if err != nil {
		werr, is := wikierrors.AsWikiError(err)
		if !is {
			werr = wikierrors.DatabaseError(err)
		}
		c.AbortWithStatusJSON(werr.Code, gin.H{
			"error": werr.Details,
		})
		return
	}
	defer db.Close()
	dataDir := utils.GetDataDir()

	res, err := db.QueryContext(ctx, `
		SELECT slug FROM pages WHERE deleted_at IS NULL LIMIT $1 OFFSET $2;
	`, count, ind)
	if err != nil {
		werr, is := wikierrors.AsWikiError(err)
		if !is {
			werr = wikierrors.DatabaseError(err)
		}
		c.AbortWithStatusJSON(werr.Code, gin.H{
			"error": werr.Details,
		})
		return
	}
	defer res.Close()

	var slugs []string
	for res.Next() {
		var row string
		if err := res.Scan(&row); err != nil {
			werr := wikierrors.DatabaseError(err)
			c.AbortWithStatusJSON(werr.Code, gin.H{"error": werr.Details})
			return
		}
		slugs = append(slugs, row)
	}

	var indexable []utils.IndexInfo
	for _, page := range slugs {
		indexInfo, err := utils.GetIndexInfo(ctx, db, dataDir, page)
		if err != nil {
			werr := wikierrors.DatabaseFilesystemError(err)
			c.AbortWithStatusJSON(werr.Code, gin.H{
				"error": werr.Details,
			})
			return
		}
		if indexInfo == nil {
			continue
		}
		indexable = append(indexable, *indexInfo)
	}

	c.JSON(http.StatusOK, indexable)
}
