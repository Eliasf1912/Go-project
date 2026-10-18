package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/usecase"
)

// Vérifie à la compilation que SessionRepository implémente bien l'interface.
var _ usecase.SessionRepository = (*SessionRepository)(nil)

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, session domain.Session) error {
	query := `
		INSERT INTO sessions (token, user_id, expires_at)
		VALUES ($1, $2, $3)`

	_, err := r.db.ExecContext(ctx, query, session.Token, session.UserID, session.ExpiresAt)
	if err != nil {
		return fmt.Errorf("création de la session : %w", err)
	}
	return nil
}

func (r *SessionRepository) FindByToken(ctx context.Context, token string) (domain.Session, error) {
	query := `SELECT token, user_id, created_at, expires_at FROM sessions WHERE token = $1`

	var session domain.Session
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&session.Token, &session.UserID, &session.CreatedAt, &session.ExpiresAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Session{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Session{}, fmt.Errorf("recherche de la session : %w", err)
	}
	return session, nil
}
