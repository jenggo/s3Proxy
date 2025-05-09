package main

import (
	"os"
	"os/signal"
	"s3proxy/pkg"
	"s3proxy/server"
	"s3proxy/types"
	"syscall"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Load configuration
	if err := cleanenv.ReadEnv(&types.Config); err != nil {
		log.Fatal().Err(err).Send()
	}

	// Configure logging
	zerolog.SetGlobalLevel(zerolog.Level(types.Config.App.LogLevel))
	log.Logger = zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "Mon 2006-01-02 15:04:05",
	}).With().Timestamp().Logger()

	// Initialize MinIO client
	if err := pkg.InitMinio(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize MinIO client")
	}
	log.Info().Msg("MinIO client initialized successfully")

	// Start the server
	app, err := server.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
	defer func() {
		if err := app.ShutdownWithTimeout(5 * time.Second); err != nil {
			log.Error().Err(err).Send()
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server")
}
