// Package main is the entry point for the Clawland Fleet Manager.
// Fleet Manager handles Cloud-Edge orchestration: node registration,
// heartbeat monitoring, event collection, and command dispatch.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Clawland-AI/clawland-fleet/pkg/fleet"
)

const version = "0.1.0"

func main() {
	addr := listenAddress(os.Getenv("PORT"))

	fmt.Printf("Clawland Fleet Manager v%s\n", version)
	fmt.Printf("   Cloud-Edge orchestration starting on %s...\n", addr)
	fmt.Println("   Waiting for edge agent registrations...")

	log.Printf("Fleet Manager listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, fleet.NewServer(fleet.NewRegistry())))
}

func listenAddress(port string) string {
	if port == "" {
		port = "8080"
	}
	return ":" + port
}
