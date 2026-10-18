package client

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/Eliasf1912/go-project/internal/apiclient"
	"github.com/Eliasf1912/go-project/internal/dto"
)

type registerDoneMsg struct{ err error }

type confirmDoneMsg struct{ err error }

type loginDoneMsg struct {
	api  *apiclient.Client
	user dto.MeResponse
	err  error
}

func registerCmd(api *apiclient.Client, email, password string) tea.Cmd {
	return func() tea.Msg {
		err := api.Register(context.Background(), email, password)
		return registerDoneMsg{err: err}
	}
}

func confirmCmd(api *apiclient.Client, email, code string) tea.Cmd {
	return func() tea.Msg {
		err := api.Confirm(context.Background(), email, code)
		return confirmDoneMsg{err: err}
	}
}

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
		return loginDoneMsg{api: authed, user: user}
	}
}
