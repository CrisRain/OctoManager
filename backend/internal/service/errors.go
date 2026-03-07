package service

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgerrcode"
	"gorm.io/gorm"
	"octomanger/backend/pkg/apperror"
)

func invalidInput(message string) error {
	return apperror.New(apperror.CodeInvalidInput, message)
}

func unauthorized(message string) error {
	return apperror.New(apperror.CodeUnauthorized, message)
}

func notFound(message string) error {
	return apperror.New(apperror.CodeNotFound, message)
}

func conflict(message string) error {
	return apperror.New(apperror.CodeConflict, message)
}

func internalError(message string, err error) error {
	return apperror.Wrap(apperror.CodeInternal, message, err)
}

func wrapRepoError(err error, notFoundMsg string) error {
	if isNotFound(err) {
		return notFound(notFoundMsg)
	}
	if isDuplicateKeyError(err) {
		return conflict("resource already exists")
	}
	return internalError("internal error", err)
}

func isNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func isDuplicateKeyError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgerrcode.UniqueViolation
	}
	return false
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgerrcode.ForeignKeyViolation
	}
	return false
}
