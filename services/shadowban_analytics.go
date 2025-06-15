package services

import (
	"time"

	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
)

type ShadowbanPeriod struct {
	AccountID     uint       `json:"account_id"`
	UserID        string     `json:"user_id"`
	AccountTitle  string     `json:"account_title"`
	StartTime     time.Time  `json:"start_time"`
	EndTime       *time.Time `json:"end_time,omitempty"`
	DurationHours *float64   `json:"duration_hours,omitempty"`
	IsCompleted   bool       `json:"is_completed"`
	StartBanID    uint       `json:"start_ban_id"`
	EndBanID      *uint      `json:"end_ban_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func TrackShadowbanTransition(accountID uint, userID string, previousStatus, newStatus models.Status, banID uint) error {
	logger.Log.Debugf("Tracking shadowban transition for account %d: %s -> %s", accountID, previousStatus, newStatus)

	if (previousStatus == models.StatusGood) &&
		(newStatus == models.StatusShadowban || newStatus == models.StatusRankLocked) {
		return startShadowbanPeriod(accountID, userID, banID)
	}

	if (previousStatus == models.StatusShadowban || previousStatus == models.StatusRankLocked) &&
		(newStatus == models.StatusGood) {
		return endShadowbanPeriod(accountID, userID, banID)
	}

	if previousStatus == models.StatusUnknown && newStatus == models.StatusGood {
		return logShadowbanRecovery(accountID, userID, banID)
	}

	if previousStatus == models.StatusUnknown &&
		(newStatus == models.StatusShadowban || newStatus == models.StatusRankLocked) {
		return startShadowbanPeriod(accountID, userID, banID)
	}

	return nil
}

func startShadowbanPeriod(accountID uint, userID string, banID uint) error {
	var existingPeriod ShadowbanPeriod
	result := database.DB.Table("shadowban_periods").
		Where("account_id = ? AND is_completed = ?", accountID, false).
		First(&existingPeriod)

	if result.Error == nil {
		logger.Log.Warnf("Account %d already has an active shadowban period, completing the previous one", accountID)
		if err := forceCompleteShadowbanPeriod(accountID); err != nil {
			logger.Log.WithError(err).Error("Failed to complete previous shadowban period")
		}
	}

	var account models.Account
	if err := database.DB.First(&account, accountID).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to get account info for shadowban tracking")
		return err
	}

	period := ShadowbanPeriod{
		AccountID:    accountID,
		UserID:       userID,
		AccountTitle: account.Title,
		StartTime:    time.Now(),
		IsCompleted:  false,
		StartBanID:   banID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := database.DB.Table("shadowban_periods").Create(&period).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to create shadowban period record")
		return err
	}

	logger.Log.Infof("Started tracking shadowban period for account %s (ID: %d)", account.Title, accountID)

	shardMgr := GetAppShardManager()
	LogAnalyticsEvent("shadowban_started", userID, "", "", "shadowban_tracking",
		shardMgr.ShardID, shardMgr.InstanceID, map[string]interface{}{
			"account_id":    accountID,
			"account_title": account.Title,
			"start_ban_id":  banID,
		})

	return nil
}

func endShadowbanPeriod(accountID uint, userID string, banID uint) error {
	var period ShadowbanPeriod
	result := database.DB.Table("shadowban_periods").
		Where("account_id = ? AND is_completed = ?", accountID, false).
		First(&period)

	if result.Error != nil {
		logger.Log.Warnf("No active shadowban period found for account %d when trying to end it", accountID)
		return logShadowbanRecovery(accountID, userID, banID)
	}

	now := time.Now()
	duration := now.Sub(period.StartTime)
	durationHours := duration.Hours()

	period.EndTime = &now
	period.DurationHours = &durationHours
	period.IsCompleted = true
	period.EndBanID = &banID
	period.UpdatedAt = now

	if err := database.DB.Table("shadowban_periods").Save(&period).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to complete shadowban period")
		return err
	}

	logger.Log.Infof("Completed shadowban period for account %s (ID: %d): %.2f hours (%.1f days)",
		period.AccountTitle, accountID, durationHours, durationHours/24)

	shardMgr := GetAppShardManager()
	LogAnalyticsEvent("shadowban_ended", userID, "", "", "shadowban_tracking",
		shardMgr.ShardID, shardMgr.InstanceID, map[string]interface{}{
			"account_id":     accountID,
			"account_title":  period.AccountTitle,
			"duration_hours": durationHours,
			"duration_days":  durationHours / 24,
			"start_ban_id":   period.StartBanID,
			"end_ban_id":     banID,
		})

	return nil
}

func logShadowbanRecovery(accountID uint, userID string, banID uint) error {
	var account models.Account
	if err := database.DB.First(&account, accountID).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to get account info for shadowban recovery logging")
		return err
	}

	logger.Log.Infof("Account %s (ID: %d) recovered from shadowban/unknown state (no reliable duration data)",
		account.Title, accountID)

	shardMgr := GetAppShardManager()
	LogAnalyticsEvent("shadowban_recovery", userID, "", "", "shadowban_tracking",
		shardMgr.ShardID, shardMgr.InstanceID, map[string]interface{}{
			"account_id":      accountID,
			"account_title":   account.Title,
			"recovery_ban_id": banID,
			"note":            "No reliable start time available",
		})

	return nil
}

func forceCompleteShadowbanPeriod(accountID uint) error {
	now := time.Now()

	result := database.DB.Table("shadowban_periods").
		Where("account_id = ? AND is_completed = ?", accountID, false).
		Updates(map[string]interface{}{
			"is_completed": true,
			"updated_at":   now,
			"end_time":     now,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		logger.Log.Infof("Force completed %d hanging shadowban periods for account %d", result.RowsAffected, accountID)
	}

	return nil
}

func GetShadowbanStatistics(userID string, accountID *uint, days int) (map[string]interface{}, error) {
	if days <= 0 {
		days = 30
	}

	startDate := time.Now().AddDate(0, 0, -days)
	stats := map[string]interface{}{
		"period_days": days,
		"start_date":  startDate.Format("2006-01-02"),
		"end_date":    time.Now().Format("2006-01-02"),
		"user_id":     userID,
	}

	query := database.DB.Table("shadowban_periods").Where("user_id = ? AND created_at >= ?", userID, startDate)
	if accountID != nil {
		query = query.Where("account_id = ?", *accountID)
		stats["account_id"] = *accountID
	}

	var completedPeriods []ShadowbanPeriod
	if err := query.Where("is_completed = ?", true).Find(&completedPeriods).Error; err != nil {
		return stats, err
	}

	stats["total_completed_periods"] = len(completedPeriods)

	var activePeriods int64
	if err := query.Where("is_completed = ?", false).Count(&activePeriods).Error; err != nil {
		return stats, err
	}
	stats["active_periods"] = activePeriods

	if len(completedPeriods) > 0 {
		var totalHours float64
		var minHours, maxHours float64
		durations := make([]float64, 0, len(completedPeriods))

		for i, period := range completedPeriods {
			if period.DurationHours != nil {
				hours := *period.DurationHours
				totalHours += hours
				durations = append(durations, hours)

				if i == 0 {
					minHours = hours
					maxHours = hours
				} else {
					if hours < minHours {
						minHours = hours
					}
					if hours > maxHours {
						maxHours = hours
					}
				}
			}
		}

		if len(durations) > 0 {
			avgHours := totalHours / float64(len(durations))
			stats["average_duration_hours"] = avgHours
			stats["average_duration_days"] = avgHours / 24
			stats["min_duration_hours"] = minHours
			stats["max_duration_hours"] = maxHours
			stats["min_duration_days"] = minHours / 24
			stats["max_duration_days"] = maxHours / 24
			stats["total_shadowban_hours"] = totalHours
			stats["total_shadowban_days"] = totalHours / 24
		}

		var recentPeriods []map[string]interface{}
		for _, period := range completedPeriods {
			periodInfo := map[string]interface{}{
				"account_title": period.AccountTitle,
				"start_time":    period.StartTime.Format("2006-01-02 15:04:05"),
				"end_time":      period.EndTime.Format("2006-01-02 15:04:05"),
			}
			if period.DurationHours != nil {
				periodInfo["duration_hours"] = *period.DurationHours
				periodInfo["duration_days"] = *period.DurationHours / 24
			}
			recentPeriods = append(recentPeriods, periodInfo)
		}
		stats["recent_periods"] = recentPeriods
	}

	return stats, nil
}

func GetGlobalShadowbanStats(days int) (map[string]interface{}, error) {
	if days <= 0 {
		days = 30
	}

	startDate := time.Now().AddDate(0, 0, -days)
	stats := map[string]interface{}{
		"period_days": days,
		"start_date":  startDate.Format("2006-01-02"),
		"end_date":    time.Now().Format("2006-01-02"),
	}

	var totalPeriods int64
	if err := database.DB.Table("shadowban_periods").
		Where("created_at >= ?", startDate).Count(&totalPeriods).Error; err != nil {
		return stats, err
	}
	stats["total_periods"] = totalPeriods

	var completedPeriods []ShadowbanPeriod
	if err := database.DB.Table("shadowban_periods").
		Where("created_at >= ? AND is_completed = ? AND duration_hours IS NOT NULL", startDate, true).
		Find(&completedPeriods).Error; err != nil {
		return stats, err
	}
	stats["completed_periods"] = len(completedPeriods)

	var activePeriods int64
	if err := database.DB.Table("shadowban_periods").
		Where("created_at >= ? AND is_completed = ?", startDate, false).
		Count(&activePeriods).Error; err != nil {
		return stats, err
	}
	stats["active_periods"] = activePeriods

	if len(completedPeriods) > 0 {
		var totalHours float64
		durationRanges := map[string]int{
			"under_24h":    0,
			"1_3_days":     0,
			"3_7_days":     0,
			"7_14_days":    0,
			"over_14_days": 0,
		}

		for _, period := range completedPeriods {
			if period.DurationHours != nil {
				hours := *period.DurationHours
				totalHours += hours

				days := hours / 24
				switch {
				case days < 1:
					durationRanges["under_24h"]++
				case days <= 3:
					durationRanges["1_3_days"]++
				case days <= 7:
					durationRanges["3_7_days"]++
				case days <= 14:
					durationRanges["7_14_days"]++
				default:
					durationRanges["over_14_days"]++
				}
			}
		}

		avgHours := totalHours / float64(len(completedPeriods))
		stats["average_duration_hours"] = avgHours
		stats["average_duration_days"] = avgHours / 24
		stats["duration_distribution"] = durationRanges
	}

	var uniqueUsers int64
	if err := database.DB.Table("shadowban_periods").
		Where("created_at >= ?", startDate).
		Distinct("user_id").Count(&uniqueUsers).Error; err != nil {
		return stats, err
	}
	stats["unique_users_affected"] = uniqueUsers

	return stats, nil
}

func CleanupOldShadowbanPeriods(retentionDays int) error {
	if retentionDays <= 0 {
		logger.Log.Warn("Invalid retention days for shadowban period cleanup, skipping")
		return nil
	}

	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	var oldRecordsCount int64
	if err := database.DB.Table("shadowban_periods").
		Where("created_at < ?", cutoffDate).Count(&oldRecordsCount).Error; err != nil {
		return err
	}

	if oldRecordsCount == 0 {
		logger.Log.Debug("No old shadowban period data to clean up")
		return nil
	}

	logger.Log.Infof("Cleaning up %d shadowban period records older than %d days", oldRecordsCount, retentionDays)

	result := database.DB.Table("shadowban_periods").Where("created_at < ?", cutoffDate).Delete(&ShadowbanPeriod{})
	if result.Error != nil {
		return result.Error
	}

	logger.Log.Infof("Successfully cleaned up %d old shadowban period records", result.RowsAffected)
	return nil
}

func CreateShadowbanPeriodsTable() error {
	sql := `
	CREATE TABLE IF NOT EXISTS shadowban_periods (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		account_id BIGINT NOT NULL,
		user_id VARCHAR(255) NOT NULL,
		account_title VARCHAR(255) NOT NULL,
		start_time DATETIME(3) NOT NULL,
		end_time DATETIME(3) NULL,
		duration_hours DOUBLE NULL,
		is_completed BOOLEAN NOT NULL DEFAULT FALSE,
		start_ban_id BIGINT NOT NULL,
		end_ban_id BIGINT NULL,
		created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
		updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
		INDEX idx_account_id (account_id),
		INDEX idx_user_id (user_id),
		INDEX idx_created_at (created_at),
		INDEX idx_is_completed (is_completed),
		INDEX idx_start_time (start_time),
		INDEX idx_end_time (end_time),
		FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`

	if err := database.DB.Exec(sql).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to create shadowban_periods table")
		return err
	}

	logger.Log.Info("Successfully created or verified shadowban_periods table")
	return nil
}
