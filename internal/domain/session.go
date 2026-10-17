package domain

import "time"

type Session struct {
	Token     string
	UserID    int
	CreatedAt time.Time
	ExpiresAt time.Time
}

func (s Session) IsExpired(now time.Time) bool {
	return now.After(s.ExpiresAt)
}
