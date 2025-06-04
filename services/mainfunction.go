package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bwmarrin/discordgo"
)

// TODO: Remove old constants uses cfg now
/*
const (
	_            = 5  // maxConsecutiveErrors (use cfg.ErrorHandling.MaxConsecutiveErrors)
	_            = 24 // cookieExpirationWarning (use cfg.ErrorHandling.CookieExpirationWarningHours)
)
*/

var (
	DBMutex sync.Mutex
)

func init() {}

func InitializeServices() {
	cfg := configuration.Get()
	initDefaultSettings()
	logger.Log.Infof("Loaded rate limits and intervals: CHECK_INTERVAL=%d, NOTIFICATION_INTERVAL=%.2f, "+
		"COOLDOWN_DURATION=%.2f, SLEEP_DURATION=%d, COOKIE_CHECK_INTERVAL_PERMABAN=%.2f, "+
		"STATUS_CHANGE_COOLDOWN=%.2f, GLOBAL_NOTIFICATION_COOLDOWN=%.2f, COOKIE_EXPIRATION_WARNING=%.2f, "+
		"TEMP_BAN_UPDATE_INTERVAL=%.2f, CHECK_NOW_RATE_LIMIT=%v, DEFAULT_RATE_LIMIT=%v",
		cfg.Intervals.Check, cfg.Intervals.Notification, cfg.Intervals.Cooldown, cfg.Intervals.Sleep,
		cfg.Intervals.PermaBanCheck, cfg.Intervals.StatusChange, cfg.Intervals.GlobalNotification,
		cfg.Intervals.CookieExpiration, cfg.Intervals.TempBanUpdate, cfg.RateLimits.CheckNow, cfg.RateLimits.Default)
}

func CheckAccounts(s *discordgo.Session) {
	logger.Log.Info("Starting periodic account check")
	shardMgr := GetAppShardManager()
	if !shardMgr.Initialized {
		if err := shardMgr.Initialize(); err != nil {
			logger.Log.WithError(err).Error("Failed to initialize app shard manager")
			return
		}
	}

	var accounts []models.Account
	if err := database.DB.Where("is_check_disabled = ? AND is_expired_cookie = ?", false, false).Find(&accounts).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to fetch accounts from database")
		return
	}

	accountsByUser := make(map[string][]models.Account)
	for _, account := range accounts {
		accountsByUser[account.UserID] = append(accountsByUser[account.UserID], account)
	}

	processedCount := 0
	skippedCount := 0
	start := time.Now()

	for userID, userAccounts := range accountsByUser {
		if !shardMgr.IsUserAssignedToShard(userID) {
			skippedCount++
			continue
		}
		processUserAccounts(s, userID, userAccounts)
		processedCount++
	}

	duration := time.Since(start).Seconds()
	logger.Log.Infof("Completed periodic account check: processed %d users, skipped %d users in %.2f seconds",
		processedCount, skippedCount, duration)

	if err := updateShardStats(shardMgr.InstanceID, processedCount, duration); err != nil {
		logger.Log.WithError(err).Error("Failed to update shard stats")
	}
}

func updateShardStats(instanceID string, processedUsers int, durationSec float64) error {
	stats := map[string]interface{}{
		"last_check_time": time.Now(),
		"processed_users": processedUsers,
		"duration_sec":    durationSec,
	}
	statJSON, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	return database.DB.Model(&models.ShardInfo{}).
		Where("instance_id = ?", instanceID).
		Update("stats", string(statJSON)).Error
}

func HandleStatusChange(s *discordgo.Session, account models.Account, newStatus models.Status, userSettings *models.UserSettings) {
	if account.IsPermabanned && newStatus == models.StatusPermaban {
		if account.LastNotification != 0 {
			logger.Log.Debugf("Account %s already notified of permaban, skipping notification", account.Title)
			return
		}
	}

	statusChanged := account.LastStatus != newStatus && account.LastStatus != models.StatusUnknown
	gameSpecificChanged := false

	var latestAccount models.Account
	if err := database.DB.First(&latestAccount, account.ID).Error; err == nil {
		account = latestAccount
		if len(account.GameSpecificBans) > 0 {
			gameSpecificChanged = true
		}
	}

	if statusChanged || gameSpecificChanged {
		logger.Log.Debugf("Status change detected for account %s: %s -> %s (Game-specific changes: %v)",
			account.Title, account.LastStatus, newStatus, gameSpecificChanged)

		DBMutex.Lock()
		defer DBMutex.Unlock()

		now := time.Now()
		previousStatus := account.LastStatus
		account.LastStatus = newStatus
		account.LastStatusChange = now.Unix()
		account.IsPermabanned = newStatus == models.StatusPermaban
		account.IsShadowbanned = newStatus == models.StatusShadowban || newStatus == models.StatusRankLocked
		account.IsTempbanned = newStatus == models.StatusTempban
		account.LastSuccessfulCheck = now
		account.ConsecutiveErrors = 0

		if err := database.DB.Save(&account).Error; err != nil {
			logger.Log.WithError(err).Error("Failed to update account status")
			return
		}

		statusLog := models.Ban{
			AccountID:        account.ID,
			Status:           newStatus,
			PreviousStatus:   previousStatus,
			LogType:          "status_change",
			Message:          fmt.Sprintf("Status changed from %s to %s", previousStatus, newStatus),
			Timestamp:        now,
			Initiator:        "auto_check",
			GameSpecificBans: account.GameSpecificBans,
		}

		if len(account.GameSpecificBans) > 0 {
			var games []string
			for title := range account.GameSpecificBans {
				games = append(games, title)
			}
			statusLog.AffectedGames = strings.Join(games, ", ")
		} else {
			statusLog.AffectedGames = getAffectedGames(account.SSOCookie)
		}

		if newStatus == models.StatusTempban {
			statusLog.TempBanDuration = calculateBanDuration(time.Now().Add(24 * time.Hour))
		}

		if err := database.DB.Create(&statusLog).Error; err != nil {
			logger.Log.WithError(err).Error("Failed to create status log")
		}

		ban := models.Ban{
			AccountID:        account.ID,
			Status:           newStatus,
			GameSpecificBans: account.GameSpecificBans,
		}

		if newStatus == models.StatusTempban {
			ban.TempBanDuration = calculateBanDuration(time.Now().Add(24 * time.Hour))
			ban.AffectedGames = statusLog.AffectedGames
		} else if newStatus != models.StatusGood {
			ban.AffectedGames = statusLog.AffectedGames
		}

		if err := database.DB.Create(&ban).Error; err != nil {
			logger.Log.WithError(err).Error("Failed to create ban record")
		} else {
			logger.Log.Infof("Created ban record for account %s: %s -> %s", account.Title, previousStatus, newStatus)
		}

		LogStatusChange(account.ID, account.UserID, newStatus, previousStatus)

		embed := createStatusChangeEmbed(account, newStatus, previousStatus, ban)
		notificationType := getNotificationType(newStatus)
		err := SendNotification(s, account, embed, fmt.Sprintf("<@%s>", account.UserID), notificationType)
		if err != nil {
			logger.Log.WithError(err).Errorf("Failed to send status update message for account %s", account.Title)
		} else {
			userSettings.LastStatusChangeNotification = now
			if err := database.DB.Save(userSettings).Error; err != nil {
				logger.Log.WithError(err).Errorf("Failed to update LastStatusChangeNotification for user %s", account.UserID)
			}
		}

		switch newStatus {
		case models.StatusTempban:
			go ScheduleTempBanNotification(s, account, ban.TempBanDuration)
		case models.StatusPermaban:
			handlePermaBanNotification(s, account, ban)
			account.LastNotification = now.Unix()
		case models.StatusRankLocked:
			handleRankLockedNotification(s, account, ban)
		}

		if err := database.DB.Save(&account).Error; err != nil {
			logger.Log.WithError(err).Error("Failed to save final account status")
		}
	}
}

func createStatusChangeEmbed(account models.Account, newStatus models.Status, previousStatus models.Status, ban models.Ban) *discordgo.MessageEmbed {
	now := time.Now()
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s - %s", account.Title, EmbedTitleFromStatus(newStatus)),
		Description: GetStatusDescription(newStatus, account.Title, ban),
		Color:       GetColorForStatus(newStatus, account.IsExpiredCookie, account.IsCheckDisabled),
		Fields:      getStatusFields(account, newStatus, ban),
		Timestamp:   now.Format(time.RFC3339),
	}

	if previousStatus != models.StatusUnknown {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Previous Status",
			Value:  string(previousStatus),
			Inline: true,
		})
	}

	if len(account.GameSpecificBans) > 0 {
		var gameDetails []string

		isBo6CampaignShadowban := account.IsCampaignOnlyShadowban
		if !isBo6CampaignShadowban && (newStatus == models.StatusShadowban || newStatus == models.StatusRankLocked) {
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
				Name:   "📋 BO6 Campaign Note",
				Value:  "This appears to be a BO6 campaign-specific shadowban. The game will show 'Under Review' for campaign permanently, but multiplayer restrictions will be lifted after the normal shadowban period.",
				Inline: false,
			})
		}
	}

	if newStatus == models.StatusRankLocked {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "⚠️ Ranked Play Restriction",
			Value:  "This account is restricted from ranked play only. Regular multiplayer and other modes are available.",
			Inline: false,
		})
	}

	return embed
}

func handleRankLockedNotification(s *discordgo.Session, account models.Account, ban models.Ban) {
	isBo6CampaignShadowban := account.IsCampaignOnlyShadowban
	if !isBo6CampaignShadowban && len(account.GameSpecificBans) > 0 {
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

	description := "Your account has a persistent ranked play restriction. " +
		"You can still play regular multiplayer and other game modes, but ranked play is disabled."

	if isBo6CampaignShadowban {
		description = "Your account has a BO6 campaign-specific restriction. " +
			"This type of restriction shows as 'Under Review' permanently for campaign mode, " +
			"but the shadowban effects on multiplayer will disappear after the normal timeframe."
	} else {
		description += "\n\nThis is typically a permanent restriction that remains even after shadowbans are lifted."
	}

	rankLockedEmbed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s - Ranked Play Restriction", account.Title),
		Description: description,
		Color:       GetColorForStatus(models.StatusRankLocked, false, false),
		Timestamp:   time.Now().Format(time.RFC3339),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name: "Restriction Type",
				Value: func() string {
					if isBo6CampaignShadowban {
						return "Campaign Only"
					} else {
						return "Ranked Play Only"
					}
				}(),
				Inline: true,
			},
			{
				Name:   "Other Modes",
				Value:  "Available ✅",
				Inline: true,
			},
		},
	}

	if len(account.GameSpecificBans) > 0 {
		var gameDetails []string
		for title, enforcement := range account.GameSpecificBans {
			gameDetails = append(gameDetails, fmt.Sprintf("**%s**: %s", formatGameTitle(title), formatEnforcement(enforcement)))
		}
		rankLockedEmbed.Fields = append(rankLockedEmbed.Fields, &discordgo.MessageEmbedField{
			Name:   "Detailed Status",
			Value:  strings.Join(gameDetails, "\n"),
			Inline: false,
		})
	}

	if err := SendNotification(s, account, rankLockedEmbed, "", "rank_locked_notice"); err != nil {
		logger.Log.WithError(err).Error("Failed to send rank locked notice")
	}
}

func getAffectedGames(ssoCookie string) string {
	cfg := configuration.Get()
	req, err := http.NewRequest("GET", cfg.API.CheckEndpoint, nil)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to create request for affected games")
		return "All Games"
	}

	headers := GenerateHeaders(ssoCookie)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := DoRequest(req)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to get affected games")
		return "All Games"
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Log.WithError(err).Error("Failed to close response body")
		}
	}(resp.Body)

	var data struct {
		Bans []struct {
			Title string `json:"title"`
		} `json:"bans"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		logger.Log.WithError(err).Error("Failed to decode affected games response")
		return "All Games"
	}

	affectedGames := make(map[string]bool)
	for _, ban := range data.Bans {
		affectedGames[ban.Title] = true
	}

	var games []string
	for game := range affectedGames {
		games = append(games, game)
	}

	if len(games) == 0 {
		return "All Games"
	}

	return strings.Join(games, ", ")
}

func getStatusFields(account models.Account, status models.Status, ban models.Ban) []*discordgo.MessageEmbedField {
	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Account Status",
			Value:  string(status),
			Inline: true,
		},
		{
			Name:   "Last Checked",
			Value:  time.Unix(account.LastCheck, 0).Format(time.RFC1123),
			Inline: true,
		},
	}

	if ban.AffectedGames != "" && status != models.StatusGood {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Affected Games",
			Value:  ban.AffectedGames,
			Inline: false,
		})
	}

	if isVIP, err := CheckVIPStatus(account.SSOCookie); err == nil {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "VIP Status",
			Value:  formatVIPStatus(isVIP),
			Inline: true,
		})
	}

	if !account.IsExpiredCookie {
		timeUntilExpiration, err := CheckSSOCookieExpiration(account.SSOCookieExpiration)
		if err == nil {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "Cookie Expires",
				Value:  FormatDuration(timeUntilExpiration),
				Inline: true,
			})
		}
	}

	if account.Created > 0 {
		creationDate := time.Unix(account.Created, 0)
		accountAge := time.Since(creationDate)
		years := int(accountAge.Hours() / 24 / 365)
		months := int(accountAge.Hours()/24/30.44) % 12
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Account Age",
			Value:  fmt.Sprintf("%d years, %d months", years, months),
			Inline: true,
		})
	}

	switch status {
	case models.StatusPermaban:
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Ban Type",
			Value:  "Permanent",
			Inline: true,
		})
	case models.StatusTempban:
		var latestBan models.Ban
		if err := database.DB.Where("account_id = ?", account.ID).
			Order("created_at DESC").
			First(&latestBan).Error; err == nil && latestBan.TempBanDuration != "" {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "Ban Duration",
				Value:  latestBan.TempBanDuration,
				Inline: true,
			})
		}
	case models.StatusShadowban:
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Review Status",
			Value:  "Account Under Review",
			Inline: true,
		})
	case models.StatusRankLocked:
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Restriction",
			Value:  "Ranked Play Disabled",
			Inline: true,
		})
	}

	if account.ConsecutiveErrors > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Check Errors",
			Value:  fmt.Sprintf("%d consecutive errors", account.ConsecutiveErrors),
			Inline: true,
		})
	}

	return fields
}

func disableAccount(s *discordgo.Session, account models.Account, reason string) {
	account.IsCheckDisabled = true
	account.DisabledReason = reason
	account.ConsecutiveErrors = 0

	if err := database.DB.Save(&account).Error; err != nil {
		logger.Log.WithError(err).Errorf("Failed to disable account %s", account.Title)
		return
	}

	logger.Log.Infof("Account %s has been disabled. Reason: %s", account.Title, reason)
	NotifyUserAboutDisabledAccount(s, account, reason)
}

func handlePermaBanNotification(s *discordgo.Session, account models.Account, ban models.Ban) {
	permaBanEmbed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("%s - Permanent Ban Detected", account.Title),
		Description: "This account has been permanently banned. The account will no longer be checked automatically.\n" +
			"It's recommended to remove it from monitoring using the /removeaccount command to free up your account slot.",
		Color:     GetColorForStatus(models.StatusPermaban, false, false),
		Timestamp: time.Now().Format(time.RFC3339),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Account Status",
				Value:  "Permanently Banned",
				Inline: true,
			},
			{
				Name:   "Suggested Action",
				Value:  "Remove account using /removeaccount",
				Inline: true,
			},
		},
	}

	if ban.AffectedGames != "" {
		permaBanEmbed.Fields = append(permaBanEmbed.Fields, &discordgo.MessageEmbedField{
			Name:   "Affected Games",
			Value:  ban.AffectedGames,
			Inline: false,
		})
	}

	if len(account.GameSpecificBans) > 0 {
		var gameDetails []string
		for title, enforcement := range account.GameSpecificBans {
			gameDetails = append(gameDetails, fmt.Sprintf("**%s**: %s", formatGameTitle(title), formatEnforcement(enforcement)))
		}
		permaBanEmbed.Fields = append(permaBanEmbed.Fields, &discordgo.MessageEmbedField{
			Name:   "Game-Specific Details",
			Value:  strings.Join(gameDetails, "\n"),
			Inline: false,
		})
	}

	if err := SendNotification(s, account, permaBanEmbed, "", "permaban_notice"); err != nil {
		logger.Log.WithError(err).Error("Failed to send permaban notice")
	}
}

func handleShadowBanNotification(s *discordgo.Session, account models.Account, ban models.Ban) {
	isBo6CampaignShadowban := account.IsCampaignOnlyShadowban
	if !isBo6CampaignShadowban && len(account.GameSpecificBans) > 0 {
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

	description := fmt.Sprintf("Your account has been placed under review (shadowban). " +
		"This typically means your account is being investigated.")

	if isBo6CampaignShadowban {
		description += "\n\n**BO6 Campaign Note:**\nThis appears to be a campaign-specific shadowban. The game will continue to show 'Under Review' for campaign mode, but multiplayer restrictions will be lifted after the normal shadowban period."
	}

	shadowBanEmbed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s - Account Under Review", account.Title),
		Description: description,
		Color:       GetColorForStatus(models.StatusShadowban, false, false),
		Timestamp:   time.Now().Format(time.RFC3339),
		Fields:      getStatusFields(account, models.StatusShadowban, ban),
	}

	if len(account.GameSpecificBans) > 0 {
		var gameDetails []string
		for title, enforcement := range account.GameSpecificBans {
			gameDetails = append(gameDetails, fmt.Sprintf("**%s**: %s", formatGameTitle(title), formatEnforcement(enforcement)))
		}

		shadowBanEmbed.Fields = append(shadowBanEmbed.Fields, &discordgo.MessageEmbedField{
			Name:   "Game-Specific Status",
			Value:  strings.Join(gameDetails, "\n"),
			Inline: false,
		})
	}

	if err := SendNotification(s, account, shadowBanEmbed, "", "shadowban_notice"); err != nil {
		logger.Log.WithError(err).Error("Failed to send shadowban notice")
	}
}

func ScheduleTempBanNotification(s *discordgo.Session, account models.Account, duration string) {
	parts := strings.Split(duration, ",")
	if len(parts) != 2 {
		logger.Log.Errorf("Invalid duration format for account %s: %s", account.Title, duration)
		return
	}

	days, _ := strconv.Atoi(strings.TrimSpace(strings.Split(parts[0], " ")[0]))
	hours, _ := strconv.Atoi(strings.TrimSpace(strings.Split(parts[1], " ")[0]))
	sleepDuration := time.Duration(days)*24*time.Hour + time.Duration(hours)*time.Hour

	for remainingTime := sleepDuration; remainingTime > 0; remainingTime -= 24 * time.Hour {
		if remainingTime > 24*time.Hour {
			time.Sleep(24 * time.Hour)
		} else {
			time.Sleep(remainingTime)
		}

		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s - Temporary Ban Update", account.Title),
			Description: fmt.Sprintf("Your account is still temporarily banned. Remaining time: %v", remainingTime),
			Color:       GetColorForStatus(models.StatusTempban, false, account.IsCheckDisabled),
			Timestamp:   time.Now().Format(time.RFC3339),
		}

		err := SendNotification(s, account, embed, "", "temp_ban_update")
		if err != nil {
			logger.Log.WithError(err).Errorf("Failed to send temporary ban update for account %s", account.Title)
		}
	}

	result, err := CheckAccount(account.SSOCookie, account.UserID, "")
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to check account %s after temporary ban duration", account.Title)
		return
	}

	var embed *discordgo.MessageEmbed
	if result == models.StatusGood {
		embed = &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s - Temporary Ban Lifted", account.Title),
			Description: fmt.Sprintf("The temporary ban for account %s has been lifted. The account is now in good standing.", account.Title),
			Color:       GetColorForStatus(result, false, account.IsCheckDisabled),
			Timestamp:   time.Now().Format(time.RFC3339),
		}
	} else if result == models.StatusPermaban {
		embed = &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s - Temporary Ban Escalated", account.Title),
			Description: fmt.Sprintf("The temporary ban for account %s has been escalated to a permanent ban.", account.Title),
			Color:       GetColorForStatus(result, false, account.IsCheckDisabled),
			Timestamp:   time.Now().Format(time.RFC3339),
		}
	} else {
		embed = &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s - Temporary Ban Update", account.Title),
			Description: fmt.Sprintf("The temporary ban for account %s is still in effect. Current status: %s", account.Title, result),
			Color:       GetColorForStatus(result, false, account.IsCheckDisabled),
			Timestamp:   time.Now().Format(time.RFC3339),
		}
	}

	err = SendNotification(s, account, embed, fmt.Sprintf("<@%s>", account.UserID), "temp_ban_update")
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to send temporary ban update message for account %s", account.Title)
	}
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

func calculateBanDuration(endTime time.Time) string {
	duration := time.Until(endTime)
	if duration < 0 {
		return "Expired"
	}

	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24

	if days > 0 {
		return fmt.Sprintf("%d days, %d hours", days, hours)
	}
	return fmt.Sprintf("%d hours", hours)
}
