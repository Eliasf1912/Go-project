package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // enregistre le driver "pgx" auprès de database/sql
)

// Open ouvre la connexion à PostgreSQL et vérifie qu'elle répond.
func Open(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("ouverture de la base : %w", err)
	}

	// sql.Open ne se connecte pas encore : Ping force une vraie connexion.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connexion à la base : %w", err)
	}

	return db, nil
}
