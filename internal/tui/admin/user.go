package admin

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/Eliasf1912/go-project/internal/apiclient"
	"github.com/Eliasf1912/go-project/internal/dto"
	"github.com/Eliasf1912/go-project/internal/tui/components"
)

type usersLoadedMsg struct {
	users []dto.UserResponse
	err   error
}

type userDeletedMsg struct {
	anonymized bool
	err        error
}

func loadUsersCmd(api *apiclient.Client) tea.Cmd {
	return func() tea.Msg {
		users, err := api.ListUsers(context.Background())
		return usersLoadedMsg{users: users, err: err}
	}
}

func deleteUserCmd(api *apiclient.Client, id int) tea.Cmd {
	return func() tea.Msg {
		anonymized, err := api.DeleteUser(context.Background(), id)
		return userDeletedMsg{anonymized: anonymized, err: err}
	}
}

// userLabel affiche un utilisateur dans la liste : email, rôle et état.
func userLabel(u dto.UserResponse) string {
	label := u.Email + " (" + u.Role + ")"
	if u.Deleted {
		label += " [anonymisé]"
	} else if !u.Confirmed {
		label += " [non confirmé]"
	}
	return label
}

func (m *Model) goUsers() {
	items := make([]string, 0, len(m.users)+1)
	for _, u := range m.users {
		items = append(items, userLabel(u))
	}
	items = append(items, "← Retour")

	m.screen = screenUsers
	m.menu = components.NewMenu(items...)
}

// Même ordre que dans goUserActions.
const (
	userActionDelete = iota
	userActionBack
)

func (m *Model) goUserActions() {
	m.screen = screenUserActions
	m.menu = components.NewMenu("Supprimer", "← Retour")
	m.err, m.info = "", ""
}

func (m Model) updateUsers(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	choice, selected := m.menu.Update(key.String())
	if !selected {
		return m, nil
	}

	if choice == len(m.users) { // ← Retour (la dernière ligne)
		m.goMain()
		return m, nil
	}
	m.selectedUser = m.users[choice]
	m.goUserActions()
	return m, nil
}

func (m Model) updateUserActions(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	choice, selected := m.menu.Update(key.String())
	if !selected {
		return m, nil
	}

	switch choice {
	case userActionDelete:
		m.loading = true
		return m, deleteUserCmd(m.api, m.selectedUser.ID)
	case userActionBack:
		m.goUsers()
	}
	return m, nil
}
