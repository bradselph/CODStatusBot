package globalannouncement

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

func SendGlobalAnnouncement(s *discordgo.Session, userID string) error {
	var userSettings models.UserSettings
	result := database.DB.Where(models.UserSettings{UserID: userID}).FirstOrCreate(&userSettings)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error getting user settings for global announcement")
		return result.Error
	}

	if !userSettings.HasSeenAnnouncement {
		channelID, err := getChannelForAnnouncement(s, userID, userSettings)
		if err != nil {
			logger.Log.WithError(err).Error("Error finding channel for user")
			return err
		}

		announcementEmbed := services.CreateAnnouncementEmbed()

		_, err = s.ChannelMessageSendEmbed(channelID, announcementEmbed)
		if err != nil {
			logger.Log.WithError(err).Error("Error sending global announcement")
			return err
		}

		userSettings.HasSeenAnnouncement = true
		if err := database.DB.Save(&userSettings).Error; err != nil {
			logger.Log.WithError(err).Error("Error updating user settings after sending announcement")
			return err
		}
	}

	return nil
}

func CommandGlobalAnnouncement(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cfg := configuration.Get()
	developerID := cfg.Discord.DeveloperID
	if developerID == "" {
		logger.Log.Error("DEVELOPER_ID not set in environment variables")
		respondToInteraction(s, i, "Error: Developer ID not configured.")
		return
	}

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

	if userID != developerID {
		logger.Log.Warnf("Unauthorized user %s attempted to use global announcement command", userID)
		respondToInteraction(s, i, "You don't have permission to use this command. Only the bot developer can send global announcements.")
		return
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "global_announcement_modal",
			Title:    "Send Global Announcement",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:  "announcement_title",
							Label:     "Announcement Title",
							Style:     discordgo.TextInputShort,
							Required:  true,
							MinLength: 1,
							MaxLength: 100,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:  "announcement_content",
							Label:     "Announcement Content",
							Style:     discordgo.TextInputParagraph,
							Required:  true,
							MinLength: 1,
							MaxLength: 4000,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:  "reset_flags",
							Label:     "Reset Initial Announcement Flag (yes/no)",
							Style:     discordgo.TextInputShort,
							Required:  true,
							MinLength: 2,
							MaxLength: 3,
							Value:     "no",
						},
					},
				},
			},
		},
	})

	if err != nil {
		logger.Log.WithError(err).Error("Error showing announcement modal")
		respondToInteraction(s, i, "Error creating announcement modal. Please try again.")
	}
}

func HandleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()

	var title, content, resetFlags string
	for _, comp := range data.Components {
		if row, ok := comp.(*discordgo.ActionsRow); ok {
			for _, rowComp := range row.Components {
				if textInput, ok := rowComp.(*discordgo.TextInput); ok {
					switch textInput.CustomID {
					case "announcement_title":
						title = utils.SanitizeInput(strings.TrimSpace(textInput.Value))
					case "announcement_content":
						content = utils.SanitizeAnnouncement(strings.TrimSpace(textInput.Value))
					case "reset_flags":
						resetFlags = strings.ToLower(utils.SanitizeInput(strings.TrimSpace(textInput.Value)))
					}
				}
			}
		}
	}

	if resetFlags == "yes" {
		logger.Log.Info("Resetting announcement flags for all users")
		if err := database.DB.Exec("UPDATE user_settings SET has_seen_announcement = ?", false).Error; err != nil {
			logger.Log.WithError(err).Error("Failed to reset announcement flags")
			respondToInteraction(s, i, "Error resetting announcement flags. Please try again.")
			return
		}
		logger.Log.Info("Reset all users' announcement flags successfully")
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: content,
		Color:       0xFFD700,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "COD Status Bot Announcement",
		},
	}

	var users []models.UserSettings
	if err := database.DB.Find(&users).Error; err != nil {
		logger.Log.WithError(err).Error("Error fetching users for announcement")
		respondToInteraction(s, i, "Error fetching users. Please try again.")
		return
	}

	successCount := 0
	failCount := 0

	for _, user := range users {
		if err := sendDynamicAnnouncementToUser(s, user.UserID, embed); err != nil {
			logger.Log.WithError(err).Errorf("Failed to send announcement to user %s", user.UserID)
			failCount++
		} else {
			successCount++
		}
	}

	respondToInteraction(s, i, fmt.Sprintf("Announcement sent successfully to %d users. %d users could not be reached.", successCount, failCount))
}

func sendDynamicAnnouncementToUser(s *discordgo.Session, userID string, embed *discordgo.MessageEmbed) error {
	var userSettings models.UserSettings
	result := database.DB.Where(models.UserSettings{UserID: userID}).FirstOrCreate(&userSettings)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error getting user settings for global announcement")
		return result.Error
	}

	channelID, err := getChannelForAnnouncement(s, userID, userSettings)
	if err != nil {
		return err
	}

	_, err = s.ChannelMessageSendEmbed(channelID, embed)
	return err
}

func getChannelForAnnouncement(s *discordgo.Session, userID string, userSettings models.UserSettings) (string, error) {
	if userSettings.NotificationType == "dm" {
		channel, err := s.UserChannelCreate(userID)
		if err != nil {
			return "", fmt.Errorf("failed to create DM channel: %w", err)
		}
		return channel.ID, nil
	}

	var account models.Account
	if err := database.DB.Where("user_id = ?", userID).Order("updated_at DESC").First(&account).Error; err != nil {
		channel, err := s.UserChannelCreate(userID)
		if err != nil {
			return "", fmt.Errorf("both channel lookup and DM creation failed: %w", err)
		}
		return channel.ID, nil
	}
	return account.ChannelID, nil
}

func respondToInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := services.RespondWithPreference(s, i, message, nil, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error responding to interaction")
	}
}
