package handlers

import (
	"net/http"
	"wiki/database"
	wikierrors "wiki/errors"
	"wiki/utils"

	"github.com/gin-gonic/gin"
)

func CategoriesHandler(c *gin.Context) {
	ctx := c.Request.Context()
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

	tree := c.DefaultQuery("tree", "false") == "true"
	rootOnly := c.DefaultQuery("root", "false") == "true"

	var categories []database.Category
	if rootOnly {
		categories, err = database.GetRootCategories(ctx, db)
	} else if tree {
		categories, err = database.GetCategoryTree(ctx, db)
	} else {
		categories, err = database.ListCategories(ctx, db)
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

	c.JSON(http.StatusOK, categories)
}

func GetPageCategoriesHandler(c *gin.Context) {
	ctx := c.Request.Context()
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

	id := c.Param("id")

	var categories []database.Category
	categories, err = database.GetPageCategories(ctx, db, id)
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

	c.JSON(http.StatusOK, categories)
}

func SetPageCategoriesHandler(c *gin.Context) {
	ctx := c.Request.Context()
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

	id := c.Param("id")

	var categories []string
	err = c.ShouldBindJSON(&categories)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "bad request format",
		})
		return
	}

	err = database.SetPageCategories(ctx, db, id, categories)
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

