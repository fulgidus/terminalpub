package main

import (
	"fmt"
	"log"
	"os"

	"github.com/fulgidus/terminalpub/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: migrate [up|down|version]")
		fmt.Println("")
		fmt.Println("Commands:")
		fmt.Println("  up       Run all pending migrations")
		fmt.Println("  down     Rollback the last migration")
		fmt.Println("  version  Show current migration version")
		os.Exit(1)
	}

	command := os.Args[1]

	// Load configuration
	cfg := config.LoadOrDefault("config/config.yaml")

	// Build database URL
	dbURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.Postgres.User,
		cfg.Database.Postgres.Password,
		cfg.Database.Postgres.Host,
		cfg.Database.Postgres.Port,
		cfg.Database.Postgres.Database,
		cfg.Database.Postgres.SSLMode,
	)

	// Create migration instance
	m, err := migrate.New(
		"file://migrations",
		dbURL,
	)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer m.Close()

	// Execute command
	switch command {
	case "up":
		fmt.Println("Running migrations...")
		if err := m.Up(); err != nil {
			if err == migrate.ErrNoChange {
				fmt.Println("No migrations to run")
				return
			}
			log.Fatalf("Migration failed: %v", err)
		}
		fmt.Println("Migrations completed successfully!")

	case "down":
		fmt.Println("Rolling back last migration...")
		if err := m.Steps(-1); err != nil {
			if err == migrate.ErrNoChange {
				fmt.Println("No migrations to rollback")
				return
			}
			log.Fatalf("Rollback failed: %v", err)
		}
		fmt.Println("Rollback completed successfully!")

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			if err == migrate.ErrNilVersion {
				fmt.Println("No migrations have been run yet")
				return
			}
			log.Fatalf("Failed to get version: %v", err)
		}
		fmt.Printf("Current version: %d", version)
		if dirty {
			fmt.Println(" (dirty)")
		} else {
			fmt.Println()
		}

	default:
		log.Fatalf("Unknown command: %s. Use 'up', 'down', or 'version'", command)
	}
}
