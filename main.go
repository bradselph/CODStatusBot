package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/bradselph/CODStatusBot/bot"
	"github.com/bradselph/CODStatusBot/command/verdansk"
	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bradselph/CODStatusBot/services"
	"github.com/bwmarrin/discordgo"
)

var discord *discordgo.Session

func loadEnv(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening config file: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("Error closing config file: %v\n", err)
		}
	}(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		value = strings.Trim(value, `"'`)

		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("error setting environment variable %s: %w", key, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	return nil
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic: %v\n%s\n", r, debug.Stack())
		}
	}()

	if err := run(); err != nil {
		fmt.Printf("Bot encountered an error and is shutting down: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Println("Starting COD Status Bot...")

	if err := loadEnv("config.env"); err != nil {
		return fmt.Errorf("failed to load environment variables: %w", err)
	}

	if err := configuration.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := logger.InitializeLogger(); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	logger.Log.Info("Starting COD Status Bot...")

	services.InitHTTPClients()
	services.InitializeServices()
	cfg := configuration.Get()

	if !cfg.CaptchaService.Capsolver.Enabled && !cfg.CaptchaService.EZCaptcha.Enabled && !cfg.CaptchaService.TwoCaptcha.Enabled {
		logger.Log.Warn("No captcha services are enabled - functionality will be limited")
	} else {
		var enabledServices []string
		if cfg.CaptchaService.Capsolver.Enabled && cfg.CaptchaService.Capsolver.ClientKey != "" {
			enabledServices = append(enabledServices, "Capsolver")
			if err := services.ValidateDefaultCapsolverConfig(); err != nil {
				logger.Log.WithError(err).Error("Capsolver service enabled but configuration is invalid")
				cfg.CaptchaService.Capsolver.Enabled = false
			} else {
				logger.Log.Info("Capsolver service enabled and configured correctly")
			}
		}
		if cfg.CaptchaService.EZCaptcha.Enabled && cfg.CaptchaService.EZCaptcha.ClientKey != "" {
			enabledServices = append(enabledServices, "EZCaptcha")
			if services.VerifyEZCaptchaConfig() {
				logger.Log.Info("EZCaptcha service enabled and configured correctly")
			} else {
				logger.Log.Error("EZCaptcha service enabled but configuration is invalid")
				cfg.CaptchaService.EZCaptcha.Enabled = false
			}
		}
		if cfg.CaptchaService.TwoCaptcha.Enabled && cfg.CaptchaService.TwoCaptcha.ClientKey != "" {
			enabledServices = append(enabledServices, "2Captcha")
			logger.Log.Info("2Captcha service enabled and configured correctly")
		}

		if len(enabledServices) == 0 {
			logger.Log.Error("No properly configured captcha services found")
		} else {
			logger.Log.Infof("Enabled captcha services: %s", strings.Join(enabledServices, ", "))
		}
	}

	if err := database.Databaselogin(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	logger.Log.Info("Database connection established successfully")

	appShardManager := services.GetAppShardManager()
	if err := appShardManager.Initialize(); err != nil {
		logger.Log.WithError(err).Error("Failed to initialize app shard manager")
	}

	shardCtx, shardCancel := context.WithCancel(context.Background())
	defer shardCancel()

	appShardManager.StartHeartbeat(shardCtx)
	logger.Log.Infof("Application shard %d of %d initialized successfully",
		appShardManager.ShardID, appShardManager.TotalShards)

	services.StartAdminAPI()
	logger.Log.Info("Admin API started successfully")

	var err error
	discord, err = bot.StartBot()
	if err != nil {
		return fmt.Errorf("failed to start Discord bot: %w", err)
	}
	logger.Log.Info("Discord bot started successfully")

	services.StartNotificationProcessor(discord)
	logger.Log.Info("Notification processor started successfully")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	periodicTasksCtx, cancelPeriodicTasks := context.WithCancel(ctx)
	go startPeriodicTasks(periodicTasksCtx, discord)

	verdansk.InitCleanupRoutine()

	logger.Log.Info("COD Status Bot startup complete")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Log.Info("Shutting down COD Status Bot...")

	cancelPeriodicTasks()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	done := make(chan struct{})
	go func() {
		close(done)
	}()

	select {
	case <-done:
		logger.Log.Info("All goroutines terminated gracefully")
	case <-shutdownCtx.Done():
		logger.Log.Warn("Shutdown timed out, forcing exit")
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

func startPeriodicTasks(ctx context.Context, s *discordgo.Session) {
	cfg := configuration.Get()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				services.CheckAccounts(s)
				time.Sleep(time.Duration(cfg.Intervals.Sleep) * time.Minute)
			}
		}
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var users []models.UserSettings
				if err := database.DB.Find(&users).Error; err != nil {
					logger.Log.WithError(err).Error("Failed to fetch users for consolidated updates")
					time.Sleep(time.Hour)
					continue
				}

				for _, user := range users {
					var accounts []models.Account
					if err := database.DB.Where("user_id = ? AND is_check_disabled = ? AND is_expired_cookie = ?",
						user.UserID, false, false).Find(&accounts).Error; err != nil {
						logger.Log.WithError(err).Error("Failed to fetch accounts for user")
						continue
					}

					if time.Since(user.LastDailyUpdateNotification) >=
						time.Duration(cfg.Intervals.Notification)*time.Hour {
						services.SendConsolidatedDailyUpdate(s, user.UserID, user, accounts)
					}
				}

				time.Sleep(time.Hour)
			}
		}
	}()

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

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := s.UpdateWatchStatus(0, bot.StatusMessage); err != nil {
					logger.Log.WithError(err).Error("Failed to refresh presence status")
				}
				time.Sleep(60 * time.Minute)
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(12 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				services.CleanupOldRateLimitData()
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(cfg.Users.CleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				services.CleanupInactiveUsers()
				logger.Log.Info("Ran inactive users cleanup")
				services.LogInstallationStats(s)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := s.UpdateWatchStatus(0, bot.StatusMessage); err != nil {
					logger.Log.WithError(err).Error("Failed to refresh presence status")
				}
				time.Sleep(60 * time.Minute)
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cfg := configuration.Get()
				if err := services.CleanupOldAnalyticsData(cfg.Admin.RetentionDays); err != nil {
					logger.Log.WithError(err).Error("Failed to clean up old analytics data")
				}
			}
		}
	}()

	logger.Log.Info("Periodic tasks started successfully")
}
