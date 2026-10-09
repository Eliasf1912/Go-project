package hasher

import (
	"golang.org/x/crypto/bcrypt"
)

type BcryptHasher struct{}

func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{}
}

func (h *BcryptHasher) Hash(password string) (string, error) {
	passwordBytes := []byte(password)

	HashedBytes, err := bcrypt.GenerateFromPassword(passwordBytes, bcrypt.DefaultCost)

	if err != nil {
		return "", err
	}

	return string(HashedBytes), nil
}

func (h *BcryptHasher) CompareHash(password, hash string) bool {

	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil

}
