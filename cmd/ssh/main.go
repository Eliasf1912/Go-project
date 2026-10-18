package main

import (
	"flag"
	"log"
	"net"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/ssh"
	"charm.land/wish/v2"
	"charm.land/wish/v2/activeterm"
	"charm.land/wish/v2/bubbletea"
	"charm.land/wish/v2/logging"

	"github.com/Eliasf1912/go-project/internal/apiclient"
	"github.com/Eliasf1912/go-project/internal/tui/admin"
	"github.com/Eliasf1912/go-project/internal/tui/client"
)

func main() {
	defaultURL := os.Getenv("API_URL")
	if defaultURL == "" {
		defaultURL = "http://localhost:8080"
	}
	apiURL := flag.String("api-url", defaultURL, "adresse du serveur HTTP")
	clientPort := flag.String("client-port", "23234", "port SSH du CLI client")
	adminPort := flag.String("admin-port", "23235", "port SSH du CLI admin")
	hostKey := flag.String("host-key", "ssh_host_key", "fichier de la clé d'hôte (créé s'il n'existe pas)")
	flag.Parse()

	// Chaque connexion SSH reçoit son propre Model et son propre client HTTP :
	// le jeton d'un utilisateur n'est jamais partagé avec un autre.
	clientHandler := func(ssh.Session) (tea.Model, []tea.ProgramOption) {
		return client.New(apiclient.New(*apiURL)), nil
	}
	adminHandler := func(ssh.Session) (tea.Model, []tea.ProgramOption) {
		return admin.New(apiclient.New(*apiURL)), nil
	}

	clientServer, err := newServer(*clientPort, *hostKey, clientHandler)
	if err != nil {
		log.Fatalf("serveur SSH client : %v", err)
	}
	adminServer, err := newServer(*adminPort, *hostKey, adminHandler)
	if err != nil {
		log.Fatalf("serveur SSH admin : %v", err)
	}

	// Les deux serveurs tournent en parallèle ; le premier qui s'arrête arrête le programme.
	errs := make(chan error, 2)
	go func() { errs <- clientServer.ListenAndServe() }()
	go func() { errs <- adminServer.ListenAndServe() }()

	log.Printf("SSH client sur le port %s, SSH admin sur le port %s (API : %s)", *clientPort, *adminPort, *apiURL)
	log.Fatal(<-errs)
}

func newServer(port, hostKey string, handler bubbletea.Handler) (*ssh.Server, error) {
	return wish.NewServer(
		wish.WithAddress(net.JoinHostPort("", port)),
		wish.WithHostKeyPath(hostKey),
		wish.WithMiddleware(
			bubbletea.Middleware(handler),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
}
