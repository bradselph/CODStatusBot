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
	isLeader      bool
	lastRebalance time.Time
}

var appShardManager *AppShardManager

func GetAppShardManager() *AppShardManager {
	if appShardManager == nil {
		appShardManager = &AppShardManager{
			InstanceID:    generateInstanceID(),
			Initialized:   false,
			lastRebalance: time.Now(),
		}
	}
	return appShardManager
}

func (asm *AppShardManager) Initialize() error {
	asm.Lock()
	defer asm.Unlock()

	if asm.Initialized {
		return nil
	}

	shardID := os.Getenv("SHARD_ID")
	totalShards := os.Getenv("TOTAL_SHARDS")

	if shardID == "" || totalShards == "" {
		return asm.initializeAutoShard()
	}

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

	return asm.registerShard()
}

func (asm *AppShardManager) initializeAutoShard() error {
	if !asm.ensureShardInfoTable() {
		return fmt.Errorf("failed to ensure shard_infos table exists")
	}

	var activeShards []models.ShardInfo
	if err := database.DB.Where("status = 'active' AND last_heartbeat > ?",
		time.Now().Add(-2*time.Minute)).Find(&activeShards).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to query active shards")
		asm.ShardID = 0
		asm.TotalShards = 1
	} else {
		asm.ShardID = len(activeShards)
		asm.TotalShards = len(activeShards) + 1

		for _, shard := range activeShards {
			if shard.ShardID >= asm.ShardID {
				asm.ShardID = shard.ShardID + 1
			}
		}
	}

	logger.Log.Infof("Auto-assigned shard %d of %d", asm.ShardID, asm.TotalShards)
	return asm.registerShard()
}

func (asm *AppShardManager) ensureShardInfoTable() bool {
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
			startup_time datetime(3),
			process_id bigint,
			hostname varchar(255),
			INDEX idx_shard_infos_deleted_at (deleted_at),
			INDEX idx_shard_infos_shard_id (shard_id),
			UNIQUE INDEX idx_shard_infos_instance_id (instance_id),
			INDEX idx_shard_infos_last_heartbeat (last_heartbeat),
			INDEX idx_shard_infos_status (status)
		)`

		if err := database.DB.Exec(shardInfosTableSQL).Error; err != nil {
			logger.Log.WithError(err).Error("Failed to create shard_infos table")
			return false
		}
		logger.Log.Info("Created shard_infos table successfully")
	}
	return true
}

func (asm *AppShardManager) registerShard() error {
	if !asm.ensureShardInfoTable() {
		return fmt.Errorf("failed to ensure shard_infos table exists")
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	shardInfo := models.ShardInfo{
		ShardID:       asm.ShardID,
		TotalShards:   asm.TotalShards,
		InstanceID:    asm.InstanceID,
		LastHeartbeat: time.Now(),
		Status:        "active",
		StartupTime:   time.Now(),
		ProcessID:     int64(os.Getpid()),
		Hostname:      hostname,
	}

	if err := database.DB.Where("instance_id = ?", asm.InstanceID).
		Assign(shardInfo).
		FirstOrCreate(&shardInfo).Error; err != nil {
		return fmt.Errorf("failed to register shard: %w", err)
	}

	logger.Log.Infof("Registered application shard %d of %d with instance ID %s on host %s (PID: %d)",
		asm.ShardID, asm.TotalShards, asm.InstanceID, hostname, os.Getpid())

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

	ticker := time.NewTicker(15 * time.Second)
	rebalanceTicker := time.NewTicker(60 * time.Second)

	go func() {
		defer ticker.Stop()
		defer rebalanceTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				asm.cleanup()
				return
			case <-ticker.C:
				if err := asm.updateHeartbeat(); err != nil {
					logger.Log.WithError(err).Error("Failed to update shard heartbeat")
				}
			case <-rebalanceTicker.C:
				asm.performMaintenance()
			}
		}
	}()

	logger.Log.Info("Started application shard heartbeat with maintenance")
}

func (asm *AppShardManager) updateHeartbeat() error {
	asm.Lock()
	defer asm.Unlock()

	asm.HeartbeatTime = time.Now()

	result := database.DB.Model(&models.ShardInfo{}).
		Where("instance_id = ?", asm.InstanceID).
		Updates(map[string]interface{}{
			"last_heartbeat": time.Now(),
			"total_shards":   asm.TotalShards,
			"status":         "active",
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		logger.Log.Warn("Heartbeat update affected 0 rows, re-registering shard")
		return asm.registerShard()
	}

	return nil
}

func (asm *AppShardManager) performMaintenance() {
	if time.Since(asm.lastRebalance) < 30*time.Second {
		return
	}

	asm.healShards()
	asm.rebalanceShards()
	asm.electLeader()
	asm.lastRebalance = time.Now()
}

func (asm *AppShardManager) healShards() {
	heartbeatTimeout := 90 * time.Second

	var deadShards []models.ShardInfo
	if err := database.DB.Where("last_heartbeat < ? AND status = 'active'",
		time.Now().Add(-heartbeatTimeout)).
		Find(&deadShards).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to query for dead shards")
		return
	}

	if len(deadShards) > 0 {
		logger.Log.Infof("Found %d dead shards, marking as inactive", len(deadShards))

		var instanceIDs []string
		for _, shard := range deadShards {
			instanceIDs = append(instanceIDs, shard.InstanceID)
			logger.Log.Infof("Marking shard %d (instance %s, host %s, PID %d) as inactive (last heartbeat: %s)",
				shard.ShardID, shard.InstanceID, shard.Hostname, shard.ProcessID, shard.LastHeartbeat)
		}

		if err := database.DB.Model(&models.ShardInfo{}).
			Where("instance_id IN ?", instanceIDs).
			Update("status", "inactive").Error; err != nil {
			logger.Log.WithError(err).Error("Failed to mark dead shards as inactive")
		}
	}
}

func (asm *AppShardManager) rebalanceShards() {
	var activeShards []models.ShardInfo
	if err := database.DB.Where("status = 'active'").
		Order("shard_id ASC").Find(&activeShards).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to query active shards for rebalancing")
		return
	}

	if len(activeShards) == 0 {
		logger.Log.Error("No active shards found during rebalancing")
		return
	}

	needsRebalance := false
	newTotalShards := len(activeShards)

	for i, shard := range activeShards {
		if shard.ShardID != i || shard.TotalShards != newTotalShards {
			needsRebalance = true
			break
		}
	}

	if !needsRebalance {
		if asm.TotalShards != newTotalShards {
			asm.Lock()
			asm.TotalShards = newTotalShards
			asm.Unlock()
			logger.Log.Infof("Updated local total shards to %d", newTotalShards)
		}
		return
	}

	logger.Log.Infof("Rebalancing %d active shards", len(activeShards))

	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.Log.Errorf("Panic during shard rebalancing: %v", r)
		}
	}()

	for i, shard := range activeShards {
		newShardID := i
		if err := tx.Model(&models.ShardInfo{}).
			Where("instance_id = ?", shard.InstanceID).
			Updates(map[string]interface{}{
				"shard_id":     newShardID,
				"total_shards": newTotalShards,
			}).Error; err != nil {
			tx.Rollback()
			logger.Log.WithError(err).Error("Failed to update shard during rebalancing")
			return
		}

		if shard.InstanceID == asm.InstanceID {
			asm.Lock()
			asm.ShardID = newShardID
			asm.TotalShards = newTotalShards
			asm.Unlock()
			logger.Log.Infof("Updated local shard assignment to %d of %d", newShardID, newTotalShards)
		}
	}

	if err := tx.Commit().Error; err != nil {
		logger.Log.WithError(err).Error("Failed to commit shard rebalancing transaction")
		return
	}

	logger.Log.Infof("Successfully rebalanced shards: total active shards = %d", newTotalShards)
}

func (asm *AppShardManager) electLeader() {
	var leader models.ShardInfo
	if err := database.DB.Where("status = 'active'").
		Order("startup_time ASC, instance_id ASC").
		First(&leader).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to elect leader")
		return
	}

	asm.Lock()
	wasLeader := asm.isLeader
	asm.isLeader = (leader.InstanceID == asm.InstanceID)
	asm.Unlock()

	if asm.isLeader && !wasLeader {
		logger.Log.Infof("Elected as cluster leader (instance %s)", asm.InstanceID)
	} else if !asm.isLeader && wasLeader {
		logger.Log.Infof("No longer cluster leader, new leader is %s", leader.InstanceID)
	}
}

func (asm *AppShardManager) IsLeader() bool {
	asm.RLock()
	defer asm.RUnlock()
	return asm.isLeader
}

func (asm *AppShardManager) cleanup() {
	asm.Lock()
	defer asm.Unlock()

	if err := database.DB.Model(&models.ShardInfo{}).
		Where("instance_id = ?", asm.InstanceID).
		Update("status", "shutdown").Error; err != nil {
		logger.Log.WithError(err).Error("Failed to mark shard as shutdown")
	} else {
		logger.Log.Info("Marked shard as shutdown in database")
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
	timestamp := time.Now().UnixNano()

	return fmt.Sprintf("%s-%d-%d", hostname, pid, timestamp)
}

func (asm *AppShardManager) GetShardingStatus() map[string]interface{} {
	asm.RLock()
	defer asm.RUnlock()

	return map[string]interface{}{
		"shard_id":       asm.ShardID,
		"total_shards":   asm.TotalShards,
		"instance_id":    asm.InstanceID,
		"initialized":    asm.Initialized,
		"is_leader":      asm.isLeader,
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
