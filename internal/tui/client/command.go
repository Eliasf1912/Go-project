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
type cartLoadedMsg struct {
	cart dto.CartResponse
	err  error
}

type checkoutDoneMsg struct {
	order dto.OrderResponse
	err   error
}

type ordersLoadedMsg struct {
	orders []dto.OrderResponse
	info   string // message à afficher avec la liste (ex : commande créée)
	err    error
}

type payDoneMsg struct {
	order dto.OrderResponse
	err   error
}

type cancelDoneMsg struct {
	order dto.OrderResponse
	err   error
}

func loadCartCmd(api *apiclient.Client) tea.Cmd {
	return func() tea.Msg {
		cart, err := api.GetCart(context.Background())
		return cartLoadedMsg{cart: cart, err: err}
	}
}

func checkoutCmd(api *apiclient.Client) tea.Cmd {
	return func() tea.Msg {
		order, err := api.Checkout(context.Background())
		return checkoutDoneMsg{order: order, err: err}
	}
}

func loadOrdersCmd(api *apiclient.Client, info string) tea.Cmd {
	return func() tea.Msg {
		orders, err := api.ListMyOrders(context.Background())
		return ordersLoadedMsg{orders: orders, info: info, err: err}
	}
}

func payCmd(api *apiclient.Client, orderID int, cardNumber, expiry, cvc string) tea.Cmd {
	return func() tea.Msg {
		order, err := api.Pay(context.Background(), orderID, cardNumber, expiry, cvc)
		return payDoneMsg{order: order, err: err}
	}
}

func cancelOrderCmd(api *apiclient.Client, orderID int, reason string) tea.Cmd {
	return func() tea.Msg {
		order, err := api.CancelOrder(context.Background(), orderID, reason)
		return cancelDoneMsg{order: order, err: err}
	}
}
