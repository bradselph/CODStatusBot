package services

import (
	"fmt"
	"time"

	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
)

func ShardHealthCheck() (*ShardHealthReport, error) {
	report := &ShardHealthReport{
		CheckTime: time.Now(),
		Healthy:   true,
		Issues:    []string{},
		Warnings:  []string{},
	}

	shardMgr := GetAppShardManager()
	if !shardMgr.Initialized {
		report.Healthy = false
		report.Issues = append(report.Issues, "Shard manager not initialized")
		return report, nil
	}

	report.CurrentShard = shardMgr.ShardID
	report.TotalShards = shardMgr.TotalShards
	report.InstanceID = shardMgr.InstanceID
	report.IsLeader = shardMgr.IsLeader()

	var activeShards []models.ShardInfo
	if err := database.DB.Where("status = 'active'").Find(&activeShards).Error; err != nil {
		report.Healthy = false
		report.Issues = append(report.Issues, fmt.Sprintf("Failed to query active shards: %v", err))
		return report, err
	}

	report.ActiveShards = len(activeShards)

	heartbeatTimeout := 90 * time.Second
	var deadShards []models.ShardInfo
	if err := database.DB.Where("status = 'active' AND last_heartbeat < ?",
		time.Now().Add(-heartbeatTimeout)).Find(&deadShards).Error; err != nil {
		report.Warnings = append(report.Warnings, fmt.Sprintf("Failed to check for dead shards: %v", err))
	} else if len(deadShards) > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("Found %d potentially dead shards", len(deadShards)))
		report.DeadShards = len(deadShards)
	}

	userStats, err := ValidateUserShardAssignments()
	if err != nil {
		report.Warnings = append(report.Warnings, fmt.Sprintf("Failed to validate user assignments: %v", err))
	} else {
		report.UserDistribution = userStats

		if distribution, ok := userStats["shard_distribution"].(map[int]int); ok {
			maxUsers := 0
			minUsers := int(^uint(0) >> 1)

			for _, count := range distribution {
				if count > maxUsers {
					maxUsers = count
				}
				if count < minUsers {
					minUsers = count
				}
			}

			if maxUsers > 0 && minUsers >= 0 {
				ratio := float64(maxUsers) / float64(minUsers+1)
				if ratio > 2.0 {
					report.Warnings = append(report.Warnings,
						fmt.Sprintf("Uneven user distribution detected (ratio: %.2f)", ratio))
				}
			}
		}
	}

	var totalAccounts int64
	if err := database.DB.Model(&models.Account{}).Count(&totalAccounts).Error; err != nil {
		report.Warnings = append(report.Warnings, fmt.Sprintf("Failed to count total accounts: %v", err))
	} else {
		report.TotalAccounts = int(totalAccounts)

		var accounts []models.Account
		if err := database.DB.Find(&accounts).Error; err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("Failed to fetch accounts: %v", err))
		} else {
			filteredAccounts := FilterAccountsByShardAssignment(accounts)
			report.ShardAccounts = len(filteredAccounts)

			if report.TotalShards > 0 {
				expectedRatio := float64(report.ShardAccounts) / float64(report.TotalAccounts)
				idealRatio := 1.0 / float64(report.TotalShards)

				if expectedRatio < idealRatio*0.5 || expectedRatio > idealRatio*2.0 {
					report.Warnings = append(report.Warnings,
						fmt.Sprintf("Account distribution deviation: %.2f%% vs expected %.2f%%",
							expectedRatio*100, idealRatio*100))
				}
			}
		}
	}

	if time.Since(shardMgr.HeartbeatTime) > 2*time.Minute {
		report.Warnings = append(report.Warnings, "Heartbeat is stale")
	}

	if shardMgr.rebalancing {
		report.Warnings = append(report.Warnings, "Shard rebalancing in progress")
	}

	if len(report.Issues) > 0 {
		report.Healthy = false
	} else if len(report.Warnings) > 3 {
		report.Healthy = false
		report.Issues = append(report.Issues, "Too many warnings detected")
	}

	return report, nil
}

type ShardHealthReport struct {
	CheckTime        time.Time              `json:"check_time"`
	Healthy          bool                   `json:"healthy"`
	Issues           []string               `json:"issues"`
	Warnings         []string               `json:"warnings"`
	CurrentShard     int                    `json:"current_shard"`
	TotalShards      int                    `json:"total_shards"`
	ActiveShards     int                    `json:"active_shards"`
	DeadShards       int                    `json:"dead_shards"`
	InstanceID       string                 `json:"instance_id"`
	IsLeader         bool                   `json:"is_leader"`
	TotalAccounts    int                    `json:"total_accounts"`
	ShardAccounts    int                    `json:"shard_accounts"`
	UserDistribution map[string]interface{} `json:"user_distribution"`
}

func (r *ShardHealthReport) String() string {
	status := "HEALTHY"
	if !r.Healthy {
		status = "UNHEALTHY"
	}

	summary := fmt.Sprintf("Shard Health Report [%s]\n", status)
	summary += fmt.Sprintf("Time: %s\n", r.CheckTime.Format("2006-01-02 15:04:05"))
	summary += fmt.Sprintf("Shard: %d/%d (Instance: %s)\n", r.CurrentShard, r.TotalShards, r.InstanceID)
	summary += fmt.Sprintf("Active Shards: %d, Dead Shards: %d\n", r.ActiveShards, r.DeadShards)
	summary += fmt.Sprintf("Accounts: %d/%d assigned to this shard\n", r.ShardAccounts, r.TotalAccounts)
	summary += fmt.Sprintf("Leader: %v\n", r.IsLeader)

	if len(r.Issues) > 0 {
		summary += "\nISSUES:\n"
		for _, issue := range r.Issues {
			summary += fmt.Sprintf("  - %s\n", issue)
		}
	}

	if len(r.Warnings) > 0 {
		summary += "\nWARNINGS:\n"
		for _, warning := range r.Warnings {
			summary += fmt.Sprintf("  - %s\n", warning)
		}
	}

	return summary
}

func PerformShardHealthCheck() {
	report, err := ShardHealthCheck()
	if err != nil {
		logger.Log.WithError(err).Error("Failed to perform shard health check")
		return
	}

	if report.Healthy {
		logger.Log.Info("Shard health check passed")
		logger.Log.Debugf("Health Report:\n%s", report.String())
	} else {
		logger.Log.Warn("Shard health check failed")
		logger.Log.Warnf("Health Report:\n%s", report.String())
	}

	shardMgr := GetAppShardManager()
	metadata := map[string]interface{}{
		"healthy":        report.Healthy,
		"issues_count":   len(report.Issues),
		"warnings_count": len(report.Warnings),
		"active_shards":  report.ActiveShards,
		"dead_shards":    report.DeadShards,
		"total_accounts": report.TotalAccounts,
		"shard_accounts": report.ShardAccounts,
	}

	status := "healthy"
	if !report.Healthy {
		status = "unhealthy"
	}

	LogAnalyticsEvent("shard_health_check", "", "", "", status,
		shardMgr.ShardID, shardMgr.InstanceID, metadata)
}

func GetShardDistributionStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var activeShards []models.ShardInfo
	if err := database.DB.Where("status = 'active'").Find(&activeShards).Error; err != nil {
		return stats, err
	}

	stats["active_shards"] = len(activeShards)
	stats["shard_details"] = activeShards

	userStats, err := ValidateUserShardAssignments()
	if err != nil {
		return stats, err
	}
	stats["user_distribution"] = userStats

	var allAccounts []models.Account
	if err := database.DB.Select("user_id").Find(&allAccounts).Error; err != nil {
		return stats, err
	}

	accountDistribution := make(map[int]int)
	totalShards := len(activeShards)
	if totalShards == 0 {
		totalShards = 1
	}

	for _, account := range allAccounts {
		if account.UserID == "" {
			continue
		}
		shard := getUserShard(account.UserID, totalShards)
		accountDistribution[shard]++
	}

	stats["account_distribution"] = accountDistribution
	stats["total_accounts"] = len(allAccounts)

	return stats, nil
}

func FixShardAssignments() error {
	shardMgr := GetAppShardManager()
	if !shardMgr.IsLeader() {
		return fmt.Errorf("only leader shard can fix assignments")
	}

	logger.Log.Info("Starting shard assignment fixes")

	shardMgr.performMaintenance()

	CleanupUsersByShardAssignment()
	CleanupOldShardInfo()

	stats, err := ValidateUserShardAssignments()
	if err != nil {
		return fmt.Errorf("failed to validate assignments after fix: %w", err)
	}

	logger.Log.Infof("Shard assignment fixes completed. Stats: %+v", stats)

	return nil
}
