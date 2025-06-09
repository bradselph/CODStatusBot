package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bwmarrin/discordgo"
)

type ErrorHandler struct {
	mu           sync.RWMutex
	errorCounts  map[string]*ErrorStats
	lastCleanup  time.Time
	cleanupMutex sync.Mutex
}

type ErrorStats struct {
	Count             int
	LastError         time.Time
	LastNotified      time.Time
	ConsecutiveErrors int
	ErrorTypes        map[string]int
}

var globalErrorHandler = &ErrorHandler{
	errorCounts: make(map[string]*ErrorStats),
	lastCleanup: time.Now(),
}

func GetErrorHandler() *ErrorHandler {
	return globalErrorHandler
}

func (eh *ErrorHandler) RecordError(userID string, errorType string, err error) {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	stats, exists := eh.errorCounts[userID]
	if !exists {
		stats = &ErrorStats{
			ErrorTypes: make(map[string]int),
		}
		eh.errorCounts[userID] = stats
	}

	stats.Count++
	stats.ConsecutiveErrors++
	stats.LastError = time.Now()
	stats.ErrorTypes[errorType]++

	if time.Since(eh.lastCleanup) > 24*time.Hour {
		go eh.cleanupOldErrors()
	}
}

func (eh *ErrorHandler) ResetConsecutiveErrors(userID string) {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	if stats, exists := eh.errorCounts[userID]; exists {
		stats.ConsecutiveErrors = 0
	}
}

func (eh *ErrorHandler) GetErrorStats(userID string) *ErrorStats {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	if stats, exists := eh.errorCounts[userID]; exists {
		return stats
	}
	return nil
}

func (eh *ErrorHandler) ShouldNotifyUser(userID string, errorType string) bool {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	cfg := configuration.Get()
	stats, exists := eh.errorCounts[userID]
	if !exists {
		return false
	}

	if stats.ConsecutiveErrors < cfg.ErrorHandling.AccountErrorThreshold {
		return false
	}

	cooldown := time.Duration(cfg.ErrorHandling.ErrorNotificationCooldownHours) * time.Hour
	if time.Since(stats.LastNotified) < cooldown {
		return false
	}

	return true
}

func (eh *ErrorHandler) MarkNotified(userID string) {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	if stats, exists := eh.errorCounts[userID]; exists {
		stats.LastNotified = time.Now()
	}
}

func (eh *ErrorHandler) cleanupOldErrors() {
	eh.cleanupMutex.Lock()
	defer eh.cleanupMutex.Unlock()

	eh.mu.Lock()
	defer eh.mu.Unlock()

	cutoff := time.Now().Add(-7 * 24 * time.Hour)

	for userID, stats := range eh.errorCounts {
		if stats.LastError.Before(cutoff) {
			delete(eh.errorCounts, userID)
		}
	}

	eh.lastCleanup = time.Now()
	logger.Log.Debug("Cleaned up old error statistics")
}

func HandleAccountError(s *discordgo.Session, account models.Account, errorType string, err error) {
	errorHandler := GetErrorHandler()
	cfg := configuration.Get()

	errorHandler.RecordError(account.UserID, errorType, err)

	account.ConsecutiveErrors++
	account.LastErrorTime = time.Now()

	if account.ConsecutiveErrors >= cfg.ErrorHandling.MaxConsecutiveErrors {
		account.IsCheckDisabled = true
		account.DisabledReason = fmt.Sprintf("Too many consecutive %s errors (%d)", errorType, account.ConsecutiveErrors)

		embed := &discordgo.MessageEmbed{
			Title: "Account Automatically Disabled",
			Description: fmt.Sprintf("Account '%s' has been automatically disabled due to repeated errors.\n\n"+
				"**Error Type:** %s\n"+
				"**Consecutive Errors:** %d\n"+
				"**Reason:** %s\n\n"+
				"Please resolve the issue and use `/togglecheck` to re-enable monitoring.",
				account.Title, errorType, account.ConsecutiveErrors, account.DisabledReason),
			Color:     0xFF0000,
			Timestamp: time.Now().Format(time.RFC3339),
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Next Steps",
					Value:  "1. Fix the underlying issue (expired cookie, invalid credentials, etc.)\n2. Use `/updateaccount` if needed\n3. Use `/togglecheck` to re-enable",
					Inline: false,
				},
			},
		}

		if sendErr := SendNotification(s, account, embed, "", "account_auto_disabled"); sendErr != nil {
			logger.Log.WithError(sendErr).Error("Failed to send account auto-disabled notification")
		}
	}

	if dbErr := database.DB.Save(&account).Error; dbErr != nil {
		logger.Log.WithError(dbErr).Error("Failed to save account after error handling")
	}

	if errorHandler.ShouldNotifyUser(account.UserID, errorType) {
		notifyUserAboutErrors(s, account.UserID, errorType)
		errorHandler.MarkNotified(account.UserID)
	}

	logger.Log.WithError(err).WithField("userID", account.UserID).
		WithField("accountID", account.ID).
		WithField("accountTitle", account.Title).
		WithField("errorType", errorType).
		WithField("consecutiveErrors", account.ConsecutiveErrors).
		Error("Account check error handled")
}

func notifyUserAboutErrors(s *discordgo.Session, userID string, errorType string) {
	errorHandler := GetErrorHandler()
	stats := errorHandler.GetErrorStats(userID)
	if stats == nil {
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Repeated Account Check Errors",
		Description: fmt.Sprintf("Multiple accounts are experiencing %s errors. This may indicate a systemic issue that needs attention.", errorType),
		Color:       0xFFA500,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Error Statistics",
				Value:  fmt.Sprintf("**Total Errors:** %d\n**Consecutive Errors:** %d\n**Last Error:** %s", stats.Count, stats.ConsecutiveErrors, stats.LastError.Format("2006-01-02 15:04:05")),
				Inline: true,
			},
			{
				Name:   "Common Solutions",
				Value:  getErrorSolutions(errorType),
				Inline: false,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	var account models.Account
	if err := database.DB.Where("user_id = ?", userID).First(&account).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to find account for error notification")
		return
	}

	if err := SendNotification(s, account, embed, "", "repeated_errors"); err != nil {
		logger.Log.WithError(err).Error("Failed to send repeated errors notification")
	}
}

func getErrorSolutions(errorType string) string {
	switch errorType {
	case "cookie_expired":
		return "• Update SSO cookies using `/updateaccount`\n• Check cookie expiration dates\n• Ensure cookies are copied correctly"
	case "captcha_failed":
		return "• Check your captcha service balance\n• Verify API key is correct\n• Try switching captcha providers"
	case "rate_limited":
		return "• Reduce check frequency\n• Use your own API key for higher limits\n• Wait for rate limits to reset"
	case "network_error":
		return "• Check internet connection\n• Verify Activision servers are online\n• Try again in a few minutes"
	default:
		return "• Check account credentials\n• Update account information\n• Contact support if issues persist"
	}
}

func HandleSuccessfulCheck(account models.Account) {
	errorHandler := GetErrorHandler()

	if account.ConsecutiveErrors > 0 {
		account.ConsecutiveErrors = 0
		account.LastSuccessfulCheck = time.Now()

		if err := database.DB.Save(&account).Error; err != nil {
			logger.Log.WithError(err).Error("Failed to save account after successful check")
		}

		errorHandler.ResetConsecutiveErrors(account.UserID)

		logger.Log.WithField("userID", account.UserID).
			WithField("accountID", account.ID).
			WithField("accountTitle", account.Title).
			Debug("Reset consecutive errors after successful check")
	}
}

func StartErrorCleanupRoutine(ctx context.Context) {
	errorHandler := GetErrorHandler()
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			errorHandler.cleanupOldErrors()
		}
	}
}

func ValidateInteractionContext(i *discordgo.InteractionCreate) error {
	if i == nil {
		return errors.New("interaction is nil")
	}

	if i.Type == discordgo.InteractionApplicationCommand && i.ApplicationCommandData().Name == "" {
		return errors.New("command name is empty")
	}

	if i.Type == discordgo.InteractionMessageComponent && i.MessageComponentData().CustomID == "" {
		return errors.New("component custom ID is empty")
	}

	return nil
}

func SafeInteractionRespond(s *discordgo.Session, i *discordgo.InteractionCreate, response *discordgo.InteractionResponse) error {
	if err := ValidateInteractionContext(i); err != nil {
		return fmt.Errorf("invalid interaction context: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- s.InteractionRespond(i.Interaction, response)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("interaction response timed out: %w", ctx.Err())
	}
}

func RecoverFromPanic(operation string) {
	if r := recover(); r != nil {
		logger.Log.WithField("operation", operation).
			WithField("panic", r).
			Error("Recovered from panic")
	}
}
