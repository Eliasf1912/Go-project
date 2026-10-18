package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/usecase"
)

var (
	_ usecase.CartRepository = (*fakeCartRepo)(nil)
	_ usecase.ProductLookup  = (*fakeProductLookup)(nil)
)

// fakeProductLookup simule la vérification "produit actif" : seuls les
// produits présents dans la map sont considérés actifs.
type fakeProductLookup struct {
	active map[int]domain.Product
}

func (r *fakeProductLookup) FindActiveByID(_ context.Context, id int) (domain.Product, error) {
	p, ok := r.active[id]
	if !ok {
		return domain.Product{}, domain.ErrNotFound
	}
	return p, nil
}

// fakeCartRepo reproduit la contrainte réelle : un seul panier par
// utilisateur (clé = userID), créé seulement par Create.
type fakeCartRepo struct {
	carts  map[int]domain.Cart // clé : userID
	nextID int
}

func newFakeCartRepo() *fakeCartRepo {
	return &fakeCartRepo{carts: map[int]domain.Cart{}, nextID: 1}
}

func (r *fakeCartRepo) FindByUserID(_ context.Context, userID int) (domain.Cart, error) {
	cart, ok := r.carts[userID]
	if !ok {
		return domain.Cart{}, domain.ErrNotFound
	}
	return cart, nil
}

func (r *fakeCartRepo) Create(_ context.Context, userID int) (domain.Cart, error) {
	cart := domain.Cart{ID: r.nextID, Reference: "BSK-TEST00", UserID: userID}
	r.nextID++
	r.carts[userID] = cart
	return cart, nil
}

func (r *fakeCartRepo) findCartByID(cartID int) (int, domain.Cart, bool) {
	for userID, cart := range r.carts {
		if cart.ID == cartID {
			return userID, cart, true
		}
	}
	return 0, domain.Cart{}, false
}

func (r *fakeCartRepo) AddItem(_ context.Context, cartID, productID, quantity int) error {
	userID, cart, ok := r.findCartByID(cartID)
	if !ok {
		return domain.ErrNotFound
	}
	for i := range cart.Items {
		if cart.Items[i].ProductID == productID {
			cart.Items[i].Quantity += quantity
			r.carts[userID] = cart
			return nil
		}
	}
	cart.Items = append(cart.Items, domain.CartItem{ProductID: productID, Quantity: quantity})
	r.carts[userID] = cart
	return nil
}

func (r *fakeCartRepo) SetItemQuantity(_ context.Context, cartID, productID, quantity int) error {
	userID, cart, ok := r.findCartByID(cartID)
	if !ok {
		return domain.ErrNotFound
	}
	for i := range cart.Items {
		if cart.Items[i].ProductID == productID {
			if quantity == 0 {
				cart.Items = append(cart.Items[:i], cart.Items[i+1:]...)
			} else {
				cart.Items[i].Quantity = quantity
			}
			r.carts[userID] = cart
			return nil
		}
	}
	return domain.ErrNotFound
}

func (r *fakeCartRepo) RemoveItem(_ context.Context, cartID, productID int) error {
	userID, cart, ok := r.findCartByID(cartID)
	if !ok {
		return domain.ErrNotFound
	}
	for i := range cart.Items {
		if cart.Items[i].ProductID == productID {
			cart.Items = append(cart.Items[:i], cart.Items[i+1:]...)
			r.carts[userID] = cart
			return nil
		}
	}
	return domain.ErrNotFound
}

func TestCartGetMine_PanierVideSiInexistant(t *testing.T) {
	service := usecase.NewCartService(newFakeCartRepo(), &fakeProductLookup{})

	cart, err := service.GetMine(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetMine() erreur inattendue : %v", err)
	}
	if cart.UserID != 10 || len(cart.Items) != 0 {
		t.Errorf("GetMine() = %+v, attendu un panier vide pour l'utilisateur 10", cart)
	}
}

func TestCartAddItem_CreeLePanierSiAbsent(t *testing.T) {
	repo := newFakeCartRepo()
	products := &fakeProductLookup{active: map[int]domain.Product{1: {ID: 1}}}
	service := usecase.NewCartService(repo, products)

	cart, err := service.AddItem(context.Background(), 10, 1, 2)
	if err != nil {
		t.Fatalf("AddItem() erreur inattendue : %v", err)
	}
	if len(cart.Items) != 1 || cart.Items[0].Quantity != 2 {
		t.Errorf("AddItem() = %+v, attendu une ligne quantité 2", cart.Items)
	}
}

func TestCartAddItem_AugmenteQuantiteExistante(t *testing.T) {
	repo := newFakeCartRepo()
	products := &fakeProductLookup{active: map[int]domain.Product{1: {ID: 1}}}
	service := usecase.NewCartService(repo, products)

	if _, err := service.AddItem(context.Background(), 10, 1, 2); err != nil {
		t.Fatalf("premier AddItem() erreur inattendue : %v", err)
	}
	cart, err := service.AddItem(context.Background(), 10, 1, 3)
	if err != nil {
		t.Fatalf("second AddItem() erreur inattendue : %v", err)
	}
	if len(cart.Items) != 1 || cart.Items[0].Quantity != 5 {
		t.Errorf("AddItem() = %+v, attendu une seule ligne quantité 5", cart.Items)
	}
}

func TestCartAddItem_QuantiteInvalide(t *testing.T) {
	repo := newFakeCartRepo()
	products := &fakeProductLookup{active: map[int]domain.Product{1: {ID: 1}}}
	service := usecase.NewCartService(repo, products)

	for _, q := range []int{0, -1} {
		if _, err := service.AddItem(context.Background(), 10, 1, q); !errors.Is(err, domain.ErrInvalidQuantity) {
			t.Errorf("AddItem(quantity=%d) = %v, attendu %v", q, err, domain.ErrInvalidQuantity)
		}
	}
	if _, err := repo.FindByUserID(context.Background(), 10); !errors.Is(err, domain.ErrNotFound) {
		t.Error("un panier a été créé malgré une quantité invalide")
	}
}

func TestCartAddItem_ProduitInactifOuInexistant(t *testing.T) {
	repo := newFakeCartRepo()
	service := usecase.NewCartService(repo, &fakeProductLookup{}) // aucun produit actif

	if _, err := service.AddItem(context.Background(), 10, 1, 1); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("AddItem() = %v, attendu %v", err, domain.ErrNotFound)
	}
	if _, err := repo.FindByUserID(context.Background(), 10); !errors.Is(err, domain.ErrNotFound) {
		t.Error("un panier a été créé malgré un produit invalide")
	}
}

func TestCartUpdateItem_FixeQuantite(t *testing.T) {
	repo := newFakeCartRepo()
	products := &fakeProductLookup{active: map[int]domain.Product{1: {ID: 1}}}
	service := usecase.NewCartService(repo, products)
	if _, err := service.AddItem(context.Background(), 10, 1, 2); err != nil {
		t.Fatalf("AddItem() erreur inattendue : %v", err)
	}

	cart, err := service.UpdateItem(context.Background(), 10, 1, 7)
	if err != nil {
		t.Fatalf("UpdateItem() erreur inattendue : %v", err)
	}
	if len(cart.Items) != 1 || cart.Items[0].Quantity != 7 {
		t.Errorf("UpdateItem() = %+v, attendu une ligne quantité 7", cart.Items)
	}
}

func TestCartUpdateItem_ZeroSupprimeLaLigne(t *testing.T) {
	repo := newFakeCartRepo()
	products := &fakeProductLookup{active: map[int]domain.Product{1: {ID: 1}}}
	service := usecase.NewCartService(repo, products)
	if _, err := service.AddItem(context.Background(), 10, 1, 2); err != nil {
		t.Fatalf("AddItem() erreur inattendue : %v", err)
	}

	cart, err := service.UpdateItem(context.Background(), 10, 1, 0)
	if err != nil {
		t.Fatalf("UpdateItem() erreur inattendue : %v", err)
	}
	if len(cart.Items) != 0 {
		t.Errorf("UpdateItem(quantity=0) = %+v, attendu un panier sans cette ligne", cart.Items)
	}
}

func TestCartUpdateItem_QuantiteNegativeInvalide(t *testing.T) {
	repo := newFakeCartRepo()
	service := usecase.NewCartService(repo, &fakeProductLookup{})

	if _, err := service.UpdateItem(context.Background(), 10, 1, -1); !errors.Is(err, domain.ErrInvalidQuantity) {
		t.Errorf("UpdateItem(quantity=-1) = %v, attendu %v", err, domain.ErrInvalidQuantity)
	}
}

func TestCartUpdateItem_PanierOuLigneInexistante(t *testing.T) {
	repo := newFakeCartRepo()
	service := usecase.NewCartService(repo, &fakeProductLookup{})

	// Pas de panier du tout pour cet utilisateur.
	if _, err := service.UpdateItem(context.Background(), 10, 1, 5); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("UpdateItem() sans panier = %v, attendu %v", err, domain.ErrNotFound)
	}

	// Panier existant mais sans cette ligne.
	if _, err := repo.Create(context.Background(), 10); err != nil {
		t.Fatalf("Create() erreur inattendue : %v", err)
	}
	if _, err := service.UpdateItem(context.Background(), 10, 1, 5); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("UpdateItem() sans cette ligne = %v, attendu %v", err, domain.ErrNotFound)
	}
}

func TestCartRemoveItem_Succes(t *testing.T) {
	repo := newFakeCartRepo()
	products := &fakeProductLookup{active: map[int]domain.Product{1: {ID: 1}}}
	service := usecase.NewCartService(repo, products)
	if _, err := service.AddItem(context.Background(), 10, 1, 2); err != nil {
		t.Fatalf("AddItem() erreur inattendue : %v", err)
	}

	cart, err := service.RemoveItem(context.Background(), 10, 1)
	if err != nil {
		t.Fatalf("RemoveItem() erreur inattendue : %v", err)
	}
	if len(cart.Items) != 0 {
		t.Errorf("RemoveItem() = %+v, attendu un panier vide", cart.Items)
	}
}

func TestCartRemoveItem_ProduitAbsent(t *testing.T) {
	repo := newFakeCartRepo()
	if _, err := repo.Create(context.Background(), 10); err != nil {
		t.Fatalf("Create() erreur inattendue : %v", err)
	}
	service := usecase.NewCartService(repo, &fakeProductLookup{})

	if _, err := service.RemoveItem(context.Background(), 10, 1); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("RemoveItem() = %v, attendu %v", err, domain.ErrNotFound)
	}
}
