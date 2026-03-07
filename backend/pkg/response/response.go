package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"octomanger/backend/pkg/apperror"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: 0, Message: "success", Data: data})
}

func Fail(c *gin.Context, httpStatus int, msg string) {
	c.JSON(httpStatus, Response{Code: httpStatus, Message: msg, Data: nil})
	c.Abort()
}

func FailWithError(c *gin.Context, err error) {
	if appErr, ok := apperror.As(err); ok {
		message := appErr.Message
		if message == "" {
			message = appErr.Error()
		}
		c.JSON(http.StatusOK, Response{Code: appErr.Code, Message: message, Data: nil})
	} else {
		c.JSON(http.StatusInternalServerError, Response{Code: http.StatusInternalServerError, Message: "internal server error", Data: nil})
	}
	c.Abort()
}
