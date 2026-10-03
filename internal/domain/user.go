package domain

import "time"

type User struct {
	ID                     int
	Email                  string
	PasswordHash           string
	Role                   Role
	Confirmed              bool
	ConfirmationCode       string
	ConfirmationExpiresAt  *time.Time
	PasswordResetCode      string
	PasswordResetExpiresAt *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
	DeletedAt              *time.Time
}

func (u User) CanLogin() bool {
	return u.Confirmed && u.DeletedAt == nil
}
