package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bradselph/CODStatusBot/utils"
	"github.com/bwmarrin/discordgo"
)

type EdgeCaseHandler struct {
	mu                  sync.RWMutex
	pendingInteractions map[string]time.Time
	rateLimitedUsers    map[string]time.Time
	temporaryFailures   map[string]int
	lastCleanup         time.Time
}

var globalEdgeCaseHandler = &EdgeCaseHandler{
	pendingInteractions: make(map[string]time.Time),
	rateLimitedUsers:    make(map[string]time.Time),
	temporaryFailures:   make(map[string]int),
	lastCleanup:         time.Now(),
}

func GetEdgeCaseHandler() *EdgeCaseHandler {
	return globalEdgeCaseHandler
}

func (ech *EdgeCaseHandler) StartInteraction(userID string) bool {
	ech.mu.Lock()
	defer ech.mu.Unlock()

	if lastStart, exists := ech.pendingInteractions[userID]; exists {
		if time.Since(lastStart) < 5*time.Second {
			return false
		}
	}

	ech.pendingInteractions[userID] = time.Now()
	return true
}

func (ech *EdgeCaseHandler) EndInteraction(userID string) {
	ech.mu.Lock()
	defer ech.mu.Unlock()

	delete(ech.pendingInteractions, userID)
}

func (ech *EdgeCaseHandler) IsRateLimited(userID string) bool {
	ech.mu.RLock()
	defer ech.mu.RUnlock()

	if limitUntil, exists := ech.rateLimitedUsers[userID]; exists {
		return time.Now().Before(limitUntil)
	}
	return false
}

func (ech *EdgeCaseHandler) SetRateLimit(userID string, duration time.Duration) {
	ech.mu.Lock()
	defer ech.mu.Unlock()

	ech.rateLimitedUsers[userID] = time.Now().Add(duration)
}

func (ech *EdgeCaseHandler) RecordTemporaryFailure(identifier string) bool {
	ech.mu.Lock()
	defer ech.mu.Unlock()

	ech.temporaryFailures[identifier]++
	return ech.temporaryFailures[identifier] >= 3
}

func (ech *EdgeCaseHandler) ResetTemporaryFailures(identifier string) {
	ech.mu.Lock()
	defer ech.mu.Unlock()

	delete(ech.temporaryFailures, identifier)
}

func (ech *EdgeCaseHandler) Cleanup() {
	ech.mu.Lock()
	defer ech.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-10 * time.Minute)

	for userID, startTime := range ech.pendingInteractions {
		if startTime.Before(cutoff) {
			delete(ech.pendingInteractions, userID)
		}
	}

	for userID, limitUntil := range ech.rateLimitedUsers {
		if now.After(limitUntil) {
			delete(ech.rateLimitedUsers, userID)
		}
	}

	for identifier := range ech.temporaryFailures {
		delete(ech.temporaryFailures, identifier)
	}

	ech.lastCleanup = now
}

func HandleDuplicateInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	userID, err := GetUserID(i)
	if err != nil {
		return false
	}

	edgeHandler := GetEdgeCaseHandler()
	if !edgeHandler.StartInteraction(userID) {
		logger.Log.WithField("userID", userID).Debug("Duplicate interaction blocked")
		if err := RespondWithPreference(s, i, "Please wait before trying again.", nil, true); err != nil {
			logger.Log.WithError(err).Error("Failed to respond to duplicate interaction")
		}
		return true
	}

	go func() {
		time.Sleep(2 * time.Second)
		edgeHandler.EndInteraction(userID)
	}()

	return false
}

func HandleUserRateLimit(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	userID, err := GetUserID(i)
	if err != nil {
		return false
	}

	edgeHandler := GetEdgeCaseHandler()
	if edgeHandler.IsRateLimited(userID) {
		logger.Log.WithField("userID", userID).Debug("User is rate limited")
		if err := RespondWithPreference(s, i, "You are being rate limited. Please try again later.", nil, true); err != nil {
			logger.Log.WithError(err).Error("Failed to respond to rate limited user")
		}
		return true
	}

	return false
}

func ValidateAndSanitizeInput(input string, maxLength int) string {
	if input == "" {
		return ""
	}

	sanitized := utils.NormalizeInput(input)
	if len(sanitized) > maxLength {
		sanitized = utils.TruncateString(sanitized, maxLength)
	}

	return sanitized
}

func HandleDatabaseConnectionLoss() {
	logger.Log.Error("Database connection lost, attempting to reconnect")

	for attempts := 0; attempts < 5; attempts++ {
		if err := database.CheckConnection(); err == nil {
			logger.Log.Info("Database connection restored")
			return
		}

		backoff := time.Duration(1<<attempts) * time.Second
		logger.Log.Warnf("Database reconnection attempt %d failed, retrying in %v", attempts+1, backoff)
		time.Sleep(backoff)
	}

	logger.Log.Fatal("Failed to restore database connection after 5 attempts")
}

func HandleDiscordAPIError(err error, operation string) {
	if err == nil {
		return
	}

	logger.Log.WithError(err).WithField("operation", operation).Error("Discord API error")

	edgeHandler := GetEdgeCaseHandler()
	if edgeHandler.RecordTemporaryFailure("discord_api") {
		logger.Log.Warn("Multiple Discord API failures detected, implementing backoff")
		time.Sleep(30 * time.Second)
		edgeHandler.ResetTemporaryFailures("discord_api")
	}
}

func ValidateAccountLimits(userID string) error {
	cfg := configuration.Get()

	var accountCount int64
	if err := database.DB.Model(&models.Account{}).Where("user_id = ?", userID).Count(&accountCount).Error; err != nil {
		return err
	}

	userSettings, err := GetUserSettings(userID)
	if err != nil {
		return err
	}

	maxAccounts := cfg.RateLimits.DefaultMaxAccounts
	if userSettings.CapSolverAPIKey != "" || userSettings.EZCaptchaAPIKey != "" || userSettings.TwoCaptchaAPIKey != "" {
		maxAccounts = cfg.RateLimits.PremiumMaxAccounts
	}

	if int(accountCount) >= maxAccounts {
		return utils.ValidationError{Field: "account_limit", Message: fmt.Sprintf("You have reached the maximum limit of %d accounts", maxAccounts)}
	}

	return nil
}

func HandleShardFailover(s *discordgo.Session) {
	logger.Log.Info("Handling shard failover")

	shardMgr := GetAppShardManager()
	if !shardMgr.Initialized {
		logger.Log.Error("Shard manager not initialized during failover")
		return
	}

	for attempts := 0; attempts < 3; attempts++ {
		if err := shardMgr.Initialize(); err == nil {
			logger.Log.Info("Shard failover completed successfully")
			return
		}

		logger.Log.Warnf("Shard failover attempt %d failed, retrying", attempts+1)
		time.Sleep(time.Duration(attempts+1) * 5 * time.Second)
	}

	logger.Log.Error("Shard failover failed after 3 attempts")
}

func StartEdgeCaseCleanupRoutine(ctx context.Context) {
	edgeHandler := GetEdgeCaseHandler()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			edgeHandler.Cleanup()
		}
	}
}

func ValidateComponentsForDiscord(components []discordgo.MessageComponent) []discordgo.MessageComponent {
	cfg := configuration.Get()

	if len(components) == 0 {
		return components
	}

	maxRows := cfg.Message.MaxComponentRows
	if len(components) > maxRows {
		logger.Log.Warnf("Too many component rows (%d), truncating to %d", len(components), maxRows)
		return components[:maxRows]
	}

	return components
}

func HandleGracefulShutdown(s *discordgo.Session) {
	logger.Log.Info("Starting graceful shutdown")

	if s != nil {
		if err := s.UpdateWatchStatus(0, "Shutting down..."); err != nil {
			logger.Log.WithError(err).Error("Failed to update status during shutdown")
		}

		time.Sleep(2 * time.Second)

		if err := s.Close(); err != nil {
			logger.Log.WithError(err).Error("Error closing Discord session")
		}
	}

	if err := database.CloseConnection(); err != nil {
		logger.Log.WithError(err).Error("Error closing database connection")
	}

	logger.Log.Info("Graceful shutdown completed")
}

func ValidateRuntimeHealth() error {
	if err := database.CheckConnection(); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	cfg := configuration.Get()
	if cfg.Discord.Token == "" {
		return fmt.Errorf("discord token is not configured")
	}

	return nil
}
