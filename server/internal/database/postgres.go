package database

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func InitDB(host string) {
	dbUser := os.Getenv("POSTGRES_USER")
	if dbUser == "" {
		log.Fatal("FATAL STARTUP ERROR: POSTGRES_USER environment variable is missing")
	}

	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	if dbPassword == "" {
		log.Fatal("FATAL STARTUP ERROR: POSTGRES_PASSWORD environment variable is missing")
	}

	// Safely construct the connection URL using Go's native struct
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(dbUser, dbPassword),
		Host:     fmt.Sprintf("%s:5432", host),
		Path:     "CampusCompile_db",
		RawQuery: "sslmode=disable",
	}
	dbURL := u.String()

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

			// --- NEW: AUTO-MIGRATION LOGIC ---
			var tableExists bool
			checkQuery := `SELECT EXISTS (
				SELECT FROM pg_tables 
				WHERE schemaname = 'public' 
				AND tablename  = 'users'
			)`

			// Use pgxpool QueryRow to check for the table
			err = Pool.QueryRow(ctx, checkQuery).Scan(&tableExists)
			if err != nil {
				log.Fatalf("Fatal: Failed to check if tables exist: %v", err)
			}

			if !tableExists {
				fmt.Println("[*] No tables found. Initializing CampusCompile schema...")

				// Read the DDL file. Ensure this path is correct relative to the compiled binary!
				ddlBytes, err := os.ReadFile("./scripts/ddl.sql")
				if err != nil {
					log.Fatalf("Fatal: Could not read ddl.sql file: %v", err)
				}

				// Execute the SQL schema using pgxpool Exec
				_, err = Pool.Exec(ctx, string(ddlBytes))
				if err != nil {
					log.Fatalf("Fatal: Failed to execute ddl.sql: %v", err)
				}

				fmt.Println("[*] Database schema initialized successfully!")
			} else {
				fmt.Println("[*] Database schema already exists. Skipping DDL execution.")
			}
			// ---------------------------------

			return
		}

		fmt.Printf("[!] Database not ready (Attempt %d/5). Waiting 2 seconds...\n", i)
		time.Sleep(2 * time.Second)

		if i == 5 {
			log.Fatalf("Fatal: Could not connect to database after 5 attempts: %v\n", err)
		}
	}
}
