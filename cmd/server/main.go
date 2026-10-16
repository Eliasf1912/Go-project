package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/Eliasf1912/go-project/internal/adapter/httpserver"
)

func main() {
	addr := flag.String("addr", ":8080", "adresse d'écoute du serveur HTTP")
	flag.Parse()

	srv := &http.Server{
		Addr:              *addr,
		Handler:           httpserver.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("serveur HTTP démarré sur %s", *addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
