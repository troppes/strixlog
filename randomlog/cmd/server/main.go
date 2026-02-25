package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/troppes/strixlog/randomlog/internal/server"
)

var levels = []string{"INFO", "WARN", "ERROR", "DEBUG"}
var messages = []string{
	"user login successful",
	"database query executed",
	"cache miss",
	"request timeout",
	"connection established",
	"file not found",
	"rate limit exceeded",
	"service restarted",
}

type logEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Source    string `json:"source"`
}

func emitLogs() {
	enc := json.NewEncoder(os.Stdout)
	for {
		entry := logEntry{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Level:     levels[rand.Intn(len(levels))],
			Message:   messages[rand.Intn(len(messages))],
			Source:    "randomlog",
		}
		enc.Encode(entry)
		time.Sleep(time.Second)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	go emitLogs()

	log.Printf("randomlog starting on port %s", port)
	if err := server.NewServer(port).Start(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
