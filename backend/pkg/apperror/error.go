package apperror

import "errors"

const (
    CodeInvalidInput  = 40000
    CodeUnauthorized  = 40100
    CodeNotFound      = 40400
    CodeConflict      = 40900
    CodeInternal      = 50000
)

type AppError struct {
    Code    int
    Message string
    Err     error
}

func (e *AppError) Error() string {
    if e == nil {
        return ""
    }
    if e.Message != "" {
        return e.Message
    }
    if e.Err != nil {
        return e.Err.Error()
    }
    return "error"
}

func (e *AppError) Unwrap() error {
    if e == nil {
        return nil
    }
    return e.Err
}

func New(code int, message string) *AppError {
    return &AppError{Code: code, Message: message}
}

func Wrap(code int, message string, err error) *AppError {
    return &AppError{Code: code, Message: message, Err: err}
}

func As(err error) (*AppError, bool) {
    if err == nil {
        return nil, false
    }
    var appErr *AppError
    if errors.As(err, &appErr) {
        return appErr, true
    }
    return nil, false
}
