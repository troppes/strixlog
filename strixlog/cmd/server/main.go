package main

import (
	"log"
	"os"

	"github.com/troppes/strixlog/strixlog/internal/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("strixlog starting on port %s", port)
	if err := server.NewServer(port).Start(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
