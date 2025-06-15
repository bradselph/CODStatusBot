package database

import (
	"errors"
	"fmt"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Databaselogin() error {
	logger.Log.Info("Connecting to database...")

	cfg := configuration.Get()
	dbConfig := cfg.Database

	if dbConfig.User == "" || dbConfig.Password == "" || dbConfig.Host == "" ||
		dbConfig.Port == "" || dbConfig.Name == "" || dbConfig.Var == "" {
		err := errors.New("one or more database configuration values not set")
		logger.Log.WithError(err).WithField("Bot Startup ", "database configuration ").Error()
		return err
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s%s",
		dbConfig.User,
		dbConfig.Password,
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.Name,
		dbConfig.Var)

	var db *gorm.DB
	var err error
	maxRetries := 5

	for retries := 0; retries < maxRetries; retries++ {
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}

		logger.Log.WithError(err).Warnf("Database connection attempt %d/%d failed, retrying...",
			retries+1, maxRetries)

		if retries < maxRetries-1 {
			time.Sleep(time.Duration(2<<retries) * time.Second)
		}
	}
	if err != nil {
		logger.Log.WithError(err).WithField("Bot Startup ", "MySQL Config ").Error()
		return err
	}

	DB = db

	sqlDB, err := DB.DB()
	if err != nil {
		logger.Log.WithError(err).Error("Failed to get database instance")
		return err
	}

	sqlDB.SetMaxIdleConns(cfg.Performance.DbMaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.Performance.DbMaxOpenConns)

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

	err = DB.AutoMigrate(
		&models.Account{},
		&models.Ban{},
		&models.UserSettings{},
		&models.SuppressedNotification{},
		&models.Analytics{},
		&models.BotStatistics{},
		&models.CommandStatistics{},
		&models.ProxyStats{},
		&models.ShardInfo{},
	)
	if err != nil {
		logger.Log.WithError(err).WithField("Bot Startup ", "Database Models Problem ").Error()
		return err
	}

	CleanupInvalidTimestamps()
	RunMigrations()

	return nil
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
func RunMigrations() {
	logger.Log.Info("Running migrations")

	if err := createShadowbanPeriodsTable(); err != nil {
		logger.Log.WithError(err).Error("Failed to create shadowban_periods table")
	}

	CleanupInvalidTimestamps()

	if !DB.Migrator().HasColumn(&models.Analytics{}, "shard_id") {
		logger.Log.Info("Adding shard_id column to Analytics table")
		if err := DB.Exec("ALTER TABLE analytics ADD COLUMN shard_id INT DEFAULT 0").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to add shard_id column to Analytics table")
		}
		if err := DB.Exec("CREATE INDEX idx_analytics_shard_id ON analytics (shard_id)").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to create shard_id index on Analytics table")
		}
	}

	if !DB.Migrator().HasColumn(&models.Analytics{}, "instance_id") {
		logger.Log.Info("Adding instance_id column to Analytics table")
		if err := DB.Exec("ALTER TABLE analytics ADD COLUMN instance_id VARCHAR(255) DEFAULT ''").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to add instance_id column to Analytics table")
		}
		if err := DB.Exec("CREATE INDEX idx_analytics_instance_id ON analytics (instance_id)").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to create instance_id index on Analytics table")
		}
	}

	if !DB.Migrator().HasColumn(&models.ShardInfo{}, "startup_time") {
		logger.Log.Info("Adding startup_time column to ShardInfo table")
		if err := DB.Exec("ALTER TABLE shard_infos ADD COLUMN startup_time datetime(3)").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to add startup_time column to ShardInfo table")
		}
	}

	if !DB.Migrator().HasColumn(&models.ShardInfo{}, "process_id") {
		logger.Log.Info("Adding process_id column to ShardInfo table")
		if err := DB.Exec("ALTER TABLE shard_infos ADD COLUMN process_id bigint").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to add process_id column to ShardInfo table")
		}
	}

	if !DB.Migrator().HasColumn(&models.ShardInfo{}, "hostname") {
		logger.Log.Info("Adding hostname column to ShardInfo table")
		if err := DB.Exec("ALTER TABLE shard_infos ADD COLUMN hostname varchar(255)").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to add hostname column to ShardInfo table")
		}
		if err := DB.Exec("CREATE INDEX idx_shard_infos_hostname ON shard_infos (hostname)").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to create hostname index on ShardInfo table")
		}
	}

	// Continue with existing migrations...
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

	if !DB.Migrator().HasColumn(&models.UserSettings{}, "has_seen_fallback_notice") {
		logger.Log.Info("Adding has_seen_fallback_notice column to UserSettings table")
		if err := DB.Exec("ALTER TABLE user_settings ADD COLUMN has_seen_fallback_notice BOOLEAN DEFAULT FALSE").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to add has_seen_fallback_notice column to UserSettings table")
		}
	}

	if !DB.Migrator().HasColumn(&models.Account{}, "is_og_verdansk") {
		logger.Log.Info("Adding IsOGVerdansk column to Account table")
		if err := DB.Migrator().AddColumn(&models.Account{}, "is_og_verdansk"); err != nil {
			logger.Log.WithError(err).Error("Failed to add IsOGVerdansk column to Account table")
		}
	}

	if err := DB.Exec("UPDATE accounts SET game_specific_bans = '{}' WHERE game_specific_bans IS NULL OR game_specific_bans = ''").Error; err != nil {
		logger.Log.WithError(err).Error("Failed to initialize game_specific_bans")
	}

	if err := DB.Exec("UPDATE user_settings SET notification_times = '{}' WHERE notification_times IS NULL OR notification_times = ''").Error; err != nil {
		logger.Log.WithError(err).Error("Failed to initialize notification_times")
	}

	if err := DB.Exec("UPDATE user_settings SET action_counts = '{}' WHERE action_counts IS NULL OR action_counts = ''").Error; err != nil {
		logger.Log.WithError(err).Error("Failed to initialize action_counts")
	}

	if err := DB.Exec("UPDATE user_settings SET last_action_times = '{}' WHERE last_action_times IS NULL OR last_action_times = ''").Error; err != nil {
		logger.Log.WithError(err).Error("Failed to initialize last_action_times")
	}

	if err := DB.Exec("UPDATE user_settings SET last_command_times = '{}' WHERE last_command_times IS NULL OR last_command_times = ''").Error; err != nil {
		logger.Log.WithError(err).Error("Failed to initialize last_command_times")
	}

	if err := DB.Exec("UPDATE user_settings SET rate_limit_expiration = '{}' WHERE rate_limit_expiration IS NULL OR rate_limit_expiration = ''").Error; err != nil {
		logger.Log.WithError(err).Error("Failed to initialize rate_limit_expiration")
	}

	if err := DB.Exec("UPDATE user_settings SET rate_limit_backoff = '{}' WHERE rate_limit_backoff IS NULL OR rate_limit_backoff = ''").Error; err != nil {
		logger.Log.WithError(err).Error("Failed to initialize rate_limit_backoff")
	}

	if err := DB.Exec("UPDATE user_settings SET last_rate_limit_hit = '{}' WHERE last_rate_limit_hit IS NULL OR last_rate_limit_hit = ''").Error; err != nil {
		logger.Log.WithError(err).Error("Failed to initialize last_rate_limit_hit")
	}

	MigrateEphemeralDefaults()
	MigrateFallbackDefaults()

	logger.Log.Info("Database migrations completed")
}
func CheckConnection() error {
	if DB == nil {
		return errors.New("database connection is nil")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

func CloseConnection() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		if err := sqlDB.Close(); err != nil {
			return err
		}
		logger.Log.Info("Database connection closed successfully")
	}
	return nil
}

func CloseDB() error {
	return CloseConnection()
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

func createShadowbanPeriodsTable() error {
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
		INDEX idx_end_time (end_time)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`

	if err := DB.Exec(sql).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to create shadowban_periods table")
		return err
	}

	logger.Log.Info("Successfully created or verified shadowban_periods table")
	return nil
}
