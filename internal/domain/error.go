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
)
