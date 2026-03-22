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
			var tableCount int
			checkQuery := `SELECT count(*) FROM pg_tables WHERE schemaname = 'public'`

			// Start a transaction for the DDL
			tx, err := Pool.Begin(ctx)
			if err != nil {
				log.Fatalf("Fatal: Could not begin transaction: %v", err)
			}
			defer tx.Rollback(context.Background())

			// Acquire transaction-level advisory lock
			_, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(1337)")
			if err != nil {
				log.Fatalf("Fatal: Could not acquire lock: %v", err)
			}

			// Use tx.QueryRow to check for the setup inside the lock
			err = tx.QueryRow(ctx, checkQuery).Scan(&tableCount)
			if err != nil {
				log.Fatalf("Fatal: Failed to count tables: %v", err)
			}

			if tableCount == 0 {
				fmt.Println("[*] Empty database found. Initializing CampusCompile schema...")

				// Read the DDL file. Ensure this path is correct relative to the compiled binary!
				ddlBytes, err := os.ReadFile("./scripts/ddl.sql")
				if err != nil {
					log.Fatalf("Fatal: Could not read ddl.sql file: %v", err)
				}

				// Execute the SQL schema using pgx simple protocol on the transaction
				_, err = tx.Conn().PgConn().Exec(ctx, string(ddlBytes)).ReadAll()
				if err != nil {
					log.Fatalf("Fatal: Failed to execute ddl.sql: %v", err)
				}

				err = tx.Commit(ctx)
				if err != nil {
					log.Fatalf("Fatal: Failed to commit schema transaction: %v", err)
				}

				fmt.Println("[*] Database schema initialized successfully!")
			} else if tableCount < 9 {
				log.Fatalf("FATAL STARTUP ERROR: Database is in a partially initialized state (%d/9 tables found). Please drop the public schema to re-initialize.", tableCount)
			} else {
				fmt.Println("[*] Database schema already completely initialized. Skipping DDL execution.")
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
