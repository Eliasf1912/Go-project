package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Eliasf1912/go-project/internal/adapter/hasher"
	"github.com/Eliasf1912/go-project/internal/adapter/httpserver"
	"github.com/Eliasf1912/go-project/internal/adapter/postgres"
	"github.com/Eliasf1912/go-project/internal/usecase"
)

func main() {
	addr := flag.String("addr", ":8080", "adresse d'écoute du serveur HTTP")
	flag.Parse()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("la variable d'environnement DATABASE_URL est vide")
	}

	db, err := postgres.Open(databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	log.Println("connecté à PostgreSQL")

	// Adapters
	userRepo := postgres.NewUserRepository(db)
	sessionRepo := postgres.NewSessionRepository(db)
	passwordHasher := &hasher.BcryptHasher{}

	// Usecases
	authService := usecase.NewAuthService(userRepo, passwordHasher, sessionRepo)

	// Handlers HTTP
	authHandler := httpserver.NewAuthHandler(authService)

	srv := &http.Server{
		Addr:              *addr,
		Handler:           httpserver.NewRouter(authHandler),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("serveur HTTP démarré sur %s", *addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
