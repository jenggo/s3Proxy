package main

import (
	"os"
	"os/signal"
	"s3proxy/infrastructure/config"
	"s3proxy/infrastructure/repository"
	"s3proxy/interfaces/http"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Load configuration
	if err := config.LoadConfig(); err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Configure logging
	zerolog.SetGlobalLevel(zerolog.Level(config.GetLogLevel()))
	log.Logger = zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "Mon 2006-01-02 15:04:05",
	}).With().Timestamp().Logger()

	// Initialize S3 repository
	s3repo, err := repository.NewS3Repository()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize S3 repository")
	}
	log.Info().Msg("S3 repository initialized successfully")

	// Create and start HTTP server
	server := http.NewServer(s3repo)
	go func() {
		if err := server.Start(); err != nil {
			log.Error().Err(err).Msg("Error starting server")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	
	log.Info().Msg("Shutting down server")
	
	// Graceful shutdown
	if err := server.ShutdownWithTimeout(5 * time.Second); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}
	
	log.Info().Msg("Server exited")
}