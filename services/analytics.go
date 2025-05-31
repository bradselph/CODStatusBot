package services

import (
	"fmt"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
)

func LogCommandExecution(commandName, userID, guildID string, success bool, responseTimeMs int64, errorDetails string) {
	shardMgr := GetAppShardManager()
	shardID := 0
	instanceID := ""

	if shardMgr.Initialized {
		shardID = shardMgr.ShardID
		instanceID = shardMgr.InstanceID
	}

	analytics := models.Analytics{
		Type:           "command",
		UserID:         userID,
		GuildID:        guildID,
		CommandName:    commandName,
		Success:        success,
		ResponseTimeMs: responseTimeMs,
		ErrorDetails:   errorDetails,
		Timestamp:      time.Now(),
		Day:            time.Now().Format("2006-01-02"),
		ShardID:        shardID,
		InstanceID:     instanceID,
	}

	if err := database.DB.Create(&analytics).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to log command analytics")
	}

	updateCommandStatistics(commandName, success, responseTimeMs)
}

func LogAccountCheck(accountID uint, userID string, status models.Status, success bool,
	captchaProvider string, captchaCost float64, responseTimeMs int64) {
	shardMgr := GetAppShardManager()
	shardID := 0
	instanceID := ""

	if shardMgr.Initialized {
		shardID = shardMgr.ShardID
		instanceID = shardMgr.InstanceID
	}

	analytics := models.Analytics{
		Type:            "account_check",
		UserID:          userID,
		AccountID:       accountID,
		Status:          string(status),
		Success:         success,
		ResponseTimeMs:  responseTimeMs,
		CaptchaProvider: captchaProvider,
		CaptchaCost:     captchaCost,
		Timestamp:       time.Now(),
		Day:             time.Now().Format("2006-01-02"),
		ShardID:         shardID,
		InstanceID:      instanceID,
	}

	if err := database.DB.Create(&analytics).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to log account check analytics")
	}

	updateBotStatistics("account_check", success, responseTimeMs)
}

func LogStatusChange(accountID uint, userID string, status models.Status,
	previousStatus models.Status) {
	shardMgr := GetAppShardManager()
	shardID := 0
	instanceID := ""

	if shardMgr.Initialized {
		shardID = shardMgr.ShardID
		instanceID = shardMgr.InstanceID
	}

	analytics := models.Analytics{
		Type:           "status_change",
		UserID:         userID,
		AccountID:      accountID,
		Status:         string(status),
		PreviousStatus: string(previousStatus),
		Success:        true,
		Timestamp:      time.Now(),
		Day:            time.Now().Format("2006-01-02"),
		ShardID:        shardID,
		InstanceID:     instanceID,
	}

	if err := database.DB.Create(&analytics).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to log status change analytics")
	}

	updateBotStatistics("status_change", true, 0)
}

func LogNotification(userID string, accountID uint, notificationType string, success bool) {
	shardMgr := GetAppShardManager()
	shardID := 0
	instanceID := ""

	if shardMgr.Initialized {
		shardID = shardMgr.ShardID
		instanceID = shardMgr.InstanceID
	}

	analytics := models.Analytics{
		Type:        "notification",
		UserID:      userID,
		AccountID:   accountID,
		CommandName: notificationType,
		Success:     success,
		Timestamp:   time.Now(),
		Day:         time.Now().Format("2006-01-02"),
		ShardID:     shardID,
		InstanceID:  instanceID,
	}

	if err := database.DB.Create(&analytics).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to log notification analytics")
	}

	updateBotStatistics("notification", success, 0)
}

func updateBotStatistics(eventType string, success bool, responseTimeMs int64) {
	tableExists := database.DB.Migrator().HasTable(&models.BotStatistics{})
	if !tableExists {
		logger.Log.Warn("Bot statistics table doesn't exist yet - skipping update")
		return
	}

	today := time.Now().Format("2006-01-02")
	todayTime, _ := time.Parse("2006-01-02", today)

	var stats models.BotStatistics
	stats.Date = todayTime

	result := database.DB.Where("date = ?", todayTime).FirstOrCreate(&stats)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Failed to get/create bot statistics")
		return
	}

	switch eventType {
	case "command":
		stats.CommandsUsed++
	case "account_check":
		stats.AccountsChecked++
		if !success {
			stats.CaptchaErrors++
		} else {
			stats.CaptchaUsed++
		}

		if responseTimeMs > 0 {
			if stats.AverageCheckTime == 0 {
				stats.AverageCheckTime = float64(responseTimeMs)
			} else {
				stats.AverageCheckTime = (stats.AverageCheckTime*float64(stats.AccountsChecked-1) + float64(responseTimeMs)) / float64(stats.AccountsChecked)
			}
		}
	case "status_change":
		stats.StatusChanges++
	}

	var activeUsers int64
	database.DB.Model(&models.Analytics{}).
		Where("day = ?", today).
		Distinct("user_id").
		Count(&activeUsers)
	stats.ActiveUsers = int(activeUsers)

	if err := database.DB.Save(&stats).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to update bot statistics")
	}
}

func updateCommandStatistics(commandName string, success bool, responseTimeMs int64) {
	tableExists := database.DB.Migrator().HasTable(&models.CommandStatistics{})
	if !tableExists {
		logger.Log.Warn("Command statistics table doesn't exist yet - skipping update")
		return
	}

	today := time.Now().Format("2006-01-02")
	todayTime, _ := time.Parse("2006-01-02", today)

	var stats models.CommandStatistics
	stats.CommandName = commandName
	stats.Date = todayTime

	result := database.DB.Where("date = ? AND command_name = ?", todayTime, commandName).FirstOrCreate(&stats)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Failed to get/create command statistics")
		return
	}

	stats.UsageCount++
	if success {
		stats.SuccessCount++
	} else {
		stats.ErrorCount++
	}

	if responseTimeMs > 0 {
		if stats.AverageTimeMs == 0 {
			stats.AverageTimeMs = float64(responseTimeMs)
		} else {
			stats.AverageTimeMs = (stats.AverageTimeMs*float64(stats.UsageCount-1) + float64(responseTimeMs)) / float64(stats.UsageCount)
		}
	}

	if err := database.DB.Save(&stats).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to update command statistics")
	}
}

func GetDailyStats(day string) (map[string]interface{}, error) {
	var stats struct {
		CommandCount      int64
		AccountCheckCount int64
		StatusChangeCount int64
		NotificationCount int64
		UniqueUsers       int64
		SuccessRate       float64
		AvgResponseTime   float64
	}

	if err := database.DB.Model(&models.Analytics{}).
		Where("day = ? AND type = ?", day, "command").
		Count(&stats.CommandCount).Error; err != nil {
		return nil, err
	}

	if err := database.DB.Model(&models.Analytics{}).
		Where("day = ? AND type = ?", day, "account_check").
		Count(&stats.AccountCheckCount).Error; err != nil {
		return nil, err
	}

	if err := database.DB.Model(&models.Analytics{}).
		Where("day = ? AND type = ?", day, "status_change").
		Count(&stats.StatusChangeCount).Error; err != nil {
		return nil, err
	}

	if err := database.DB.Model(&models.Analytics{}).
		Where("day = ? AND type = ?", day, "notification").
		Count(&stats.NotificationCount).Error; err != nil {
		return nil, err
	}

	database.DB.Model(&models.Analytics{}).
		Where("day = ?", day).
		Distinct("user_id").
		Count(&stats.UniqueUsers)

	var successCount int64
	var totalCount int64
	database.DB.Model(&models.Analytics{}).
		Where("day = ?", day).
		Count(&totalCount)
	database.DB.Model(&models.Analytics{}).
		Where("day = ? AND success = ?", day, true).
		Count(&successCount)

	if totalCount > 0 {
		stats.SuccessRate = float64(successCount) / float64(totalCount) * 100
	}

	database.DB.Model(&models.Analytics{}).
		Where("day = ?", day).
		Select("AVG(response_time_ms) as avg_response_time").
		Scan(&stats.AvgResponseTime)

	var botStats models.BotStatistics
	dayTime, _ := time.Parse("2006-01-02", day)
	if err := database.DB.Where("date = ?", dayTime).First(&botStats).Error; err == nil {
		return map[string]interface{}{
			"date":                day,
			"command_count":       botStats.CommandsUsed,
			"account_check_count": botStats.AccountsChecked,
			"status_change_count": botStats.StatusChanges,
			"captcha_used":        botStats.CaptchaUsed,
			"captcha_errors":      botStats.CaptchaErrors,
			"unique_users":        botStats.ActiveUsers,
			"avg_check_time":      botStats.AverageCheckTime,
		}, nil
	}

	return map[string]interface{}{
		"date":                day,
		"command_count":       stats.CommandCount,
		"account_check_count": stats.AccountCheckCount,
		"status_change_count": stats.StatusChangeCount,
		"notification_count":  stats.NotificationCount,
		"unique_users":        stats.UniqueUsers,
		"success_rate":        stats.SuccessRate,
		"avg_response_time":   stats.AvgResponseTime,
	}, nil
}

func GetUserStats(userID string, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)
	startDateStr := startDate.Format("2006-01-02")

	var stats struct {
		CommandCount      int64
		AccountCheckCount int64
		StatusChangeCount int64
		SuccessRate       float64
		CaptchaCost       float64
	}

	if err := database.DB.Model(&models.Analytics{}).
		Where("user_id = ? AND type = ? AND day >= ?", userID, "command", startDateStr).
		Count(&stats.CommandCount).Error; err != nil {
		return nil, err
	}

	if err := database.DB.Model(&models.Analytics{}).
		Where("user_id = ? AND type = ? AND day >= ?", userID, "account_check", startDateStr).
		Count(&stats.AccountCheckCount).Error; err != nil {
		return nil, err
	}

	if err := database.DB.Model(&models.Analytics{}).
		Where("user_id = ? AND type = ? AND day >= ?", userID, "status_change", startDateStr).
		Count(&stats.StatusChangeCount).Error; err != nil {
		return nil, err
	}

	var successCount int64
	var totalCount int64
	database.DB.Model(&models.Analytics{}).
		Where("user_id = ? AND day >= ?", userID, startDateStr).
		Count(&totalCount)
	database.DB.Model(&models.Analytics{}).
		Where("user_id = ? AND success = ? AND day >= ?", userID, true, startDateStr).
		Count(&successCount)

	if totalCount > 0 {
		stats.SuccessRate = float64(successCount) / float64(totalCount) * 100
	}

	database.DB.Model(&models.Analytics{}).
		Where("user_id = ? AND day >= ?", userID, startDateStr).
		Select("SUM(captcha_cost) as captcha_cost").
		Scan(&stats.CaptchaCost)

	var commands []struct {
		CommandName string `json:"command_name"`
		Count       int64  `json:"count"`
	}

	database.DB.Model(&models.Analytics{}).
		Select("command_name, COUNT(*) as count").
		Where("user_id = ? AND type = ? AND day >= ?", userID, "command", startDateStr).
		Group("command_name").
		Order("count DESC").
		Limit(10).
		Scan(&commands)

	return map[string]interface{}{
		"user_id":             userID,
		"days":                days,
		"command_count":       stats.CommandCount,
		"account_check_count": stats.AccountCheckCount,
		"status_change_count": stats.StatusChangeCount,
		"success_rate":        stats.SuccessRate,
		"captcha_cost":        stats.CaptchaCost,
		"top_commands":        commands,
	}, nil
}

func CleanupOldAnalyticsData(retentionDays int) error {
	cfg := configuration.Get()

	if retentionDays <= 0 {
		retentionDays = cfg.Admin.RetentionDays
	}

	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	cutoffDateStr := cutoffDate.Format("2006-01-02")

	if err := database.DB.Where("day < ?", cutoffDateStr).Delete(&models.Analytics{}).Error; err != nil {
		return fmt.Errorf("failed to clean up old analytics data: %w", err)
	}

	if err := database.DB.Where("date < ?", cutoffDate).Delete(&models.BotStatistics{}).Error; err != nil {
		return fmt.Errorf("failed to clean up old bot statistics: %w", err)
	}

	if err := database.DB.Where("date < ?", cutoffDate).Delete(&models.CommandStatistics{}).Error; err != nil {
		return fmt.Errorf("failed to clean up old command statistics: %w", err)
	}

	logger.Log.Infof("Cleaned up analytics data older than %s", cutoffDateStr)
	return nil
}
