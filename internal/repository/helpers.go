package repository

import "strings"

// isDuplicateKeyError reports whether the error is a PostgreSQL
// unique-constraint violation (SQLSTATE 23505).
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint")
}

// isForeignKeyError reports whether the error is a PostgreSQL
// foreign-key violation (SQLSTATE 23503).
func isForeignKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "23503") ||
		strings.Contains(msg, "foreign key") ||
		strings.Contains(msg, "violates foreign key")
}

// isNotNullError reports whether the error is a PostgreSQL
// NOT NULL violation (SQLSTATE 23502).
func isNotNullError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "23502") ||
		strings.Contains(msg, "not-null") ||
		strings.Contains(msg, "null value in column")
}

// isCheckConstraintError reports whether the error is a PostgreSQL
// CHECK constraint violation (SQLSTATE 23514).
func isCheckConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "23514") ||
		strings.Contains(msg, "check constraint") ||
		strings.Contains(msg, "violates check constraint")
}

// nullableString returns nil for empty strings, so the DB stores NULL
// instead of an empty string. Useful for optional text columns.
func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
