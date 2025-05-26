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
