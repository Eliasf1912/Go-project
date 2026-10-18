package client

import (
	"errors"
	"net/http"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Eliasf1912/go-project/internal/apiclient"
	"github.com/Eliasf1912/go-project/internal/dto"
	"github.com/Eliasf1912/go-project/internal/tui/components"
	"github.com/Eliasf1912/go-project/internal/tui/styles"
)

const appName = "Charm Bracelet"

type screen int

const (
	screenHome screen = iota
	screenRegister
	screenConfirm
	screenLogin
	screenMain
	screenForgot
	screenReset
)

// Même ordre que dans goHome.
const (
	homeLogin = iota
	homeRegister
	homeForgot
	homeQuit
)

// Même ordre que dans goMain.
const (
	mainProducts = iota
	mainLogout
	mainQuit
)

type Model struct {
	api    *apiclient.Client
	screen screen
	menu   components.Menu
	form   components.Form

	email    string
	user     dto.MeResponse
	loading  bool
	quitting bool
	err      string
	info     string
}

func New(api *apiclient.Client) Model {
	m := Model{api: api}
	m.goHome("")
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) goHome(info string) {
	m.screen = screenHome
	m.menu = components.NewMenu("Se connecter", "Créer un compte", "Mot de passe oublié", "Quitter")
	m.err, m.info = "", info
}

func (m *Model) goRegister() {
	m.screen = screenRegister
	m.form = components.NewForm(
		components.Field{Label: "Email", Placeholder: "nom@exemple.fr", Value: m.email},
		components.Field{Label: "Mot de passe", Secret: true},
	)
	m.err, m.info = "", ""
}

func (m *Model) goConfirm(info string) {
	m.screen = screenConfirm
	m.form = components.NewForm(
		components.Field{Label: "Email", Placeholder: "nom@exemple.fr", Value: m.email},
		components.Field{Label: "Code de confirmation", Placeholder: "6 chiffres"},
	)
	m.err, m.info = "", info
}

func (m *Model) goLogin(info string) {
	m.screen = screenLogin
	m.form = components.NewForm(
		components.Field{Label: "Email", Placeholder: "nom@exemple.fr", Value: m.email},
		components.Field{Label: "Mot de passe", Secret: true},
	)
	m.err, m.info = "", info
}

func (m *Model) goForgot() {
	m.screen = screenForgot
	m.form = components.NewForm(
		components.Field{Label: "Email", Placeholder: "nom@exemple.fr", Value: m.email},
	)
	m.err, m.info = "", ""
}

func (m *Model) goReset(info string) {
	m.screen = screenReset
	m.form = components.NewForm(
		components.Field{Label: "Email", Placeholder: "nom@exemple.fr", Value: m.email},
		components.Field{Label: "Code reçu", Placeholder: "6 chiffres"},
		components.Field{Label: "Nouveau mot de passe", Secret: true},
	)
	m.err, m.info = "", info
}

func (m *Model) goMain() {
	m.screen = screenMain
	m.menu = components.NewMenu("Produits", "Se déconnecter", "Quitter")
	m.err, m.info = "", ""
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.loading {
			return m, nil // on ignore les touches pendant une requête
		}
		if msg.String() == "esc" && m.screen != screenHome && m.screen != screenMain {
			m.goHome("")
			return m, nil
		}

	case registerDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.goConfirm("Compte créé ! Entre le code de confirmation reçu.")
		return m, nil

	case confirmDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.goLogin("Compte confirmé, tu peux te connecter.")
		return m, nil

	case loginDoneMsg:
		m.loading = false
		var apiErr *apiclient.APIError
		if errors.As(msg.err, &apiErr) && apiErr.Status == http.StatusForbidden {
			m.goConfirm("Ton compte n'est pas encore confirmé : entre le code reçu.")
			return m, nil
		}
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.api = msg.api
		m.user = msg.user
		m.goMain()
		return m, nil

	case forgotDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.goReset("Si ce compte existe, un code a été envoyé.")
		return m, nil

	case resetDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.goLogin("Mot de passe modifié, tu peux te connecter.")
		return m, nil

	case logoutDoneMsg:
		m.loading = false
		if m.quitting {
			return m, tea.Quit
		}
		// On oublie le jeton même si le serveur n'a pas répondu :
		// la session expirera d'elle-même.
		m.api = m.api.WithToken("")
		m.user = dto.MeResponse{}
		m.goHome("Tu es déconnecté.")
		if msg.err != nil {
			m.err = "Le serveur n'a pas pu fermer la session : " + msg.err.Error()
		}
		return m, nil
	}

	switch m.screen {
	case screenHome:
		return m.updateHome(msg)
	case screenRegister, screenConfirm, screenLogin, screenForgot, screenReset:
		return m.updateForm(msg)
	case screenMain:
		return m.updateMain(msg)
	}
	return m, nil
}

func (m Model) updateHome(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	choice, selected := m.menu.Update(key.String())
	if !selected {
		return m, nil
	}
	switch choice {
	case homeLogin:
		m.goLogin("")
	case homeRegister:
		m.goRegister()
	case homeForgot:
		m.goForgot()
	case homeQuit:
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	submitted, cmd := m.form.Update(msg)
	if !submitted {
		return m, cmd
	}

	m.email = m.form.Value(0)
	m.err, m.info = "", ""
	m.loading = true

	switch m.screen {
	case screenRegister:
		return m, registerCmd(m.api, m.email, m.form.Value(1))
	case screenConfirm:
		return m, confirmCmd(m.api, m.email, m.form.Value(1))
	case screenLogin:
		return m, loginCmd(m.api, m.email, m.form.Value(1))
	case screenForgot:
		return m, forgotCmd(m.api, m.email)
	case screenReset:
		return m, resetCmd(m.api, m.email, m.form.Value(1), m.form.Value(2))
	}
	return m, nil
}

func (m Model) updateMain(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	choice, selected := m.menu.Update(key.String())
	if !selected {
		return m, nil
	}
	switch choice {
	case mainProducts:
		m.info = "Bientôt disponible."
	case mainLogout:
		m.loading = true
		return m, logoutCmd(m.api)
	case mainQuit:
		m.loading = true
		m.quitting = true
		return m, logoutCmd(m.api)
	}
	return m, nil
}

func (m Model) View() tea.View {
	var b strings.Builder

	b.WriteString(styles.Title.Render(m.title()))
	b.WriteString("\n")
	if m.screen == screenMain {
		b.WriteString(styles.Help.Render("Connecté : " + m.user.Email))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	switch m.screen {
	case screenHome, screenMain:
		b.WriteString(m.menu.View())
	default:
		b.WriteString(m.form.View())
	}

	b.WriteString("\n")
	switch {
	case m.loading:
		b.WriteString("Chargement…\n")
	case m.err != "":
		b.WriteString(styles.Error.Render(m.err))
		b.WriteString("\n")
	case m.info != "":
		b.WriteString(styles.Success.Render(m.info))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.Help.Render(m.help()))

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

func (m Model) title() string {
	switch m.screen {
	case screenRegister:
		return "Créer un compte"
	case screenConfirm:
		return "Confirmer le compte"
	case screenLogin:
		return "Se connecter"
	case screenForgot:
		return "Mot de passe oublié"
	case screenReset:
		return "Nouveau mot de passe"
	default:
		return appName
	}
}

func (m Model) help() string {
	switch m.screen {
	case screenHome, screenMain:
		return "↑/↓ : choisir • Entrée : valider • Ctrl+C : quitter"
	default:
		return "Tab : champ suivant • Entrée : valider • Échap : retour • Ctrl+C : quitter"
	}
}
