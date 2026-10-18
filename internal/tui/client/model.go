package client

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Eliasf1912/go-project/internal/apiclient"
	"github.com/Eliasf1912/go-project/internal/dto"
	"github.com/Eliasf1912/go-project/internal/tui/components"
	"github.com/Eliasf1912/go-project/internal/tui/format"
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
	screenCart
	screenOrders
	screenOrderActions
	screenPay
	screenCancel
)

// Même ordre que dans goHome.
const (
	homeLogin = iota
	homeRegister
	homeQuit
)

// Même ordre que dans goMain.
const (
	mainProducts = iota
	mainCart
	mainOrders
	mainLogout
	mainQuit
)

// orderActionKind identifie l'action choisie sur l'écran d'une commande
// (screenOrderActions) : la liste proposée dépend du statut de la
// commande, donc on la stocke avec son libellé plutôt que de se fier à
// l'index du menu.
type orderActionKind int

const (
	actionPay orderActionKind = iota
	actionCancel
	actionBack
)

type orderAction struct {
	kind  orderActionKind
	label string
}

type Model struct {
	api      *apiclient.Client
	screen   screen
	menu     components.Menu
	form     components.Form
	quitting bool
	email    string
	user     dto.MeResponse
	loading  bool
	err      string
	info     string

	cart          dto.CartResponse
	orders        []dto.OrderResponse
	selectedOrder dto.OrderResponse
	orderActions  []orderAction
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
	m.menu = components.NewMenu("Se connecter", "Créer un compte", "Quitter")
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

func (m *Model) goMain() {
	m.screen = screenMain
	m.menu = components.NewMenu("Produits", "Mon panier", "Mes commandes", "Se déconnecter", "Quitter")
	m.err, m.info = "", ""
}

// goCart affiche le panier reçu de l'API. "Valider le panier" n'est
// proposé que s'il contient au moins une ligne (un panier vide ne peut
// pas être validé, voir ErrCartEmpty côté serveur).
func (m *Model) goCart(cart dto.CartResponse) {
	m.screen = screenCart
	m.cart = cart

	items := []string{}
	if len(cart.Items) > 0 {
		items = append(items, "Valider le panier")
	}
	items = append(items, "Retour")
	m.menu = components.NewMenu(items...)
	m.err, m.info = "", ""
}

// goOrders affiche la liste des commandes de l'utilisateur, les plus
// récentes en premier (déjà trié par l'API).
func (m *Model) goOrders(orders []dto.OrderResponse, info string) {
	m.screen = screenOrders
	m.orders = orders

	labels := make([]string, 0, len(orders)+1)
	for _, o := range orders {
		labels = append(labels, fmt.Sprintf("%s — %s — %s", o.Reference, format.OrderStatusLabel(o.Status), format.Euros(o.TotalTTC)))
	}
	labels = append(labels, "Retour")
	m.menu = components.NewMenu(labels...)
	m.err, m.info = "", info
}

// goOrderActions propose les actions possibles sur une commande, selon
// son statut : payer et annuler depuis "pending", annuler seulement
// depuis "paid", rien depuis les autres statuts (juste "Retour").
func (m *Model) goOrderActions(order dto.OrderResponse) {
	m.screen = screenOrderActions
	m.selectedOrder = order

	m.orderActions = nil
	switch order.Status {
	case "pending":
		m.orderActions = []orderAction{{actionPay, "Payer"}, {actionCancel, "Annuler"}}
	case "paid":
		m.orderActions = []orderAction{{actionCancel, "Annuler"}}
	}
	m.orderActions = append(m.orderActions, orderAction{actionBack, "Retour"})

	labels := make([]string, len(m.orderActions))
	for i, a := range m.orderActions {
		labels[i] = a.label
	}
	m.menu = components.NewMenu(labels...)
	m.err, m.info = "", ""
}

func (m *Model) goPay() {
	m.screen = screenPay
	m.form = components.NewForm(
		components.Field{Label: "Numéro de carte", Placeholder: "4242424242424242"},
		components.Field{Label: "Expiration", Placeholder: "MM/AA"},
		components.Field{Label: "CVC", Placeholder: "123", Secret: true},
	)
	m.err, m.info = "", ""
}

func (m *Model) goCancel() {
	m.screen = screenCancel
	m.form = components.NewForm(
		components.Field{Label: "Raison", Placeholder: "Je me suis trompé"},
	)
	m.err, m.info = "", ""
}

// goBack revient à l'écran précédent (touche Échap). Une fois connecté,
// on remonte d'un niveau sans jamais repasser par l'accueil : l'accueil
// ferait croire à une déconnexion alors que la session est toujours
// ouverte.
func (m *Model) goBack() {
	switch m.screen {
	case screenPay, screenCancel:
		m.goOrderActions(m.selectedOrder)
	case screenOrderActions:
		m.goOrders(m.orders, "")
	case screenCart, screenOrders:
		m.goMain()
	default: // inscription, confirmation, connexion
		m.goHome("")
	}
}

// replaceOrder met à jour une commande dans la liste déjà chargée (après
// un paiement ou une annulation), pour éviter un aller-retour réseau en
// plus juste pour rafraîchir l'écran.
func (m *Model) replaceOrder(updated dto.OrderResponse) {
	for i, o := range m.orders {
		if o.ID == updated.ID {
			m.orders[i] = updated
			return
		}
	}
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
			m.goBack()
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

	case cartLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.goMain() // avant m.err : goMain efface l'erreur
			m.err = msg.err.Error()
			return m, nil
		}
		m.goCart(msg.cart)
		return m, nil

	case checkoutDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		// On recharge la liste depuis le serveur : celle en mémoire peut
		// être incomplète si "Mes commandes" n'a pas encore été ouvert.
		m.loading = true
		info := fmt.Sprintf("Commande %s créée (%s).", msg.order.Reference, format.Euros(msg.order.TotalTTC))
		return m, loadOrdersCmd(m.api, info)

	case ordersLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.goMain() // avant m.err : goMain efface l'erreur
			m.err = msg.err.Error()
			return m, nil
		}
		m.goOrders(msg.orders, msg.info)
		return m, nil

	case payDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.replaceOrder(msg.order)
		m.goOrders(m.orders, "Commande payée.")
		return m, nil

	case cancelDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.replaceOrder(msg.order)
		m.goOrders(m.orders, "Commande annulée.")
		return m, nil
	}

	switch m.screen {
	case screenHome:
		return m.updateHome(msg)
	case screenRegister, screenConfirm, screenLogin, screenPay, screenCancel:
		return m.updateForm(msg)
	case screenMain:
		return m.updateMain(msg)
	case screenCart:
		return m.updateCart(msg)
	case screenOrders:
		return m.updateOrders(msg)
	case screenOrderActions:
		return m.updateOrderActions(msg)
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

	m.err, m.info = "", ""
	m.loading = true

	switch m.screen {
	case screenRegister:
		m.email = m.form.Value(0)
		return m, registerCmd(m.api, m.email, m.form.Value(1))
	case screenConfirm:
		m.email = m.form.Value(0)
		return m, confirmCmd(m.api, m.email, m.form.Value(1))
	case screenLogin:
		m.email = m.form.Value(0)
		return m, loginCmd(m.api, m.email, m.form.Value(1))
	case screenPay:
		return m, payCmd(m.api, m.selectedOrder.ID, m.form.Value(0), m.form.Value(1), m.form.Value(2))
	case screenCancel:
		return m, cancelOrderCmd(m.api, m.selectedOrder.ID, m.form.Value(0))
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
	case mainCart:
		m.loading = true
		return m, loadCartCmd(m.api)
	case mainOrders:
		m.loading = true
		return m, loadOrdersCmd(m.api, "")
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

func (m Model) updateCart(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	choice, selected := m.menu.Update(key.String())
	if !selected {
		return m, nil
	}
	// "Valider le panier" n'existe (en position 0) que si le panier a des
	// lignes ; sinon le seul choix est "Retour", géré par défaut ci-dessous.
	if len(m.cart.Items) > 0 && choice == 0 {
		m.loading = true
		return m, checkoutCmd(m.api)
	}
	m.goMain()
	return m, nil
}

func (m Model) updateOrders(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	choice, selected := m.menu.Update(key.String())
	if !selected {
		return m, nil
	}
	if choice == len(m.orders) { // dernier élément du menu = "Retour"
		m.goMain()
		return m, nil
	}
	m.goOrderActions(m.orders[choice])
	return m, nil
}

func (m Model) updateOrderActions(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	choice, selected := m.menu.Update(key.String())
	if !selected {
		return m, nil
	}
	switch m.orderActions[choice].kind {
	case actionPay:
		m.goPay()
	case actionCancel:
		m.goCancel()
	case actionBack:
		m.goOrders(m.orders, "")
	}
	return m, nil
}

func (m Model) View() tea.View {
	var b strings.Builder

	b.WriteString(styles.Title.Render(m.title()))
	b.WriteString("\n")
	if m.screen != screenHome && m.screen != screenRegister && m.screen != screenConfirm && m.screen != screenLogin {
		b.WriteString(styles.Help.Render("Connecté : " + m.user.Email))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	switch m.screen {
	case screenHome, screenMain, screenOrders:
		b.WriteString(m.menu.View())
	case screenCart:
		b.WriteString(m.viewCart())
	case screenOrderActions:
		b.WriteString(m.viewOrderActions())
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

// viewCart affiche les lignes du panier (nom, quantité, total de la
// ligne) et son total TTC, avant le menu.
func (m Model) viewCart() string {
	var b strings.Builder
	if len(m.cart.Items) == 0 {
		b.WriteString("Ton panier est vide.\n\n")
	} else {
		for _, item := range m.cart.Items {
			lineTotal := item.PriceTTC * item.Quantity
			b.WriteString(fmt.Sprintf("  %-30s x%-3d %s\n", item.ProductName, item.Quantity, format.Euros(lineTotal)))
		}
		b.WriteString("\n" + styles.Title.Render("Total : "+format.Euros(m.cart.TotalTTC)) + "\n\n")
	}
	b.WriteString(m.menu.View())
	return b.String()
}

// viewOrderActions affiche le détail d'une commande (lignes, raison
// d'annulation éventuelle) avant le menu de ses actions possibles.
func (m Model) viewOrderActions() string {
	var b strings.Builder
	o := m.selectedOrder

	b.WriteString(fmt.Sprintf("%s — %s\n\n", format.OrderStatusLabel(o.Status), format.Euros(o.TotalTTC)))
	for _, item := range o.Items {
		lineTotal := item.PriceTTC * item.Quantity
		b.WriteString(fmt.Sprintf("  %-30s x%-3d %s\n", item.ProductName, item.Quantity, format.Euros(lineTotal)))
	}
	if o.CancelReason != nil {
		b.WriteString("\nRaison d'annulation : " + *o.CancelReason + "\n")
	}
	b.WriteString("\n")
	b.WriteString(m.menu.View())
	return b.String()
}

func (m Model) title() string {
	switch m.screen {
	case screenRegister:
		return "Créer un compte"
	case screenConfirm:
		return "Confirmer le compte"
	case screenLogin:
		return "Se connecter"
	case screenCart:
		return "Mon panier"
	case screenOrders:
		return "Mes commandes"
	case screenOrderActions:
		return m.selectedOrder.Reference
	case screenPay:
		return "Payer la commande"
	case screenCancel:
		return "Annuler la commande"
	default:
		return appName
	}
}

func (m Model) help() string {
	switch m.screen {
	case screenHome, screenMain:
		return "↑/↓ : choisir • Entrée : valider • Ctrl+C : quitter"
	case screenCart, screenOrders, screenOrderActions:
		return "↑/↓ : choisir • Entrée : valider • Échap : retour • Ctrl+C : quitter"
	default:
		return "Tab : champ suivant • Entrée : valider • Échap : retour • Ctrl+C : quitter"
	}
}
