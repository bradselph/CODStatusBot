package accountlogs

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"

	"github.com/bwmarrin/discordgo"
)

func CommandAccountLogs(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	// Create buttons for each account
	var components []discordgo.MessageComponent
	var currentRow []discordgo.MessageComponent

	for _, account := range accounts {
		currentRow = append(currentRow, discordgo.Button{
			Label:    account.Title,
			Style:    discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("account_logs_%d", account.ID),
		})

		if len(currentRow) == 5 {
			components = append(components, discordgo.ActionsRow{Components: currentRow})
			currentRow = []discordgo.MessageComponent{}
		}
	}

	// Create View All Logs button
	if len(currentRow) < 5 {
		currentRow = append(currentRow, discordgo.Button{
			Label:    "View All Logs",
			Style:    discordgo.SuccessButton,
			CustomID: "account_logs_all",
		})
	} else {
		components = append(components, discordgo.ActionsRow{Components: currentRow})
		currentRow = []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "View All Logs",
				Style:    discordgo.SuccessButton,
				CustomID: "account_logs_all",
			},
		}
	}

	// Add the last row
	components = append(components, discordgo.ActionsRow{Components: currentRow})

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    "Select an account to view its logs, or 'View All Logs' to see logs for all accounts:",
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

	if customID == "account_logs_all" {
		handleAllAccountLogs(s, i)
		return
	}

	accountID, err := strconv.Atoi(strings.TrimPrefix(customID, "account_logs_"))
	if err != nil {
		logger.Log.WithError(err).Error("Error parsing account ID")
		respondToInteraction(s, i, "Error processing your selection. Please try again.")
		return
	}

	var account models.Account
	result := database.DB.First(&account, accountID)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error fetching account")
		respondToInteraction(s, i, "Error: Account not found or you don't have permission to view its logs.")
		return
	}

	embed := createAccountLogEmbed(account)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{},
		},
	})
	if err != nil {
		logger.Log.WithError(err).Error("Error responding to interaction with account logs")
		respondToInteraction(s, i, "Error displaying account logs. Please try again.")
	}
}

func handleAllAccountLogs(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	var embeds []*discordgo.MessageEmbed
	for _, account := range accounts {
		embed := createAccountLogEmbed(account)
		embeds = append(embeds, embed)
	}

	// Send embeds in batches of 10 (Discord's limit)
	for j := 0; j < len(embeds); j += 10 {
		end := j + 10
		if end > len(embeds) {
			end = len(embeds)
		}

		var err error
		if j == 0 {
			err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Content:    "",
					Embeds:     embeds[j:end],
					Components: []discordgo.MessageComponent{},
				},
			})
		} else {
			_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Embeds: embeds[j:end],
				Flags:  discordgo.MessageFlagsEphemeral,
			})
		}

		if err != nil {
			logger.Log.WithError(err).Error("Error sending account logs")
		}
	}
}

func createAccountLogEmbed(account models.Account) *discordgo.MessageEmbed {
	var logs []models.Ban
	database.DB.Where("account_id = ?", account.ID).Order("created_at desc").Limit(10).Find(&logs)

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s - Account Logs", account.Title),
		Description: "The last 10 status changes for this account",
		Color:       0x00ff00,
		Fields:      make([]*discordgo.MessageEmbedField, len(logs)),
	}

	for i, log := range logs {
		embed.Fields[i] = &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("Status Change %d", i+1),
			Value:  fmt.Sprintf("Status: %s\nTime: %s", log.Status, log.CreatedAt.Format(time.RFC1123)),
			Inline: false,
		}
	}

	if len(logs) == 0 {
		embed.Description = "No status changes logged for this account yet."
	}

	return embed
}

func respondToInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		logger.Log.WithError(err).Error("Error responding to interaction")
	}
}
