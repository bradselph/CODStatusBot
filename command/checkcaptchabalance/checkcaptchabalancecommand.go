package checkcaptchabalance

import (
	"fmt"
	"strings"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/services"
	"github.com/bwmarrin/discordgo"
)

func CommandCheckCaptchaBalance(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	userSettings, err := services.GetUserSettings(userID)
	if err != nil {
		logger.Log.WithError(err).Error("Error fetching user settings")
		respondToInteraction(s, i, "Error fetching your settings. Please try again.")
		return
	}

	cfg := configuration.Get()

	// Initial check for any enabled services
	if !services.IsServiceEnabled("capsolver") &&
		!services.IsServiceEnabled("ezcaptcha") &&
		!services.IsServiceEnabled("2captcha") {
		respondToInteraction(s, i, "No captcha services are currently available. Please try again later.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:     "Captcha Service Status",
		Color:     0x00ff00,
		Timestamp: time.Now().Format(time.RFC3339),
		Fields:    []*discordgo.MessageEmbedField{},
	}

	var availableServices []string
	if services.IsServiceEnabled("capsolver") {
		availableServices = append(availableServices, "Capsolver")
	}
	if services.IsServiceEnabled("ezcaptcha") {
		availableServices = append(availableServices, "EZCaptcha")
	}
	if services.IsServiceEnabled("2captcha") {
		availableServices = append(availableServices, "2Captcha")
	}

	if !services.IsServiceEnabled(userSettings.PreferredCaptchaProvider) {
		embed.Description = fmt.Sprintf("Warning: Your preferred service (%s) is currently disabled.\n"+
			"Available services: %s\n"+
			"Use /setcaptchaservice to switch to an available service.",
			userSettings.PreferredCaptchaProvider,
			strings.Join(availableServices, ", "))
		embed.Color = 0xFFA500
	}

	hasUserKey := false

	if userSettings.CapSolverAPIKey != "" && services.IsServiceEnabled("capsolver") {
		isValid, balance, err := services.ValidateCaptchaKey(userSettings.CapSolverAPIKey, "capsolver")
		if err == nil && isValid {
			hasUserKey = true
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Capsolver Balance",
				Value:  fmt.Sprintf("$%.3f", balance),
				Inline: true,
			})
		}
	}

	if userSettings.EZCaptchaAPIKey != "" && services.IsServiceEnabled("ezcaptcha") {
		isValid, balance, err := services.ValidateCaptchaKey(userSettings.EZCaptchaAPIKey, "ezcaptcha")
		if err == nil && isValid {
			hasUserKey = true
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "EZCaptcha Balance",
				Value:  fmt.Sprintf("%.0f points", balance),
				Inline: true,
			})
		}
	}

	if userSettings.TwoCaptchaAPIKey != "" && services.IsServiceEnabled("2captcha") {
		isValid, balance, err := services.ValidateCaptchaKey(userSettings.TwoCaptchaAPIKey, "2captcha")
		if err == nil && isValid {
			hasUserKey = true
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "2Captcha Balance",
				Value:  fmt.Sprintf("$%.2f", balance),
				Inline: true,
			})
		}
	}

	if !hasUserKey {
		embed.Description = "You are currently using the bot's default API key. Consider setting up your own key using /setcaptchaservice for unlimited checks."
		embed.Color = 0xFFA500

		if services.IsServiceEnabled(userSettings.PreferredCaptchaProvider) {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Default Service",
				Value:  userSettings.PreferredCaptchaProvider,
				Inline: true,
			})
		}
	} else {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Preferred Provider",
			Value:  userSettings.PreferredCaptchaProvider,
			Inline: false,
		})
	}

	var thresholds []string
	if services.IsServiceEnabled("capsolver") {
		thresholds = append(thresholds, fmt.Sprintf("Capsolver: $%.3f", cfg.CaptchaService.Capsolver.BalanceMin))
	}
	if services.IsServiceEnabled("ezcaptcha") {
		thresholds = append(thresholds, fmt.Sprintf("EZCaptcha: %.0f points", cfg.CaptchaService.EZCaptcha.BalanceMin))
	}
	if services.IsServiceEnabled("2captcha") {
		thresholds = append(thresholds, fmt.Sprintf("2Captcha: $%.2f", cfg.CaptchaService.TwoCaptcha.BalanceMin))
	}

	if len(thresholds) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Minimum Balance Thresholds",
			Value:  strings.Join(thresholds, "\n"),
			Inline: false,
		})
	}

	err = services.RespondWithPreference(s, i, "", []*discordgo.MessageEmbed{embed}, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error responding to interaction")
	}
}

func respondToInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := services.RespondWithPreference(s, i, message, nil, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error responding to interaction")
	}
}
