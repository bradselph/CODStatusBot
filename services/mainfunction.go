package services

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"codstatusbot/database"
	"codstatusbot/logger"
	"codstatusbot/models"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var notificationInterval string

func init() {
	err := godotenv.Load()
	if err != nil {
		logger.Log.WithError(err).Error("Failed to load .env file")
	}
	notificationInterval = os.Getenv("NOTIFICATION_INTERVAL")
}

// SendDailyUpdate sends a daily update to the Discord channel associated with the given account.
func SendDailyUpdate(account models.Account, discord *discordgo.Session) {
	var description string
	if account.IsExpiredCookie {
		description = fmt.Sprintf("The SSO cookie for account %s has expired. Please update the cookie using the /updateaccount command or delete the account using the /removeaccount command.", account.Title)
	} else {
		description = fmt.Sprintf("The last status of account named %s was %s.", account.Title, account.LastStatus)
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s Hour Update - %s", notificationInterval, account.Title),
		Description: description,
		Color:       GetColorForStatus(account.LastStatus, account.IsExpiredCookie),
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	_, err := discord.ChannelMessageSendComplex(account.ChannelID, &discordgo.MessageSend{
		Embed: embed,
	})
	if err != nil {
		logger.Log.WithError(err).Error("Failed to send daily update message for account named", account.Title)
	}

	account.LastCheck = time.Now().Unix()
	account.LastNotification = time.Now().Unix()
	if err := database.DB.Save(&account).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to save account changes for account named", account.Title)
	}
}

// CheckAccounts periodically checks all accounts in the database and sends notifications if necessary.
// It also updates the LastCheck and LastNotification fields for each account.
func CheckAccounts(s *discordgo.Session) {
	for {
		logger.Log.Info("Starting periodic account check")

		var accounts []models.Account
		if err := database.DB.Find(&accounts).Error; err != nil {
			logger.Log.WithError(err).Error("Failed to fetch accounts from the database")
			continue
		}

		for _, account := range accounts {
			var lastCheck time.Time
			if account.LastCheck != 0 {
				lastCheck = time.Unix(account.LastCheck, 0)
			}
			var lastNotification time.Time
			if account.LastNotification != 0 {
				lastNotification = time.Unix(account.LastNotification, 0)
			}

			if account.IsExpiredCookie {
				logger.Log.WithField("account", account.Title).Info("Skipping account with expired cookie")
				notificationInterval, _ := strconv.ParseFloat(os.Getenv("NOTIFICATION_INTERVAL"), 64)
				if time.Since(lastNotification).Hours() > notificationInterval {
					go SendDailyUpdate(account, s)
				} else {
					logger.Log.WithField("account", account.Title).Info("Owner of account named", account.Title, "recently notified within 24 hours, skipping")
				}
				continue
			}

			// Check if account ha been checked in the last 15 minutes.
			checkInterval, _ := strconv.ParseFloat(os.Getenv("CHECK_INTERVAL"), 64)
			if time.Since(lastCheck).Minutes() > checkInterval {
				go CheckSingleAccount(account, s)
			} else {
				logger.Log.WithField("account", account.Title).Info("Account named", account.Title, "checked recently, skipping")
			}

			// Send a daily update if the account hasn't been notified in the last 24 hours.
			notificationInterval, _ := strconv.ParseFloat(os.Getenv("NOTIFICATION_INTERVAL"), 64)
			if time.Since(lastNotification).Hours() > notificationInterval {
				go SendDailyUpdate(account, s)
			} else {
				logger.Log.WithField("account", account.Title).Info("Owner of Account Named", account.Title, "recently notified within 24Hours already, skipping")
			}
		}

		// Wait for SLEEP_DURATION in minutes before checking all accounts again.
		sleepDuration, _ := strconv.Atoi(os.Getenv("SLEEP_DURATION"))
		time.Sleep(time.Duration(sleepDuration) * time.Minute)
	}
}

// CheckSingleAccount checks the status of a single account and sends a notification
// if the status has changed. It also updates the account's LastCheck and IsExpiredCookie fields in the database.
func CheckSingleAccount(account models.Account, discord *discordgo.Session) {
	result, err := CheckAccount(account.SSOCookie)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to check account named ", account.Title, "possible expired SSO Cookie")
		return
	}

	// If the account has an invalid cookie, send a notification and update the
	// account's LastCookieNotification and IsExpiredCookie fields in the database.
	if result == models.StatusInvalidCookie {
		cooldownDuration, _ := strconv.ParseFloat(os.Getenv("COOLDOWN_DURATION"), 64)
		lastNotification := time.Unix(account.LastCookieNotification, 0)
		if time.Since(lastNotification) >= time.Duration(cooldownDuration)*time.Hour || account.LastCookieNotification == 0 {
			logger.Log.Infof("Account named %s has an invalid SSO cookie", account.Title)
			embed := &discordgo.MessageEmbed{
				Title:       fmt.Sprintf("%s - Invalid SSO Cookie", account.Title),
				Description: fmt.Sprintf("The SSO cookie for account %s has expired. Please update the cookie using the /updateaccount command or delete the account using the /deleteaccount command.", account.Title),
				Color:       0xff0000,
				Timestamp:   time.Now().Format(time.RFC3339),
			}
			_, err = discord.ChannelMessageSendComplex(account.ChannelID, &discordgo.MessageSend{
				Embed: embed,
			})
			if err != nil {
				logger.Log.WithError(err).Error("Failed to send invalid cookie notification for account named", account.Title)
			}

			account.LastCookieNotification = time.Now().Unix() // Store the current time as the last notification time
			account.IsExpiredCookie = true                     // Mark the account as having an expired cookie
			if err := database.DB.Save(&account).Error; err != nil {
				logger.Log.WithError(err).Error("Failed to save account changes for account named", account.Title)
			}
		} else {
			logger.Log.Infof("Skipping expired cookie notification for account named %s (cooldown)", account.Title)
		}
		return
	}

	lastStatus := account.LastStatus

	account.LastCheck = time.Now().Unix()
	account.IsExpiredCookie = false // Reset the expired cookie status if the account is successfully checked
	if err := database.DB.Save(&account).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to save account changes for account named", account.Title)
		return
	}
	if result != lastStatus {
		account.LastStatus = result
		if err := database.DB.Save(&account).Error; err != nil {
			logger.Log.WithError(err).Error("Failed to save account changes for account named", account.Title)
			return
		}
		logger.Log.Infof("Account named %s status changed to %s", account.Title, result)
		ban := models.Ban{
			Account:   account,
			Status:    result,
			AccountID: account.ID,
		}
		if err := database.DB.Create(&ban).Error; err != nil {
			logger.Log.WithError(err).Error("Failed to create new ban record for account named", account.Title)
		}
		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s - %s", account.Title, EmbedTitleFromStatus(result)),
			Description: fmt.Sprintf("The status of account named %s has changed to %s <@%s>", account.Title, result, account.UserID),
			Color:       GetColorForStatus(result, account.IsExpiredCookie),
			Timestamp:   time.Now().Format(time.RFC3339),
		}
		_, err = discord.ChannelMessageSendComplex(account.ChannelID, &discordgo.MessageSend{
			Embed:   embed,
			Content: fmt.Sprintf("<@%s>", account.UserID),
		})
		if err != nil {
			logger.Log.WithError(err).Error("Failed to send status update message for account named", account.Title)
		}
	}
}

// GetColorForStatus returns a color code based on the account's status or cookie validity.
func GetColorForStatus(status models.Status, isExpiredCookie bool) int {
	if isExpiredCookie {
		return 0xff0000 // Red for expired cookie
	}
	switch status {
	case models.StatusPermaban:
		return 0xff0000 // Red for permanent ban
	case models.StatusShadowban:
		return 0xffff00 // Yellow for shadowban
	default:
		return 0x00ff00 // Green for no ban
	}
}

// EmbedTitleFromStatus returns a string title based on the ban status.
func EmbedTitleFromStatus(status models.Status) string {
	switch status {
	case models.StatusPermaban:
		return "PERMANENT BAN DETECTED"
	case models.StatusShadowban:
		return "SHADOWBAN DETECTED"
	default:
		return "ACCOUNT NOT BANNED"
	}
}