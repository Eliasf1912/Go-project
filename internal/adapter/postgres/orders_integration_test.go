package postgres

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/ids"
)

// Ce test a besoin d'une vraie base PostgreSQL (schéma de sql/init.sql).
// Il est ignoré si TEST_DATABASE_URL n'est pas défini, par exemple :
//
//	TEST_DATABASE_URL="postgres://ecommerce:ecommerce@localhost:5432/ecommerce?sslmode=disable" go test ./internal/adapter/postgres/
func openTestDB(t *testing.T) *OrderRepository {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL non défini : test d'intégration ignoré")
	}
	db, err := Open(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return NewOrderRepository(db)
}

func TestCreate_ValidationsSimultaneesDuMemePanier(t *testing.T) {
	repo := openTestDB(t)
	ctx := context.Background()

	// Données de test dédiées, supprimées à la fin.
	var categoryID, productID, userID, cartID int
	mustScan := func(query string, dest *int, args ...any) {
		if err := repo.db.QueryRowContext(ctx, query, args...).Scan(dest); err != nil {
			t.Fatalf("préparation des données : %v", err)
		}
	}
	mustScan(`INSERT INTO categories (name) VALUES ('test-checkout') RETURNING id`, &categoryID)
	mustScan(`INSERT INTO products (reference, name, price_ht, category_id)
		VALUES ($1, 'Produit de test', 1000, $2) RETURNING id`, &productID, ids.NewProductRef(), categoryID)
	mustScan(`INSERT INTO users (email, password_hash, confirmed)
		VALUES ('checkout.test@test.com', 'x', true) RETURNING id`, &userID)
	mustScan(`INSERT INTO carts (reference, user_id) VALUES ($1, $2) RETURNING id`, &cartID, ids.NewCartRef(), userID)
	if _, err := repo.db.ExecContext(ctx,
		`INSERT INTO cart_items (cart_id, product_id, quantity) VALUES ($1, $2, 2)`, cartID, productID); err != nil {
		t.Fatalf("préparation des données : %v", err)
	}
	t.Cleanup(func() {
		repo.db.Exec(`DELETE FROM orders WHERE user_id = $1`, userID)
		repo.db.Exec(`DELETE FROM users WHERE id = $1`, userID) // supprime aussi le panier (ON DELETE CASCADE)
		repo.db.Exec(`DELETE FROM products WHERE id = $1`, productID)
		repo.db.Exec(`DELETE FROM categories WHERE id = $1`, categoryID)
	})

	// 20 validations simultanées, qui ont toutes lu le même panier plein :
	// exactement ce qui se passe quand l'utilisateur double-clique.
	const attempts = 20
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
		refused   int
	)
	for range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			order := domain.Order{
				Reference: ids.NewOrderRef(),
				UserID:    userID,
				CartID:    &cartID,
				Status:    domain.OrderStatusPending,
				Items:     []domain.OrderItem{{ProductID: productID, ProductName: "Produit de test", Quantity: 2, PriceHT: 1000}},
			}
			_, err := repo.Create(ctx, order)

			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				succeeded++
			case errors.Is(err, domain.ErrCartEmpty):
				refused++
			default:
				t.Errorf("Create() erreur inattendue : %v", err)
			}
		}()
	}
	wg.Wait()

	if succeeded != 1 || refused != attempts-1 {
		t.Errorf("%d validation(s) réussie(s), %d refusée(s) ; attendu 1 et %d", succeeded, refused, attempts-1)
	}

	var orders int
	mustScan(`SELECT count(*) FROM orders WHERE user_id = $1`, &orders, userID)
	if orders != 1 {
		t.Errorf("%d commande(s) en base, attendu 1", orders)
	}
}

func TestCreate_ClientInexistant(t *testing.T) {
	repo := openTestDB(t)
	ctx := context.Background()

	var categoryID, productID int
	if err := repo.db.QueryRowContext(ctx,
		`INSERT INTO categories (name) VALUES ('test-client-inexistant') RETURNING id`).Scan(&categoryID); err != nil {
		t.Fatalf("préparation des données : %v", err)
	}
	if err := repo.db.QueryRowContext(ctx, `INSERT INTO products (reference, name, price_ht, category_id)
		VALUES ($1, 'Produit de test', 1000, $2) RETURNING id`, ids.NewProductRef(), categoryID).Scan(&productID); err != nil {
		t.Fatalf("préparation des données : %v", err)
	}
	t.Cleanup(func() {
		repo.db.Exec(`DELETE FROM products WHERE id = $1`, productID)
		repo.db.Exec(`DELETE FROM categories WHERE id = $1`, categoryID)
	})

	// Commande créée par l'admin (sans panier) pour un client qui n'existe pas.
	reference := ids.NewOrderRef()
	_, err := repo.Create(ctx, domain.Order{
		Reference: reference,
		UserID:    999999,
		Status:    domain.OrderStatusPending,
		Items:     []domain.OrderItem{{ProductID: productID, ProductName: "Produit de test", Quantity: 1, PriceHT: 1000}},
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Create() = %v, attendu %v", err, domain.ErrNotFound)
	}

	var orders int
	if err := repo.db.QueryRowContext(ctx, `SELECT count(*) FROM orders WHERE reference = $1`, reference).Scan(&orders); err != nil {
		t.Fatal(err)
	}
	if orders != 0 {
		t.Errorf("%d commande(s) créée(s), attendu aucune", orders)
	}
}
