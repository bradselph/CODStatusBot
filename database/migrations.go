package database

import (
	"time"

	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
)

func RunMigrations() {
	logger.Log.Info("Running migrations")

	CleanupInvalidTimestamps()

	if !DB.Migrator().HasColumn(&models.Analytics{}, "shard_id") {
		logger.Log.Info("Adding shard_id column to Analytics table")
		if err := DB.Exec("ALTER TABLE analytics ADD COLUMN shard_id INT DEFAULT 0").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to add shard_id column to Analytics table")
		}
	}

	if !DB.Migrator().HasColumn(&models.Analytics{}, "instance_id") {
		logger.Log.Info("Adding instance_id column to Analytics table")
		if err := DB.Exec("ALTER TABLE analytics ADD COLUMN instance_id VARCHAR(255) DEFAULT ''").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to add instance_id column to Analytics table")
		}
	}

	if !DB.Migrator().HasColumn(&models.Account{}, "game_specific_bans") {
		logger.Log.Info("Adding game_specific_bans column to Account table")
		if err := DB.Exec("ALTER TABLE accounts ADD COLUMN game_specific_bans JSON").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to add game_specific_bans column to Account table")
		}
	}

	if !DB.Migrator().HasColumn(&models.Account{}, "is_rank_locked") {
		logger.Log.Info("Adding is_rank_locked column to Account table")
		if err := DB.Exec("ALTER TABLE accounts ADD COLUMN is_rank_locked BOOLEAN DEFAULT FALSE").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to add is_rank_locked column to Account table")
		}
	}

	if !DB.Migrator().HasColumn(&models.UserSettings{}, "prefer_ephemeral_responses") {
		logger.Log.Info("Adding prefer_ephemeral_responses column to UserSettings table")
		if err := DB.Exec("ALTER TABLE user_settings ADD COLUMN prefer_ephemeral_responses BOOLEAN DEFAULT FALSE").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to add prefer_ephemeral_responses column to UserSettings table")
		}
	}

	if !DB.Migrator().HasColumn(&models.UserSettings{}, "rate_limit_backoff") {
		logger.Log.Info("Adding rate_limit_backoff column to UserSettings table")
		if err := DB.Exec("ALTER TABLE user_settings ADD COLUMN rate_limit_backoff JSON").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to add rate_limit_backoff column to UserSettings table")
		}
	}

	if !DB.Migrator().HasColumn(&models.UserSettings{}, "last_rate_limit_hit") {
		logger.Log.Info("Adding last_rate_limit_hit column to UserSettings table")
		if err := DB.Exec("ALTER TABLE user_settings ADD COLUMN last_rate_limit_hit JSON").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to add last_rate_limit_hit column to UserSettings table")
		}
	}

	if !DB.Migrator().HasColumn(&models.Ban{}, "game_specific_bans") {
		logger.Log.Info("Adding game_specific_bans column to Ban table")
		if err := DB.Exec("ALTER TABLE bans ADD COLUMN game_specific_bans JSON").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to add game_specific_bans column to Ban table")
		}
	}

	if !DB.Migrator().HasColumn(&models.Account{}, "is_campaign_only_shadowban") {
		logger.Log.Info("Adding is_campaign_only_shadowban column to Account table")
		if err := DB.Exec("ALTER TABLE accounts ADD COLUMN is_campaign_only_shadowban BOOLEAN DEFAULT FALSE").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to add is_campaign_only_shadowban column to Account table")
		}
	}
}

func CleanupInvalidTimestamps() {
	logger.Log.Info("Running timestamp cleanup migration")

	result := DB.Model(&models.Ban{}).
		Where("timestamp <= '1970-01-01' OR timestamp IS NULL").
		Update("timestamp", time.Now())
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Failed to clean up invalid Ban timestamps")
	} else {
		logger.Log.Infof("Fixed %d invalid Ban timestamps", result.RowsAffected)
	}

	if !DB.Migrator().HasColumn(&models.Account{}, "is_og_verdansk") {
		logger.Log.Info("Adding IsOGVerdansk column to Account table")
		if err := DB.Migrator().AddColumn(&models.Account{}, "is_og_verdansk"); err != nil {
			logger.Log.WithError(err).Error("Failed to add IsOGVerdansk column to Account table")
		}
	}

	result = DB.Model(&models.Account{}).
		Where("created <= 0 OR created IS NULL").
		Update("created", time.Now().Unix())
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Failed to clean up invalid Account created timestamps")
	} else {
		logger.Log.Infof("Fixed %d invalid Account created timestamps", result.RowsAffected)
	}

	result = DB.Model(&models.Account{}).
		Where("last_check <= 0 OR last_check IS NULL").
		Update("last_check", time.Now().Unix())
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Failed to clean up invalid Account last_check timestamps")
	} else {
		logger.Log.Infof("Fixed %d invalid Account last_check timestamps", result.RowsAffected)
	}
}
