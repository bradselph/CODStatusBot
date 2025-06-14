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

	cfg := configuration.Get()
	if cfg.Discord.Token == "" {
		return fmt.Errorf("DISCORD_TOKEN is required but not set")
	}

	if cfg.Database.Host == "" || cfg.Database.User == "" || cfg.Database.Password == "" || cfg.Database.Name == "" {
		return fmt.Errorf("database configuration is incomplete")
	}

	services.InitHTTPClients()

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

	services.InitializeProxyStatsAfterDB()
	logger.Log.Info("Proxy stats initialization completed")

	appShardManager := services.GetAppShardManager()
	if err := appShardManager.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize app shard manager: %w", err)
	}

	shardCtx, shardCancel := context.WithCancel(context.Background())
	defer shardCancel()

	appShardManager.StartHeartbeat(shardCtx)
	logger.Log.Infof("Application shard %d of %d initialized successfully (Instance: %s)",
		appShardManager.ShardID, appShardManager.TotalShards, appShardManager.InstanceID)

	if appShardManager.IsLeader() {
		services.StartAdminAPI()
		logger.Log.Info("Started Admin API (leader shard)")
	} else {
		logger.Log.Info("Skipping Admin API startup (not leader shard)")
	}

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
	go startPeriodicTasks(periodicTasksCtx, discord, appShardManager)

	errorCleanupCtx, cancelErrorCleanup := context.WithCancel(ctx)
	if appShardManager.IsLeader() {
		go services.StartErrorCleanupRoutine(errorCleanupCtx)
		logger.Log.Info("Started error cleanup routine (leader shard)")
	}

	edgeCaseCtx, cancelEdgeCase := context.WithCancel(ctx)
	if appShardManager.IsLeader() {
		go services.StartEdgeCaseCleanupRoutine(edgeCaseCtx)
		logger.Log.Info("Started edge case cleanup routine (leader shard)")
	}

	if appShardManager.IsLeader() {
		verdansk.InitCleanupRoutine()
		logger.Log.Info("Initialized Verdansk cleanup routine (leader shard)")
	}

	logger.Log.Info("COD Status Bot startup complete")

	go startHealthCheckRoutine(discord, appShardManager)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	startupComplete := make(chan bool, 1)
	go func() {
		time.Sleep(5 * time.Second)
		startupComplete <- true
	}()

	select {
	case <-startupComplete:
		logger.Log.Info("All services are ready")
	case <-time.After(time.Duration(cfg.Startup.TimeoutSeconds) * time.Second):
		logger.Log.Warn("Startup timeout reached, continuing anyway")
	}

	services.LogAnalyticsEvent("shard_startup_complete", "", "", "", "",
		appShardManager.ShardID, appShardManager.InstanceID, map[string]interface{}{
			"is_leader":                appShardManager.IsLeader(),
			"startup_duration_seconds": 5,
		})

	<-stop

	logger.Log.Info("Shutting down COD Status Bot...")

	cancelPeriodicTasks()
	cancelErrorCleanup()
	cancelEdgeCase()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Startup.ShutdownTimeout)
	defer shutdownCancel()
	done := make(chan struct{})
	go func() {
		services.HandleGracefulShutdown(discord)
		close(done)
	}()

	select {
	case <-done:
		logger.Log.Info("All goroutines terminated gracefully")
	case <-shutdownCtx.Done():
		logger.Log.Warn("Shutdown timed out, forcing exit")
	}

	services.LogAnalyticsEvent("shard_shutdown", "", "", "", "",
		appShardManager.ShardID, appShardManager.InstanceID, map[string]interface{}{
			"shutdown_reason": "signal_received",
		})

	logger.Log.Info("Shutdown complete")
	return nil
}

func startPeriodicTasks(ctx context.Context, s *discordgo.Session, shardManager *services.AppShardManager) {
	cfg := configuration.Get()

	go func() {
		checkTicker := time.NewTicker(time.Duration(cfg.Intervals.Sleep) * time.Minute)
		defer checkTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-checkTicker.C:
				if shardManager.Initialized {
					logger.Log.Debugf("Shard %d starting account check cycle", shardManager.ShardID)
					services.CheckAccounts(s)
				}
			}
		}
	}()

	if shardManager.IsLeader() {
		go func() {
			updateTicker := time.NewTicker(time.Hour)
			defer updateTicker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-updateTicker.C:
					logger.Log.Debug("Leader shard processing consolidated daily updates")
					var users []models.UserSettings
					if err := database.DB.Find(&users).Error; err != nil {
						logger.Log.WithError(err).Error("Failed to fetch users for consolidated updates")
						continue
					}
					for _, user := range users {
						if !shardManager.IsUserAssignedToShard(user.UserID) {

							processedUsers := 0
							continue
						}

						var accounts []models.Account
						if err := database.DB.Where("user_id = ? AND is_check_disabled = ? AND is_expired_cookie = ?",
							user.UserID, false, false).Find(&accounts).Error; err != nil {
							logger.Log.WithError(err).Error("Failed to fetch accounts for user")
							continue
						}

						if time.Since(user.LastDailyUpdateNotification) >=
							time.Duration(cfg.Intervals.Notification)*time.Hour {
							services.SendConsolidatedDailyUpdate(s, user.UserID, user, accounts)
							processedUsers++
						}
					}
					logger.Log.Debugf("Leader shard processed %d users for daily updates", processedUsers)
				}
			}
		}()

		go services.ScheduleBalanceChecks(s)

		go func() {
			announcementTicker := time.NewTicker(24 * time.Hour)
			defer announcementTicker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-announcementTicker.C:
					if err := services.SendAnnouncementToAllUsers(s); err != nil {
						logger.Log.WithError(err).Error("Failed to send global announcement")
					}
				}
			}
		}()

		go func() {
			cleanupTicker := time.NewTicker(12 * time.Hour)
			defer cleanupTicker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-cleanupTicker.C:
					services.CleanupOldRateLimitData()
					logger.Log.Debug("Leader shard completed rate limit cleanup")
				}
			}
		}()

		go func() {
			userCleanupTicker := time.NewTicker(cfg.Users.CleanupInterval)
			defer userCleanupTicker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-userCleanupTicker.C:
					services.CleanupInactiveUsers()
					logger.Log.Info("Leader shard completed inactive users cleanup")
					services.LogInstallationStats(s)
				}
			}
		}()

		go func() {
			analyticsTicker := time.NewTicker(24 * time.Hour)
			defer analyticsTicker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-analyticsTicker.C:
					if err := services.CleanupOldAnalyticsData(cfg.Admin.RetentionDays); err != nil {
						logger.Log.WithError(err).Error("Failed to clean up old analytics data")
					}
				}
			}
		}()
	}

	go func() {
		statusTicker := time.NewTicker(60 * time.Minute)
		defer statusTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-statusTicker.C:
				if err := s.UpdateWatchStatus(0, bot.StatusMessage); err != nil {
					logger.Log.WithError(err).Error("Failed to refresh presence status")
				}
			}
		}
	}()

	logger.Log.Infof("Shard %d periodic tasks started (leader: %v)",
		shardManager.ShardID, shardManager.IsLeader())
}

func startHealthCheckRoutine(s *discordgo.Session, shardManager *services.AppShardManager) {
	cfg := configuration.Get()
	ticker := time.NewTicker(cfg.Startup.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		if s.DataReady == false {
			logger.Log.Errorf("Shard %d Discord connection is not ready", shardManager.ShardID)
			continue
		}

		if err := database.CheckConnection(); err != nil {
			logger.Log.WithError(err).Errorf("Shard %d database health check failed", shardManager.ShardID)
			continue
		}

		if !shardManager.Initialized {
			logger.Log.Errorf("Shard %d manager is not initialized", shardManager.ShardID)
			continue
		}

		logger.Log.Debugf("Shard %d health check passed", shardManager.ShardID)
	}
}
