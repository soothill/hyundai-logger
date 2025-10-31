// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/soothill/hyundai-logger/internal/alerts"
	"github.com/soothill/hyundai-logger/internal/api"
	"github.com/soothill/hyundai-logger/internal/config"
	"github.com/soothill/hyundai-logger/internal/database"
	"github.com/soothill/hyundai-logger/internal/logging"
	"github.com/soothill/hyundai-logger/internal/retry"
)

var (
	configPath = flag.String("config", "config.yaml", "Path to configuration file")
	initDB     = flag.Bool("init-db", false, "Initialize database schema and exit")
	version    = "1.0.0"
)

func main() {
	flag.Parse()

	// Initial logging to stdout before we set up file logging
	log.Printf("Hyundai Logger v%s starting...", version)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Determine log file path
	logFilePath := cfg.Logging.File
	if logFilePath == "" {
		// Default to /var/log if running as service, otherwise local directory
		if _, err := os.Stat("/var/log/hyundai-logger"); err == nil {
			logFilePath = "/var/log/hyundai-logger/hyundai-logger.log"
		} else {
			logFilePath = "hyundai-logger.log"
		}
	}

	// Initialize logger
	logger, err := logging.New(logging.Config{
		LogFilePath: logFilePath,
		LogLevel:    cfg.Logging.Level,
		LogToFile:   true,
		LogToStdout: true,
	})
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Close()

	// Log startup information
	logger.LogStartup(version, cfg.Hyundai.Region, cfg.Hyundai.Brand, cfg.RateLimit.PollIntervalMinutes)
	logger.Info("Loaded configuration from %s", *configPath)
	logger.Info("Log file: %s", logFilePath)
	logger.LogRateLimit(cfg.RateLimit.RequestsPerHour)

	// Create context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to database
	logger.Info("Connecting to InfluxDB at %s...", cfg.Database.URL)
	db, err := database.New(ctx, cfg.Database.URL, cfg.Database.Token, cfg.Database.Organization, cfg.Database.Bucket)
	if err != nil {
		logger.Fatal("Failed to connect to database: %v", err)
	}
	defer db.Close()
	logger.LogDatabaseReady()

	// Initialize schema if requested
	if *initDB {
		logger.LogDatabaseInit()
		if err := db.InitSchema(ctx); err != nil {
			logger.Fatal("Failed to initialize database schema: %v", err)
		}
		logger.Info("Database schema initialized successfully")
		logger.Info("You can now run the logger without the -init-db flag")
		return
	}

	// Create API client with retry configuration
	apiClient := api.NewClient(
		cfg.Hyundai.Username,
		cfg.Hyundai.Password,
		cfg.Hyundai.PIN,
		cfg.Hyundai.Brand,
		cfg.Hyundai.Region,
		cfg.RateLimit.RequestsPerHour,
		retry.Config{
			MaxAttempts:       cfg.Retry.MaxAttempts,
			InitialDelayMs:    cfg.Retry.InitialDelayMs,
			MaxDelayMs:        cfg.Retry.MaxDelayMs,
			BackoffMultiplier: cfg.Retry.BackoffMultiplier,
		},
	)

	// Create email alerter for persistent errors
	alerter := alerts.NewAlerter(alerts.Config{
		Enabled:           cfg.Alerts.Enabled,
		SMTPHost:          cfg.Alerts.SMTPHost,
		SMTPPort:          cfg.Alerts.SMTPPort,
		SMTPUsername:      cfg.Alerts.SMTPUsername,
		SMTPPassword:      cfg.Alerts.SMTPPassword,
		FromEmail:         cfg.Alerts.FromEmail,
		ToEmail:           cfg.Alerts.ToEmail,
		AlertThreshold:    cfg.Alerts.AlertThreshold,
		AlertCooldownMins: cfg.Alerts.AlertCooldownMins,
	})

	if cfg.Alerts.Enabled {
		logger.Info("Email alerts enabled (threshold: %d failures, cooldown: %d minutes)",
			cfg.Alerts.AlertThreshold, cfg.Alerts.AlertCooldownMins)
	}

	// Create data logger with intelligent scheduling and alerting
	dataLogger := database.NewLogger(db, apiClient, &cfg.RateLimit, &cfg.RateLimit.ChargingConfig, alerter, cfg.Alerts.AlertThreshold, logger)

	// Start data logger
	if err := dataLogger.Start(ctx); err != nil {
		logger.Fatal("Failed to start data logger: %v", err)
	}

	// Setup signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Wait for interrupt signal
	sig := <-sigCh
	logger.Info("Received signal %v, shutting down gracefully...", sig)

	// Stop data logger
	dataLogger.Stop()

	// Cancel context
	cancel()

	// Log shutdown
	logger.LogShutdown()
}
