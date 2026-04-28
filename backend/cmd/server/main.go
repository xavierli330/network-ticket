package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/xavierli/network-ticket/internal/config"
	"go.uber.org/zap"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to configuration file")
	flag.Parse()

	// Load configuration.
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger.
	logger, err := config.InitLogger(&cfg.Log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Replace global logger so other packages can use zap.L().
	zap.ReplaceGlobals(logger)

	logger.Info("network-ticket server starting",
		zap.Int("port", cfg.Server.Port),
		zap.String("mode", cfg.Server.Mode),
	)
}
