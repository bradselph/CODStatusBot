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
		if err := DB.Exec("ALTER TABLE user_settings ADD COLUMN prefer_ephemeral_responses BOOLEAN DEFAULT TRUE").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to add prefer_ephemeral_responses column to UserSettings table")
		}

		logger.Log.Info("Setting prefer_ephemeral_responses to TRUE for all existing users")
		if err := DB.Exec("UPDATE user_settings SET prefer_ephemeral_responses = TRUE WHERE prefer_ephemeral_responses IS NULL OR prefer_ephemeral_responses = FALSE").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to update existing users to prefer ephemeral responses")
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

	if !DB.Migrator().HasColumn(&models.UserSettings{}, "fallback_captcha_provider") {
		logger.Log.Info("Adding fallback_captcha_provider column to UserSettings table")
		if err := DB.Exec("ALTER TABLE user_settings ADD COLUMN fallback_captcha_provider VARCHAR(255) DEFAULT ''").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to add fallback_captcha_provider column to UserSettings table")
		}
	}

	if !DB.Migrator().HasColumn(&models.UserSettings{}, "enable_fallback") {
		logger.Log.Info("Adding enable_fallback column to UserSettings table")
		if err := DB.Exec("ALTER TABLE user_settings ADD COLUMN enable_fallback BOOLEAN DEFAULT TRUE").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to add enable_fallback column to UserSettings table")
		}
	}

	if !DB.Migrator().HasColumn(&models.UserSettings{}, "use_fallback_for_default") {
		logger.Log.Info("Adding use_fallback_for_default column to UserSettings table")
		if err := DB.Exec("ALTER TABLE user_settings ADD COLUMN use_fallback_for_default BOOLEAN DEFAULT TRUE").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to add use_fallback_for_default column to UserSettings table")
		}
	}

	if !DB.Migrator().HasColumn(&models.UserSettings{}, "fallback_captcha_balance") {
		logger.Log.Info("Adding fallback_captcha_balance column to UserSettings table")
		if err := DB.Exec("ALTER TABLE user_settings ADD COLUMN fallback_captcha_balance DOUBLE DEFAULT 0").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to add fallback_captcha_balance column to UserSettings table")
		}
	}

	MigrateEphemeralDefaults()
	MigrateFallbackDefaults()
}

func MigrateEphemeralDefaults() {
	logger.Log.Info("Running ephemeral defaults migration")

	var count int64
	if err := DB.Model(&models.UserSettings{}).Where("prefer_ephemeral_responses = FALSE OR prefer_ephemeral_responses IS NULL").Count(&count).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to count users needing ephemeral migration")
		return
	}

	if count > 0 {
		logger.Log.Infof("Migrating %d users to prefer ephemeral responses by default", count)
		result := DB.Model(&models.UserSettings{}).Where("prefer_ephemeral_responses = FALSE OR prefer_ephemeral_responses IS NULL").Update("prefer_ephemeral_responses", true)
		if result.Error != nil {
			logger.Log.WithError(result.Error).Error("Failed to migrate users to ephemeral default")
		} else {
			logger.Log.Infof("Successfully migrated %d users to ephemeral default", result.RowsAffected)
		}
	}
}

func MigrateFallbackDefaults() {
	logger.Log.Info("Running fallback defaults migration")

	var count int64
	if err := DB.Model(&models.UserSettings{}).Where("enable_fallback IS NULL").Count(&count).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to count users needing fallback migration")
		return
	}

	if count > 0 {
		logger.Log.Infof("Migrating %d users to enable fallback by default", count)
		result := DB.Model(&models.UserSettings{}).Where("enable_fallback IS NULL").Update("enable_fallback", true)
		if result.Error != nil {
			logger.Log.WithError(result.Error).Error("Failed to migrate users to fallback default")
		} else {
			logger.Log.Infof("Successfully migrated %d users to fallback default", result.RowsAffected)
		}
	}

	var defaultCount int64
	if err := DB.Model(&models.UserSettings{}).Where("use_fallback_for_default IS NULL").Count(&defaultCount).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to count users needing default fallback migration")
		return
	}

	if defaultCount > 0 {
		logger.Log.Infof("Migrating %d users to enable default fallback", defaultCount)
		result := DB.Model(&models.UserSettings{}).Where("use_fallback_for_default IS NULL").Update("use_fallback_for_default", true)
		if result.Error != nil {
			logger.Log.WithError(result.Error).Error("Failed to migrate users to default fallback setting")
		} else {
			logger.Log.Infof("Successfully migrated %d users to default fallback setting", result.RowsAffected)
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
