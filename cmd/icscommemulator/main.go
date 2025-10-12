package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"icscommemulator/pkg/logger"
	"icscommemulator/pkg/web"
)

func main() {
	// Parse command line flags
	host := flag.String("host", "127.0.0.1", "Host to bind the server to")
	port := flag.Int("port", 8080, "Port to bind the server to")
	logLevel := flag.String("log", "DEBUG", "Log level (DEBUG, INFO, WARNING, ERROR)")
	flag.Parse()

	// Configure logging
	logger.SetLevel(*logLevel)
	logger.Info("ICS Communication Emulator starting...")

	// Create and configure the server
	server, err := web.NewServer(*host, *port)
	if err != nil {
		logger.Error("Failed to create server: %v", err)
		os.Exit(1)
	}

	// Start server in goroutine
	go func() {
		logger.Info("Server starting on http://%s:%d", *host, *port)
		if err := server.Start(); err != nil {
			logger.Error("Server error: %v", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown: %v", err)
		os.Exit(1)
	}

	logger.Info("Server exited successfully")
}
