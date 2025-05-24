package database

import (
	"time"

	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
)

func RunMigrations() {
	logger.Log.Info("Running migrations")

	if DB.Migrator().HasTable("shard_infos") {
		logger.Log.Info("Cleaning up shard_infos table before migrations")
		if err := DB.Exec("DROP TABLE IF EXISTS shard_infos").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to drop shard_infos table")
		}
	}

	if DB.Migrator().HasTable("proxy_stats") {
		logger.Log.Info("Cleaning up proxy_stats table before migrations")
		if err := DB.Exec("DROP TABLE IF EXISTS proxy_stats").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to drop proxy_stats table")
		}
	}

	shardInfosTableSQL := `CREATE TABLE IF NOT EXISTS shard_infos (
		id bigint unsigned AUTO_INCREMENT PRIMARY KEY,
		created_at datetime(3) NULL,
		updated_at datetime(3) NULL,
		deleted_at datetime(3) NULL,
		shard_id bigint,
		total_shards bigint,
		instance_id varchar(191),
		last_heartbeat datetime(3) NULL,
		status varchar(191) DEFAULT 'active',
		stats text,
		INDEX idx_shard_infos_deleted_at (deleted_at),
		INDEX idx_shard_infos_shard_id (shard_id),
		UNIQUE INDEX idx_shard_infos_instance_id (instance_id),
		INDEX idx_shard_infos_last_heartbeat (last_heartbeat)
	)`

	if err := DB.Exec(shardInfosTableSQL).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to create shard_infos table")
	} else {
		logger.Log.Info("Successfully created shard_infos table")
	}

	proxyStatsTableSQL := `CREATE TABLE IF NOT EXISTS proxy_stats (
		id bigint unsigned AUTO_INCREMENT PRIMARY KEY,
		created_at datetime(3) NULL,
		updated_at datetime(3) NULL,
		deleted_at datetime(3) NULL,
		proxy_url varchar(191),
		status varchar(191) DEFAULT 'active',
		success_count bigint,
		failure_count bigint,
		consecutive_failures bigint DEFAULT 0,
		last_check datetime(3) NULL,
		last_error longtext,
		rate_limited_until datetime(3) NULL,
		stats text,
		INDEX idx_proxy_stats_deleted_at (deleted_at),
		UNIQUE INDEX idx_proxy_stats_proxy_url (proxy_url)
	)`

	if err := DB.Exec(proxyStatsTableSQL).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to create proxy_stats table")
	} else {
		logger.Log.Info("Successfully created proxy_stats table")
	}

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
