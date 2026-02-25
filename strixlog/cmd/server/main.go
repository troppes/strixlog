package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/troppes/strixlog/strixlog/internal/printer"
	"github.com/troppes/strixlog/strixlog/internal/server"
	"github.com/troppes/strixlog/strixlog/internal/source/docker"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	src, err := docker.NewDockerSource()
	if err != nil {
		log.Fatalf("docker source init: %v", err)
	}
	if err := src.Start(ctx); err != nil {
		log.Fatalf("docker source start: %v", err)
	}
	defer src.Stop() //nolint:errcheck

	go printer.PrintLogs(ctx, src.Logs())

	log.Printf("strixlog starting on port %s", port)
	if err := server.NewServer(port).Start(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
