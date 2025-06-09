package accountlogs

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

	var components []discordgo.MessageComponent

	for _, account := range accounts {
		components = append(components, discordgo.Button{
			Label:    account.Title,
			Style:    discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("account_logs_%d", account.ID),
		})
	}

	components = append(components, discordgo.Button{
		Label:    "View All Logs",
		Style:    discordgo.SuccessButton,
		CustomID: "account_logs_all",
	})

	err := services.RespondWithPreferenceAndComponents(s, i, "Select an account to view its logs, or 'View All Logs' to see logs for all accounts:", nil, components, false)
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

	if !account.IsExpiredCookie && !services.VerifySSOCookie(account.SSOCookie) {
		account.IsExpiredCookie = true
		database.DB.Save(&account)
	}

	embed := createAccountLogEmbed(account)

	err = services.UpdateMessageWithPreference(s, i, "", []*discordgo.MessageEmbed{embed}, []discordgo.MessageComponent{})
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
		if !account.IsExpiredCookie && !services.VerifySSOCookie(account.SSOCookie) {
			account.IsExpiredCookie = true
			database.DB.Save(&account)
		}

		embed := createAccountLogEmbed(account)
		embeds = append(embeds, embed)
	}

	for j := 0; j < len(embeds); j += 10 {
		end := j + 10
		if end > len(embeds) {
			end = len(embeds)
		}

		var err error
		if j == 0 {
			err = services.UpdateMessageWithPreference(s, i, "", embeds[j:end], []discordgo.MessageComponent{})
		} else {
			_, err = services.FollowupWithPreference(s, i, "", embeds[j:end], nil, false)
		}

		if err != nil {
			logger.Log.WithError(err).Error("Error sending account logs")
		}
	}
}

func createAccountLogEmbed(account models.Account) *discordgo.MessageEmbed {
	var logs []models.Ban
	database.DB.Where("account_id = ?", account.ID).Order("timestamp desc").Limit(15).Find(&logs)

	var validLogs []models.Ban
	for _, log := range logs {
		if !log.Timestamp.IsZero() && log.Timestamp.Year() > 1970 {
			validLogs = append(validLogs, log)
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s - Account History", account.Title),
		Description: fmt.Sprintf("Recent account history (showing last %d entries)", len(validLogs)),
		Color:       services.GetColorForStatus(account.LastStatus, account.IsExpiredCookie, account.IsCheckDisabled),
		Fields:      make([]*discordgo.MessageEmbedField, 0),
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	if len(validLogs) == 0 {
		embed.Description = "No valid history found for this account."
		return embed
	}

	createdTime := "Unknown"
	if account.Created > 0 {
		createdTime = time.Unix(account.Created, 0).Format("Jan 02, 2006 15:04:05 MST")
	}

	lastCheckTime := "Never checked"
	if account.LastCheck > 0 {
		lastCheckTime = time.Unix(account.LastCheck, 0).Format("Jan 02, 2006 15:04:05 MST")
	}

	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name: "Account Information",
		Value: fmt.Sprintf("Current Status: %s\nChecks: %s\nCreated: %s\nLast Checked: %s",
			account.LastStatus,
			services.GetCheckStatus(account.IsCheckDisabled),
			createdTime,
			lastCheckTime),
		Inline: false,
	})

	for _, log := range logs {
		var fieldValue strings.Builder

		if log.Timestamp.IsZero() {
			continue
		}

		switch log.LogType {
		case "account_added":
			fieldValue.WriteString("Account added to monitoring\n")
		case "status_change":
			fieldValue.WriteString(fmt.Sprintf("%s -> %s\n", log.PreviousStatus, log.Status))
			if log.Message != "" {
				fieldValue.WriteString(fmt.Sprintf("%s\n", log.Message))
			}
			if log.AffectedGames != "" {
				fieldValue.WriteString(fmt.Sprintf("Affected Games: %s\n", log.AffectedGames))
			}
			if log.TempBanDuration != "" {
				fieldValue.WriteString(fmt.Sprintf("Duration: %s\n", log.TempBanDuration))
			}
		case "check_status":
			fieldValue.WriteString(log.Message + "\n")
		case "cookie_update":
			fieldValue.WriteString("SSO Cookie updated\n")
		case "error":
			fieldValue.WriteString(fmt.Sprintf("Error: %s\n", log.ErrorDetails))
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s", log.Timestamp.Format("Jan 02, 2006 15:04:05 MST")),
			Value:  fieldValue.String(),
			Inline: false,
		})
	}

	return embed
}

func formatTimeField(timestamp int64) string {
	if timestamp <= 0 {
		return "Not available"
	}
	return time.Unix(timestamp, 0).Format("Jan 02, 2006 15:04:05 MST")
}

func respondToInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	err := services.RespondWithPreference(s, i, content, nil, false)

	if err != nil {
		logger.Log.WithError(err).Error("Error responding with channel message, trying update message")

		err = services.UpdateMessageWithPreference(s, i, content, nil, []discordgo.MessageComponent{})

		if err != nil {
			logger.Log.WithError(err).Error("Error updating message, trying followup")

			_, err = services.FollowupWithPreference(s, i, content, nil, nil, false)

			if err != nil {
				logger.Log.WithError(err).Error("All response methods failed")
			}
		}
	}
}
