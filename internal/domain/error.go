package domain

import "errors"

var (
	ErrInvalidCategoryName     = errors.New("le nom de la catégorie est obligatoire")
	ErrNotFound                = errors.New("ressource non trouvé")
	ErrEmailTaken              = errors.New("l'email est déjà utilisé")
	ErrInvalidEmail            = errors.New("l'email est invalide")
	ErrPasswordTooShort        = errors.New("le mot de passe fait moins de 8 caractères")
	ErrAccountAlreadyConfirmed = errors.New("ce compte à déja été confirmé")
	ErrInvalidCode             = errors.New("le code est invalide")
	ErrCodeExpired             = errors.New("le code a expiré")
	ErrInvalidCredentials      = errors.New("les identifiants sont mauvais")
	ErrAccountNotConfirmed     = errors.New("Le compte n'est pas comfirmé")
	ErrUnauthorized            = errors.New("session invalide ou expirée")
	ErrOrderNotPending         = errors.New("la commande n'est pas en attente de paiement")
	ErrOrderStatusChanged      = errors.New("le statut de la commande a changé entre-temps, réessayez")
	ErrInvalidCardNumber       = errors.New("le numéro de carte est invalide")
	ErrInvalidCardExpiry       = errors.New("la date d'expiration doit être au format MM/AA")
	ErrCardExpired             = errors.New("la carte a expiré")
	ErrInvalidCVC              = errors.New("le CVC doit contenir 3 chiffres")
	ErrOrderNotCancellable     = errors.New("la commande ne peut plus être annulée")
	ErrCancelReasonRequired    = errors.New("la raison de l'annulation est obligatoire")
	ErrInvalidPriceRange       = errors.New("le prix minimum doit être inférieur ou égal au prix maximum")
)
