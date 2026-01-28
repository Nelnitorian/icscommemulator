package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"icscommemulator/pkg/appconfig"
	"icscommemulator/pkg/logger"
	"icscommemulator/pkg/scenario"
	"icscommemulator/pkg/web"
)

func main() {
	host := flag.String("host", "127.0.0.1", "Host to bind the server to")
	port := flag.Int("port", 8080, "Port to bind the server to")
	logLevel := flag.String("log", "INFO", "Log level (DEBUG, INFO, WARNING, ERROR)")
	configPath := flag.String("config", "", "Path to config file (YAML or JSON)")
	flag.Parse()

	logger.SetLevel(*logLevel)
	logger.Info("ICS Communication Emulator starting...")

	if *configPath != "" {
		cfg, err := appconfig.Load(*configPath)
		if err != nil {
			logger.Error("Failed to load config: %v", err)
			os.Exit(1)
		}
		baseDir := filepath.Dir(*configPath)
		if err := cfg.ImportScenarios(scenario.NewStorage(), baseDir); err != nil {
			logger.Error("Failed to import scenarios: %v", err)
			os.Exit(1)
		}
	}

	server, err := web.NewServer(*host, *port)
	if err != nil {
		logger.Error("Failed to create server: %v", err)
		os.Exit(1)
	}

	go func() {
		logger.Info("Server starting on http://%s:%d", *host, *port)
		if err := server.Start(); err != nil {
			logger.Error("Server error: %v", err)
			os.Exit(1)
		}
	}()

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
