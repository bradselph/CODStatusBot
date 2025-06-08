package setcaptchaservice

import (
	"fmt"
	"strings"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bradselph/CODStatusBot/services"
	"github.com/bradselph/CODStatusBot/utils"
	"github.com/bwmarrin/discordgo"
)

var providerLabels = map[string]string{
	"capsolver": "Capsolver",
	"ezcaptcha": "EZCaptcha",
	"2captcha":  "2Captcha",
}

func CommandSetCaptchaService(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cfg := configuration.Get()
	var components []discordgo.MessageComponent

	if cfg.CaptchaService.Capsolver.Enabled {
		components = append(components, createProviderButton("capsolver"))
	}

	if cfg.CaptchaService.EZCaptcha.Enabled {
		components = append(components, createProviderButton("ezcaptcha"))
	}

	if cfg.CaptchaService.TwoCaptcha.Enabled {
		components = append(components, createProviderButton("2captcha"))
	}

	components = append(components, services.CreateV2Button("Remove API Key", "set_captcha_remove", discordgo.DangerButton))

	components = append(components, services.CreateV2Button("Fallback Settings", "set_captcha_fallback", discordgo.SecondaryButton))

	if len(components) == 1 {
		respondToInteraction(s, i, "No captcha services are currently enabled. Please contact the bot administrator.")
		return
	}

	err := services.RespondWithPreferenceAndComponents(s, i, "Select a captcha service provider:", nil, components, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error responding with service selection")
	}
}

func HandleCaptchaServiceSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID

	if customID == "set_captcha_remove" {
		handleAPIKeyRemoval(s, i)
		return
	}

	if customID == "set_captcha_fallback" {
		showFallbackSettings(s, i)
		return
	}

	provider := strings.TrimPrefix(customID, "set_captcha_")
	if _, ok := providerLabels[provider]; !ok {
		respondToInteraction(s, i, "Invalid service selection")
		return
	}

	showAPIKeyModal(s, i, provider)
}

func HandleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	provider := strings.TrimPrefix(data.CustomID, "set_captcha_service_modal_")

	userID, err := services.GetUserID(i)
	if err != nil {
		respondToInteraction(s, i, "An error occurred while processing your request.")
		return
	}

	apiKey := getAPIKeyFromModal(data)
	if err := validateAndSaveAPIKey(s, i, userID, provider, apiKey); err != nil {
		respondToInteraction(s, i, fmt.Sprintf("Error: %v", err))
		return
	}
}

func createProviderButton(provider string) discordgo.MessageComponent {
	return services.CreateV2Button(providerLabels[provider], fmt.Sprintf("set_captcha_%s", provider), discordgo.PrimaryButton)
}

func handleAPIKeyRemoval(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID, err := services.GetUserID(i)
	if err != nil {
		respondToInteraction(s, i, "An error occurred while processing your request.")
		return
	}

	if err := services.RemoveCaptchaKey(userID); err != nil {
		respondToInteraction(s, i, "Error removing API key. Please try again.")
		return
	}

	respondToInteraction(s, i, "Your API key has been removed. The bot's default API key will be used. Your check interval and notification settings have been reset to default values.")
}

func showAPIKeyModal(s *discordgo.Session, i *discordgo.InteractionCreate, provider string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("set_captcha_service_modal_%s", provider),
			Title:    fmt.Sprintf("Set %s API Key", providerLabels[provider]),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "api_key",
							Label:       fmt.Sprintf("Enter your %s API key", providerLabels[provider]),
							Style:       discordgo.TextInputShort,
							Placeholder: "Enter your new API key",
							Required:    true,
							MinLength:   32,
							MaxLength:   90,
						},
					},
				},
			},
		},
	})
	if err != nil {
		logger.Log.WithError(err).Error("Error showing API key modal")
	}
}

func getAPIKeyFromModal(data discordgo.ModalSubmitInteractionData) string {
	for _, comp := range data.Components {
		if row, ok := comp.(*discordgo.ActionsRow); ok {
			for _, rowComp := range row.Components {
				if textInput, ok := rowComp.(*discordgo.TextInput); ok && textInput.CustomID == "api_key" {
					return utils.SanitizeInput(strings.TrimSpace(textInput.Value))
				}
			}
		}
	}
	return ""
}

func validateAndSaveAPIKey(s *discordgo.Session, i *discordgo.InteractionCreate, userID, provider, apiKey string) error {
	isValid, balance, err := services.ValidateCaptchaKey(apiKey, provider)
	if err != nil {
		return fmt.Errorf("error validating the %s API key: %v", provider, err)
	}
	if !isValid {
		return fmt.Errorf("the provided %s API key is invalid", provider)
	}

	settings := models.UserSettings{UserID: userID}
	if err := database.DB.Where("user_id = ?", userID).FirstOrCreate(&settings).Error; err != nil {
		return fmt.Errorf("error updating settings")
	}

	cfg := configuration.Get()
	settings.PreferredCaptchaProvider = provider
	if settings.FallbackCaptchaProvider == "" || settings.FallbackCaptchaProvider == provider {
		settings.FallbackCaptchaProvider = services.GetFallbackProvider(provider)
	}
	settings.CheckInterval = cfg.Intervals.Check
	settings.NotificationInterval = cfg.Intervals.Notification
	settings.CustomSettings = true
	settings.CaptchaBalance = balance
	settings.LastBalanceCheck = time.Now()

	updateAPIKeys(&settings, provider, apiKey)

	if err := database.DB.Save(&settings).Error; err != nil {
		return fmt.Errorf("error saving settings")
	}

	if apiKey != cfg.CaptchaService.Capsolver.ClientKey &&
		apiKey != cfg.CaptchaService.EZCaptcha.ClientKey &&
		apiKey != cfg.CaptchaService.TwoCaptcha.ClientKey {

		embed := &discordgo.MessageEmbed{
			Title:       "API Key Configuration Updated",
			Description: fmt.Sprintf("Your %s API key has been configured successfully!", providerLabels[provider]),
			Color:       0x00ff00,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Premium Features Unlocked",
					Value:  "• Faster check intervals\n• Increased account limits\n• Priority status updates",
					Inline: false,
				},
				{
					Name:   "Service Provider",
					Value:  providerLabels[provider],
					Inline: true,
				},
				{
					Name:   "Current Balance",
					Value:  fmt.Sprintf("%.2f points", balance),
					Inline: true,
				},
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}
		respondToInteractionWithEmbed(s, i, "", embed)
	} else {
		embed := &discordgo.MessageEmbed{
			Title:       "API Key Configuration Updated",
			Description: fmt.Sprintf("Your %s API key has been configured successfully!", providerLabels[provider]),
			Color:       0x00ff00,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Service Provider",
					Value:  providerLabels[provider],
					Inline: true,
				},
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}
		respondToInteractionWithEmbed(s, i, "", embed)
	}

	return nil
}

func updateAPIKeys(settings *models.UserSettings, provider, apiKey string) {
	settings.CapSolverAPIKey = ""
	settings.EZCaptchaAPIKey = ""
	settings.TwoCaptchaAPIKey = ""

	switch provider {
	case "capsolver":
		settings.CapSolverAPIKey = apiKey
	case "ezcaptcha":
		settings.EZCaptchaAPIKey = apiKey
	case "2captcha":
		settings.TwoCaptchaAPIKey = apiKey
	}
}

func respondToInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := services.RespondWithPreference(s, i, message, nil, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error responding to interaction")
	}
}

func respondToInteractionWithEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, message string, embed *discordgo.MessageEmbed) {
	embeds := []*discordgo.MessageEmbed{}
	if embed != nil {
		embeds = []*discordgo.MessageEmbed{embed}
	}

	err := services.RespondWithPreference(s, i, message, embeds, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error responding to interaction with embed")
	}
}

func showFallbackSettings(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID, err := services.GetUserID(i)
	if err != nil {
		respondToInteraction(s, i, "An error occurred while processing your request.")
		return
	}

	settings, err := services.GetUserSettings(userID)
	if err != nil {
		respondToInteraction(s, i, "Failed to get your current settings.")
		return
	}

	var components []discordgo.MessageComponent

	currentFallback := "Auto (" + services.GetFallbackProvider(settings.PreferredCaptchaProvider) + ")"
	if settings.FallbackCaptchaProvider != "" {
		currentFallback = providerLabels[settings.FallbackCaptchaProvider]
	}

	fallbackStatus := "Disabled"
	if settings.EnableFallback {
		fallbackStatus = "Enabled"
	}

	enabledLabel := "Enable Fallback"
	enabledStyle := discordgo.PrimaryButton
	if settings.EnableFallback {
		enabledLabel = "Disable Fallback"
		enabledStyle = discordgo.SecondaryButton
	}

	components = append(components, services.CreateV2Button(enabledLabel, "toggle_fallback_enabled", enabledStyle))

	cfg := configuration.Get()
	if cfg.CaptchaService.Capsolver.Enabled && settings.PreferredCaptchaProvider != "capsolver" {
		components = append(components, services.CreateV2Button("Set Capsolver as Fallback", "set_fallback_capsolver", discordgo.SecondaryButton))
	}

	if cfg.CaptchaService.EZCaptcha.Enabled && settings.PreferredCaptchaProvider != "ezcaptcha" {
		components = append(components, services.CreateV2Button("Set EZCaptcha as Fallback", "set_fallback_ezcaptcha", discordgo.SecondaryButton))
	}

	if cfg.CaptchaService.TwoCaptcha.Enabled && settings.PreferredCaptchaProvider != "2captcha" {
		components = append(components, services.CreateV2Button("Set 2Captcha as Fallback", "set_fallback_2captcha", discordgo.SecondaryButton))
	}

	components = append(components, services.CreateV2Button("Back to Main Menu", "captcha_main_menu", discordgo.SecondaryButton))

	embed := &discordgo.MessageEmbed{
		Title:       "🔄 Fallback Captcha Settings",
		Description: "Configure automatic fallback when your primary captcha service fails.",
		Color:       0x00D4AA,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "📊 Current Configuration",
				Value:  fmt.Sprintf("**Primary:** %s\n**Fallback:** %s\n**Status:** %s", providerLabels[settings.PreferredCaptchaProvider], currentFallback, fallbackStatus),
				Inline: false,
			},
			{
				Name:   "💡 How It Works",
				Value:  "When your primary service fails, the bot automatically tries the fallback service. This improves reliability and reduces failed checks.",
				Inline: false,
			},
			{
				Name:   "⚙️ Benefits",
				Value:  "• Higher success rate\n• Automatic failover\n• No manual intervention needed\n• Better uptime for checks",
				Inline: false,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Components v2 • No ActionRow limits",
		},
	}

	err = services.UpdateMessageWithPreference(s, i, "", []*discordgo.MessageEmbed{embed}, components)
	if err != nil {
		logger.Log.WithError(err).Error("Error updating message with fallback settings")
	}
}

func HandleFallbackSettingsInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	userID, err := services.GetUserID(i)
	if err != nil {
		respondToInteraction(s, i, "An error occurred while processing your request.")
		return
	}

	switch customID {
	case "toggle_fallback_enabled":
		toggleFallbackEnabled(s, i, userID)
	case "set_fallback_capsolver":
		setFallbackProvider(s, i, userID, "capsolver")
	case "set_fallback_ezcaptcha":
		setFallbackProvider(s, i, userID, "ezcaptcha")
	case "set_fallback_2captcha":
		setFallbackProvider(s, i, userID, "2captcha")
	case "captcha_main_menu":
		CommandSetCaptchaService(s, i)
	default:
		respondToInteraction(s, i, "Unknown fallback setting.")
	}
}

func toggleFallbackEnabled(s *discordgo.Session, i *discordgo.InteractionCreate, userID string) {
	settings, err := services.GetUserSettings(userID)
	if err != nil {
		respondToInteraction(s, i, "Failed to get your current settings.")
		return
	}

	settings.EnableFallback = !settings.EnableFallback

	if err := database.DB.Save(&settings).Error; err != nil {
		respondToInteraction(s, i, "Failed to update your settings.")
		return
	}

	status := "disabled"
	if settings.EnableFallback {
		status = "enabled"
	}

	respondToInteraction(s, i, fmt.Sprintf("Fallback captcha has been %s.", status))
}

func setFallbackProvider(s *discordgo.Session, i *discordgo.InteractionCreate, userID, provider string) {
	settings, err := services.GetUserSettings(userID)
	if err != nil {
		respondToInteraction(s, i, "Failed to get your current settings.")
		return
	}

	if settings.PreferredCaptchaProvider == provider {
		respondToInteraction(s, i, "You cannot set the same provider as both primary and fallback.")
		return
	}

	settings.FallbackCaptchaProvider = provider

	if err := database.DB.Save(&settings).Error; err != nil {
		respondToInteraction(s, i, "Failed to update your settings.")
		return
	}

	respondToInteraction(s, i, fmt.Sprintf("Fallback provider set to %s.", providerLabels[provider]))
}

func HandleFallbackNoticeInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	userID, err := services.GetUserID(i)
	if err != nil {
		respondToInteraction(s, i, "An error occurred while processing your request.")
		return
	}

	switch customID {
	case "dismiss_fallback_notice":
		dismissFallbackNotice(s, i, userID)
	case "set_captcha_from_notice":
		CommandSetCaptchaService(s, i)
	default:
		respondToInteraction(s, i, "Unknown action.")
	}
}

func dismissFallbackNotice(s *discordgo.Session, i *discordgo.InteractionCreate, userID string) {
	settings, err := services.GetUserSettings(userID)
	if err != nil {
		respondToInteraction(s, i, "Failed to get your current settings.")
		return
	}

	settings.HasSeenFallbackNotice = true

	if err := database.DB.Save(&settings).Error; err != nil {
		respondToInteraction(s, i, "Failed to update your settings.")
		return
	}

	respondToInteraction(s, i, "You will no longer receive fallback service notifications. You can still configure fallback settings using /setcaptchaservice.")
}
