package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Eliasf1912/go-project/internal/domain"
	"github.com/Eliasf1912/go-project/internal/usecase"
)

var (
	_ usecase.UserRepository    = (*fakeUserRepo)(nil)
	_ usecase.SessionRepository = (*fakeSessionRepo)(nil)
	_ usecase.PasswordHasher    = fakeHasher{}
)

type fakeUserRepo struct {
	users  map[int]domain.User
	nextID int
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: map[int]domain.User{}, nextID: 1}
}

func (r *fakeUserRepo) FindByEmail(_ context.Context, email string) (domain.User, error) {
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}

func (r *fakeUserRepo) FindByID(_ context.Context, id int) (domain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

func (r *fakeUserRepo) Create(ctx context.Context, user domain.User) (int, error) {
	if _, err := r.FindByEmail(ctx, user.Email); err == nil {
		return 0, domain.ErrEmailTaken
	}
	user.ID = r.nextID
	r.nextID++
	r.users[user.ID] = user
	return user.ID, nil
}

func (r *fakeUserRepo) Update(_ context.Context, user domain.User) error {
	if _, ok := r.users[user.ID]; !ok {
		return domain.ErrNotFound
	}
	r.users[user.ID] = user
	return nil
}

type fakeSessionRepo struct {
	sessions map[string]domain.Session
}

func newFakeSessionRepo() *fakeSessionRepo {
	return &fakeSessionRepo{sessions: map[string]domain.Session{}}
}

func (r *fakeSessionRepo) Create(_ context.Context, s domain.Session) error {
	r.sessions[s.Token] = s
	return nil
}

func (r *fakeSessionRepo) DeleteByUser(_ context.Context, userID int) error {
	for token, s := range r.sessions {
		if s.UserID == userID {
			delete(r.sessions, token)
		}
	}
	return nil
}

func (r *fakeSessionRepo) Delete(_ context.Context, token string) error {
	delete(r.sessions, token)
	return nil
}

func (r *fakeSessionRepo) FindByToken(_ context.Context, token string) (domain.Session, error) {
	s, ok := r.sessions[token]
	if !ok {
		return domain.Session{}, domain.ErrNotFound
	}
	return s, nil
}

// fakeHasher remplace bcrypt : instantané et prévisible.
// On teste la logique du usecase ici, pas bcrypt.
type fakeHasher struct{}

func (fakeHasher) Hash(password string) (string, error)   { return "hash:" + password, nil }
func (fakeHasher) CompareHash(password, hash string) bool { return hash == "hash:"+password }

const (
	testEmail    = "elias@test.fr"
	testPassword = "motdepasse123"
)

type testEnv struct {
	auth     *usecase.AuthService
	users    *fakeUserRepo
	sessions *fakeSessionRepo
}

func newTestEnv() testEnv {
	users := newFakeUserRepo()
	sessions := newFakeSessionRepo()
	return testEnv{
		auth:     usecase.NewAuthService(users, fakeHasher{}, sessions),
		users:    users,
		sessions: sessions,
	}
}

func register(t *testing.T, env testEnv) domain.User {
	t.Helper()
	ctx := context.Background()
	if err := env.auth.Register(ctx, testEmail, testPassword); err != nil {
		t.Fatalf("Register : erreur inattendue : %v", err)
	}
	user, err := env.users.FindByEmail(ctx, testEmail)
	if err != nil {
		t.Fatalf("utilisateur absent après Register : %v", err)
	}
	return user
}

func registerAndConfirm(t *testing.T, env testEnv) {
	t.Helper()
	user := register(t, env)
	if err := env.auth.Confirm(context.Background(), testEmail, user.ConfirmationCode); err != nil {
		t.Fatalf("Confirm : erreur inattendue : %v", err)
	}
}

func TestRegister_CreeUnUtilisateurNonConfirme(t *testing.T) {
	env := newTestEnv()
	user := register(t, env)

	if user.Confirmed {
		t.Error("l'utilisateur ne devrait pas être confirmé juste après l'inscription")
	}
	if user.PasswordHash != "hash:"+testPassword {
		t.Errorf("mot de passe mal haché : obtenu %q", user.PasswordHash)
	}
	if len(user.ConfirmationCode) != 6 {
		t.Errorf("le code devrait faire 6 caractères, obtenu %q", user.ConfirmationCode)
	}
	if user.ConfirmationExpiresAt == nil {
		t.Error("le code de confirmation devrait avoir une date d'expiration")
	}
	if user.Role != domain.RoleClient {
		t.Errorf("rôle attendu %q, obtenu %q", domain.RoleClient, user.Role)
	}
}

func TestRegister_EntreesInvalides(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		wantErr  error
	}{
		{"email invalide", "pas-un-email", testPassword, domain.ErrInvalidEmail},
		{"mot de passe trop court", testEmail, "court", domain.ErrPasswordTooShort},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv()
			err := env.auth.Register(context.Background(), tt.email, tt.password)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("erreur attendue %v, obtenu %v", tt.wantErr, err)
			}
			if len(env.users.users) != 0 {
				t.Error("aucun utilisateur ne devrait être créé")
			}
		})
	}
}

func TestConfirm_BonCode(t *testing.T) {
	env := newTestEnv()
	registerAndConfirm(t, env)

	user, _ := env.users.FindByEmail(context.Background(), testEmail)
	if !user.Confirmed {
		t.Error("l'utilisateur devrait être confirmé")
	}
	if user.ConfirmationCode != "" || user.ConfirmationExpiresAt != nil {
		t.Error("le code et son expiration devraient être effacés après confirmation")
	}
}

func TestConfirm_Erreurs(t *testing.T) {
	ctx := context.Background()

	t.Run("mauvais code", func(t *testing.T) {
		env := newTestEnv()
		register(t, env)
		err := env.auth.Confirm(ctx, testEmail, "000000")
		if !errors.Is(err, domain.ErrInvalidCode) {
			t.Errorf("attendu ErrInvalidCode, obtenu %v", err)
		}
	})

	t.Run("email inconnu", func(t *testing.T) {
		env := newTestEnv()
		err := env.auth.Confirm(ctx, "inconnu@test.fr", "123456")
		if !errors.Is(err, domain.ErrInvalidCode) {
			t.Errorf("attendu ErrInvalidCode (ne pas révéler que l'email n'existe pas), obtenu %v", err)
		}
	})

	t.Run("code expiré", func(t *testing.T) {
		env := newTestEnv()
		user := register(t, env)
		past := time.Now().Add(-time.Minute)
		user.ConfirmationExpiresAt = &past
		env.users.users[user.ID] = user

		err := env.auth.Confirm(ctx, testEmail, user.ConfirmationCode)
		if !errors.Is(err, domain.ErrCodeExpired) {
			t.Errorf("attendu ErrCodeExpired, obtenu %v", err)
		}
	})

	t.Run("déjà confirmé", func(t *testing.T) {
		env := newTestEnv()
		registerAndConfirm(t, env)
		err := env.auth.Confirm(ctx, testEmail, "123456")
		if !errors.Is(err, domain.ErrAccountAlreadyConfirmed) {
			t.Errorf("attendu ErrAccountAlreadyConfirmed, obtenu %v", err)
		}
	})
}

func TestLogin_Succes(t *testing.T) {
	env := newTestEnv()
	registerAndConfirm(t, env)

	token, err := env.auth.Login(context.Background(), testEmail, testPassword)
	if err != nil {
		t.Fatalf("Login : erreur inattendue : %v", err)
	}
	if len(token) != 64 {
		t.Errorf("le jeton devrait faire 64 caractères hexadécimaux, obtenu %d", len(token))
	}

	session, ok := env.sessions.sessions[token]
	if !ok {
		t.Fatal("aucune session enregistrée pour ce jeton")
	}
	user, _ := env.users.FindByEmail(context.Background(), testEmail)
	if session.UserID != user.ID {
		t.Errorf("session liée à l'utilisateur %d, attendu %d", session.UserID, user.ID)
	}
	ecart := time.Until(session.ExpiresAt) - 24*time.Hour
	if ecart > time.Minute || ecart < -time.Minute {
		t.Errorf("la session devrait expirer dans 24 h, expire le %v", session.ExpiresAt)
	}
}

func TestLogin_Erreurs(t *testing.T) {
	tests := []struct {
		name     string
		confirm  bool
		email    string
		password string
		wantErr  error
	}{
		{"compte non confirmé", false, testEmail, testPassword, domain.ErrAccountNotConfirmed},
		{"mauvais mot de passe", true, testEmail, "mauvais-mdp", domain.ErrInvalidCredentials},
		{"email inconnu", true, "inconnu@test.fr", testPassword, domain.ErrInvalidCredentials},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv()
			if tt.confirm {
				registerAndConfirm(t, env)
			} else {
				register(t, env)
			}

			token, err := env.auth.Login(context.Background(), tt.email, tt.password)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("erreur attendue %v, obtenu %v", tt.wantErr, err)
			}
			if token != "" {
				t.Error("aucun jeton ne devrait être renvoyé en cas d'erreur")
			}
			if len(env.sessions.sessions) != 0 {
				t.Error("aucune session ne devrait être créée")
			}
		})
	}
}

func TestAuthenticate(t *testing.T) {
	ctx := context.Background()

	t.Run("jeton valide", func(t *testing.T) {
		env := newTestEnv()
		registerAndConfirm(t, env)
		token, err := env.auth.Login(ctx, testEmail, testPassword)
		if err != nil {
			t.Fatalf("Login : %v", err)
		}

		user, err := env.auth.Authenticate(ctx, token)
		if err != nil {
			t.Fatalf("Authenticate : erreur inattendue : %v", err)
		}
		if user.Email != testEmail {
			t.Errorf("utilisateur attendu %q, obtenu %q", testEmail, user.Email)
		}
	})

	t.Run("jeton inconnu", func(t *testing.T) {
		env := newTestEnv()
		_, err := env.auth.Authenticate(ctx, "jeton-qui-n-existe-pas")
		if !errors.Is(err, domain.ErrUnauthorized) {
			t.Errorf("attendu ErrUnauthorized, obtenu %v", err)
		}
	})

	t.Run("session expirée", func(t *testing.T) {
		env := newTestEnv()
		registerAndConfirm(t, env)
		token, err := env.auth.Login(ctx, testEmail, testPassword)
		if err != nil {
			t.Fatalf("Login : %v", err)
		}
		s := env.sessions.sessions[token]
		s.ExpiresAt = time.Now().Add(-time.Minute)
		env.sessions.sessions[token] = s

		_, err = env.auth.Authenticate(ctx, token)
		if !errors.Is(err, domain.ErrUnauthorized) {
			t.Errorf("attendu ErrUnauthorized, obtenu %v", err)
		}
	})
}

func TestLogout(t *testing.T) {
	ctx := context.Background()
	env := newTestEnv()
	registerAndConfirm(t, env)
	token, err := env.auth.Login(ctx, testEmail, testPassword)
	if err != nil {
		t.Fatalf("Login : %v", err)
	}

	if err := env.auth.Logout(ctx, token); err != nil {
		t.Fatalf("Logout : erreur inattendue : %v", err)
	}
	if _, ok := env.sessions.sessions[token]; ok {
		t.Error("la session devrait être supprimée")
	}

	_, err = env.auth.Authenticate(ctx, token)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("le jeton ne devrait plus être accepté, obtenu %v", err)
	}
}

func TestResetPassword(t *testing.T) {
	ctx := context.Background()
	env := newTestEnv()
	registerAndConfirm(t, env)
	oldToken, err := env.auth.Login(ctx, testEmail, testPassword)
	if err != nil {
		t.Fatalf("Login : %v", err)
	}

	if err := env.auth.ForgotPassword(ctx, "inconnu@test.fr"); err != nil {
		t.Errorf("un email inconnu ne doit pas renvoyer d'erreur, obtenu %v", err)
	}
	if err := env.auth.ForgotPassword(ctx, testEmail); err != nil {
		t.Fatalf("ForgotPassword : %v", err)
	}
	user, _ := env.users.FindByEmail(ctx, testEmail)

	err = env.auth.ResetPassword(ctx, testEmail, "000000", "nouveaumdp123")
	if !errors.Is(err, domain.ErrInvalidCode) {
		t.Errorf("mauvais code : attendu ErrInvalidCode, obtenu %v", err)
	}

	if err := env.auth.ResetPassword(ctx, testEmail, user.PasswordResetCode, "nouveaumdp123"); err != nil {
		t.Fatalf("ResetPassword : %v", err)
	}
	if _, err := env.auth.Authenticate(ctx, oldToken); !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("l'ancienne session devrait être supprimée, obtenu %v", err)
	}
	if _, err := env.auth.Login(ctx, testEmail, "nouveaumdp123"); err != nil {
		t.Errorf("connexion avec le nouveau mot de passe : %v", err)
	}
	if _, err := env.auth.Login(ctx, testEmail, testPassword); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("l'ancien mot de passe ne devrait plus marcher, obtenu %v", err)
	}
}
