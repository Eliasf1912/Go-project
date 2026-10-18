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

type logoutDoneMsg struct{ err error }

func logoutCmd(api *apiclient.Client) tea.Cmd {
	return func() tea.Msg {
		err := api.Logout(context.Background())
		return logoutDoneMsg{err: err}
	}
}

type forgotDoneMsg struct{ err error }

type resetDoneMsg struct{ err error }

func forgotCmd(api *apiclient.Client, email string) tea.Cmd {
	return func() tea.Msg {
		err := api.ForgotPassword(context.Background(), email)
		return forgotDoneMsg{err: err}
	}
}

func resetCmd(api *apiclient.Client, email, code, password string) tea.Cmd {
	return func() tea.Msg {
		err := api.ResetPassword(context.Background(), email, code, password)
		return resetDoneMsg{err: err}
	}
}
