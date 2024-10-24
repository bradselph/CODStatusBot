package accountage

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bradselph/CODStatusBot/services"

	"github.com/bwmarrin/discordgo"
)

func CommandAccountAge(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	var (
		components []discordgo.MessageComponent
		currentRow []discordgo.MessageComponent
	)

	for _, account := range accounts {
		currentRow = append(currentRow, discordgo.Button{
			Label:    account.Title,
			Style:    discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("account_age_%d", account.ID),
		})

		if len(currentRow) == 5 {
			components = append(components, discordgo.ActionsRow{Components: currentRow})
			currentRow = []discordgo.MessageComponent{}
		}
	}

	if len(currentRow) > 0 {
		components = append(components, discordgo.ActionsRow{Components: currentRow})
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    "Select an account to check its age:",
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
	accountID, err := strconv.Atoi(strings.TrimPrefix(customID, "account_age_"))
	if err != nil {
		logger.Log.WithError(err).Error("Error parsing account ID")
		respondToInteraction(s, i, "Error processing your selection. Please try again.")
		return
	}

	var account models.Account
	result := database.DB.First(&account, accountID)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error fetching account")
		respondToInteraction(s, i, "Error: Account not found or you don't have permission to check its age.")
		return
	}

	if !services.VerifySSOCookie(account.SSOCookie) {
		account.IsExpiredCookie = true // Update account's IsExpiredCookie flag
		database.DB.Save(&account)
		respondToInteraction(s, i, "Invalid SSOCookie. Account's cookie status updated.")
		return
	}

	years, months, days, createdEpoch, err := services.CheckAccountAge(account.SSOCookie)
	if err != nil {
		logger.Log.WithError(err).Errorf("Error checking account age for account %s", account.Title)
		respondToInteraction(s, i, "There was an error checking the account age.")
		return
	}

	account.Created = createdEpoch
	if err := database.DB.Save(&account).Error; err != nil {
		logger.Log.WithError(err).Errorf("Error saving account creation timestamp for account %s", account.Title)
	}

	creationDate := time.Unix(createdEpoch, 0).UTC().Format("January 2, 2006")

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s - Account Age", account.Title),
		Description: fmt.Sprintf("The account is %d years, %d months, and %d days old.", years, months, days),
		Color:       0x00ff00,
		Timestamp:   time.Now().Format(time.RFC3339),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Last Status",
				Value:  string(account.LastStatus),
				Inline: true,
			},
			{
				Name:   "Creation Date",
				Value:  creationDate,
				Inline: true,
			},
		},
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{},
		},
	})
	if err != nil {
		logger.Log.WithError(err).Error("Error responding to interaction with account age")
		respondToInteraction(s, i, "Error displaying account age. Please try again.")
	}
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
