package services

import (
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
)

func CleanupInactiveUsers() {
	shardMgr := GetAppShardManager()
	if !shardMgr.IsLeader() {
		logger.Log.Debug("Skipping user cleanup - not leader shard")
		return
	}

	cfg := configuration.Get()
	inactivePeriod := time.Now().Add(-cfg.Users.InactiveUserPeriod)

	logger.Log.Info("Starting inactive user cleanup process")

	var allInactiveUsers []models.UserSettings
	if err := database.DB.Where("last_guild_interaction < ? AND last_direct_interaction < ?",
		inactivePeriod, inactivePeriod).Find(&allInactiveUsers).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to fetch inactive users")
		return
	}

	logger.Log.Infof("Found %d potentially inactive users to process", len(allInactiveUsers))

	processedCount := 0
	archivedCount := 0
	errorCount := 0

	for _, user := range allInactiveUsers {
		if user.UserID == "" {
			logger.Log.Warn("Skipping user with empty UserID")
			errorCount++
			continue
		}

		user.IsUnreachable = true
		if user.UnreachableSince.IsZero() {
			user.UnreachableSince = time.Now()
		}

		if err := database.DB.Save(&user).Error; err != nil {
			logger.Log.WithError(err).Errorf("Failed to mark user %s as unreachable", user.UserID)
			errorCount++
			continue
		}

		archivedCount++
		processedCount++

		if processedCount%100 == 0 {
			logger.Log.Infof("Processed %d inactive users so far", processedCount)
		}
	}

	logger.Log.Infof("Completed inactive user cleanup: processed %d users, archived %d as unreachable, %d errors",
		processedCount, archivedCount, errorCount)

	cleanupOldUnreachableUsers(cfg)

	if err := CleanupOldShadowbanPeriods(365); err != nil {
		logger.Log.WithError(err).Error("Failed to cleanup old shadowban period data")
	}
}

func cleanupOldUnreachableUsers(cfg *configuration.Config) {
	cutoffTime := time.Now().Add(-cfg.Users.UnreachableResetPeriod)

	var oldUnreachableCount int64
	if err := database.DB.Model(&models.UserSettings{}).
		Where("is_unreachable = ? AND unreachable_since < ?", true, cutoffTime).
		Count(&oldUnreachableCount).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to count old unreachable users")
		return
	}

	if oldUnreachableCount == 0 {
		logger.Log.Debug("No old unreachable users to reset")
		return
	}

	logger.Log.Infof("Resetting %d users that have been unreachable for more than %v",
		oldUnreachableCount, cfg.Users.UnreachableResetPeriod)

	result := database.DB.Model(&models.UserSettings{}).
		Where("is_unreachable = ? AND unreachable_since < ?", true, cutoffTime).
		Updates(map[string]interface{}{
			"is_unreachable":       false,
			"unreachable_since":    time.Time{},
			"message_failures":     0,
			"last_message_failure": time.Time{},
		})

	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Failed to reset old unreachable users")
	} else {
		logger.Log.Infof("Successfully reset %d old unreachable users", result.RowsAffected)
	}
}

func CleanupUsersByShardAssignment() {
	shardMgr := GetAppShardManager()
	if !shardMgr.IsLeader() {
		logger.Log.Debug("Skipping shard assignment cleanup - not leader shard")
		return
	}

	logger.Log.Info("Starting shard assignment cleanup")

	var activeShards []models.ShardInfo
	if err := database.DB.Where("status = 'active'").Find(&activeShards).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to get active shards for cleanup")
		return
	}

	if len(activeShards) == 0 {
		logger.Log.Warn("No active shards found during cleanup")
		return
	}

	totalShards := len(activeShards)
	logger.Log.Infof("Validating user assignments against %d active shards", totalShards)

	var allUsers []models.UserSettings
	if err := database.DB.Select("user_id").Find(&allUsers).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to get users for shard assignment cleanup")
		return
	}

	orphanedUsers := 0
	processedUsers := 0

	for _, user := range allUsers {
		if user.UserID == "" {
			continue
		}

		userShard := getUserShard(user.UserID, totalShards)

		shardExists := false
		for _, activeShard := range activeShards {
			if activeShard.ShardID == userShard {
				shardExists = true
				break
			}
		}

		if !shardExists {
			logger.Log.Debugf("User %s assigned to non-existent shard %d (total: %d)",
				user.UserID, userShard, totalShards)
			orphanedUsers++
		}

		processedUsers++
	}

	logger.Log.Infof("Shard assignment cleanup complete: processed %d users, found %d orphaned users",
		processedUsers, orphanedUsers)

	if orphanedUsers > 0 {
		logger.Log.Warnf("Found %d users assigned to non-existent shards - they may not be processed until shard rebalancing occurs", orphanedUsers)
	}
}

func CleanupOldShardInfo() {
	shardMgr := GetAppShardManager()
	if !shardMgr.IsLeader() {
		logger.Log.Debug("Skipping shard info cleanup - not leader shard")
		return
	}

	cutoffTime := time.Now().Add(-24 * time.Hour)

	var oldShardCount int64
	if err := database.DB.Model(&models.ShardInfo{}).
		Where("status != 'active' AND last_heartbeat < ?", cutoffTime).
		Count(&oldShardCount).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to count old shard records")
		return
	}

	if oldShardCount == 0 {
		logger.Log.Debug("No old shard records to clean up")
		return
	}

	logger.Log.Infof("Cleaning up %d old shard info records", oldShardCount)

	result := database.DB.Where("status != 'active' AND last_heartbeat < ?", cutoffTime).
		Delete(&models.ShardInfo{})

	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Failed to clean up old shard records")
	} else {
		logger.Log.Infof("Successfully cleaned up %d old shard records", result.RowsAffected)
	}
}

func ValidateUserShardAssignments() (map[string]interface{}, error) {
	shardMgr := GetAppShardManager()

	stats := map[string]interface{}{
		"total_users":        0,
		"valid_users":        0,
		"invalid_users":      0,
		"empty_userids":      0,
		"shard_distribution": make(map[int]int),
	}

	var users []models.UserSettings
	if err := database.DB.Select("user_id").Find(&users).Error; err != nil {
		return stats, err
	}

	stats["total_users"] = len(users)

	for _, user := range users {
		if user.UserID == "" {
			stats["empty_userids"] = stats["empty_userids"].(int) + 1
			continue
		}

		userShard := getUserShard(user.UserID, shardMgr.TotalShards)

		distribution := stats["shard_distribution"].(map[int]int)
		distribution[userShard]++
		stats["shard_distribution"] = distribution

		if userShard >= 0 && userShard < shardMgr.TotalShards {
			stats["valid_users"] = stats["valid_users"].(int) + 1
		} else {
			stats["invalid_users"] = stats["invalid_users"].(int) + 1
		}
	}

	return stats, nil
}
