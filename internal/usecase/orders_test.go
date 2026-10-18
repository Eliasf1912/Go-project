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

func (r *fakeOrderRepo) ListAll(_ context.Context) ([]domain.Order, error) {
	orders := make([]domain.Order, 0, len(r.orders))
	for _, o := range r.orders {
		orders = append(orders, o)
	}
	return orders, nil
}

func (r *fakeOrderRepo) Delete(_ context.Context, id int) error {
	if _, ok := r.orders[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.orders, id)
	return nil
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

func TestListAll_RenvoieToutesLesCommandes(t *testing.T) {
	repo := newFakeOrderRepo(
		domain.Order{ID: 1, UserID: 10},
		domain.Order{ID: 2, UserID: 20},
	)
	service := usecase.NewOrderService(repo, newFakeCartRepo(), &fakeProductLookup{})

	orders, err := service.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll() erreur inattendue : %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("attendu 2 commandes, obtenu %d", len(orders))
	}
}

func TestCreateForUser_Succes(t *testing.T) {
	products := &fakeProductLookup{active: map[int]domain.Product{
		1: {ID: 1, Name: "Clavier", PriceHT: 4999},
	}}
	orderRepo := newFakeOrderRepo()
	service := usecase.NewOrderService(orderRepo, newFakeCartRepo(), products)

	order, err := service.CreateForUser(context.Background(), 10, []usecase.OrderLine{
		{ProductID: 1, Quantity: 2},
	})
	if err != nil {
		t.Fatalf("CreateForUser() erreur inattendue : %v", err)
	}
	if order.UserID != 10 || order.Status != domain.OrderStatusPending {
		t.Errorf("commande = %+v, inattendue", order)
	}
	if order.CartID != nil {
		t.Errorf("CartID = %v, attendu nil (pas de panier)", order.CartID)
	}
	if len(order.Items) != 1 || order.Items[0].ProductName != "Clavier" || order.Items[0].PriceHT != 4999 {
		t.Errorf("lignes = %+v, inattendues", order.Items)
	}
}

func TestCreateForUser_Erreurs(t *testing.T) {
	products := &fakeProductLookup{active: map[int]domain.Product{1: {ID: 1}}}

	t.Run("produit archivé ou inexistant", func(t *testing.T) {
		service := usecase.NewOrderService(newFakeOrderRepo(), newFakeCartRepo(), products)
		_, err := service.CreateForUser(context.Background(), 10, []usecase.OrderLine{{ProductID: 99, Quantity: 1}})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("CreateForUser() = %v, attendu %v", err, domain.ErrNotFound)
		}
	})

	t.Run("quantité invalide", func(t *testing.T) {
		service := usecase.NewOrderService(newFakeOrderRepo(), newFakeCartRepo(), products)
		_, err := service.CreateForUser(context.Background(), 10, []usecase.OrderLine{{ProductID: 1, Quantity: 0}})
		if !errors.Is(err, domain.ErrInvalidQuantity) {
			t.Errorf("CreateForUser() = %v, attendu %v", err, domain.ErrInvalidQuantity)
		}
	})

	t.Run("aucune ligne", func(t *testing.T) {
		service := usecase.NewOrderService(newFakeOrderRepo(), newFakeCartRepo(), products)
		_, err := service.CreateForUser(context.Background(), 10, nil)
		if !errors.Is(err, domain.ErrOrderHasNoItems) {
			t.Errorf("CreateForUser() = %v, attendu %v", err, domain.ErrOrderHasNoItems)
		}
	})
}

func TestShip(t *testing.T) {
	tests := []struct {
		name    string
		from    domain.OrderStatus
		wantErr error
	}{
		{"commande payée", domain.OrderStatusPaid, nil},
		{"commande en attente", domain.OrderStatusPending, domain.ErrOrderNotPaid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeOrderRepo(domain.Order{ID: 1, Status: tt.from})
			service := usecase.NewOrderService(repo, newFakeCartRepo(), &fakeProductLookup{})

			order, err := service.Ship(context.Background(), 1)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Ship() = %v, attendu %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if order.Status != domain.OrderStatusShipping {
					t.Errorf("statut renvoyé = %s, attendu shipping", order.Status)
				}
				if got := repo.orders[1].Status; got != domain.OrderStatusShipping {
					t.Errorf("statut enregistré = %s, attendu shipping", got)
				}
			}
		})
	}
}

func TestDeliver(t *testing.T) {
	tests := []struct {
		name    string
		from    domain.OrderStatus
		wantErr error
	}{
		{"commande en livraison", domain.OrderStatusShipping, nil},
		{"commande payée mais pas expédiée", domain.OrderStatusPaid, domain.ErrOrderNotShipping},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeOrderRepo(domain.Order{ID: 1, Status: tt.from})
			service := usecase.NewOrderService(repo, newFakeCartRepo(), &fakeProductLookup{})

			order, err := service.Deliver(context.Background(), 1)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Deliver() = %v, attendu %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && order.Status != domain.OrderStatusDelivered {
				t.Errorf("statut renvoyé = %s, attendu delivered", order.Status)
			}
		})
	}
}

func TestCancelAsAdmin(t *testing.T) {
	// À la différence de Cancel, aucune vérification de propriétaire :
	// l'admin annule la commande d'un client qu'il n'est pas lui-même.
	repo := newFakeOrderRepo(domain.Order{ID: 1, UserID: 10, Status: domain.OrderStatusPaid})
	service := usecase.NewOrderService(repo, newFakeCartRepo(), &fakeProductLookup{})

	order, err := service.CancelAsAdmin(context.Background(), 1, "Rupture de stock")
	if err != nil {
		t.Fatalf("CancelAsAdmin() erreur inattendue : %v", err)
	}
	if order.Status != domain.OrderStatusCancelled {
		t.Errorf("statut = %s, attendu cancelled", order.Status)
	}
	if order.CancelReason == nil || *order.CancelReason != "Rupture de stock" {
		t.Errorf("raison = %v, attendu %q", order.CancelReason, "Rupture de stock")
	}
}

func TestDelete(t *testing.T) {
	t.Run("commande existante", func(t *testing.T) {
		repo := newFakeOrderRepo(domain.Order{ID: 1, UserID: 10})
		service := usecase.NewOrderService(repo, newFakeCartRepo(), &fakeProductLookup{})

		if err := service.Delete(context.Background(), 1); err != nil {
			t.Fatalf("Delete() erreur inattendue : %v", err)
		}
		if _, ok := repo.orders[1]; ok {
			t.Error("la commande existe encore après Delete()")
		}
	})

	t.Run("commande inexistante", func(t *testing.T) {
		repo := newFakeOrderRepo()
		service := usecase.NewOrderService(repo, newFakeCartRepo(), &fakeProductLookup{})

		if err := service.Delete(context.Background(), 99); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("Delete() = %v, attendu %v", err, domain.ErrNotFound)
		}
	})
}
