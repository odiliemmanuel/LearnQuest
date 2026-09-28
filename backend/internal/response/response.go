package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Envelope struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{Success: true, Data: data})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Envelope{Success: true, Data: data})
}

func Fail(c *gin.Context, status int, msg string) {
	c.JSON(status, Envelope{Success: false, Error: msg})
}

func BadRequest(c *gin.Context, msg string)   { Fail(c, http.StatusBadRequest, msg) }
func NotFound(c *gin.Context, msg string)     { Fail(c, http.StatusNotFound, msg) }
func Unauthorized(c *gin.Context, msg string) { Fail(c, http.StatusUnauthorized, msg) }
func Forbidden(c *gin.Context, msg string)    { Fail(c, http.StatusForbidden, msg) }
func Internal(c *gin.Context, msg string)     { Fail(c, http.StatusInternalServerError, msg) }
func TooMany(c *gin.Context, msg string)      { Fail(c, http.StatusTooManyRequests, msg) }
