package usecase

import (
	"context"
	"errors"

	"github.com/Eliasf1912/go-project/internal/domain"
)

// CartRepository est le port implémenté par internal/adapter/postgres
type CartRepository interface {
	// FindByUserID renvoie le panier de l'utilisateur, avec ses lignes
	// (prix courants des produits). ErrNotFound si l'utilisateur n'a pas
	// encore de panier (aucun ajout fait).
	FindByUserID(ctx context.Context, userID int) (domain.Cart, error)
	// Create crée le panier de l'utilisateur (une seule fois : carts.user_id
	// est UNIQUE en base), avec une référence générée (BSK-XXXXXX).
	Create(ctx context.Context, userID int) (domain.Cart, error)
	// AddItem ajoute quantity au panier pour ce produit : augmente la
	// quantité si la ligne existe déjà, la crée sinon.
	AddItem(ctx context.Context, cartID, productID, quantity int) error
	// SetItemQuantity fixe la quantité d'une ligne déjà présente dans le
	// panier (quantity > 0). ErrNotFound si le produit n'y est pas.
	SetItemQuantity(ctx context.Context, cartID, productID, quantity int) error
	// RemoveItem supprime une ligne du panier. ErrNotFound si elle n'y est pas.
	RemoveItem(ctx context.Context, cartID, productID int) error
}

// ProductLookup est le sous-ensemble des produits dont le panier a besoin :
// vérifier qu'un produit existe et n'est pas archivé avant de l'ajouter.
// Implémenté par le même adapter postgres que ProductRepository.
type ProductLookup interface {
	FindActiveByID(ctx context.Context, id int) (domain.Product, error)
}

type CartService struct {
	carts    CartRepository
	products ProductLookup
}

func NewCartService(carts CartRepository, products ProductLookup) *CartService {
	return &CartService{carts: carts, products: products}
}

// GetMine renvoie le panier de l'utilisateur. S'il n'en a pas encore,
// renvoie un panier vide sans le créer en base : la ligne carts n'est
// créée qu'au premier ajout (AddItem).
func (s *CartService) GetMine(ctx context.Context, userID int) (domain.Cart, error) {
	cart, err := s.carts.FindByUserID(ctx, userID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.Cart{UserID: userID}, nil
	}
	return cart, err
}

// AddItem ajoute quantity exemplaires d'un produit actif au panier de
// l'utilisateur, en créant le panier s'il n'existe pas encore. Si le
// produit est déjà dans le panier, sa quantité est augmentée.
func (s *CartService) AddItem(ctx context.Context, userID, productID, quantity int) (domain.Cart, error) {
	if quantity < 1 {
		return domain.Cart{}, domain.ErrInvalidQuantity
	}

	if _, err := s.products.FindActiveByID(ctx, productID); err != nil {
		return domain.Cart{}, err
	}

	cart, err := s.getOrCreateCart(ctx, userID)
	if err != nil {
		return domain.Cart{}, err
	}

	if err := s.carts.AddItem(ctx, cart.ID, productID, quantity); err != nil {
		return domain.Cart{}, err
	}

	return s.carts.FindByUserID(ctx, userID)
}

// UpdateItem fixe la quantité d'un produit déjà présent dans le panier de
// l'utilisateur. Une quantité de 0 supprime la ligne.
func (s *CartService) UpdateItem(ctx context.Context, userID, productID, quantity int) (domain.Cart, error) {
	if quantity < 0 {
		return domain.Cart{}, domain.ErrInvalidQuantity
	}

	cart, err := s.carts.FindByUserID(ctx, userID)
	if err != nil {
		return domain.Cart{}, err
	}

	if err := s.carts.SetItemQuantity(ctx, cart.ID, productID, quantity); err != nil {
		return domain.Cart{}, err
	}

	return s.carts.FindByUserID(ctx, userID)
}

// RemoveItem supprime un produit du panier de l'utilisateur.
func (s *CartService) RemoveItem(ctx context.Context, userID, productID int) (domain.Cart, error) {
	cart, err := s.carts.FindByUserID(ctx, userID)
	if err != nil {
		return domain.Cart{}, err
	}

	if err := s.carts.RemoveItem(ctx, cart.ID, productID); err != nil {
		return domain.Cart{}, err
	}

	return s.carts.FindByUserID(ctx, userID)
}

// getOrCreateCart renvoie le panier de l'utilisateur, ou le crée s'il n'en
// a pas encore.
func (s *CartService) getOrCreateCart(ctx context.Context, userID int) (domain.Cart, error) {
	cart, err := s.carts.FindByUserID(ctx, userID)
	if errors.Is(err, domain.ErrNotFound) {
		return s.carts.Create(ctx, userID)
	}
	return cart, err
}
