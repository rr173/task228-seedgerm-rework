package store

import (
	"strings"

	"task228-seedgerm/internal/model"
)

// isUniqueErr 判断 SQLite 错误是否为唯一约束冲突。
func isUniqueErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "constraint failed")
}

// wrapNotFound 将 sql.ErrNoRows 归一为领域 ErrNotFound。
func wrapNotFound(err error) error {
	if err == nil {
		return nil
	}
	if err.Error() == "sql: no rows in result set" {
		return model.ErrNotFound
	}
	return err
}
