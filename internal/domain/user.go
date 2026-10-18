package domain

import (
	"strings"
	"time"
)

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

func (u *User) Confirm(code string, now time.Time) error {

	if u.Confirmed {
		return ErrAccountAlreadyConfirmed
	}

	if code == "" || code != u.ConfirmationCode {
		return ErrInvalidCode
	}

	if u.ConfirmationExpiresAt == nil || u.ConfirmationExpiresAt.Before(now) {
		return ErrCodeExpired
	}

	u.ConfirmationCode = ""
	u.Confirmed = true
	u.ConfirmationExpiresAt = nil

	return nil
}

func NormalizeEmail(email string) (string, error) {
	email = strings.ToLower(email)
	email = strings.TrimSpace(email)
	if !strings.Contains(email, "@") {
		return "", ErrInvalidEmail
	}
	return email, nil
}

// TODO: limite de 72 octets (bcrypt)
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}

	return nil
}

// CheckResetCode vérifie le code de réinitialisation du mot de passe.
func (u User) CheckResetCode(code string, now time.Time) error {
	if code == "" || code != u.PasswordResetCode {
		return ErrInvalidCode
	}
	if u.PasswordResetExpiresAt == nil || u.PasswordResetExpiresAt.Before(now) {
		return ErrCodeExpired
	}
	return nil
}
