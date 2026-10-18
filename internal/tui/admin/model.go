package admin

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Eliasf1912/go-project/internal/apiclient"
	"github.com/Eliasf1912/go-project/internal/dto"
	"github.com/Eliasf1912/go-project/internal/tui/components"
	"github.com/Eliasf1912/go-project/internal/tui/format"
	"github.com/Eliasf1912/go-project/internal/tui/styles"
)

const appName = "Charm Bracelet — Administration"

type screen int

const (
	screenLogin screen = iota
	screenMain
	screenCategories
	screenCategoryActions
	screenCategoryForm
	screenUsers
	screenUserActions
	screenOrders
	screenOrderActions
	screenCreateOrder
	screenCancelOrder
	screenDeleteConfirm
)

// Même ordre que dans goMain.
const (
	mainCategories = iota
	mainUsers
	mainProducts
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
	actionShip orderActionKind = iota
	actionDeliver
	actionCancel
	actionDelete
	actionBack
)

type orderAction struct {
	kind  orderActionKind
	label string
}

// Même ordre que dans goDeleteConfirm.
const (
	deleteConfirmYes = iota
	deleteConfirmNo
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

	categories []dto.CategoryResponse
	selected   dto.CategoryResponse // catégorie choisie (ID 0 : nouvelle)
	users        []dto.UserResponse
	selectedUser dto.UserResponse
	orders        []dto.OrderResponse
	selectedOrder dto.OrderResponse
	orderActions  []orderAction
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

// goOrders affiche la liste de toutes les commandes, tous clients
// confondus, les plus récentes en premier (déjà trié par l'API).
func (m *Model) goOrders(orders []dto.OrderResponse, info string) {
	m.screen = screenOrders
	m.orders = orders

	labels := make([]string, 0, len(orders)+2)
	for _, o := range orders {
		labels = append(labels, fmt.Sprintf("%s — client #%d — %s — %s", o.Reference, o.UserID, format.OrderStatusLabel(o.Status), format.Euros(o.TotalTTC)))
	}
	labels = append(labels, "Créer une commande pour un client", "Retour")
	m.menu = components.NewMenu(labels...)
	m.err, m.info = "", info
}

// goOrderActions propose les actions possibles sur une commande, selon son
// statut : expédier depuis "paid", livrer depuis "shipping", annuler
// depuis "pending" ou "paid" seulement (une commande en livraison, livrée
// ou déjà annulée ne peut plus l'être). La suppression, elle, reste
// toujours possible : c'est une action admin indépendante du cycle de vie.
func (m *Model) goOrderActions(order dto.OrderResponse) {
	m.screen = screenOrderActions
	m.selectedOrder = order

	m.orderActions = nil
	switch order.Status {
	case "paid":
		m.orderActions = append(m.orderActions, orderAction{actionShip, "Expédier"})
	case "shipping":
		m.orderActions = append(m.orderActions, orderAction{actionDeliver, "Livrer"})
	}
	if order.Status == "pending" || order.Status == "paid" {
		m.orderActions = append(m.orderActions, orderAction{actionCancel, "Annuler"})
	}
	m.orderActions = append(m.orderActions,
		orderAction{actionDelete, "Supprimer"},
		orderAction{actionBack, "Retour"},
	)

	labels := make([]string, len(m.orderActions))
	for i, a := range m.orderActions {
		labels[i] = a.label
	}
	m.menu = components.NewMenu(labels...)
	m.err, m.info = "", ""
}

// goCreateOrder ouvre le formulaire de création d'une commande pour un
// client. Une seule ligne produit à la fois : c'est suffisant pour les
// besoins du support client (ajouter une commande oubliée, dépanner un
// client par téléphone...), et ça évite un formulaire à nombre de champs
// variable.
func (m *Model) goCreateOrder() {
	m.screen = screenCreateOrder
	m.form = components.NewForm(
		components.Field{Label: "ID client", Placeholder: "1"},
		components.Field{Label: "ID produit", Placeholder: "1"},
		components.Field{Label: "Quantité", Placeholder: "1"},
	)
	m.err, m.info = "", ""
}

func (m *Model) goCancelOrder() {
	m.screen = screenCancelOrder
	m.form = components.NewForm(
		components.Field{Label: "Raison", Placeholder: "Rupture de stock"},
	)
	m.err, m.info = "", ""
}

func (m *Model) goDeleteConfirm() {
	m.screen = screenDeleteConfirm
	m.menu = components.NewMenu("Confirmer la suppression", "Annuler")
	m.err, m.info = "", ""
}

// goBack revient à l'écran précédent (touche Échap). Une fois connecté, on
// remonte d'un niveau sans jamais repasser par le login : ça ferait croire
// à une déconnexion alors que la session est toujours ouverte.
func (m *Model) goBack() {
	switch m.screen {
	case screenCreateOrder:
		m.goOrders(m.orders, "")
	case screenCancelOrder, screenDeleteConfirm:
		m.goOrderActions(m.selectedOrder)
	case screenOrderActions:
		m.goOrders(m.orders, "")
	case screenOrders:
		m.goMain()
	}
}

// replaceOrder met à jour une commande dans la liste déjà chargée (après
// une expédition, livraison ou annulation), pour éviter un aller-retour
// réseau en plus juste pour rafraîchir l'écran.
func (m *Model) replaceOrder(updated dto.OrderResponse) {
	for i, o := range m.orders {
		if o.ID == updated.ID {
			m.orders[i] = updated
			return
		}
	}
}

// removeOrder retire une commande supprimée de la liste déjà chargée.
func (m *Model) removeOrder(id int) {
	kept := m.orders[:0]
	for _, o := range m.orders {
		if o.ID != id {
			kept = append(kept, o)
		}
	}
	m.orders = kept
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
		if msg.String() == "esc" {
			return m.back()
		if msg.String() == "esc" && m.screen != screenLogin && m.screen != screenMain {
			m.goBack()
			return m, nil
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

	case categoriesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.categories = msg.categories
		m.goCategories()
		return m, nil

	case categoryChangedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		// On recharge la liste depuis le serveur pour afficher l'état réel.
		m.info = msg.info
		m.loading = true
		return m, loadCategoriesCmd(m.api)

	case usersLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.users = msg.users
		m.goUsers()
		return m, nil

	case userDeletedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		if msg.anonymized {
			m.info = "Compte anonymisé (il avait des commandes)."
		} else {
			m.info = "Compte supprimé."
		}
		m.loading = true
		return m, loadUsersCmd(m.api)

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

	case ordersLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.goMain() // avant m.err : goMain efface l'erreur
			m.err = msg.err.Error()
			return m, nil
		}
		m.goOrders(msg.orders, msg.info)
		return m, nil

	case createOrderDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		// On recharge la liste depuis le serveur : la nouvelle commande n'y
		// est pas encore si "Commandes" n'a pas été ouvert depuis.
		m.loading = true
		info := fmt.Sprintf("Commande %s créée pour le client #%d.", msg.order.Reference, msg.order.UserID)
		return m, loadOrdersCmd(m.api, info)

	case shipDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.replaceOrder(msg.order)
		m.goOrders(m.orders, "Commande expédiée.")
		return m, nil

	case deliverDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.replaceOrder(msg.order)
		m.goOrders(m.orders, "Commande livrée.")
		return m, nil

	case cancelOrderDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.replaceOrder(msg.order)
		m.goOrders(m.orders, "Commande annulée.")
		return m, nil

	case deleteOrderDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.removeOrder(msg.orderID)
		m.goOrders(m.orders, "Commande supprimée.")
		return m, nil
	}

	switch m.screen {
	case screenLogin:
		return m.updateLogin(msg)
	case screenMain:
		return m.updateMain(msg)
	case screenCategories:
		return m.updateCategories(msg)
	case screenCategoryActions:
		return m.updateCategoryActions(msg)
	case screenCategoryForm:
		return m.updateCategoryForm(msg)
	case screenUsers:
		return m.updateUsers(msg)
	case screenUserActions:
		return m.updateUserActions(msg)
	case screenOrders:
		return m.updateOrders(msg)
	case screenOrderActions:
		return m.updateOrderActions(msg)
	case screenCreateOrder:
		return m.updateCreateOrder(msg)
	case screenCancelOrder:
		return m.updateCancelOrder(msg)
	case screenDeleteConfirm:
		return m.updateDeleteConfirm(msg)
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
	case mainCategories:
		m.err, m.info = "", ""
		m.loading = true
		return m, loadCategoriesCmd(m.api)
	case mainUsers:
		m.err, m.info = "", ""
		m.loading = true
		return m, loadUsersCmd(m.api)
	case mainProducts, mainOrders:
	case mainCategories, mainUsers, mainProducts:
		m.info = "Bientôt disponible."
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

func (m Model) updateOrders(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	choice, selected := m.menu.Update(key.String())
	if !selected {
		return m, nil
	}
	switch choice {
	case len(m.orders): // avant-dernier élément du menu = "Créer une commande..."
		m.goCreateOrder()
	case len(m.orders) + 1: // dernier élément du menu = "Retour"
		m.goMain()
	default:
		m.goOrderActions(m.orders[choice])
	}
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
	case actionShip:
		m.loading = true
		return m, shipCmd(m.api, m.selectedOrder.ID)
	case actionDeliver:
		m.loading = true
		return m, deliverCmd(m.api, m.selectedOrder.ID)
	case actionCancel:
		m.goCancelOrder()
	case actionDelete:
		m.goDeleteConfirm()
	case actionBack:
		m.goOrders(m.orders, "")
	}
	return m, nil
}

// updateCreateOrder valide les trois champs (tous numériques) avant
// d'appeler l'API : un ID ou une quantité mal formés restent une erreur
// de saisie locale, inutile de faire un aller-retour réseau pour ça.
func (m Model) updateCreateOrder(msg tea.Msg) (tea.Model, tea.Cmd) {
	submitted, cmd := m.form.Update(msg)
	if !submitted {
		return m, cmd
	}

	userID, errUser := strconv.Atoi(strings.TrimSpace(m.form.Value(0)))
	productID, errProduct := strconv.Atoi(strings.TrimSpace(m.form.Value(1)))
	quantity, errQuantity := strconv.Atoi(strings.TrimSpace(m.form.Value(2)))
	if errUser != nil || errProduct != nil || errQuantity != nil {
		m.err = "ID client, ID produit et quantité doivent être des nombres entiers."
		return m, nil
	}

	m.err, m.info = "", ""
	m.loading = true
	lines := []apiclient.AdminOrderLine{{ProductID: productID, Quantity: quantity}}
	return m, createOrderCmd(m.api, userID, lines)
}

func (m Model) updateCancelOrder(msg tea.Msg) (tea.Model, tea.Cmd) {
	submitted, cmd := m.form.Update(msg)
	if !submitted {
		return m, cmd
	}

	m.err, m.info = "", ""
	m.loading = true
	return m, cancelOrderCmd(m.api, m.selectedOrder.ID, m.form.Value(0))
}

func (m Model) updateDeleteConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	choice, selected := m.menu.Update(key.String())
	if !selected {
		return m, nil
	}
	switch choice {
	case deleteConfirmYes:
		m.loading = true
		return m, deleteOrderCmd(m.api, m.selectedOrder.ID)
	case deleteConfirmNo:
		m.goOrderActions(m.selectedOrder)
	}
	return m, nil
}

func (m Model) View() tea.View {
	var b strings.Builder

	b.WriteString(styles.Title.Render(m.title()))
	b.WriteString("\n")
	if m.screen != screenLogin {
		b.WriteString(styles.Help.Render("Connecté : " + m.user.Email))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	switch m.screen {
	case screenLogin, screenCategoryForm:
		b.WriteString(m.form.View())
	case screenMain, screenOrders, screenDeleteConfirm:
		b.WriteString(m.menu.View())
	case screenOrderActions:
		b.WriteString(m.viewOrderActions())
	default: // screenCreateOrder, screenCancelOrder
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
	case screenCategories:
		return "Catégories"
	case screenUsers:
		return "Utilisateurs"
	case screenUserActions:
		return "Utilisateur : " + m.selectedUser.Email
	case screenCategoryActions:
		return "Catégorie : " + m.selected.Name
	case screenCategoryForm:
		if m.selected.ID == 0 {
			return "Nouvelle catégorie"
		}
		return "Renommer : " + m.selected.Name
// viewOrderActions affiche le détail d'une commande (lignes, raison
// d'annulation éventuelle) avant le menu de ses actions possibles.
func (m Model) viewOrderActions() string {
	var b strings.Builder
	o := m.selectedOrder

	b.WriteString(fmt.Sprintf("Client #%d — %s — %s\n\n", o.UserID, format.OrderStatusLabel(o.Status), format.Euros(o.TotalTTC)))
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
	case screenOrders:
		return "Commandes"
	case screenOrderActions:
		return m.selectedOrder.Reference
	case screenCreateOrder:
		return "Créer une commande"
	case screenCancelOrder:
		return "Annuler la commande"
	case screenDeleteConfirm:
		return "Supprimer " + m.selectedOrder.Reference + " ?"
	default:
		return appName
	}
}

func (m Model) help() string {
	switch m.screen {
	case screenLogin:
		return "Tab : champ suivant • Entrée : valider • Ctrl+C : quitter"
	case screenCategoryForm:
		return "Entrée : valider • Échap : retour • Ctrl+C : quitter"
	case screenMain:
		return "↑/↓ : choisir • Entrée : valider • Ctrl+C : quitter"
	default:
		return "↑/↓ : choisir • Entrée : valider • Échap : retour • Ctrl+C : quitter"
	}
}

// back gère la touche Échap : on remonte d'un écran.
func (m Model) back() (tea.Model, tea.Cmd) {
	m.err, m.info = "", ""
	switch m.screen {
	case screenCategories, screenUsers:
		m.goMain()
	case screenUserActions:
		m.goUsers()
	case screenCategoryActions, screenCategoryForm:
		m.goCategories()
	}
	return m, nil
	case screenMain:
		return "↑/↓ : choisir • Entrée : valider • Ctrl+C : quitter"
	case screenOrders, screenOrderActions, screenDeleteConfirm:
		return "↑/↓ : choisir • Entrée : valider • Échap : retour • Ctrl+C : quitter"
	default:
		return "Tab : champ suivant • Entrée : valider • Échap : retour • Ctrl+C : quitter"
	}
}
