package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/usecase"
)

var _ usecase.OrderRepository = (*fakeOrderRepo)(nil)

type fakeOrderRepo struct {
	orders map[int]domain.Order
	// beforeUpdate, si défini, est appelé au début de UpdateStatus : il
	// permet de simuler une autre requête qui modifie la commande entre
	// la lecture (FindByID) et l'écriture (UpdateStatus).
	beforeUpdate func()
}

func newFakeOrderRepo(orders ...domain.Order) *fakeOrderRepo {
	r := &fakeOrderRepo{orders: map[int]domain.Order{}}
	for _, o := range orders {
		r.orders[o.ID] = o
	}
	return r
}

func (r *fakeOrderRepo) ListByUserID(_ context.Context, userID int) ([]domain.Order, error) {
	var orders []domain.Order
	for _, o := range r.orders {
		if o.UserID == userID {
			orders = append(orders, o)
		}
	}
	return orders, nil
}

func (r *fakeOrderRepo) FindByID(_ context.Context, id int) (domain.Order, error) {
	o, ok := r.orders[id]
	if !ok {
		return domain.Order{}, domain.ErrNotFound
	}
	return o, nil
}

// UpdateStatus reproduit le "WHERE status = from" du vrai repository.
func (r *fakeOrderRepo) UpdateStatus(_ context.Context, id int, from, to domain.OrderStatus, cancelReason *string) error {
	if r.beforeUpdate != nil {
		r.beforeUpdate()
	}
	o, ok := r.orders[id]
	if !ok || o.Status != from {
		return domain.ErrOrderStatusChanged
	}
	o.Status = to
	o.CancelReason = cancelReason
	r.orders[id] = o
	return nil
}

var validCard = domain.Card{Number: "4242424242424242", Expiry: "12/99", CVC: "123"}

func TestListMine_RenvoieSeulementSesCommandes(t *testing.T) {
	repo := newFakeOrderRepo(
		domain.Order{ID: 1, UserID: 10},
		domain.Order{ID: 2, UserID: 10},
		domain.Order{ID: 3, UserID: 20},
	)
	service := usecase.NewOrderService(repo)

	orders, err := service.ListMine(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListMine() erreur inattendue : %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("attendu 2 commandes, obtenu %d", len(orders))
	}
	for _, o := range orders {
		if o.UserID != 10 {
			t.Errorf("commande %d d'un autre utilisateur renvoyée", o.ID)
		}
	}
}

func TestPay_Succes(t *testing.T) {
	repo := newFakeOrderRepo(domain.Order{ID: 1, UserID: 10, Status: domain.OrderStatusPending})
	service := usecase.NewOrderService(repo)

	order, err := service.Pay(context.Background(), 10, 1, validCard)
	if err != nil {
		t.Fatalf("Pay() erreur inattendue : %v", err)
	}
	if order.Status != domain.OrderStatusPaid {
		t.Errorf("statut renvoyé = %s, attendu paid", order.Status)
	}
	if got := repo.orders[1].Status; got != domain.OrderStatusPaid {
		t.Errorf("statut enregistré = %s, attendu paid", got)
	}
}

func TestPay_Erreurs(t *testing.T) {
	tests := []struct {
		name    string
		status  domain.OrderStatus
		userID  int
		orderID int
		card    domain.Card
		wantErr error
	}{
		{"carte invalide", domain.OrderStatusPending, 10, 1,
			domain.Card{Number: "1234", Expiry: "12/99", CVC: "123"}, domain.ErrInvalidCardNumber},
		{"commande d'un autre utilisateur", domain.OrderStatusPending, 20, 1, validCard, domain.ErrNotFound},
		{"commande inexistante", domain.OrderStatusPending, 10, 99, validCard, domain.ErrNotFound},
		{"déjà payée", domain.OrderStatusPaid, 10, 1, validCard, domain.ErrOrderNotPending},
		{"annulée", domain.OrderStatusCancelled, 10, 1, validCard, domain.ErrOrderNotPending},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeOrderRepo(domain.Order{ID: 1, UserID: 10, Status: tt.status})
			service := usecase.NewOrderService(repo)

			_, err := service.Pay(context.Background(), tt.userID, tt.orderID, tt.card)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Pay() = %v, attendu %v", err, tt.wantErr)
			}
			// Un paiement refusé ne doit jamais modifier la commande.
			if got := repo.orders[1].Status; got != tt.status {
				t.Errorf("statut enregistré = %s, attendu %s (inchangé)", got, tt.status)
			}
		})
	}
}

func TestPay_StatutChangeEntreTemps(t *testing.T) {
	repo := newFakeOrderRepo(domain.Order{ID: 1, UserID: 10, Status: domain.OrderStatusPending})
	// Une autre requête paie la commande juste avant notre UpdateStatus.
	repo.beforeUpdate = func() {
		o := repo.orders[1]
		o.Status = domain.OrderStatusPaid
		repo.orders[1] = o
	}
	service := usecase.NewOrderService(repo)

	_, err := service.Pay(context.Background(), 10, 1, validCard)
	if !errors.Is(err, domain.ErrOrderStatusChanged) {
		t.Errorf("Pay() = %v, attendu %v", err, domain.ErrOrderStatusChanged)
	}
}
