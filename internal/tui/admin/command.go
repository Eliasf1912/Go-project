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

// ordersLoadedMsg porte, en plus de la liste, un message optionnel (ex:
// "Commande créée") à afficher une fois revenu sur l'écran de liste.
type ordersLoadedMsg struct {
	orders []dto.OrderResponse
	info   string
	err    error
}

type createOrderDoneMsg struct {
	order dto.OrderResponse
	err   error
}

type shipDoneMsg struct {
	order dto.OrderResponse
	err   error
}

type deliverDoneMsg struct {
	order dto.OrderResponse
	err   error
}

type cancelOrderDoneMsg struct {
	order dto.OrderResponse
	err   error
}

type deleteOrderDoneMsg struct {
	orderID int
	err     error
}

func loadOrdersCmd(api *apiclient.Client, info string) tea.Cmd {
	return func() tea.Msg {
		orders, err := api.AdminListOrders(context.Background())
		return ordersLoadedMsg{orders: orders, info: info, err: err}
	}
}

func createOrderCmd(api *apiclient.Client, userID int, lines []apiclient.AdminOrderLine) tea.Cmd {
	return func() tea.Msg {
		order, err := api.AdminCreateOrder(context.Background(), userID, lines)
		return createOrderDoneMsg{order: order, err: err}
	}
}

func shipCmd(api *apiclient.Client, orderID int) tea.Cmd {
	return func() tea.Msg {
		order, err := api.AdminShipOrder(context.Background(), orderID)
		return shipDoneMsg{order: order, err: err}
	}
}

func deliverCmd(api *apiclient.Client, orderID int) tea.Cmd {
	return func() tea.Msg {
		order, err := api.AdminDeliverOrder(context.Background(), orderID)
		return deliverDoneMsg{order: order, err: err}
	}
}

func cancelOrderCmd(api *apiclient.Client, orderID int, reason string) tea.Cmd {
	return func() tea.Msg {
		order, err := api.AdminCancelOrder(context.Background(), orderID, reason)
		return cancelOrderDoneMsg{order: order, err: err}
	}
}

func deleteOrderCmd(api *apiclient.Client, orderID int) tea.Cmd {
	return func() tea.Msg {
		err := api.AdminDeleteOrder(context.Background(), orderID)
		return deleteOrderDoneMsg{orderID: orderID, err: err}
	}
}
