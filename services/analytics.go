package services

import (
	"time"

	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
)

func LogCommandExecution(commandName, userID, guildID string, success bool, responseTimeMs int64, errorDetails string) {
	shardMgr := GetAppShardManager()

	metadata := map[string]interface{}{
		"success":          success,
		"response_time_ms": responseTimeMs,
		"command_name":     commandName,
	}

	if errorDetails != "" {
		metadata["error_details"] = errorDetails
	}

	status := "success"
	if !success {
		status = "error"
	}

	LogAnalyticsEvent("command_usage", userID, guildID, commandName, status,
		shardMgr.ShardID, shardMgr.InstanceID, metadata)
}

func LogShardEvent(eventType string, metadata map[string]interface{}) {
	shardMgr := GetAppShardManager()

	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	metadata["shard_id"] = shardMgr.ShardID
	metadata["total_shards"] = shardMgr.TotalShards
	metadata["is_leader"] = shardMgr.IsLeader()

	LogAnalyticsEvent(eventType, "", "", "", "info",
		shardMgr.ShardID, shardMgr.InstanceID, metadata)
}

func CleanupOldAnalyticsData(retentionDays int) error {
	if retentionDays <= 0 {
		logger.Log.Warn("Invalid retention days for analytics cleanup, skipping")
		return nil
	}

	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	var oldRecordsCount int64
	if err := database.DB.Model(&models.Analytics{}).
		Where("timestamp < ?", cutoffDate).Count(&oldRecordsCount).Error; err != nil {
		return err
	}

	if oldRecordsCount == 0 {
		logger.Log.Debug("No old analytics data to clean up")
		return nil
	}

	logger.Log.Infof("Cleaning up %d analytics records older than %d days", oldRecordsCount, retentionDays)

	result := database.DB.Where("timestamp < ?", cutoffDate).Delete(&models.Analytics{})
	if result.Error != nil {
		return result.Error
	}

	logger.Log.Infof("Successfully cleaned up %d old analytics records", result.RowsAffected)

	shardMgr := GetAppShardManager()
	metadata := map[string]interface{}{
		"cleaned_records": result.RowsAffected,
		"retention_days":  retentionDays,
		"cutoff_date":     cutoffDate.Format("2006-01-02"),
	}

	LogAnalyticsEvent("analytics_cleanup", "", "", "", "success",
		shardMgr.ShardID, shardMgr.InstanceID, metadata)

	return nil
}

func GetAnalyticsStats(days int) (map[string]interface{}, error) {
	if days <= 0 {
		days = 7
	}

	startDate := time.Now().AddDate(0, 0, -days)

	stats := map[string]interface{}{
		"period_days": days,
		"start_date":  startDate.Format("2006-01-02"),
		"end_date":    time.Now().Format("2006-01-02"),
	}

	var totalEvents int64
	if err := database.DB.Model(&models.Analytics{}).
		Where("timestamp >= ?", startDate).Count(&totalEvents).Error; err != nil {
		return stats, err
	}
	stats["total_events"] = totalEvents

	var eventTypes []struct {
		Type  string `json:"type"`
		Count int    `json:"count"`
	}
	if err := database.DB.Model(&models.Analytics{}).
		Select("type, COUNT(*) as count").
		Where("timestamp >= ?", startDate).
		Group("type").
		Scan(&eventTypes).Error; err != nil {
		return stats, err
	}
	stats["events_by_type"] = eventTypes

	var shardDistribution []struct {
		ShardID int `json:"shard_id"`
		Count   int `json:"count"`
	}
	if err := database.DB.Model(&models.Analytics{}).
		Select("shard_id, COUNT(*) as count").
		Where("timestamp >= ?", startDate).
		Group("shard_id").
		Scan(&shardDistribution).Error; err != nil {
		return stats, err
	}
	stats["events_by_shard"] = shardDistribution

	var successCount int64
	if err := database.DB.Model(&models.Analytics{}).
		Where("timestamp >= ? AND success = ?", startDate, true).Count(&successCount).Error; err != nil {
		return stats, err
	}

	successRate := float64(0)
	if totalEvents > 0 {
		successRate = float64(successCount) / float64(totalEvents) * 100
	}
	stats["success_rate_percent"] = successRate

	return stats, nil
}

func GetDailyStats(date string) (map[string]interface{}, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	stats := map[string]interface{}{
		"date": date,
	}

	var commandCount int64
	if err := database.DB.Model(&models.Analytics{}).
		Where("type = ? AND day = ?", "command_usage", date).
		Count(&commandCount).Error; err != nil {
		return stats, err
	}
	stats["command_count"] = commandCount

	var accountCheckCount int64
	if err := database.DB.Model(&models.Analytics{}).
		Where("type = ? AND day = ?", "account_check", date).
		Count(&accountCheckCount).Error; err != nil {
		return stats, err
	}
	stats["account_check_count"] = accountCheckCount

	var statusChangeCount int64
	if err := database.DB.Model(&models.Analytics{}).
		Where("type = ? AND day = ?", "status_change", date).
		Count(&statusChangeCount).Error; err != nil {
		return stats, err
	}
	stats["status_change_count"] = statusChangeCount

	var uniqueUsers int64
	if err := database.DB.Model(&models.Analytics{}).
		Where("day = ?", date).
		Distinct("user_id").
		Count(&uniqueUsers).Error; err != nil {
		return stats, err
	}
	stats["unique_users"] = uniqueUsers

	var successfulEvents int64
	if err := database.DB.Model(&models.Analytics{}).
		Where("day = ? AND success = ?", date, true).
		Count(&successfulEvents).Error; err != nil {
		return stats, err
	}

	var totalEvents int64
	if err := database.DB.Model(&models.Analytics{}).
		Where("day = ?", date).
		Count(&totalEvents).Error; err != nil {
		return stats, err
	}

	successRate := float64(0)
	if totalEvents > 0 {
		successRate = float64(successfulEvents) / float64(totalEvents) * 100
	}
	stats["success_rate_percent"] = successRate
	stats["total_events"] = totalEvents
	stats["successful_events"] = successfulEvents

	return stats, nil
}

func GetShardAnalytics(days int) (map[string]interface{}, error) {
	shardMgr := GetAppShardManager()

	if days <= 0 {
		days = 7
	}

	startDate := time.Now().AddDate(0, 0, -days)

	stats := map[string]interface{}{
		"shard_id":    shardMgr.ShardID,
		"instance_id": shardMgr.InstanceID,
		"period_days": days,
		"start_date":  startDate.Format("2006-01-02"),
		"end_date":    time.Now().Format("2006-01-02"),
	}

	var totalEvents int64
	if err := database.DB.Model(&models.Analytics{}).
		Where("timestamp >= ? AND shard_id = ?", startDate, shardMgr.ShardID).
		Count(&totalEvents).Error; err != nil {
		return stats, err
	}
	stats["total_events"] = totalEvents

	var eventTypes []struct {
		Type  string `json:"type"`
		Count int    `json:"count"`
	}
	if err := database.DB.Model(&models.Analytics{}).
		Select("type, COUNT(*) as count").
		Where("timestamp >= ? AND shard_id = ?", startDate, shardMgr.ShardID).
		Group("type").
		Scan(&eventTypes).Error; err != nil {
		return stats, err
	}
	stats["events_by_type"] = eventTypes

	return stats, nil
}
