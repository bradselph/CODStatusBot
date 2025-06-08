package services

import (
	"fmt"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bwmarrin/discordgo"
)

func GetUserID(i *discordgo.InteractionCreate) (string, error) {
	if i.Member != nil {
		return i.Member.User.ID, nil
	}
	if i.User != nil {
		return i.User.ID, nil
	}
	return "", fmt.Errorf("no user found in interaction")
}

func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0f seconds", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0f minutes", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1f hours", d.Hours())
	}
	return fmt.Sprintf("%.1f days", d.Hours()/24)
}

func formatVIPStatus(isVIP bool) string {
	if isVIP {
		return "VIP Member"
	}
	return "Standard Member"
}

func formatCheckStatus(isDisabled bool) string {
	if isDisabled {
		return "Disabled"
	}
	return "Enabled"
}

func TrackUserInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	userID, err := GetUserID(i)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to get user ID for context tracking")
		return err
	}

	context := GetInstallContext(i)

	var existingUser models.UserSettings
	userExists := true
	if err := database.DB.Where("user_id = ?", userID).First(&existingUser).Error; err != nil {
		userExists = false
	}

	var userSettings models.UserSettings
	result := database.DB.Where("user_id = ?", userID).FirstOrCreate(&userSettings)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error getting/creating user settings for context tracking")
		return result.Error
	}

	var guildID string
	if context == ServerContext {
		guildID = i.GuildID
	}

	userSettings.UpdateInteractionContext(guildID)

	if userSettings.InstallationType == "server" {
		logger.Log.Infof("User %s interacting in %s context (installed in server: %s)",
			userID, string(context), userSettings.InstallationGuildID)
	} else {
		logger.Log.Infof("User %s interacting in %s context (direct installation)",
			userID, string(context))
	}

	if err := database.DB.Save(&userSettings).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to save user settings after context tracking update")
		return err
	}

	if !userExists && s != nil {
		logger.Log.Infof("Sending notification for new user")
		NotifyNewInstallation(s, string(context))
	}

	return nil
}

func NotifyNewInstallation(s *discordgo.Session, context string) {
	cfg := configuration.Get()
	developerID := cfg.Discord.DeveloperID
	if developerID == "" {
		logger.Log.Error("Developer ID not configured")
		return
	}

	channel, err := s.UserChannelCreate(developerID)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to create DM channel with developer")
		return
	}

	installType := "Direct Installation"
	if context == "server" {
		installType = "Server Installation"
	}

	embed := &discordgo.MessageEmbed{
		Title:       "New Bot Installation",
		Description: "A new user has started using the bot!",
		Color:       0x00FF00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Installation Type",
				Value:  installType,
				Inline: true,
			},
			{
				Name:   "Timestamp",
				Value:  time.Now().Format(time.RFC3339),
				Inline: true,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	_, err = s.ChannelMessageSendEmbed(channel.ID, embed)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to send new installation notification to developer")
	} else {
		logger.Log.Info("New installation notification sent to developer")
	}
}

func GetInstallationStats() (serverCount int64, directCount int64, err error) {
	if err = database.DB.Model(&models.UserSettings{}).
		Where("installation_type = ?", "server").
		Count(&serverCount).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to count server installations")
		return
	}

	if err = database.DB.Model(&models.UserSettings{}).
		Where("installation_type = ?", "direct").
		Count(&directCount).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to count direct installations")
		return
	}

	return
}

func LogInstallationStats(s *discordgo.Session) {
	var stats struct {
		TotalUsers     int64
		ActiveUsers    int64
		TotalAccounts  int64
		ActiveAccounts int64
		TotalGuilds    int64
		UserInstalls   int64
		GuildInstalls  int64
	}

	database.DB.Model(&models.UserSettings{}).Count(&stats.TotalUsers)
	database.DB.Model(&models.UserSettings{}).Where("updated_at > ?", time.Now().Add(-7*24*time.Hour)).Count(&stats.ActiveUsers)
	database.DB.Model(&models.Account{}).Count(&stats.TotalAccounts)
	database.DB.Model(&models.Account{}).Where("is_check_disabled = ? AND is_expired_cookie = ?", false, false).Count(&stats.ActiveAccounts)
	database.DB.Model(&models.UserSettings{}).Where("installation_type = ?", "direct").Count(&stats.UserInstalls)
	database.DB.Model(&models.UserSettings{}).Where("installation_type = ?", "server").Count(&stats.GuildInstalls)

	distinctGuilds := make(map[string]bool)
	var userSettings []models.UserSettings
	if err := database.DB.Select("installation_guild_id").Where("installation_guild_id != ''").Find(&userSettings).Error; err == nil {
		for _, us := range userSettings {
			if us.InstallationGuildID != "" {
				distinctGuilds[us.InstallationGuildID] = true
			}
		}
	}
	stats.TotalGuilds = int64(len(distinctGuilds))

	logger.Log.Infof("Installation Stats - Total Users: %d, Active Users (7d): %d, Total Accounts: %d, Active Accounts: %d, Total Guilds: %d, User Installs: %d, Guild Installs: %d",
		stats.TotalUsers, stats.ActiveUsers, stats.TotalAccounts, stats.ActiveAccounts, stats.TotalGuilds, stats.UserInstalls, stats.GuildInstalls)
}
