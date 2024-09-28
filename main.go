package main

import (
	"CODStatusBot/admin"
	"CODStatusBot/bot"
	"CODStatusBot/database"
	"CODStatusBot/logger"
	"CODStatusBot/models"
	"CODStatusBot/services"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"
)

var discord *discordgo.Session

func main() {
	defer func() {
		if r := recover(); r != nil {
			logger.Log.Errorf("Recovered from panic: %v\n%s", r, debug.Stack())
		}
	}()

	if err := run(); err != nil {
		logger.Log.WithError(err).Error("Bot encountered an error and is shutting down")
		os.Exit(1)
	}
}

func run() error {
	logger.Log.Info("Starting COD Status Bot...")

	if err := loadEnvironmentVariables(); err != nil {
		return fmt.Errorf("failed to load environment variables: %w", err)
	}
	logger.Log.Info("Environment variables loaded successfully")

	if err := services.LoadEnvironmentVariables(); err != nil {
		return fmt.Errorf("failed to initialize EZ-Captcha service: %w", err)
	}
	logger.Log.Info("EZ-Captcha service initialized successfully")

	if err := database.Databaselogin(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	logger.Log.Info("Database connection established successfully")

	if err := initializeDatabase(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	logger.Log.Info("Database initialized successfully")

	startAdminDashboard()

	var err error
	discord, err = bot.StartBot()
	if err != nil {
		return fmt.Errorf("failed to start Discord bot: %w", err)
	}
	logger.Log.Info("Discord bot started successfully")

	go startPeriodicTasks(discord)

	logger.Log.Info("COD Status Bot startup complete")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	logger.Log.Info("Shutting down COD Status Bot...")
	if err := discord.Close(); err != nil {
		logger.Log.WithError(err).Error("Error closing Discord session")
	}

	return nil
}

func loadEnvironmentVariables() error {
	logger.Log.Info("Loading environment variables...")
	if err := godotenv.Load(); err != nil {
		logger.Log.WithError(err).Error("Error loading .env file")
		return fmt.Errorf("error loading .env file: %w", err)
	}

	requiredEnvVars := []string{
		"DISCORD_TOKEN",
		"EZCAPTCHA_CLIENT_KEY",
		"RECAPTCHA_SITE_KEY",
		"RECAPTCHA_URL",
		"DB_USER",
		"DB_PASSWORD",
		"DB_HOST",
		"DB_PORT",
		"DB_NAME",
		"DB_VAR",
		"DEVELOPER_ID",
	}

	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			return fmt.Errorf("%s is not set in the environment", envVar)
		}
	}

	return nil
}

func initializeDatabase() error {
	if err := database.DB.AutoMigrate(&models.Account{}, &models.Ban{}, &models.UserSettings{}); err != nil {
		return fmt.Errorf("failed to migrate database tables: %w", err)
	}
	return nil
}

func startAdminDashboard() {
	r := mux.NewRouter()
	r.HandleFunc("/admin", admin.DashboardHandler)
	r.HandleFunc("/admin/stats", admin.StatsHandler)

	staticDir := "/home/container/"
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	port := os.Getenv("ADMIN_PORT")
	if port == "" {
		port = "8080"
	}

	go func() {
		logger.Log.Infof("Admin dashboard starting on port %s", port)
		if err := http.ListenAndServe(":"+port, r); err != nil {
			logger.Log.WithError(err).Fatal("Failed to start admin dashboard")
		}
	}()
}

func startPeriodicTasks(s *discordgo.Session) {
	go func() {
		for {
			services.CheckAccounts(s)
			time.Sleep(time.Duration(services.GetEnvInt("SLEEP_DURATION", 3)) * time.Minute)
		}
	}()

	go admin.StartStatsCaching()
	go services.ScheduleBalanceChecks(s)

	logger.Log.Info("Periodic tasks started successfully")
}
