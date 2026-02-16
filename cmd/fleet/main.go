// Package main is the entry point for the Clawland Fleet Manager.
// Fleet Manager handles Cloud-Edge orchestration: node registration,
// heartbeat monitoring, event collection, and command dispatch.
package main

import (
	"fmt"
	"log"
	"os"
)

const version = "0.1.0"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("馃 Clawland Fleet Manager v%s\n", version)
	fmt.Printf("   Cloud-Edge orchestration starting on :%s...\n", port)
	fmt.Println("   Waiting for edge agent registrations...")

	log.Printf("Fleet Manager listening on :%s", port)
}