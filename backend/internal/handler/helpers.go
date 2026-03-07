package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func parseUint64Param(c *gin.Context, name string) (uint64, error) {
	raw := strings.TrimSpace(c.Param(name))
	return strconv.ParseUint(raw, 10, 64)
}

func resolvePagination(c *gin.Context, defaultLimit, maxLimit int) (int, int, error) {
	limitStr := strings.TrimSpace(c.Query("limit"))
	offsetStr := strings.TrimSpace(c.Query("offset"))

	limit := defaultLimit
	offset := 0

	if limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil {
			return 0, 0, err
		}
		limit = parsed
	}
	if offsetStr != "" {
		parsed, err := strconv.Atoi(offsetStr)
		if err != nil {
			return 0, 0, err
		}
		offset = parsed
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if maxLimit > 0 && limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset, nil
}

func parseColonActionParam(raw string, action string) (string, bool) {
	suffix := ":" + action
	if strings.HasSuffix(raw, suffix) {
		return strings.TrimSuffix(raw, suffix), true
	}
	return "", false
}
