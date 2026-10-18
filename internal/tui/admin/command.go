package admin

import (
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"

	"github.com/Eliasf1912/go-project/internal/apiclient"
	"github.com/Eliasf1912/go-project/internal/dto"
)

var errNotAdmin = errors.New("ce compte n'est pas administrateur")

type loginDoneMsg struct {
	api  *apiclient.Client
	user dto.MeResponse
	err  error
}

type logoutDoneMsg struct{ err error }

// loginCmd se connecte, puis refuse les comptes qui ne sont pas admin.
func loginCmd(api *apiclient.Client, email, password string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		token, err := api.Login(ctx, email, password)
		if err != nil {
			return loginDoneMsg{err: err}
		}

		authed := api.WithToken(token)
		user, err := authed.Me(ctx)
		if err != nil {
			return loginDoneMsg{err: err}
		}

		if user.Role != "admin" {
			// Le login a créé une session : on la ferme avant de refuser.
			_ = authed.Logout(ctx)
			return loginDoneMsg{err: errNotAdmin}
		}
		return loginDoneMsg{api: authed, user: user}
	}
}

func logoutCmd(api *apiclient.Client) tea.Cmd {
	return func() tea.Msg {
		err := api.Logout(context.Background())
		return logoutDoneMsg{err: err}
	}
}
