package usecase

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/Eliasf1912/go-project/internal/domain"
)

type PasswordHasher interface {
	Hash(password string) (string, error)
}

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (domain.User, error)
	Create(ctx context.Context, user domain.User) (int, error)
	Update(ctx context.Context, user domain.User) error
}

type AuthService struct {
	users  UserRepository
	hasher PasswordHasher
}

func NewAuthService(users UserRepository, hasher PasswordHasher) *AuthService {
	return &AuthService{users: users, hasher: hasher}
}

func generateCode() (string, error) {
	number, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", number), nil
}

func (s *AuthService) Register(ctx context.Context, email string, password string) error {

	email, err := domain.NormalizeEmail(email)

	if err != nil {
		return err
	}

	if err := domain.ValidatePassword(password); err != nil {
		return err
	}

	_, err = s.users.FindByEmail(ctx, email)

	if err == nil {
		return domain.ErrEmailTaken
	}

	if !errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("recherche de l'utilisateur : %w", err)
	}

	hashedPassword, err := s.hasher.Hash(password)

	if err != nil {
		return fmt.Errorf("hachage du mot de passe : %w", err)
	}

	code, err := generateCode()

	if err != nil {
		return fmt.Errorf("hachage du mot de passe : %w", err)
	}

	expirationTime := time.Now().Add(time.Minute * 15)

	newUser := domain.User{
		Email:                 email,
		PasswordHash:          hashedPassword,
		Role:                  domain.RoleClient,
		Confirmed:             false,
		ConfirmationCode:      code,
		ConfirmationExpiresAt: &expirationTime,
	}

	userCreateId, err := s.users.Create(ctx, newUser)

	if err != nil {
		return err
	}

	log.Printf("Utilisateur créer : %d, code de confirmation pour %s : %s", userCreateId, email, code)

	return nil
}

func (s *AuthService) Confirm(ctx context.Context, email string, code string) error {

	normalizedEmail, err := domain.NormalizeEmail(email)

	if err != nil {
		return err
	}

	user, err := s.users.FindByEmail(ctx, normalizedEmail)

	if errors.Is(err, domain.ErrNotFound) {
		return domain.ErrInvalidCode
	}

	if err != nil {
		return fmt.Errorf("recherche de l'utilisateur : %w", err)
	}

	err = user.Confirm(code, time.Now())

	if err != nil {
		return err
	}

	err = s.users.Update(ctx, user)

	if err != nil {
		return fmt.Errorf("enregistrement de la confirmation : %w", err)
	}

	return nil
}
