package usecase

import (
	"context"

	"github.com/Eliasf1912/go-project/internal/domain"
)

// UserAdminRepository regroupe ce dont la gestion des utilisateurs a besoin.
// Le repository Postgres des utilisateurs l'implémente aussi.
type UserAdminRepository interface {
	List(ctx context.Context) ([]domain.User, error)
	DeleteOrAnonymize(ctx context.Context, id int) (anonymized bool, err error)
}

type UserService struct {
	users UserAdminRepository
}

func NewUserService(users UserAdminRepository) *UserService {
	return &UserService{users: users}
}

func (s *UserService) List(ctx context.Context) ([]domain.User, error) {
	return s.users.List(ctx)
}

// Delete supprime le compte, ou l'anonymise s'il a des commandes.
// Elle renvoie true si le compte a été anonymisé.
func (s *UserService) Delete(ctx context.Context, adminID, targetID int) (bool, error) {
	if adminID == targetID {
		return false, domain.ErrCannotDeleteSelf
	}
	return s.users.DeleteOrAnonymize(ctx, targetID)
}
