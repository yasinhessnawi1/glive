package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/glive/agent/server"
	"github.com/glive/core/config"
)

func main() {
	// Parse flags
	portFlag := flag.Int("port", 0, "Port to run the agent on")
	flag.Parse()

	// Load configuration
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// In production, allow running without local config (will use env vars)
	if !cfg.IsConfigured() && os.Getenv("GLIVE_ENV") != "production" {
		fmt.Println("⚠️  GLive is not configured yet.")
		fmt.Println("Run: glive config set api-key YOUR_API_KEY")
		os.Exit(1)
	}

	// Create and start server
	srv := server.New(cfg.Get())

	port := cfg.Get().AgentPort
	if *portFlag != 0 {
		port = *portFlag
	}

	fmt.Printf("🚀 GLive Agent starting on port %d...\n", port)

	if err := srv.Start(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
