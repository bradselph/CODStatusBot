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
		return "Yes"
	}
	return "No"
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

func LogInstallationStats(s interface{}) {
	shardMgr := GetAppShardManager()
	if !shardMgr.IsLeader() {
		return
	}

	var serverInstalls int64
	var directInstalls int64
	var totalUsers int64

	if err := database.DB.Model(&models.UserSettings{}).Count(&totalUsers).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to count total users for installation stats")
		return
	}

	if err := database.DB.Model(&models.UserSettings{}).
		Where("installation_type = ?", "server").Count(&serverInstalls).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to count server installations")
	}

	if err := database.DB.Model(&models.UserSettings{}).
		Where("installation_type = ?", "direct").Count(&directInstalls).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to count direct installations")
	}

	metadata := map[string]interface{}{
		"total_users":     totalUsers,
		"server_installs": serverInstalls,
		"direct_installs": directInstalls,
		"stats_type":      "daily_installation_summary",
	}

	LogAnalyticsEvent("installation_stats", "", "", "", "info",
		shardMgr.ShardID, shardMgr.InstanceID, metadata)

	logger.Log.Infof("Logged installation stats: Total: %d, Server: %d, Direct: %d",
		totalUsers, serverInstalls, directInstalls)
}
