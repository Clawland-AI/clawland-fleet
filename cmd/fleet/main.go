// Package main is the entry point for the Clawland Fleet Manager.
// Fleet Manager handles Cloud-Edge orchestration: node registration,
// heartbeat monitoring, event collection, and command dispatch.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Clawland-AI/clawland-fleet/pkg/fleet"
)

const version = "0.1.0"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("🍇 Clawland Fleet Manager v%s\n", version)
	fmt.Printf("   Cloud-Edge orchestration starting on :%s...\n", port)
	fmt.Println("   Waiting for edge agent registrations...")

	// Initialize registry
	registry := fleet.NewRegistry()

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/fleet/register", fleet.RegisterHandler(registry))
	mux.HandleFunc("/api/v1/fleet/heartbeat", fleet.HeartbeatHandler(registry))
	mux.HandleFunc("/api/v1/fleet/nodes", fleet.ListNodesHandler(registry))
	mux.HandleFunc("/api/v1/fleet/nodes/", fleet.GetNodeHandler(registry))

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// Start background task to mark offline nodes
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			count := registry.MarkOffline(3 * time.Minute)
			if count > 0 {
				log.Printf("[MONITOR] Marked %d nodes as offline", count)
			}
		}
	}()

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Fleet Manager listening on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
