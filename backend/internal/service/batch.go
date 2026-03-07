package service

import (
	"strconv"

	"octomanger/backend/pkg/apperror"
)

func batchError(err error) (string, string) {
	if err == nil {
		return strconv.Itoa(apperror.CodeInternal), "unknown error"
	}
	if appErr, ok := apperror.As(err); ok {
		message := appErr.Message
		if message == "" {
			message = appErr.Error()
		}
		return strconv.Itoa(appErr.Code), message
	}
	return strconv.Itoa(apperror.CodeInternal), err.Error()
}

func invalidCode() string {
	return strconv.Itoa(apperror.CodeInvalidInput)
}
