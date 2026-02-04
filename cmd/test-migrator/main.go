package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/onlyspans/events/internal/migrator"
)

func main() {
	dbURL := flag.String("db", "", "PostgreSQL connection URL")
	flag.Parse()

	if *dbURL == "" {
		fmt.Println("Usage: test-migrator -db <postgres-connection-url>")
		fmt.Println("Example: test-migrator -db 'postgres://user:pass@localhost:5432/dbname?sslmode=disable'")
		os.Exit(1)
	}

	fmt.Println("Testing migrator with embedded migrations...")
	fmt.Printf("Database URL: %s\n", *dbURL)

	if err := migrator.Run(*dbURL); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("✓ Migration completed successfully!")
}
