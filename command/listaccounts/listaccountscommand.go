package listaccounts

import (
	"fmt"
	"time"

	"CODStatusBot/database"
	"CODStatusBot/logger"
	"CODStatusBot/models"
	"CODStatusBot/services"

	"github.com/bwmarrin/discordgo"
)

func CommandListAccounts(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var userID string
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	} else {
		logger.Log.Error("Interaction doesn't have Member or User")
		respondToInteraction(s, i, "An error occurred while processing your request.")
		return
	}

	var accounts []models.Account
	result := database.DB.Where("user_id = ?", userID).Find(&accounts)

	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error fetching user accounts")
		respondToInteraction(s, i, "Error fetching your accounts. Please try again.")
		return
	}

	if len(accounts) == 0 {
		respondToInteraction(s, i, "You don't have any monitored accounts.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Your Monitored Accounts",
		Description: "Here's a detailed list of all your monitored accounts:",
		Color:       0x00ff00,
		Fields:      make([]*discordgo.MessageEmbedField, 0),
	}

	for _, account := range accounts {
		checkStatus := getCheckStatus(account.IsCheckDisabled)
		cookieExpiration := services.FormatExpirationTime(account.SSOCookieExpiration)
		creationDate := time.Unix(account.Created, 0).Format("2006-01-02")
		lastCheckTime := time.Unix(account.LastCheck, 0).Format("2006-01-02 15:04:05")

		fieldValue := fmt.Sprintf("Status: %s\n"+
			"Checks: %s\n"+
			"Notification Type: %s\n"+
			"Cookie Expires: %s\n"+
			"Created: %s\n"+
			"Last Checked: %s",
			account.LastStatus, checkStatus, account.NotificationType,
			cookieExpiration, creationDate, lastCheckTime)

		if account.IsCheckDisabled {
			fieldValue += fmt.Sprintf("\nDisabled Reason: %s", account.DisabledReason)
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s %s", account.Title, getDisabledEmoji(account.IsCheckDisabled)),
			Value:  fieldValue,
			Inline: false,
		})

		embedColor := services.GetColorForStatus(account.LastStatus, account.IsExpiredCookie, account.IsCheckDisabled)
		if embedColor != 0x00ff00 {
			embed.Color = embedColor
		}
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		logger.Log.WithError(err).Error("Error responding to interaction")
	}
}

func getCheckStatus(isDisabled bool) string {
	if isDisabled {
		return "Disabled"
	}
	return "Enabled"
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

func getBalanceInfo(userID string) string {
	apiKey, _, err := services.GetUserCaptchaKey(userID)
	if err != nil {
		logger.Log.WithError(err).Error("Error getting user captcha key")
		return ""
	}

	if apiKey != "" {
		provider := "ezcaptcha" // Default provider
		userSettings, err := services.GetUserSettings(userID)
		if err == nil && userSettings.PreferredCaptchaProvider != "" {
			provider = userSettings.PreferredCaptchaProvider
		}
		_, balance, err := services.ValidateCaptchaKey(apiKey, provider)
		if err != nil {
			logger.Log.WithError(err).Error("Error validating captcha key")
			return ""
		}
		return fmt.Sprintf("\nYour current %s balance: %.2f points", provider, balance)
	}

	return ""
}

func getDisabledEmoji(isDisabled bool) string {
	if isDisabled {
		return "🚫"
	}
	return "✅"
}
