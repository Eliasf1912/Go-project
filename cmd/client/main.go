package main

import (
	"flag"
	"log"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/Eliasf1912/go-project/internal/apiclient"
	"github.com/Eliasf1912/go-project/internal/tui/client"
)

func main() {
	defaultURL := os.Getenv("API_URL")
	if defaultURL == "" {
		defaultURL = "http://localhost:8080"
	}
	apiURL := flag.String("api-url", defaultURL, "adresse du serveur HTTP")
	flag.Parse()

	api := apiclient.New(*apiURL)

	if _, err := tea.NewProgram(client.New(api)).Run(); err != nil {
		log.Fatal(err)
	}
}
