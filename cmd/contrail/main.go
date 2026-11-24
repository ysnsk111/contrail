package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"contrail/internal/config"
	"contrail/internal/database"
	"contrail/internal/poller"
	"contrail/internal/server"
)

func main() {
	log.Println("================================================================")
	log.Println("     Contrail: E-Airline Near-Real-Time Flight API System       ")
	log.Println("================================================================")

	cfg := config.LoadConfig()
	log.Printf("[Config] Public Universal Base URL: %s", cfg.APIBaseURL)
	log.Printf("[Config] Database path: %s (Retention: %d days)", cfg.DBPath, cfg.RetentionDays)
	log.Printf("[Config] Poller enabled: %v (Upstream: %s)", cfg.PollerEnabled, cfg.UpstreamProvider)
	log.Printf("[Config] adsbdb token bucket rate: %d req/min (Cache TTL: %dh)", cfg.ADSBDbRateLimitPerMin, cfg.CallsignCacheTTLHours)

	// Initialize Database
	db, err := database.InitDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("[DB] Fatal error initializing database: %v", err)
	}
	defer db.Close()
	log.Println("[DB] SQLite database initialized with WAL mode successfully.")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start 3-month retention cleaner
	db.StartRetentionCleaner(ctx, cfg.RetentionDays, cfg.RetentionCleanupInterval)
	log.Printf("[DB] Retention cleaner worker started (purging snapshots > %d days every %v).", cfg.RetentionDays, cfg.RetentionCleanupInterval)

	// Initialize Poller
	flightPoller := poller.NewPoller(cfg, db)
	if cfg.PollerEnabled {
		flightPoller.Start(ctx)
	}

	// Initialize Server
	srv := server.NewServer(cfg, db, flightPoller)

	// Channel to listen for OS interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[Server] Fatal error in HTTP server: %v", err)
		}
	}()

	sig := <-sigChan
	log.Printf("[Main] Received signal %v, commencing graceful shutdown...", sig)

	cancel() // Stop poller and workers

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[Server] Error during server shutdown: %v", err)
	}

	log.Println("[Main] Contrail service stopped cleanly.")
}
