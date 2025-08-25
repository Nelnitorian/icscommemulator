package main

import (
	"flag"
	"os"

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
	
	// Create and configure the API
	api := web.NewNetworkAPI()
	
	// Start the server
	logger.Info("ICS Communication Emulator starting...")
	logger.Info("Server will be available at http://%s:%d", *host, *port)
	
	err := api.Run(*host, *port)
	if err != nil {
		logger.Error("Failed to start server: %v", err)
		os.Exit(1)
	}
}
