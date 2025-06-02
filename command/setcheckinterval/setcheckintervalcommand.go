package setcheckinterval

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/services"
	"github.com/bradselph/CODStatusBot/utils"

	"github.com/bwmarrin/discordgo"
)

func CommandSetCheckInterval(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	if userSettings.CapSolverAPIKey == "" && userSettings.EZCaptchaAPIKey == "" && userSettings.TwoCaptchaAPIKey == "" {
		respondToInteraction(s, i, "You need to set your own Capsolver or EZ-Captcha or 2captcha API key using the /setcaptchaservice command before you can modify these settings.")
		return
	}

	explanationEmbed := &discordgo.MessageEmbed{
		Title: "Configure Check & Notification Settings",
		Description: "You're about to configure the following settings:\n\n" +
			"**• Check Interval (1-1440 minutes)**\n" +
			"How often the bot checks your account status. Lower values mean more frequent checks but use more captcha credits.\n" +
			"Recommended: 15-30 minutes for active monitoring.\n\n" +
			"**• Notification Interval (1-24 hours)**\n" +
			"How often you receive regular status updates, even when nothing changes.\n" +
			"Recommended: 12-24 hours for daily updates.\n\n" +
			"**• Cooldown Duration (1-24 hours)**\n" +
			"Minimum time between repeated notifications of the same type to prevent spam.\n" +
			"Recommended: 6-12 hours.\n\n" +
			"**• Status Change Cooldown (1-24 hours)**\n" +
			"Minimum time between notifications when your account status changes.\n" +
			"Recommended: 1-2 hours.\n\n" +
			"Click 'Configure' to set these values.",
		Color: 0x00ff00,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Note: Lower intervals provide more frequent updates but use more captcha credits",
		},
	}

	intervalComponents := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Configure",
					Style:    discordgo.PrimaryButton,
					CustomID: "show_interval_modal",
				},
			},
		},
	}

	err = services.RespondWithPreferenceAndComponents(s, i, "", []*discordgo.MessageEmbed{explanationEmbed}, intervalComponents, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error sending explanation message")
	}
}

func HandleButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.MessageComponentData().CustomID != "show_interval_modal" {
		return
	}

	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	userSettings, err := services.GetUserSettings(userID)
	if err != nil {
		logger.Log.WithError(err).Error("Error fetching user settings")
		respondToInteraction(s, i, "Error fetching your settings. Please try again.")
		return
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "set_check_interval_modal",
			Title:    "Configure Settings",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:  "check_interval",
							Label:     "Check Interval (minutes)",
							Style:     discordgo.TextInputShort,
							Required:  false,
							MinLength: 0,
							MaxLength: 4,
							Value:     strconv.Itoa(userSettings.CheckInterval),
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:  "notification_interval",
							Label:     "Notification Interval (hours)",
							Style:     discordgo.TextInputShort,
							Required:  false,
							MinLength: 0,
							MaxLength: 2,
							Value:     fmt.Sprintf("%.0f", userSettings.NotificationInterval),
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:  "cooldown_duration",
							Label:     "Cooldown Duration (hours)",
							Style:     discordgo.TextInputShort,
							Required:  false,
							MinLength: 0,
							MaxLength: 2,
							Value:     fmt.Sprintf("%.0f", userSettings.CooldownDuration),
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:  "status_change_cooldown",
							Label:     "Status Change Cooldown (hours)",
							Style:     discordgo.TextInputShort,
							Required:  false,
							MinLength: 0,
							MaxLength: 2,
							Value:     fmt.Sprintf("%.0f", userSettings.StatusChangeCooldown),
						},
					},
				},
			},
		},
	})
	if err != nil {
		logger.Log.WithError(err).Error("Error responding with modal")
	}
}

func HandleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()

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

	defaultSettings, err := services.GetDefaultSettings()
	if err != nil {
		logger.Log.WithError(err).Error("Error fetching default settings")
		respondToInteraction(s, i, "Error fetching default settings. Please try again.")
		return
	}

	var errors []string

	for _, comp := range data.Components {
		if row, ok := comp.(*discordgo.ActionsRow); ok {
			for _, rowComp := range row.Components {
				if textInput, ok := rowComp.(*discordgo.TextInput); ok {
					value := utils.SanitizeInput(strings.TrimSpace(textInput.Value))

					switch textInput.CustomID {
					case "check_interval":
						if value == "" {
							userSettings.CheckInterval = defaultSettings.CheckInterval
						} else {
							interval, err := strconv.Atoi(value)
							if err != nil || interval < 1 || interval > 1440 {
								errors = append(errors, "Check interval must be between 1 and 1440 minutes.")
								continue
							}
							userSettings.CheckInterval = interval
						}

					case "notification_interval":
						if value == "" {
							userSettings.NotificationInterval = defaultSettings.NotificationInterval
						} else {
							interval, err := strconv.ParseFloat(value, 64)
							if err != nil || interval < 1 || interval > 24 {
								errors = append(errors, "Notification interval must be between 1 and 24 hours.")
								continue
							}
							userSettings.NotificationInterval = interval
						}

					case "cooldown_duration":
						if value == "" {
							userSettings.CooldownDuration = defaultSettings.CooldownDuration
						} else {
							duration, err := strconv.ParseFloat(value, 64)
							if err != nil || duration < 1 || duration > 24 {
								errors = append(errors, "Cooldown duration must be between 1 and 24 hours.")
								continue
							}
							userSettings.CooldownDuration = duration
						}

					case "status_change_cooldown":
						if value == "" {
							userSettings.StatusChangeCooldown = defaultSettings.StatusChangeCooldown
						} else {
							cooldown, err := strconv.ParseFloat(value, 64)
							if err != nil || cooldown < 1 || cooldown > 24 {
								errors = append(errors, "Status change cooldown must be between 1 and 24 hours.")
								continue
							}
							userSettings.StatusChangeCooldown = cooldown
						}
					}
				}
			}
		}
	}

	if len(errors) > 0 {
		respondToInteraction(s, i, fmt.Sprintf("Error updating settings:\n%s", strings.Join(errors, "\n")))
		return
	}

	if err := database.DB.Save(&userSettings).Error; err != nil {
		logger.Log.WithError(err).Error("Error saving user settings")
		respondToInteraction(s, i, "Error updating your settings. Please try again.")
		return
	}

	successEmbed := &discordgo.MessageEmbed{
		Title: "Settings Updated Successfully",
		Description: fmt.Sprintf("Your new settings:\n\n"+
			"• Check Interval: %d minutes\n"+
			"• Notification Interval: %.1f hours\n"+
			"• Cooldown Duration: %.1f hours\n"+
			"• Status Change Cooldown: %.1f hours",
			userSettings.CheckInterval,
			userSettings.NotificationInterval,
			userSettings.CooldownDuration,
			userSettings.StatusChangeCooldown),
		Color:     0x00ff00,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	respondToInteraction(s, i, "", successEmbed)
}

func respondToInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, message string, embeds ...*discordgo.MessageEmbed) {
	err := services.RespondWithPreference(s, i, message, embeds, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error responding to interaction")
	}
}
