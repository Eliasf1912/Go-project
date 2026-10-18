package postgres

import (
	"context"
	"fmt"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/usecase"
)

var _ usecase.UserAdminRepository = (*UserRepository)(nil)

func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+userColumns+` FROM users ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("liste des utilisateurs : %w", err)
	}
	defer rows.Close()

	users := []domain.User{}
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("lecture d'un utilisateur : %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("liste des utilisateurs : %w", err)
	}
	return users, nil
}

// DeleteOrAnonymize supprime l'utilisateur s'il n'a aucune commande,
// sinon l'anonymise. Tout se fait dans une transaction : soit tout réussit,
// soit rien n'est modifié.
func (r *UserRepository) DeleteOrAnonymize(ctx context.Context, id int) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("début de la transaction : %w", err)
	}
	defer tx.Rollback() // sans effet si Commit a réussi

	var hasOrders bool
	err = tx.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM orders WHERE user_id = $1)`, id,
	).Scan(&hasOrders)
	if err != nil {
		return false, fmt.Errorf("recherche des commandes : %w", err)
	}

	if !hasOrders {
		// Sessions et panier sont supprimés avec lui (ON DELETE CASCADE).
		result, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
		if err != nil {
			return false, fmt.Errorf("suppression de l'utilisateur : %w", err)
		}
		if n, err := result.RowsAffected(); err != nil {
			return false, fmt.Errorf("suppression de l'utilisateur : %w", err)
		} else if n == 0 {
			return false, domain.ErrNotFound
		}
		return false, tx.Commit()
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE orders SET status = 'cancelled', cancel_reason = 'Compte supprimé', updated_at = now()
		WHERE user_id = $1 AND status = 'pending'`, id)
	if err != nil {
		return false, fmt.Errorf("annulation des commandes en attente : %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = $1`, id); err != nil {
		return false, fmt.Errorf("suppression des sessions : %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM carts WHERE user_id = $1`, id); err != nil {
		return false, fmt.Errorf("suppression du panier : %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE users SET
			email = $2, password_hash = '', confirmed = false,
			confirmation_code = NULL, confirmation_expires_at = NULL,
			password_reset_code = NULL, password_reset_expires_at = NULL,
			deleted_at = now(), updated_at = now()
		WHERE id = $1`, id, fmt.Sprintf("supprime-%d@anonyme.local", id))
	if err != nil {
		return false, fmt.Errorf("anonymisation de l'utilisateur : %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("validation de la transaction : %w", err)
	}
	return true, nil
}
