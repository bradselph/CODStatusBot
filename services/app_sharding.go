package services

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
)

type AppShardManager struct {
	sync.RWMutex
	ShardID       int
	TotalShards   int
	InstanceID    string
	HeartbeatTime time.Time
	Initialized   bool
}

var appShardManager *AppShardManager

func GetAppShardManager() *AppShardManager {
	if appShardManager == nil {
		appShardManager = &AppShardManager{
			InstanceID:  generateInstanceID(),
			Initialized: false,
		}
	}
	return appShardManager
}

func (asm *AppShardManager) Initialize() error {
	asm.Lock()
	defer asm.Unlock()

	shardID := os.Getenv("SHARD_ID")
	totalShards := os.Getenv("TOTAL_SHARDS")

	if shardID == "" || totalShards == "" {
		logger.Log.Info("No sharding configuration found, running in single-shard mode")
		asm.ShardID = 0
		asm.TotalShards = 1
	} else {
		id, err := strconv.Atoi(shardID)
		if err != nil {
			return fmt.Errorf("invalid SHARD_ID: %w", err)
		}

		total, err := strconv.Atoi(totalShards)
		if err != nil {
			return fmt.Errorf("invalid TOTAL_SHARDS: %w", err)
		}

		if id < 0 || id >= total {
			return fmt.Errorf("SHARD_ID must be between 0 and TOTAL_SHARDS-1")
		}

		asm.ShardID = id
		asm.TotalShards = total
	}

	if !database.DB.Migrator().HasTable("shard_infos") {
		logger.Log.Warn("shard_infos table does not exist, creating it manually")
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

		if err := database.DB.Exec(shardInfosTableSQL).Error; err != nil {
			return fmt.Errorf("failed to create shard_infos table: %w", err)
		}
		logger.Log.Info("Created shard_infos table successfully")
	}

	shardInfo := models.ShardInfo{
		ShardID:       asm.ShardID,
		TotalShards:   asm.TotalShards,
		InstanceID:    asm.InstanceID,
		LastHeartbeat: time.Now(),
		Status:        "active",
	}

	if err := database.DB.Where("instance_id = ?", asm.InstanceID).
		Assign(shardInfo).
		FirstOrCreate(&shardInfo).Error; err != nil {
		return fmt.Errorf("failed to register shard: %w", err)
	}

	logger.Log.Infof("Initialized application shard %d of %d with instance ID %s",
		asm.ShardID, asm.TotalShards, asm.InstanceID)

	asm.Initialized = true
	return nil
}

func (asm *AppShardManager) StartHeartbeat(ctx context.Context) {
	if !asm.Initialized {
		if err := asm.Initialize(); err != nil {
			logger.Log.WithError(err).Error("Failed to initialize app shard manager")
			return
		}
	}

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := asm.updateHeartbeat(); err != nil {
					logger.Log.WithError(err).Error("Failed to update shard heartbeat")
				}
				asm.healShards()
			}
		}
	}()

	logger.Log.Info("Started application shard heartbeat")
}

func (asm *AppShardManager) updateHeartbeat() error {
	asm.Lock()
	defer asm.Unlock()

	asm.HeartbeatTime = time.Now()

	return database.DB.Model(&models.ShardInfo{}).
		Where("instance_id = ?", asm.InstanceID).
		Updates(map[string]interface{}{
			"last_heartbeat": time.Now(),
			"total_shards":   asm.TotalShards,
			"status":         "active",
		}).Error
}

func (asm *AppShardManager) healShards() {
	heartbeatTimeout := 2 * time.Minute

	var deadShards []models.ShardInfo
	if err := database.DB.Where("last_heartbeat < ? AND status != 'inactive'",
		time.Now().Add(-heartbeatTimeout)).
		Find(&deadShards).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to query for dead shards")
		return
	}

	if len(deadShards) > 0 {
		logger.Log.Infof("Found %d dead shards", len(deadShards))
		for _, shard := range deadShards {
			logger.Log.Infof("Marking shard %d (instance %s) as inactive (last heartbeat: %s)",
				shard.ShardID, shard.InstanceID, shard.LastHeartbeat)

			if err := database.DB.Model(&models.ShardInfo{}).
				Where("instance_id = ?", shard.InstanceID).
				Update("status", "inactive").Error; err != nil {
				logger.Log.WithError(err).Error("Failed to mark shard as inactive")
			}
		}

		var activeShardCount int64
		if err := database.DB.Model(&models.ShardInfo{}).
			Where("status = 'active'").Count(&activeShardCount).Error; err != nil {
			logger.Log.WithError(err).Error("Failed to count active shards")
			return
		}

		if activeShardCount > 0 {
			if err := database.DB.Model(&models.ShardInfo{}).
				Where("status = 'active'").
				Update("total_shards", activeShardCount).Error; err != nil {
				logger.Log.WithError(err).Error("Failed to update total shards count")
			}

			asm.Lock()
			asm.TotalShards = int(activeShardCount)
			asm.Unlock()

			logger.Log.Infof("Updated total active shards to %d", activeShardCount)
		}
	}
}

func (asm *AppShardManager) GuildBelongsToInstance(guildID string) bool {
	asm.RLock()
	defer asm.RUnlock()

	if asm.TotalShards <= 1 {
		return true
	}

	guildIDInt, err := strconv.ParseUint(guildID, 10, 64)
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to parse guildID %s as uint64", guildID)
		return false
	}

	targetShard := (guildIDInt >> 22) % uint64(asm.TotalShards)
	return int(targetShard) == asm.ShardID
}

func (asm *AppShardManager) GetGuildShardID(guildID string) int {
	asm.RLock()
	defer asm.RUnlock()

	if asm.TotalShards <= 1 {
		return 0
	}

	guildIDInt, err := strconv.ParseUint(guildID, 10, 64)
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to parse guildID %s as uint64", guildID)
		return -1
	}

	return int((guildIDInt >> 22) % uint64(asm.TotalShards))
}

func (asm *AppShardManager) ShardBelongsToInstance(userID string) bool {
	asm.RLock()
	defer asm.RUnlock()

	if asm.TotalShards <= 1 {
		return true
	}

	targetShard := getUserShard(userID, asm.TotalShards)
	return targetShard == asm.ShardID
}

func getUserShard(userID string, totalShards int) int {
	hash := sha256.Sum256([]byte(userID))
	val := binary.BigEndian.Uint64(hash[:8])
	return int(val % uint64(totalShards))
}

func (asm *AppShardManager) GetUserShardID(userID string) int {
	asm.RLock()
	defer asm.RUnlock()

	if asm.TotalShards <= 1 {
		return 0
	}

	return getUserShard(userID, asm.TotalShards)
}

func generateInstanceID() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	pid := os.Getpid()

	randBytes := make([]byte, 4)
	for i := range randBytes {
		randBytes[i] = byte(time.Now().Nanosecond() & 0xff)
		time.Sleep(time.Nanosecond)
	}

	return fmt.Sprintf("%s-%d-%x", hostname, pid, randBytes)
}

func (asm *AppShardManager) GetShardingStatus() map[string]interface{} {
	asm.RLock()
	defer asm.RUnlock()

	return map[string]interface{}{
		"shard_id":       asm.ShardID,
		"total_shards":   asm.TotalShards,
		"instance_id":    asm.InstanceID,
		"initialized":    asm.Initialized,
		"last_heartbeat": asm.HeartbeatTime,
	}
}

func (asm *AppShardManager) FilterUsersByShardAssignment(userIDs []string) []string {
	asm.RLock()
	defer asm.RUnlock()

	if asm.TotalShards <= 1 {
		return userIDs
	}

	var assignedUsers []string
	for _, userID := range userIDs {
		if getUserShard(userID, asm.TotalShards) == asm.ShardID {
			assignedUsers = append(assignedUsers, userID)
		}
	}

	return assignedUsers
}

func (asm *AppShardManager) IsUserAssignedToShard(userID string) bool {
	asm.RLock()
	defer asm.RUnlock()

	if asm.TotalShards <= 1 {
		return true
	}

	return getUserShard(userID, asm.TotalShards) == asm.ShardID
}

func (asm *AppShardManager) GetShardedUserCount() (int64, error) {
	asm.RLock()
	defer asm.RUnlock()

	if asm.TotalShards <= 1 {
		var count int64
		err := database.DB.Model(&models.UserSettings{}).Count(&count).Error
		return count, err
	}

	var userIDs []string
	if err := database.DB.Model(&models.UserSettings{}).
		Pluck("user_id", &userIDs).Error; err != nil {
		return 0, err
	}

	var count int64
	for _, userID := range userIDs {
		if getUserShard(userID, asm.TotalShards) == asm.ShardID {
			count++
		}
	}

	return count, nil
}

func FilterAccountsByShardAssignment(accounts []models.Account) []models.Account {
	shardManager := GetAppShardManager()

	if shardManager.TotalShards <= 1 {
		return accounts
	}

	var filteredAccounts []models.Account
	for _, account := range accounts {
		if shardManager.IsUserAssignedToShard(account.UserID) {
			filteredAccounts = append(filteredAccounts, account)
		}
	}

	return filteredAccounts
}

func FilterUserSettingsByShardAssignment(settings []models.UserSettings) []models.UserSettings {
	shardManager := GetAppShardManager()

	if shardManager.TotalShards <= 1 {
		return settings
	}

	var filteredSettings []models.UserSettings
	for _, setting := range settings {
		if shardManager.IsUserAssignedToShard(setting.UserID) {
			filteredSettings = append(filteredSettings, setting)
		}
	}

	return filteredSettings
}
