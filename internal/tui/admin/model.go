package admin

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Eliasf1912/go-project/internal/apiclient"
	"github.com/Eliasf1912/go-project/internal/dto"
	"github.com/Eliasf1912/go-project/internal/tui/components"
	"github.com/Eliasf1912/go-project/internal/tui/styles"
)

const appName = "Charm Bracelet — Administration"

type screen int

const (
	screenLogin screen = iota
	screenMain
)

const (
	mainCategories = iota
	mainUsers
	mainProducts
	mainOrders
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
	m.goLogin("")
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) goLogin(info string) {
	m.screen = screenLogin
	m.form = components.NewForm(
		components.Field{Label: "Email", Placeholder: "nom@exemple.fr", Value: m.email},
		components.Field{Label: "Mot de passe", Secret: true},
	)
	m.err, m.info = "", info
}

func (m *Model) goMain() {
	m.screen = screenMain
	m.menu = components.NewMenu("Catégories", "Utilisateurs", "Produits", "Commandes", "Se déconnecter", "Quitter")
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

	case loginDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.api = msg.api
		m.user = msg.user
		m.goMain()
		return m, nil

	case logoutDoneMsg:
		m.loading = false
		if m.quitting {
			return m, tea.Quit
		}
		m.api = m.api.WithToken("")
		m.user = dto.MeResponse{}
		m.goLogin("Tu es déconnecté.")
		if msg.err != nil {
			m.err = "Le serveur n'a pas pu fermer la session : " + msg.err.Error()
		}
		return m, nil
	}

	switch m.screen {
	case screenLogin:
		return m.updateLogin(msg)
	case screenMain:
		return m.updateMain(msg)
	}
	return m, nil
}

func (m Model) updateLogin(msg tea.Msg) (tea.Model, tea.Cmd) {
	submitted, cmd := m.form.Update(msg)
	if !submitted {
		return m, cmd
	}

	m.email = m.form.Value(0)
	m.err, m.info = "", ""
	m.loading = true
	return m, loginCmd(m.api, m.email, m.form.Value(1))
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
	case mainCategories, mainUsers, mainProducts, mainOrders:
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

	b.WriteString(styles.Title.Render(appName))
	b.WriteString("\n")
	if m.screen == screenMain {
		b.WriteString(styles.Help.Render("Connecté : " + m.user.Email))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	switch m.screen {
	case screenLogin:
		b.WriteString(m.form.View())
	case screenMain:
		b.WriteString(m.menu.View())
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

func (m Model) help() string {
	if m.screen == screenLogin {
		return "Tab : champ suivant • Entrée : valider • Ctrl+C : quitter"
	}
	return "↑/↓ : choisir • Entrée : valider • Ctrl+C : quitter"
}
