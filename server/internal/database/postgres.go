package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool is the globally accessible database connection pool
var Pool *pgxpool.Pool

// InitDB initializes the PostgreSQL connection
func InitDB(host string) {
	// 1. Securely fetch credentials with local-dev fallbacks
	dbUser := os.Getenv("POSTGRES_USER")
	if dbUser == "" {
		dbUser = "campus_app"
	}

	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	if dbPassword == "" {
		dbPassword = "app"
	}

	// 2. Dynamically build the connection string
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:5432/CampusCompile_db?sslmode=disable", dbUser, dbPassword, host)

	fmt.Println("[*] Attempting to connect to PostgreSQL...")
	var err error
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		Pool, err = pgxpool.New(ctx, dbURL)

		if err == nil {
			err = Pool.Ping(ctx)
		}

		if err == nil {
			fmt.Println("[*] Connected to PostgreSQL successfully!")
			return
		}

		fmt.Printf("[!] Database not ready (Attempt %d/5). Waiting 2 seconds...\n", i)
		time.Sleep(2 * time.Second)

		if i == 5 {
			log.Fatalf("Fatal: Could not connect to database after 5 attempts: %v\n", err)
		}
	}
}
