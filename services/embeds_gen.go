package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/bradselph/CODStatusBot/models"
	"github.com/bwmarrin/discordgo"
)

func CreateAnnouncementEmbed() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "Important Update: Changes to COD Status Bot's Captcha Service",
		Description: "I am excited to announce that I have upgraded the captcha service to provide better reliability and performance:",
		Color:       0xFFD700,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name: "What's Changing",
				Value: "• I have switched to Capsolver as our primary captcha service provider\n" +
					"• The bot now uses Capsolver's API for improved reliability\n" +
					"• Users can still provide their own API keys for unlimited checks",
			},
			{
				Name: "How to Get Your Own API Key",
				Value: "1. Sign up at [Capsolver](https://dashboard.capsolver.com/passport/register?inviteCode=6YjROhACQnvP)\n" +
					"2. Purchase credits (starting at just $0.001 per solve)\n" +
					"3. Use the `/setcaptchaservice` command to set your API key in the bot",
			},
			{
				Name: "Existing EZCaptcha Users",
				Value: "• If you have an existing EZCaptcha key, it will continue to work\n" +
					"• However, we highly recommend switching to Capsolver for:\n" +
					"  - Better reliability and faster solves\n" +
					"  - Lower cost per solve\n" +
					"  - Improved enterprise support",
			},
			{
				Name: "Benefits of Using Your Own API Key",
				Value: "• Uninterrupted access to the check now feature\n" +
					"• Ability to customize check intervals\n" +
					"• Support the bot's development through our referral program",
			},
			{
				Name: "Next Steps",
				Value: "1. Visit Capsolver to obtain your API key\n" +
					"2. Set up your key using the `/setcaptchaservice` command\n" +
					"3. Adjust your check interval preferences if desired",
			},
			{
				Name: "Our Commitment",
				Value: "I am committed to providing the most reliable and cost-effective service.\n" +
					"I have tried to maintain compatibility with EZCaptcha for existing users, but\n" +
					"I strongly recommend Capsolver for its superior performance and cost benefits.",
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Thank you for using CODStatusbot!",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func GetColorForStatus(status models.Status, isExpiredCookie bool, isCheckDisabled bool) int {
	if isCheckDisabled {
		return 0xA9A9A9 // Dark Gray for disabled checks
	}
	if isExpiredCookie {
		return 0xFF6347 // Tomato for expired cookie
	}
	switch status {
	case models.StatusPermaban:
		return 0xFF0000 // Red for permanent ban
	case models.StatusShadowban:
		return 0xFFA500 // Orange for shadowban
	case models.StatusTempban:
		return 0xFF8C00 // Dark Orange for temporary ban
	case models.StatusRankLocked:
		return 0xFFD700 // Gold for rank locked
	case models.StatusPartialBan:
		return 0xFF4500 // OrangeRed for partial ban
	case models.StatusGood:
		return 0x32CD32 // Lime Green for good status
	default:
		return 0x708090 // Slate Gray for unknown status
	}
}

func EmbedTitleFromStatus(status models.Status) string {
	switch status {
	case models.StatusTempban:
		return "TEMPORARY BAN DETECTED"
	case models.StatusPermaban:
		return "PERMANENT BAN DETECTED"
	case models.StatusShadowban:
		return "ACCOUNT UNDER REVIEW (SHADOWBAN)"
	case models.StatusRankLocked:
		return "RANKED PLAY RESTRICTED"
	case models.StatusPartialBan:
		return "GAME-SPECIFIC RESTRICTIONS"
	default:
		return "ACCOUNT NOT BANNED"
	}
}

func GetStatusDescription(status models.Status, accountTitle string, ban models.Ban) string {
	switch status {
	case models.StatusPermaban:
		desc := fmt.Sprintf("The account %s has been permanently banned.", accountTitle)
		if ban.AffectedGames != "" {
			desc += fmt.Sprintf("\n\n**Affected Games:**\n%s", formatAffectedGames(ban.AffectedGames))
		}
		return desc

	case models.StatusShadowban:
		desc := fmt.Sprintf("The account %s has been placed under review (shadowban).", accountTitle)

		isCampaignOnlyShadowban := false
		if len(ban.GameSpecificBans) > 0 {
			campaignBanCount := 0
			otherBanCount := 0

			for title, enforcement := range ban.GameSpecificBans {
				if enforcement == "UNDER_REVIEW" {
					if strings.Contains(title, "SP") || strings.Contains(title, "CAMPAIGN") {
						campaignBanCount++
					} else {
						otherBanCount++
					}
				}
			}

			if campaignBanCount > 0 && otherBanCount == 0 {
				isCampaignOnlyShadowban = true
			}
		}

		if isCampaignOnlyShadowban {
			desc += "\n\n**Important Note:**\nThis appears to be a campaign-only shadowban. The limited matchmaking effect will disappear after the normal shadowban timeframe, but you may still see 'Under Review' status for the campaign mode permanently."
		}

		if ban.AffectedGames != "" {
			desc += fmt.Sprintf("\n\n**Affected Games:**\n%s", formatAffectedGames(ban.AffectedGames))
		}
		return desc

	case models.StatusTempban:
		desc := fmt.Sprintf("The account %s is temporarily banned", accountTitle)
		if ban.TempBanDuration != "" {
			desc += fmt.Sprintf(" for %s", ban.TempBanDuration)
		}
		desc += "."
		if ban.AffectedGames != "" {
			desc += fmt.Sprintf("\n\n**Affected Games:**\n%s", formatAffectedGames(ban.AffectedGames))
		}
		return desc

	case models.StatusRankLocked:
		desc := fmt.Sprintf("The account %s is restricted from ranked play only.\n", accountTitle)
		desc += "Regular multiplayer and other game modes remain available.\n\n"

		isBo6CampaignShadowban := false
		for title, enforcement := range ban.GameSpecificBans {
			if strings.Contains(title, "BO6 SP") && enforcement == "UNDER_REVIEW" {
				isBo6CampaignShadowban = true
				break
			}
		}

		if isBo6CampaignShadowban {
			desc += "This appears to be a BO6 campaign-specific restriction. The account will show as 'Under Review' for the campaign mode, but this won't affect regular multiplayer matchmaking after the normal shadowban period."
		} else {
			desc += "This restriction typically persists even after shadowbans are lifted."
		}

		if len(ban.GameSpecificBans) > 0 {
			desc += "\n\n**Game-Specific Status:**"
			for title, enforcement := range ban.GameSpecificBans {
				desc += fmt.Sprintf("\n• %s: %s", formatGameTitle(title), formatEnforcement(enforcement))
			}
		}
		return desc

	case models.StatusPartialBan:
		desc := fmt.Sprintf("The account %s has game-specific restrictions.", accountTitle)
		if len(ban.GameSpecificBans) > 0 {
			desc += "\n\n**Game-Specific Status:**"
			for title, enforcement := range ban.GameSpecificBans {
				desc += fmt.Sprintf("\n• %s: %s", formatGameTitle(title), formatEnforcement(enforcement))
			}
		}
		return desc

	default:
		return fmt.Sprintf("The account %s is currently not banned.", accountTitle)
	}
}

func formatAffectedGames(games string) string {
	if games == "" || games == "All Games" {
		return "All Call of Duty titles"
	}

	gameList := strings.Split(games, ", ")
	formattedList := make([]string, len(gameList))

	for i, game := range gameList {
		formattedList[i] = fmt.Sprintf("• %s", formatGameTitle(game))
	}

	return strings.Join(formattedList, "\n")
}

func formatGameTitle(title string) string {
	replacements := map[string]string{
		"COD:BO6 SP": "Call of Duty: Black Ops 6 (Campaign)",
		"COD:BO6":    "Call of Duty: Black Ops 6 (Multiplayer)",
		"COD:MW3":    "Call of Duty: Modern Warfare III",
		"COD:MW2":    "Call of Duty: Modern Warfare II",
		"COD:WZ":     "Call of Duty: Warzone",
		"COD:WZ2":    "Call of Duty: Warzone 2.0",
		"COD:DMZ":    "Call of Duty: DMZ",
	}

	if formatted, exists := replacements[title]; exists {
		return formatted
	}

	return title
}

func formatEnforcement(enforcement string) string {
	switch enforcement {
	case "PERMANENT":
		return "Permanently Banned 🔴"
	case "UNDER_REVIEW":
		return "Under Review 🟡"
	case "TEMPORARY":
		return "Temporarily Banned 🟠"
	default:
		return enforcement
	}
}

func CreateAccountListEmbed(accounts []models.Account, userID string, page int, totalPages int) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:     fmt.Sprintf("Your Monitored Accounts (Page %d/%d)", page, totalPages),
		Color:     0x00FF00,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if len(accounts) == 0 {
		embed.Description = "You don't have any monitored accounts."
		return embed
	}

	for _, account := range accounts {
		statusEmoji := getStatusEmoji(account.LastStatus)
		fieldValue := fmt.Sprintf("Status: %s %s\nLast Check: %s",
			statusEmoji,
			account.LastStatus,
			time.Unix(account.LastCheck, 0).Format("Jan 2, 15:04"))

		if account.IsExpiredCookie {
			fieldValue += "\n⚠️ **Cookie Expired**"
		}
		if account.IsCheckDisabled {
			fieldValue += "\n🚫 **Checks Disabled**"
		}
		if account.IsRankLocked {
			fieldValue += "\n🔒 **Ranked Play Locked**"
		}

		if len(account.GameSpecificBans) > 0 {
			fieldValue += "\n📋 **Game-Specific Bans:**"
			for title, enforcement := range account.GameSpecificBans {
				fieldValue += fmt.Sprintf("\n  • %s: %s", formatGameTitle(title), formatEnforcement(enforcement))
			}
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s (ID: %d)", account.Title, account.ID),
			Value:  fieldValue,
			Inline: true,
		})
	}

	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: fmt.Sprintf("Total Accounts: %d", len(accounts)),
	}

	return embed
}

func getStatusEmoji(status models.Status) string {
	switch status {
	case models.StatusGood:
		return "✅"
	case models.StatusPermaban:
		return "🚫"
	case models.StatusShadowban:
		return "⚠️"
	case models.StatusTempban:
		return "⏳"
	case models.StatusRankLocked:
		return "🔒"
	case models.StatusPartialBan:
		return "📋"
	case models.StatusInvalidCookie:
		return "🍪"
	default:
		return "❓"
	}
}

func CreateCheckResultEmbed(account models.Account, status models.Status, userSettings models.UserSettings) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:     fmt.Sprintf("%s - Status Check", account.Title),
		Color:     GetColorForStatus(status, account.IsExpiredCookie, account.IsCheckDisabled),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if account.IsCheckDisabled {
		embed.Description = fmt.Sprintf("Checks are disabled for this account.\nReason: %s", account.DisabledReason)
		return embed
	}

	if account.IsExpiredCookie {
		embed.Description = "The SSO cookie for this account has expired. Please update it using the /updateaccount command."
		return embed
	}

	embed.Description = fmt.Sprintf("Current status: **%s**", status)

	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:   "Last Checked",
			Value:  time.Now().Format(time.RFC1123),
			Inline: true,
		},
	}

	if len(account.GameSpecificBans) > 0 {
		var gameDetails []string

		isBo6CampaignShadowban := account.IsCampaignOnlyShadowban
		if !isBo6CampaignShadowban && (status == models.StatusShadowban || status == models.StatusRankLocked) {
			campaignUnderReview := false
			otherUnderReview := false

			for title, enforcement := range account.GameSpecificBans {
				if enforcement == "UNDER_REVIEW" {
					if strings.Contains(title, "BO6 SP") {
						campaignUnderReview = true
					} else {
						otherUnderReview = true
					}
				}
			}

			if campaignUnderReview && !otherUnderReview {
				isBo6CampaignShadowban = true
			}
		}

		for title, enforcement := range account.GameSpecificBans {
			gameDetails = append(gameDetails, fmt.Sprintf("**%s**: %s", formatGameTitle(title), formatEnforcement(enforcement)))
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Game-Specific Status",
			Value:  strings.Join(gameDetails, "\n"),
			Inline: false,
		})

		if isBo6CampaignShadowban {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "📋 BO6 Campaign Restriction",
				Value:  "This appears to be a BO6 campaign-specific shadowban. The 'Under Review' status for campaign may remain even after shadowban effects disappear from multiplayer.",
				Inline: false,
			})
		}
	}

	switch status {
	case models.StatusRankLocked:
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "⚠️ Important Notice",
			Value:  "This account is restricted from ranked play only. Regular multiplayer remains available.",
			Inline: false,
		})
	case models.StatusPermaban:
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🚫 Action Required",
			Value:  "Consider removing this account using /removeaccount to free up a slot.",
			Inline: false,
		})
	}

	if isVIP, err := CheckVIPStatus(account.SSOCookie); err == nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "VIP Status",
			Value:  formatVIPStatus(isVIP),
			Inline: true,
		})
	}

	if !account.IsExpiredCookie && account.SSOCookieExpiration > 0 {
		timeUntilExpiration, err := CheckSSOCookieExpiration(account.SSOCookieExpiration)
		if err == nil {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Cookie Expires",
				Value:  FormatDuration(timeUntilExpiration),
				Inline: true,
			})
		}
	}

	return embed
}

func CreateErrorEmbed(title string, errorMessage string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: errorMessage,
		Color:       0xFF0000, // Red
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "If this error persists, please contact support",
		},
	}
}

func CreateSuccessEmbed(title string, message string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: message,
		Color:       0x00FF00, // Green
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

func CreateInfoEmbed(title string, message string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: message,
		Color:       0x0099FF, // Blue
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}
