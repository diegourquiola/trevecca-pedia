package wiki

import (
	"api-layer/config"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"
)

func PostNewPage(c *gin.Context) {
	wikiURL := fmt.Sprintf("%s/pages/new", config.WikiServiceURL)


	// get data from request
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse multipart form"})
		return
	}

	fileHeader, err := c.FormFile("new_page")
	if err != nil {
		if err == http.ErrMissingFile {
            c.JSON(http.StatusBadRequest, gin.H{"error": "new_page file is required"})
            return
        }
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file upload: " + err.Error()})
        return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot open the uploaded file"})
        return
	}
	defer file.Close()


	// create new request to wiki service
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	writer.WriteField("slug", c.PostForm("slug"))
	writer.WriteField("name", c.PostForm("name"))
	writer.WriteField("author", c.PostForm("author"))
	writer.WriteField("archive_date", c.PostForm("archive_date"))

	dstPart, err := writer.CreateFormFile("new_page", fileHeader.Filename)
	if err != nil {
		writer.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create form file part"})
        return
	}

	_, err = io.Copy(dstPart, file)
	if err != nil {
		writer.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to copy file content"})
        return
	}
	if err := writer.Close(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to finalize multipart"})
        return
    }

	req, err := http.NewRequest(http.MethodPost, wikiURL, &body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
        return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())


	// get response from request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
        c.JSON(http.StatusBadGateway, gin.H{"error": "wiki service unreachable", "detail": err.Error()})
        return
    }
	defer resp.Body.Close()

	c.Status(resp.StatusCode)
	for k, vals := range resp.Header {
		for _, v := range vals {
			c.Writer.Header().Add(k, v)
		}
	}
	io.Copy(c.Writer, resp.Body)

}

func PostDeletePage(c *gin.Context) {
	id := c.Param("id")
	wikiURL := fmt.Sprintf("%s/pages/%s/delete", config.WikiServiceURL, id)

	// get data from request
	if err := c.Request.ParseForm(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}

	// new request to wiki service
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	writer.WriteField("slug", c.PostForm("slug"))
	writer.WriteField("user", c.PostForm("user"))

	if err := writer.Close(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to finalize multipart"})
        return
    }

	req, err := http.NewRequest(http.MethodPost, wikiURL, &body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
        return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())


	// get response from request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
        c.JSON(http.StatusBadGateway, gin.H{"error": "wiki service unreachable", "detail": err.Error()})
        return
    }
	defer resp.Body.Close()

	c.Status(resp.StatusCode)
	for k, vals := range resp.Header {
		for _, v := range vals {
			c.Writer.Header().Add(k, v)
		}
	}
	io.Copy(c.Writer, resp.Body)
}

func PostPageRevision(c *gin.Context) {
	id := c.Param("id")
	wikiURL := fmt.Sprintf("%s/pages/%s/revisions", config.WikiServiceURL, id)


	// get data from request
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse multipart form"})
		return
	}

	fileHeader, err := c.FormFile("new_content")
	if err != nil {
		if err == http.ErrMissingFile {
            c.JSON(http.StatusBadRequest, gin.H{"error": "new_content file is required"})
            return
        }
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file upload: " + err.Error()})
        return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot open the uploaded file"})
        return
	}
	defer file.Close()


	// create new request to wiki service
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	writer.WriteField("page_id", c.PostForm("page_id"))
	writer.WriteField("author", c.PostForm("author"))

	writer.WriteField("slug", c.PostForm("slug"))
	writer.WriteField("name", c.PostForm("name"))
	writer.WriteField("archive_date", c.PostForm("archive_date"))

	dstPart, err := writer.CreateFormFile("new_content", fileHeader.Filename)
	if err != nil {
		writer.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create form file part"})
        return
	}

	_, err = io.Copy(dstPart, file)
	if err != nil {
		writer.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to copy file content"})
        return
	}
	if err := writer.Close(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to finalize multipart"})
        return
    }

	req, err := http.NewRequest(http.MethodPost, wikiURL, &body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
        return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())


	// get response from request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
        c.JSON(http.StatusBadGateway, gin.H{"error": "wiki service unreachable", "detail": err.Error()})
        return
    }
	defer resp.Body.Close()

	c.Status(resp.StatusCode)
	for k, vals := range resp.Header {
		for _, v := range vals {
			c.Writer.Header().Add(k, v)
		}
	}
	io.Copy(c.Writer, resp.Body)
}

