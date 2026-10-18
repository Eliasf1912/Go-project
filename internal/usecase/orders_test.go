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
	nextID int
	// beforeUpdate, si défini, est appelé au début de UpdateStatus : il
	// permet de simuler une autre requête qui modifie la commande entre
	// la lecture (FindByID) et l'écriture (UpdateStatus).
	beforeUpdate func()
	// cartRepo, si défini, reproduit ce que fait la vraie transaction de
	// Create : vider le panier d'origine après avoir créé la commande.
	cartRepo *fakeCartRepo
}

func newFakeOrderRepo(orders ...domain.Order) *fakeOrderRepo {
	r := &fakeOrderRepo{orders: map[int]domain.Order{}, nextID: 1000}
	for _, o := range orders {
		r.orders[o.ID] = o
	}
	return r
}

// Create simule l'INSERT + le vidage du panier (voir postgres.OrderRepository.Create).
func (r *fakeOrderRepo) Create(_ context.Context, order domain.Order) (domain.Order, error) {
	order.ID = r.nextID
	r.nextID++
	for i := range order.Items {
		order.Items[i].ID = i + 1
	}
	r.orders[order.ID] = order

	if r.cartRepo != nil && order.CartID != nil {
		if cart, ok := r.cartRepo.carts[order.UserID]; ok && cart.ID == *order.CartID {
			cart.Items = nil
			r.cartRepo.carts[order.UserID] = cart
		}
	}

	return order, nil
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
	service := usecase.NewOrderService(repo, newFakeCartRepo(), &fakeProductLookup{})

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
	service := usecase.NewOrderService(repo, newFakeCartRepo(), &fakeProductLookup{})

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
			service := usecase.NewOrderService(repo, newFakeCartRepo(), &fakeProductLookup{})

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
	service := usecase.NewOrderService(repo, newFakeCartRepo(), &fakeProductLookup{})

	_, err := service.Pay(context.Background(), 10, 1, validCard)
	if !errors.Is(err, domain.ErrOrderStatusChanged) {
		t.Errorf("Pay() = %v, attendu %v", err, domain.ErrOrderStatusChanged)
	}
}

func TestCancel_Succes(t *testing.T) {
	repo := newFakeOrderRepo(domain.Order{ID: 1, UserID: 10, Status: domain.OrderStatusPaid})
	service := usecase.NewOrderService(repo, newFakeCartRepo(), &fakeProductLookup{})

	order, err := service.Cancel(context.Background(), 10, 1, "Je me suis trompé")
	if err != nil {
		t.Fatalf("Cancel() erreur inattendue : %v", err)
	}
	if order.Status != domain.OrderStatusCancelled {
		t.Errorf("statut renvoyé = %s, attendu cancelled", order.Status)
	}
	saved := repo.orders[1]
	if saved.Status != domain.OrderStatusCancelled {
		t.Errorf("statut enregistré = %s, attendu cancelled", saved.Status)
	}
	if saved.CancelReason == nil || *saved.CancelReason != "Je me suis trompé" {
		t.Errorf("raison enregistrée = %v, attendu %q", saved.CancelReason, "Je me suis trompé")
	}
}

func TestCancel_Erreurs(t *testing.T) {
	tests := []struct {
		name    string
		status  domain.OrderStatus
		userID  int
		orderID int
		reason  string
		wantErr error
	}{
		{"commande d'un autre utilisateur", domain.OrderStatusPending, 20, 1, "Trop cher", domain.ErrNotFound},
		{"commande inexistante", domain.OrderStatusPending, 10, 99, "Trop cher", domain.ErrNotFound},
		{"en cours de livraison", domain.OrderStatusShipping, 10, 1, "Trop cher", domain.ErrOrderNotCancellable},
		{"déjà annulée", domain.OrderStatusCancelled, 10, 1, "Trop cher", domain.ErrOrderNotCancellable},
		{"raison faite d'espaces", domain.OrderStatusPending, 10, 1, "   ", domain.ErrCancelReasonRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeOrderRepo(domain.Order{ID: 1, UserID: 10, Status: tt.status})
			service := usecase.NewOrderService(repo, newFakeCartRepo(), &fakeProductLookup{})

			_, err := service.Cancel(context.Background(), tt.userID, tt.orderID, tt.reason)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Cancel() = %v, attendu %v", err, tt.wantErr)
			}
			// Une annulation refusée ne doit jamais modifier la commande.
			if got := repo.orders[1]; got.Status != tt.status || got.CancelReason != nil {
				t.Errorf("commande modifiée : statut %s, raison %v", got.Status, got.CancelReason)
			}
		})
	}
}

func TestCancel_StatutChangeEntreTemps(t *testing.T) {
	repo := newFakeOrderRepo(domain.Order{ID: 1, UserID: 10, Status: domain.OrderStatusPaid})
	// L'admin passe la commande en livraison juste avant notre UpdateStatus.
	repo.beforeUpdate = func() {
		o := repo.orders[1]
		o.Status = domain.OrderStatusShipping
		repo.orders[1] = o
	}
	service := usecase.NewOrderService(repo, newFakeCartRepo(), &fakeProductLookup{})

	_, err := service.Cancel(context.Background(), 10, 1, "Trop cher")
	if !errors.Is(err, domain.ErrOrderStatusChanged) {
		t.Errorf("Cancel() = %v, attendu %v", err, domain.ErrOrderStatusChanged)
	}
	if got := repo.orders[1].Status; got != domain.OrderStatusShipping {
		t.Errorf("statut = %s, attendu shipping (inchangé)", got)
	}
}

func TestCheckout_Succes(t *testing.T) {
	cartRepo := newFakeCartRepo()
	cart, err := cartRepo.Create(context.Background(), 10)
	if err != nil {
		t.Fatalf("Create() du panier, erreur inattendue : %v", err)
	}
	if err := cartRepo.AddItem(context.Background(), cart.ID, 1, 2); err != nil {
		t.Fatalf("AddItem() erreur inattendue : %v", err)
	}

	products := &fakeProductLookup{active: map[int]domain.Product{1: {ID: 1}}}
	orderRepo := newFakeOrderRepo()
	orderRepo.cartRepo = cartRepo
	service := usecase.NewOrderService(orderRepo, cartRepo, products)

	order, err := service.Checkout(context.Background(), 10)
	if err != nil {
		t.Fatalf("Checkout() erreur inattendue : %v", err)
	}
	if order.Status != domain.OrderStatusPending {
		t.Errorf("statut = %s, attendu pending", order.Status)
	}
	if order.CartID == nil || *order.CartID != cart.ID {
		t.Errorf("CartID = %v, attendu %d", order.CartID, cart.ID)
	}
	if len(order.Items) != 1 || order.Items[0].ProductID != 1 || order.Items[0].Quantity != 2 {
		t.Errorf("lignes de la commande = %+v, inattendues", order.Items)
	}

	// Le panier doit être vidé après validation.
	emptied, err := cartRepo.FindByUserID(context.Background(), 10)
	if err != nil {
		t.Fatalf("FindByUserID() erreur inattendue : %v", err)
	}
	if len(emptied.Items) != 0 {
		t.Errorf("panier après Checkout = %+v, attendu vide", emptied.Items)
	}
}

func TestCheckout_PanierVideOuAbsent(t *testing.T) {
	t.Run("aucun panier créé", func(t *testing.T) {
		cartRepo := newFakeCartRepo()
		service := usecase.NewOrderService(newFakeOrderRepo(), cartRepo, &fakeProductLookup{})

		_, err := service.Checkout(context.Background(), 10)
		if !errors.Is(err, domain.ErrCartEmpty) {
			t.Errorf("Checkout() = %v, attendu %v", err, domain.ErrCartEmpty)
		}
	})

	t.Run("panier créé mais sans ligne", func(t *testing.T) {
		cartRepo := newFakeCartRepo()
		if _, err := cartRepo.Create(context.Background(), 10); err != nil {
			t.Fatalf("Create() erreur inattendue : %v", err)
		}
		service := usecase.NewOrderService(newFakeOrderRepo(), cartRepo, &fakeProductLookup{})

		_, err := service.Checkout(context.Background(), 10)
		if !errors.Is(err, domain.ErrCartEmpty) {
			t.Errorf("Checkout() = %v, attendu %v", err, domain.ErrCartEmpty)
		}
	})
}

func TestCheckout_ProduitArchive(t *testing.T) {
	cartRepo := newFakeCartRepo()
	cart, err := cartRepo.Create(context.Background(), 10)
	if err != nil {
		t.Fatalf("Create() du panier, erreur inattendue : %v", err)
	}
	if err := cartRepo.AddItem(context.Background(), cart.ID, 1, 2); err != nil {
		t.Fatalf("AddItem() erreur inattendue : %v", err)
	}

	// Aucun produit actif : le produit 1 a été archivé depuis son ajout.
	orderRepo := newFakeOrderRepo()
	service := usecase.NewOrderService(orderRepo, cartRepo, &fakeProductLookup{})

	_, err = service.Checkout(context.Background(), 10)
	if !errors.Is(err, domain.ErrCartHasInactiveProduct) {
		t.Errorf("Checkout() = %v, attendu %v", err, domain.ErrCartHasInactiveProduct)
	}
	// Un checkout refusé ne doit ni créer de commande, ni vider le panier.
	if len(orderRepo.orders) != 0 {
		t.Errorf("une commande a été créée malgré le refus : %+v", orderRepo.orders)
	}
	stillThere, err := cartRepo.FindByUserID(context.Background(), 10)
	if err != nil {
		t.Fatalf("FindByUserID() erreur inattendue : %v", err)
	}
	if len(stillThere.Items) != 1 {
		t.Errorf("panier après refus = %+v, attendu inchangé", stillThere.Items)
	}
}
