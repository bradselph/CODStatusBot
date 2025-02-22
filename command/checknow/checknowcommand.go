package checknow

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
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
	cfg := configuration.Get()
	rateLimit = cfg.RateLimits.CheckNow
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

	if !services.IsServiceEnabled(userSettings.PreferredCaptchaProvider) {
		msg := fmt.Sprintf("Your preferred captcha service (%s) is currently disabled. ", userSettings.PreferredCaptchaProvider)
		if services.IsServiceEnabled("ezcaptcha") {
			msg += "Please switch to EZCaptcha using /setcaptchaservice."
		} else if services.IsServiceEnabled("2captcha") {
			msg += "Please switch to 2Captcha using /setcaptchaservice."
		} else {
			msg += "No captcha services are currently available. Please try again later."
		}
		respondToInteraction(s, i, msg)
		return
	}

	if userSettings.CapSolverAPIKey != "" && userSettings.EZCaptchaAPIKey == "" && userSettings.TwoCaptchaAPIKey == "" {
		if !checkRateLimit(userID) {
			respondToInteraction(s, i, fmt.Sprintf("You're using the bot's default API key and are rate limited. Please wait %v before trying again, or set up your own API key using /setcaptchaservice for unlimited checks.", rateLimit))
			return
		}
	}

	if userSettings.CapSolverAPIKey != "" || userSettings.EZCaptchaAPIKey != "" || userSettings.TwoCaptchaAPIKey != "" {
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

	isUsingDefaultKey := userSettings.CapSolverAPIKey == "" &&
		userSettings.EZCaptchaAPIKey == "" &&
		userSettings.TwoCaptchaAPIKey == ""

	if isUsingDefaultKey {
		lastCheck := userSettings.LastCommandTimes["check_now"]
		cfg := configuration.Get()

		userSettings.EnsureMapsInitialized()
		checksUsed := userSettings.ActionCounts["check_now"]
		maxChecks := cfg.RateLimits.DefaultMaxAccounts

		if !lastCheck.IsZero() && time.Since(lastCheck) >= cfg.RateLimits.CheckNow {
			checksUsed = 0
			userSettings.ActionCounts["check_now"] = 0
			userSettings.LastCommandTimes["check_now"] = time.Now()
			if err := database.DB.Save(&userSettings).Error; err != nil {
				logger.Log.WithError(err).Error("Error resetting check count")
			}
		}

		if accountIDOrAll == "all" {
			var accountCount int64
			if err := database.DB.Model(&models.Account{}).Where("user_id = ?", userID).Count(&accountCount).Error; err != nil {
				logger.Log.WithError(err).Error("Error counting accounts")
				respondToInteraction(s, i, "Error counting accounts. Please try again.")
				return
			}

			if int(accountCount) > (maxChecks - checksUsed) {
				timeUntilNext := cfg.RateLimits.CheckNow - time.Since(lastCheck)
				embed := &discordgo.MessageEmbed{
					Title: "Insufficient Checks Available",
					Description: fmt.Sprintf("You need %d checks but only have %d remaining.\n\n"+
						"Next reset in: %s\n\n"+
						"To remove this limit, set up your own API key using `/setcaptchaservice`",
						accountCount, maxChecks-checksUsed, formatDuration(timeUntilNext)),
					Color: 0xFFA500,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Available Checks",
							Value:  fmt.Sprintf("%d/%d checks remaining", maxChecks-checksUsed, maxChecks),
							Inline: true,
						},
						{
							Name:   "Remove Limits",
							Value:  "Add your own API key to get:\n• Unlimited checks\n• Faster intervals\n• More account slots",
							Inline: true,
						},
					},
					Timestamp: time.Now().Format(time.RFC3339),
				}
				respondToInteractionWithEmbed(s, i, "", embed)
				return
			}

			userSettings.ActionCounts["check_now"] += int(accountCount)
		} else {
			if checksUsed >= maxChecks {
				timeUntilNext := cfg.RateLimits.CheckNow - time.Since(lastCheck)
				embed := &discordgo.MessageEmbed{
					Title: "Rate Limit Reached",
					Description: fmt.Sprintf("You have used all available checks.\n\n"+
						"Next reset in: %s\n\n"+
						"To remove this limit, set up your own API key using `/setcaptchaservice`",
						formatDuration(timeUntilNext)),
					Color: 0xFFA500,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Check Status",
							Value:  fmt.Sprintf("%d/%d checks used", checksUsed, maxChecks),
							Inline: true,
						},
						{
							Name:   "Remove Limits",
							Value:  "Add your own API key to get:\n• Unlimited checks\n• Faster intervals\n• More account slots",
							Inline: true,
						},
					},
					Timestamp: time.Now().Format(time.RFC3339),
				}
				respondToInteractionWithEmbed(s, i, "", embed)
				return
			}
			userSettings.ActionCounts["check_now"]++
		}

		if err := database.DB.Save(&userSettings).Error; err != nil {
			logger.Log.WithError(err).Error("Error saving check count")
			respondToInteraction(s, i, "Error updating check count. Please try again.")
			return
		}
	} else {
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

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func respondToInteractionWithEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, content string, embed *discordgo.MessageEmbed) {
	responseData := &discordgo.InteractionResponseData{
		Flags: discordgo.MessageFlagsEphemeral,
	}

	if content != "" {
		responseData.Content = content
	}
	if embed != nil {
		responseData.Embeds = []*discordgo.MessageEmbed{embed}
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: responseData,
	})
	if err != nil {
		logger.Log.WithError(err).Error("Error responding to interaction with embed")
	}
}

func checkAccounts(s *discordgo.Session, i *discordgo.InteractionCreate, accounts []models.Account) {
	userID, err := services.GetUserID(i)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to get user ID")
		respondToInteraction(s, i, "An error occurred while processing your request.")
		return
	}

	userSettings, err := services.GetUserSettings(userID)
	if err != nil {
		logger.Log.WithError(err).Error("Error fetching user settings")
		respondToInteraction(s, i, "Error fetching settings. Please try again.")
		return
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		logger.Log.WithError(err).Error("Failed to defer interaction response")
		return
	}

	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: fmt.Sprintf("Starting check of %d accounts...", len(accounts)),
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		logger.Log.WithError(err).Error("Failed to send initial status message")
	}

	processedCount := 0
	for _, account := range accounts {
		var embed *discordgo.MessageEmbed

		if account.IsCheckDisabled {
			embed = &discordgo.MessageEmbed{
				Title:       fmt.Sprintf("%s - Checks Disabled", account.Title),
				Description: fmt.Sprintf("Checks are disabled for this account. Reason: %s", account.DisabledReason),
				Color:       services.GetColorForStatus(account.LastStatus, account.IsExpiredCookie, true),
				Timestamp:   time.Now().Format(time.RFC3339),
			}
		} else if account.IsExpiredCookie {
			embed = &discordgo.MessageEmbed{
				Title:       fmt.Sprintf("%s - Expired Cookie", account.Title),
				Description: "The SSO cookie for this account has expired. Please update it using the /updateaccount command.",
				Color:       services.GetColorForStatus(models.StatusUnknown, true, false),
				Timestamp:   time.Now().Format(time.RFC3339),
			}
		} else {
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

				embed = &discordgo.MessageEmbed{
					Title:       fmt.Sprintf("%s - Error", account.Title),
					Description: description,
					Color:       0xFF0000,
					Timestamp:   time.Now().Format(time.RFC3339),
				}
			} else {
				services.HandleStatusChange(s, account, status, userSettings)

				embed = &discordgo.MessageEmbed{
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
			}
		}

		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			logger.Log.WithError(err).Error("Failed to send follow-up message")
		}

		processedCount++
		time.Sleep(time.Second)
	}

	completionMessage := fmt.Sprintf("Completed checking all %d accounts.", processedCount)
	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: completionMessage,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		logger.Log.WithError(err).Error("Failed to send completion message")
	}
}

func checkRateLimit(userID string) bool {
	var userSettings models.UserSettings
	if err := database.DB.Where("user_id = ?", userID).First(&userSettings).Error; err != nil {
		logger.Log.WithError(err).Error("Error fetching user settings")
		return false
	}

	userSettings.EnsureMapsInitialized()
	now := time.Now()
	lastCheckTime := userSettings.LastCommandTimes["check_now"]

	if lastCheckTime.IsZero() || time.Since(lastCheckTime) >= rateLimit {
		userSettings.ActionCounts["check_now"] = 0
		userSettings.LastCommandTimes["check_now"] = now
	}

	cfg := configuration.Get()
	maxChecks := cfg.RateLimits.DefaultMaxAccounts

	if userSettings.ActionCounts["check_now"] >= maxChecks {
		return false
	}

	userSettings.ActionCounts["check_now"]++

	if err := database.DB.Save(&userSettings).Error; err != nil {
		logger.Log.WithError(err).Error("Error saving user settings")
		return false
	}

	return true
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
