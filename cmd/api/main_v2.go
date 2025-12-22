//go:build ignore
// +build ignore

package main

// =============================================================================
// MAIN V2 - Service-Based API Server
// This demonstrates the new architecture using ISP-compliant services
// To use: rename to main.go after full migration is complete
// =============================================================================

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chimera-pool/chimera-pool-core/internal/api"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func mainV2() {
	log.Println("🚀 Starting Chimera Pool API Server (v2 - Service Architecture)...")

	// Load configuration
	config := api.LoadServerConfig()

	// Initialize database
	db, err := initDatabaseV2(config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := runMigrationsV2(db); err != nil {
		log.Printf("Warning: Migration error: %v", err)
	}

	// Create server with all services wired up
	server, err := api.NewServerWithServices(config, db)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start server in goroutine
	go func() {
		log.Printf("✅ API Server listening on port %s", config.Port)
		if err := server.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server exited gracefully")
}

func initDatabaseV2(url string) (*sql.DB, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Println("✅ Connected to PostgreSQL database")
	return db, nil
}

func runMigrationsV2(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file:///app/migrations",
		"postgres", driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	log.Println("✅ Database migrations applied")
	return nil
}
