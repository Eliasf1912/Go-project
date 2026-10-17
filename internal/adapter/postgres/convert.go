package postgres

import (
	"database/sql"
	"time"
)

// nullTimeToPtr convertit une date SQL pouvant être NULL en pointeur Go (nil si NULL).
func nullTimeToPtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// stringToNull convertit une chaîne vide en NULL SQL.
func stringToNull(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
