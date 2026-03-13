package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"products-orchestio-li/backend/internal/app"
	"products-orchestio-li/backend/internal/model"
	"products-orchestio-li/backend/internal/store"
)

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func main() {
	databaseURL := envOrDefault("DATABASE_URL", "postgres://products:products@localhost:5432/products?sslmode=disable")
	port := envOrDefault("PORT", "8080")
	jwtSecret := envOrDefault("AUTH_JWT_SECRET", "change-me-in-env")
	adminEmail := envOrDefault("AUTH_ADMIN_EMAIL", "admin@products.local")
	adminPassword := envOrDefault("AUTH_ADMIN_PASSWORD", "admin123")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s, err := store.NewPostgresStore(ctx, databaseURL)
	if err != nil {
		log.Fatalf("store init failed: %v", err)
	}
	defer s.Close()

	if _, err := s.EnsureUser(ctx, model.CreateUserInput{Email: adminEmail, Password: adminPassword}); err != nil {
		log.Fatalf("bootstrap admin user failed: %v", err)
	}

	a, err := app.New(s, jwtSecret)
	if err != nil {
		log.Fatalf("app init failed: %v", err)
	}

	log.Printf("backend listening on :%s", port)
	if err := http.ListenAndServe(":"+port, a.Router()); err != nil {
		log.Fatal(err)
	}
}
