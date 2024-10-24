package checknow

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bradselph/CODStatusBot/services"

	"github.com/bwmarrin/discordgo"
)

var (
	rateLimiter     = make(map[string]time.Time)
	rateLimiterLock sync.Mutex
	rateLimit       time.Duration
)

func init() {
	rateLimitStr := os.Getenv("CHECK_NOW_RATE_LIMIT")
	rateLimitSeconds, err := strconv.Atoi(rateLimitStr)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to parse CHECK_NOW_RATE_LIMIT, using default of 3600 seconds")
		rateLimitSeconds = 3600
	}
	rateLimit = time.Duration(rateLimitSeconds) * time.Second
}

func CommandCheckNow(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID, err := getUserID(i)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to get user ID")
		respondToInteraction(s, i, "An error occurred while processing your request.")
		return
	}

	userSettings, err := services.GetUserSettings(userID)
	if err != nil {
		logger.Log.WithError(err).Error("Error fetching user settings")
		respondToInteraction(s, i, "Error fetching user settings. Please try again later.")
		return
	}

	if userSettings.EZCaptchaAPIKey == "" && userSettings.TwoCaptchaAPIKey == "" {
		if !checkRateLimit(userID) {
			respondToInteraction(s, i, fmt.Sprintf("You're using the bot's default API key and are rate limited. Please wait %v before trying again, or set up your own API key using /setcaptchaservice for unlimited checks.", rateLimit))
			return
		}
	}

	if userSettings.EZCaptchaAPIKey != "" || userSettings.TwoCaptchaAPIKey != "" {
		_, balance, err := services.GetUserCaptchaKey(userID)
		if err != nil {
			logger.Log.WithError(err).Error("Error getting captcha key")
			respondToInteraction(s, i, "Error validating your captcha API key. Please check your key using /setcaptchaservice.")
			return
		}

		if balance < 0 {
			respondToInteraction(s, i, fmt.Sprintf("Your captcha balance (%.2f) is too low for checking accounts. Please recharge your balance.", balance))
			return
		}
	}

	var accounts []models.Account
	query := database.DB.Where("user_id = ?", userID)
	result := query.Find(&accounts)

	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error fetching accounts")
		respondToInteraction(s, i, "Error fetching accounts. Please try again later.")
		return
	}

	if len(accounts) == 0 {
		respondToInteraction(s, i, "You don't have any monitored accounts.")
		return
	}

	showAccountButtons(s, i, accounts)
}

func showAccountButtons(s *discordgo.Session, i *discordgo.InteractionCreate, accounts []models.Account) {
	userID, err := getUserID(i)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to get user ID")
		respondToInteraction(s, i, "An error occurred while processing your request.")
		return
	}

	var components []discordgo.MessageComponent
	var currentRow []discordgo.MessageComponent

	for _, account := range accounts {
		currentRow = append(currentRow, discordgo.Button{
			Label:    account.Title,
			Style:    discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("check_now_%s_%d", userID, account.ID),
		})

		if len(currentRow) == 5 {
			components = append(components, discordgo.ActionsRow{Components: currentRow})
			currentRow = []discordgo.MessageComponent{}
		}
	}

	if len(currentRow) < 5 {
		currentRow = append(currentRow, discordgo.Button{
			Label:    "Check All",
			Style:    discordgo.SuccessButton,
			CustomID: fmt.Sprintf("check_now_%s_all", userID),
		})
	}

	if len(currentRow) > 0 {
		components = append(components, discordgo.ActionsRow{Components: currentRow})
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    "Select an account to check, or 'Check All' to check all accounts:",
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: components,
		},
	})
	if err != nil {
		logger.Log.WithError(err).Error("Error responding with account selection")
	}
}

func HandleAccountSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts := strings.Split(customID, "_")

	if len(parts) != 4 {
		logger.Log.Error("Invalid custom ID format")
		respondToInteraction(s, i, "An error occurred while processing your request.")
		return
	}

	userID := parts[2]
	accountIDOrAll := parts[3]

	userSettings, err := services.GetUserSettings(userID)
	if err != nil {
		logger.Log.WithError(err).Error("Error fetching user settings")
		respondToInteraction(s, i, "Error fetching settings. Please try again.")
		return
	}

	if userSettings.EZCaptchaAPIKey != "" || userSettings.TwoCaptchaAPIKey != "" {
		apiKey, balance, err := services.GetUserCaptchaKey(userID)
		if err != nil || apiKey == "" {
			logger.Log.WithError(err).Error("Error getting captcha key")
			respondToInteraction(s, i, "Error validating your captcha API key. Please check your key using /setcaptchaservice.")
			return
		}

		if balance < 0 {
			respondToInteraction(s, i, fmt.Sprintf("Your captcha balance (%.2f) is too low for checking accounts. Please recharge your balance.", balance))
			return
		}
	}

	var accounts []models.Account
	if accountIDOrAll == "all" {
		result := database.DB.Where("user_id = ?", userID).Find(&accounts)
		if result.Error != nil {
			logger.Log.WithError(result.Error).Error("Error fetching accounts")
			respondToInteraction(s, i, "Error fetching accounts. Please try again later.")
			return
		}
	} else {
		accountID, err := strconv.Atoi(accountIDOrAll)
		if err != nil {
			logger.Log.WithError(err).Error("Error parsing account ID")
			respondToInteraction(s, i, "Error processing your selection. Please try again.")
			return
		}

		var account models.Account
		result := database.DB.First(&account, accountID)
		if result.Error != nil {
			logger.Log.WithError(result.Error).Error("Error fetching account")
			respondToInteraction(s, i, "Error: Account not found or you don't have permission to check it.")
			return
		}

		accounts = append(accounts, account)
	}

	checkAccounts(s, i, accounts)
}

func checkAccounts(s *discordgo.Session, i *discordgo.InteractionCreate, accounts []models.Account) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		logger.Log.WithError(err).Error("Failed to defer interaction response")
		return
	}

	var embeds []*discordgo.MessageEmbed

	for _, account := range accounts {
		if account.IsCheckDisabled {
			embed := &discordgo.MessageEmbed{
				Title:       fmt.Sprintf("%s - Checks Disabled", account.Title),
				Description: fmt.Sprintf("Checks are disabled for this account. Reason: %s", account.DisabledReason),
				Color:       services.GetColorForStatus(account.LastStatus, account.IsExpiredCookie, true),
				Timestamp:   time.Now().Format(time.RFC3339),
			}
			embeds = append(embeds, embed)
			continue
		}

		if account.IsExpiredCookie {
			embed := &discordgo.MessageEmbed{
				Title:       fmt.Sprintf("%s - Expired Cookie", account.Title),
				Description: "The SSO cookie for this account has expired. Please update it using the /updateaccount command.",
				Color:       services.GetColorForStatus(models.StatusUnknown, true, false),
				Timestamp:   time.Now().Format(time.RFC3339),
			}
			embeds = append(embeds, embed)
			continue
		}

		status, err := services.CheckAccount(account.SSOCookie, account.UserID, "")
		if err != nil {
			logger.Log.WithError(err).Errorf("Error checking account %s", account.Title)
			description := "An error occurred while checking this account. "
			if strings.Contains(err.Error(), "insufficient balance") {
				description += "Your captcha balance is too low. Please recharge your balance."
			} else if strings.Contains(err.Error(), "invalid captcha") {
				description += "There was an issue with your captcha API key. Please verify it using /setcaptchaservice."
			} else {
				description += "Please try again later."
			}

			embed := &discordgo.MessageEmbed{
				Title:       fmt.Sprintf("%s - Error", account.Title),
				Description: description,
				Color:       0xFF0000,
				Timestamp:   time.Now().Format(time.RFC3339),
			}
			embeds = append(embeds, embed)
			continue
		}

		account.LastStatus = status
		account.LastCheck = time.Now().Unix()
		if err := database.DB.Save(&account).Error; err != nil {
			logger.Log.WithError(err).Errorf("Failed to update account %s after check", account.Title)
		}

		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s - Status Check", account.Title),
			Description: fmt.Sprintf("Current status: %s", status),
			Color:       services.GetColorForStatus(status, account.IsExpiredCookie, account.IsCheckDisabled),
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Last Checked",
					Value:  time.Now().Format(time.RFC1123),
					Inline: true,
				},
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}
		embeds = append(embeds, embed)
	}

	for j := 0; j < len(embeds); j += 10 {
		end := j + 10
		if end > len(embeds) {
			end = len(embeds)
		}

		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: embeds[j:end],
			Flags:  discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			logger.Log.WithError(err).Error("Failed to send follow-up message")
		}
	}
}

func checkRateLimit(userID string) bool {
	rateLimiterLock.Lock()
	defer rateLimiterLock.Unlock()

	lastUse, exists := rateLimiter[userID]
	if !exists || time.Since(lastUse) >= rateLimit {
		rateLimiter[userID] = time.Now()
		return true
	}
	return false
}

func getUserID(i *discordgo.InteractionCreate) (string, error) {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID, nil
	}
	if i.User != nil {
		return i.User.ID, nil
	}
	return "", fmt.Errorf("unable to determine user ID")
}

func respondToInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		logger.Log.WithError(err).Error("Error responding to interaction")
	}
}
