package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/usecase"
)

// Vérifie à la compilation que UserRepository implémente bien l'interface.
var _ usecase.UserRepository = (*UserRepository)(nil)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// userColumns liste les colonnes lues par scanUser, dans le même ordre que le Scan.
const userColumns = `id, email, password_hash, role, confirmed,
	confirmation_code, confirmation_expires_at,
	password_reset_code, password_reset_expires_at,
	created_at, updated_at, deleted_at`

type scanner interface {
	Scan(dest ...any) error
}

// scanUser lit une ligne de la table users et la convertit en domain.User.
func scanUser(row scanner) (domain.User, error) {
	var (
		user                   domain.User
		confirmationCode       sql.NullString
		confirmationExpiresAt  sql.NullTime
		passwordResetCode      sql.NullString
		passwordResetExpiresAt sql.NullTime
		deletedAt              sql.NullTime
	)

	err := row.Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.Confirmed,
		&confirmationCode, &confirmationExpiresAt,
		&passwordResetCode, &passwordResetExpiresAt,
		&user.CreatedAt, &user.UpdatedAt, &deletedAt,
	)
	if err != nil {
		return domain.User{}, err
	}

	user.ConfirmationCode = confirmationCode.String
	user.ConfirmationExpiresAt = nullTimeToPtr(confirmationExpiresAt)
	user.PasswordResetCode = passwordResetCode.String
	user.PasswordResetExpiresAt = nullTimeToPtr(passwordResetExpiresAt)
	user.DeletedAt = nullTimeToPtr(deletedAt)

	return user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email)
	user, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("recherche de l'utilisateur par email : %w", err)
	}
	return user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id int) (domain.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)
	user, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("recherche de l'utilisateur par id : %w", err)
	}
	return user, nil
}

func (r *UserRepository) Create(ctx context.Context, user domain.User) (int, error) {
	query := `
		INSERT INTO users (email, password_hash, role, confirmed,
		confirmation_code, confirmation_expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	var id int
	err := r.db.QueryRowContext(ctx, query,
		user.Email, user.PasswordHash, user.Role, user.Confirmed,
		stringToNull(user.ConfirmationCode), user.ConfirmationExpiresAt,
	).Scan(&id)
	if pgErrorCode(err) == codeUniqueViolation {
		return 0, domain.ErrEmailTaken
	}
	if err != nil {
		return 0, fmt.Errorf("création de l'utilisateur : %w", err)
	}
	return id, nil
}

func (r *UserRepository) Update(ctx context.Context, user domain.User) error {
	query := `
		UPDATE users SET
			email = $1, password_hash = $2, role = $3, confirmed = $4,
			confirmation_code = $5, confirmation_expires_at = $6,
			password_reset_code = $7, password_reset_expires_at = $8,
			deleted_at = $9, updated_at = now()
		WHERE id = $10`

	result, err := r.db.ExecContext(ctx, query,
		user.Email, user.PasswordHash, user.Role, user.Confirmed,
		stringToNull(user.ConfirmationCode), user.ConfirmationExpiresAt,
		stringToNull(user.PasswordResetCode), user.PasswordResetExpiresAt,
		user.DeletedAt, user.ID,
	)
	if err != nil {
		return fmt.Errorf("mise à jour de l'utilisateur : %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mise à jour de l'utilisateur : %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}
