package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
)

var (
	rateLimitMutex sync.RWMutex
	rateLimitCache = make(map[string]*models.UserSettings)
)

/*
	func validateRateLimit(userID string, action string, limit time.Duration) bool {
		userSettings, err := GetUserSettings(userID)
		if err != nil {
			logger.Log.WithError(err).Error("Failed to get user settings for rate limit check")
			return false
		}

		userSettings.EnsureMapsInitialized()

		lastAction, exists := userSettings.LastActionTimes[action]
		if !exists || time.Since(lastAction) >= limit {
			userSettings.LastActionTimes[action] = time.Now()
			if err := database.DB.Save(&userSettings).Error; err != nil {
				logger.Log.WithError(err).Error("Failed to update rate limit timestamp")
				return false
			}
			return true
		}

		return false
	}
*/
func CheckRateLimitWithBackoff(userID string, endpoint string) error {
	userSettings, err := GetUserSettings(userID)
	if err != nil {
		return fmt.Errorf("failed to get user settings: %w", err)
	}

	userSettings.EnsureMapsInitialized()

	if lastHit, exists := userSettings.LastRateLimitHit[endpoint]; exists {
		backoffMultiplier := userSettings.RateLimitBackoff[endpoint]
		if backoffMultiplier == 0 {
			backoffMultiplier = 1
		}

		backoffDuration := time.Duration(backoffMultiplier) * time.Minute

		if time.Since(lastHit) < backoffDuration {
			remainingTime := backoffDuration - time.Since(lastHit)
			return fmt.Errorf("rate limited - please wait %v before trying again", remainingTime.Round(time.Second))
		}
	}

	return nil
}

func UpdateRateLimitBackoff(userID string, endpoint string) error {
	userSettings, err := GetUserSettings(userID)
	if err != nil {
		return fmt.Errorf("failed to get user settings: %w", err)
	}

	userSettings.EnsureMapsInitialized()

	userSettings.LastRateLimitHit[endpoint] = time.Now()

	currentBackoff := userSettings.RateLimitBackoff[endpoint]
	if currentBackoff == 0 {
		currentBackoff = 1
	} else if currentBackoff < 32 {
		currentBackoff *= 2
	}

	userSettings.RateLimitBackoff[endpoint] = currentBackoff

	logger.Log.Infof("Updated rate limit backoff for user %s endpoint %s to %d minutes",
		userID, endpoint, currentBackoff)

	if err := database.DB.Save(&userSettings).Error; err != nil {
		return fmt.Errorf("failed to update rate limit backoff: %w", err)
	}

	return nil
}

func ResetRateLimitBackoff(userID string, endpoint string) error {
	userSettings, err := GetUserSettings(userID)
	if err != nil {
		return fmt.Errorf("failed to get user settings: %w", err)
	}

	userSettings.EnsureMapsInitialized()

	delete(userSettings.RateLimitBackoff, endpoint)
	delete(userSettings.LastRateLimitHit, endpoint)

	logger.Log.Infof("Reset rate limit backoff for user %s endpoint %s", userID, endpoint)

	if err := database.DB.Save(&userSettings).Error; err != nil {
		return fmt.Errorf("failed to reset rate limit backoff: %w", err)
	}

	return nil
}

func GetRateLimitStatus(userID string, action string) (remaining time.Duration, isLimited bool) {
	cfg := configuration.Get()
	userSettings, err := GetUserSettings(userID)
	if err != nil {
		return 0, true
	}

	userSettings.EnsureMapsInitialized()

	var limit time.Duration
	switch action {
	case "check_now":
		limit = cfg.RateLimits.CheckNow
	case "add_account":
		limit = cfg.RateLimits.Default
	default:
		limit = cfg.RateLimits.Default
	}

	lastAction, exists := userSettings.LastActionTimes[action]
	if !exists {
		return 0, false
	}

	elapsed := time.Since(lastAction)
	if elapsed >= limit {
		return 0, false
	}

	return limit - elapsed, true
}

func CleanupOldRateLimitData() {
	logger.Log.Info("Starting rate limit data cleanup")

	rateLimitMutex.Lock()
	defer rateLimitMutex.Unlock()

	rateLimitCache = make(map[string]*models.UserSettings)

	cutoffTime := time.Now().Add(-24 * time.Hour)

	var users []models.UserSettings
	if err := database.DB.Find(&users).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to fetch users for rate limit cleanup")
		return
	}

	cleanedCount := 0
	for _, user := range users {
		user.EnsureMapsInitialized()
		updated := false

		for endpoint, lastHit := range user.LastRateLimitHit {
			if lastHit.Before(cutoffTime) {
				delete(user.LastRateLimitHit, endpoint)
				delete(user.RateLimitBackoff, endpoint)
				updated = true
			}
		}

		for action, lastTime := range user.LastActionTimes {
			if lastTime.Before(cutoffTime) {
				delete(user.LastActionTimes, action)
				updated = true
			}
		}

		if updated {
			if err := database.DB.Save(&user).Error; err != nil {
				logger.Log.WithError(err).Errorf("Failed to clean rate limit data for user %s", user.UserID)
			} else {
				cleanedCount++
			}
		}
	}

	logger.Log.Infof("Cleaned rate limit data for %d users", cleanedCount)
}
