package errors

import "net/http"

const (
	categoryNotFound = "CategoryNotFound"
	invalidCatSlug   = "InvalidCategorySlug"
)

func CategoryNotFound() WikiError {
	return WikiError{http.StatusNotFound, categoryNotFound, "category not found", nil}
}

func InvalidCatSlug() WikiError {
	return WikiError{http.StatusBadRequest, invalidCatSlug, "invalid category slug format", nil}
}
