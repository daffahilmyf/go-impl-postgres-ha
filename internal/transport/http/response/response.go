package response

import "github.com/gin-gonic/gin"

type APIError struct {
	Message string `json:"message"`
}

type Meta struct {
	NextCursor string `json:"next_cursor,omitempty"`
}

type APIResponse struct {
	Data  any       `json:"data,omitempty"`
	Error *APIError `json:"error,omitempty"`
	Meta  *Meta     `json:"meta,omitempty"`
}

type Page[T any] struct {
	Data []T  `json:"data"`
	Meta Meta `json:"meta"`
}

func RespondOK(c *gin.Context, status int, data any, meta *Meta) {
	c.JSON(status, APIResponse{
		Data: data,
		Meta: meta,
	})
}

func RespondError(c *gin.Context, status int, message string) {
	c.JSON(status, APIResponse{
		Error: &APIError{Message: message},
	})
}
