package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/bradselph/CODStatusBot/bot"
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bradselph/CODStatusBot/services"
	"github.com/bradselph/CODStatusBot/webserver"

	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
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
		logger.Log.Fatal("Exiting due to error")
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

	server := startAdminDashboard()

	var err error
	discord, err = bot.StartBot()
	if err != nil {
		return fmt.Errorf("failed to start Discord bot: %w", err)
	}
	logger.Log.Info("Discord bot started successfully")

	periodicTasksCtx, cancelPeriodicTasks := context.WithCancel(context.Background())
	go startPeriodicTasks(periodicTasksCtx, discord)

	logger.Log.Info("COD Status Bot startup complete")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Log.Info("Shutting down COD Status Bot...")

	cancelPeriodicTasks()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Log.WithError(err).Error("Error shutting down admin dashboard server")
	}

	if err := discord.Close(); err != nil {
		logger.Log.WithError(err).Error("Error closing Discord session")
	}

	if err := database.CloseConnection(); err != nil {
		logger.Log.WithError(err).Error("Error closing database connection")
	}

	logger.Log.Info("Shutdown complete")
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
		"ADMIN_PORT",
		"ADMIN_USERNAME",
		"ADMIN_PASSWORD",
		"SESSION_KEY",
		"CHECK_INTERVAL",
		"NOTIFICATION_INTERVAL",
		"COOLDOWN_DURATION",
		"SLEEP_DURATION",
		"COOKIE_CHECK_INTERVAL_PERMABAN",
		"STATUS_CHANGE_COOLDOWN",
		"GLOBAL_NOTIFICATION_COOLDOWN",
		"COOKIE_EXPIRATION_WARNING",
		"TEMP_BAN_UPDATE_INTERVAL",
		"STATIC_DIR",
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

func startAdminDashboard() *http.Server {
	r := mux.NewRouter()
	r.HandleFunc("/", webserver.HomeHandler)
	r.HandleFunc("/help", webserver.HelpHandler)
	r.HandleFunc("/terms", webserver.TermsHandler)
	r.HandleFunc("/policy", webserver.PolicyHandler)
	r.HandleFunc("/admin/login", webserver.LoginHandler)
	r.HandleFunc("/admin/logout", webserver.LogoutHandler)
	r.HandleFunc("/admin", webserver.AuthMiddleware(webserver.DashboardHandler))
	r.HandleFunc("/admin/stats", webserver.AuthMiddleware(webserver.StatsHandler))
	r.HandleFunc("/api/server-count", webserver.ServerCountHandler)

	staticDir := os.Getenv("STATIC_DIR")
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	port := os.Getenv("ADMIN_PORT")

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 20 * time.Second,
	}

	go func() {
		logger.Log.Infof("Admin dashboard starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.WithError(err).Fatal("Failed to start admin dashboard")
		}
	}()

	return server
}

func startPeriodicTasks(ctx context.Context, s *discordgo.Session) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				services.CheckAccounts(s)
				sleepDuration := time.Duration(services.GetEnvInt("SLEEP_DURATION", 3)) * time.Minute
				time.Sleep(sleepDuration)
			}
		}
	}()

	go webserver.StartStatsCaching()
	go services.ScheduleBalanceChecks(s)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := services.SendAnnouncementToAllUsers(s); err != nil {
					logger.Log.WithError(err).Error("Failed to send global announcement")
				}
				time.Sleep(24 * time.Hour)
			}
		}
	}()

	logger.Log.Info("Periodic tasks started successfully")
}
