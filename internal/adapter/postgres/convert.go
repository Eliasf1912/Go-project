package postgres

import (
	"database/sql"
	"time"
)

func nullTimeToPtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func stringToNull(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func nullStringToPtr(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}

func stringPtrToNull(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func intPtrToNull(i *int) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*i), Valid: true}
}
