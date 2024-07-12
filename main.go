package main

import (
	"CODStatusBot/bot"
	"CODStatusBot/database"
	"CODStatusBot/logger"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	logger.Log.Info("Bot starting...") // Log that the bot is starting up.
	err := loadEnvironmentVariables()  // Load environment variables from .env file.
	if err != nil {
		logger.Log.WithError(err).WithField("Bot Startup", "Environment Variables").Error()
		os.Exit(1)
	}

	err = database.Databaselogin() // Connect to the database.
	if err != nil {
		logger.Log.WithError(err).WithField("Bot Startup", "Database login").Error()
		os.Exit(1)
	}
	err = bot.StartBot() // Start the Discord bot.
	if err != nil {
		logger.Log.WithError(err).WithField("Bot Startup", "Discord login").Error()
		os.Exit(1)
	}
	logger.Log.Info("Bot is running")                                // Log that the bot is running.
	sc := make(chan os.Signal, 1)                                    // Set up a channel to receive system signals.
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt) // Notify the channel when a SIGINT, SIGTERM, or Interrupt signal is received.
	<-sc                                                             // Block until a signal is received.

}

// loadEnvironmentVariables loads environment variables from a .env file.
func loadEnvironmentVariables() error {
	logger.Log.Info("Loading environment variables...") // Log that environment variables are being loaded.
	err := godotenv.Load()                              // Load environment variables from .env file.
	if err != nil {
		logger.Log.WithError(err).Error("Error loading .env file")
		return err
	}
	return nil
}
