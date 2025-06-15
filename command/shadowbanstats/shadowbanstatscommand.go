package shadowbanstats

import (
	"fmt"
	"time"

	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bradselph/CODStatusBot/services"
	"github.com/bradselph/CODStatusBot/utils"
	"github.com/bwmarrin/discordgo"
)

func CommandShadowbanStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := services.DeferWithPreference(s, i, false)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to defer response")
		return
	}

	userID, err := services.GetUserID(i)
	if err != nil {
		logger.Log.WithError(err).Error("Could not determine user ID")
		sendFollowup(s, i, "An error occurred while processing your request.")
		return
	}

	if err := utils.ValidateDiscordUserID(userID); err != nil {
		logger.Log.WithError(err).WithField("userID", userID).Error("Invalid user ID")
		sendFollowup(s, i, "Invalid user ID provided.")
		return
	}

	var accountCount int64
	if err := database.DB.Model(&models.Account{}).Where("user_id = ?", userID).Count(&accountCount).Error; err != nil {
		logger.Log.WithError(err).Error("Error checking user accounts")
		sendFollowup(s, i, "Error checking your accounts. Please try again.")
		return
	}

	if accountCount == 0 {
		sendFollowup(s, i, "You don't have any monitored accounts. Add an account first using /addaccount.")
		return
	}

	days := 30
	var accountID *uint

	if i.ApplicationCommandData().Options != nil {
		for _, option := range i.ApplicationCommandData().Options {
			switch option.Name {
			case "days":
				if option.IntValue() > 0 && option.IntValue() <= 365 {
					days = int(option.IntValue())
				}
			case "account":
				if option.StringValue() != "" {
					var account models.Account
					if err := database.DB.Where("user_id = ? AND title = ?", userID, option.StringValue()).First(&account).Error; err == nil {
						accountID = &account.ID
					}
				}
			}
		}
	}

	stats, err := services.GetShadowbanStatistics(userID, accountID, days)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to get shadowban statistics")
		sendFollowup(s, i, "Failed to retrieve shadowban statistics. Please try again.")
		return
	}

	embed := createShadowbanStatsEmbed(stats, accountID != nil)

	_, err = services.FollowupWithPreference(s, i, "", []*discordgo.MessageEmbed{embed}, nil, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error sending followup message")
	}
}

func createShadowbanStatsEmbed(stats map[string]interface{}, isAccountSpecific bool) *discordgo.MessageEmbed {
	title := "Your Shadowban Statistics"
	if isAccountSpecific {
		title = "Account Shadowban Statistics"
	}

	days := stats["period_days"].(int)
	startDate := stats["start_date"].(string)
	endDate := stats["end_date"].(string)

	description := fmt.Sprintf("**Period:** %s to %s (%d days)\n", startDate, endDate, days)

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0x3498db,
		Timestamp:   time.Now().Format(time.RFC3339),
		Fields:      make([]*discordgo.MessageEmbedField, 0),
	}

	completedPeriods := stats["total_completed_periods"].(int)
	activePeriods := stats["active_periods"].(int64)

	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "Overview",
		Value:  fmt.Sprintf("Completed Shadowbans: **%d**\nCurrently Active: **%d**", completedPeriods, activePeriods),
		Inline: false,
	})

	if completedPeriods > 0 {
		if avgHours, ok := stats["average_duration_hours"].(float64); ok {
			avgDays := stats["average_duration_days"].(float64)
			minHours := stats["min_duration_hours"].(float64)
			maxHours := stats["max_duration_hours"].(float64)
			totalHours := stats["total_shadowban_hours"].(float64)

			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name: "Duration Analysis",
				Value: fmt.Sprintf("Average: **%.1f hours** (%.1f days)\nShortest: **%.1f hours** (%.1f days)\nLongest: **%.1f hours** (%.1f days)\nTotal Time: **%.1f hours** (%.1f days)",
					avgHours, avgDays,
					minHours, minHours/24,
					maxHours, maxHours/24,
					totalHours, totalHours/24),
				Inline: false,
			})
		}

		if recentPeriods, ok := stats["recent_periods"].([]map[string]interface{}); ok && len(recentPeriods) > 0 {
			recentValue := ""
			maxShow := 5
			if len(recentPeriods) < maxShow {
				maxShow = len(recentPeriods)
			}

			for i := 0; i < maxShow; i++ {
				period := recentPeriods[i]
				accountTitle := period["account_title"].(string)
				startTime := period["start_time"].(string)
				endTime := period["end_time"].(string)

				if durationHours, ok := period["duration_hours"].(float64); ok {
					durationDays := period["duration_days"].(float64)
					recentValue += fmt.Sprintf("**%s**\n%s → %s\nDuration: %.1f hours (%.1f days)\n\n",
						accountTitle,
						formatTimeShort(startTime),
						formatTimeShort(endTime),
						durationHours, durationDays)
				}
			}

			if len(recentValue) > 0 {
				if len(recentPeriods) > maxShow {
					recentValue += fmt.Sprintf("*...and %d more*", len(recentPeriods)-maxShow)
				}

				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:   fmt.Sprintf("Recent Shadowbans (%d shown)", maxShow),
					Value:  recentValue,
					Inline: false,
				})
			}
		}
	} else {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "No Data",
			Value:  fmt.Sprintf("No completed shadowban periods found in the last %d days.\n\nThis could mean:\n• No shadowbans occurred\n• Shadowbans started before tracking began\n• Account status was 'Unknown' when shadowbans started", days),
			Inline: false,
		})
	}

	if activePeriods > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Active Shadowbans",
			Value:  fmt.Sprintf("You currently have **%d** account(s) under review. Duration tracking will complete when the status changes to 'Good'.", activePeriods),
			Inline: false,
		})
	}

	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: "Note: Only shadowbans with reliable start/end times are included in duration calculations",
	}

	return embed
}

func formatTimeShort(timeStr string) string {
	t, err := time.Parse("2006-01-02 15:04:05", timeStr)
	if err != nil {
		return timeStr
	}
	return t.Format("Jan 02 15:04")
}

func sendFollowup(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	_, err := services.FollowupWithPreference(s, i, content, nil, nil, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error sending followup message")
	}
}

func GetShadowbanStatsCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "shadowbanstats",
		Description: "View shadowban duration statistics for your accounts",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "days",
				Description: "Number of days to look back (1-365, default: 30)",
				Required:    false,
				MinValue:    func() *float64 { v := 1.0; return &v }(),
				MaxValue:    365,
			},
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "account",
				Description:  "Show stats for a specific account (leave blank for all accounts)",
				Required:     false,
				Autocomplete: true,
			},
		},
	}
}

func HandleShadowbanStatsAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID, err := services.GetUserID(i)
	if err != nil {
		logger.Log.WithError(err).Error("Could not determine user ID for autocomplete")
		return
	}

	var accounts []models.Account
	if err := database.DB.Where("user_id = ?", userID).Select("title").Find(&accounts).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to fetch accounts for autocomplete")
		return
	}

	var choices []*discordgo.ApplicationCommandOptionChoice
	for _, account := range accounts {
		if len(choices) >= 25 {
			break
		}
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  account.Title,
			Value: account.Title,
		})
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})

	if err != nil {
		logger.Log.WithError(err).Error("Failed to respond to autocomplete")
	}
}
