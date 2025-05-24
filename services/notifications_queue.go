package services

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bwmarrin/discordgo"
)

type NotificationQueue struct {
	items    []NotificationItem
	mutex    sync.RWMutex
	shutdown chan struct{}
	wg       sync.WaitGroup
}

type NotificationItem struct {
	UserID     string
	Content    string
	AddedAt    time.Time
	RetryCount int
	Priority   int
}

type AdaptiveRateLimits struct {
	sync.RWMutex
	UserBackoffs  map[string]*UserBackoff
	BaseLimit     int
	HistoryWindow time.Duration
}

type UserBackoff struct {
	LastSent            time.Time
	NotificationHistory []time.Time
	BackoffMultiplier   float64
	ConsecutiveCount    int
}

var notificationQueue = &NotificationQueue{
	items:    make([]NotificationItem, 0),
	shutdown: make(chan struct{}),
}

var adaptiveRateLimits = &AdaptiveRateLimits{
	UserBackoffs:  make(map[string]*UserBackoff),
	BaseLimit:     configuration.Get().Notifications.MaxPerHour,
	HistoryWindow: configuration.Get().Notifications.BackoffHistoryWindow,
}

func (a *AdaptiveRateLimits) GetBackoffDuration(userID string) time.Duration {
	cfg := configuration.Get()
	backoff, exists := a.UserBackoffs[userID]
	if !exists {
		return 0
	}

	baseBackoff := cfg.Notifications.BackoffBaseInterval
	if backoff.BackoffMultiplier <= 1.0 {
		return baseBackoff
	}

	duration := time.Duration(float64(baseBackoff) * backoff.BackoffMultiplier)
	maxBackoff := time.Hour * 1
	if duration > maxBackoff {
		return maxBackoff
	}
	return duration
}

func StartNotificationProcessor(discord *discordgo.Session) {
	notificationQueue.wg.Add(1)
	go func() {
		defer notificationQueue.wg.Done()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-notificationQueue.shutdown:
				logger.Log.Info("Notification processor shutting down")
				return
			case <-ticker.C:
				notificationQueue.processNextNotification(discord)
			}
		}
	}()
}

func (q *NotificationQueue) processNextNotification(discord *discordgo.Session) {
	q.mutex.Lock()
	if len(q.items) == 0 {
		q.mutex.Unlock()
		return
	}

	highestPriorityIndex := 0
	for i := 1; i < len(q.items); i++ {
		if q.items[i].Priority > q.items[highestPriorityIndex].Priority {
			highestPriorityIndex = i
		}
	}

	item := q.items[highestPriorityIndex]

	q.items[highestPriorityIndex] = q.items[len(q.items)-1]
	q.items = q.items[:len(q.items)-1]
	q.mutex.Unlock()

	processDone := make(chan bool, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Log.Errorf("Panic recovered in notification processing: %v", r)
				processDone <- false
				return
			}
			processDone <- true
		}()

		if IsUserRateLimited(item.UserID) {
			if item.RetryCount < 3 {
				item.RetryCount++
				item.Priority--
				q.AddNotification(item)
			} else {
				logger.Log.Warnf("Dropping notification for user %s after max retries", item.UserID)
			}
			return
		}

		ch := make(chan *discordgo.Channel, 1)
		errCh := make(chan error, 1)

		go func() {
			defer close(ch)
			defer close(errCh)
			channel, err := discord.UserChannelCreate(item.UserID)
			if err != nil {
				errCh <- err
				return
			}
			ch <- channel
		}()

		select {
		case channel := <-ch:
			if err := sendMessageWithRetry(discord, channel.ID, item.Content); err != nil {
				logger.Log.WithError(err).Errorf("Failed to send notification to user %s", item.UserID)
				if item.RetryCount < 3 {
					item.RetryCount++
					item.Priority--
					q.AddNotification(item)
				}
			}
		case err := <-errCh:
			logger.Log.WithError(err).Errorf("Error creating DM channel for user %s", item.UserID)
		case <-time.After(5 * time.Second):
			logger.Log.Warnf("Timeout creating DM channel for user %s", item.UserID)
			if item.RetryCount < 3 {
				item.RetryCount++
				item.Priority--
				q.AddNotification(item)
			}
		}

		adaptiveRateLimits.Lock()
		backoff, exists := adaptiveRateLimits.UserBackoffs[item.UserID]
		if !exists {
			backoff = &UserBackoff{
				BackoffMultiplier: 1.0,
			}
			adaptiveRateLimits.UserBackoffs[item.UserID] = backoff
		}
		backoff.LastSent = time.Now()

		now := time.Now()
		history := make([]time.Time, 0)
		for _, t := range backoff.NotificationHistory {
			if now.Sub(t) < adaptiveRateLimits.HistoryWindow {
				history = append(history, t)
			}
		}
		history = append(history, now)
		backoff.NotificationHistory = history
		adaptiveRateLimits.Unlock()
	}()

	select {
	case <-processDone:
	case <-time.After(30 * time.Second):
		logger.Log.Warnf("Notification processing timed out for user %s", item.UserID)
		if item.RetryCount < 3 {
			item.RetryCount++
			item.Priority--
			q.AddNotification(item)
		}
	}
}
func sendMessageWithRetry(s *discordgo.Session, channelID, content string) error {
	var lastErr error
	for retries := 0; retries < 3; retries++ {
		if _, err := s.ChannelMessageSend(channelID, content); err == nil {
			return nil
		} else {
			lastErr = err
			if strings.Contains(err.Error(), "Unknown Channel") ||
				strings.Contains(err.Error(), "Missing Access") {
				return err
			}
		}
		time.Sleep(time.Duration(retries+1) * time.Second)
	}
	return lastErr
}

func (q *NotificationQueue) AddNotification(item NotificationItem) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if item.Priority == 0 {
		item.Priority = 1
	}

	q.items = append(q.items, item)
}

func QueueNotification(userID string, content string) {
	item := NotificationItem{
		UserID:     userID,
		Content:    content,
		AddedAt:    time.Now(),
		RetryCount: 0,
		Priority:   1,
	}
	notificationQueue.AddNotification(item)
}

func (q *NotificationQueue) Shutdown(ctx context.Context) error {
	close(q.shutdown)

	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

/*
	func CleanupOldRateLimitData() {
		adaptiveRateLimits.Lock()
		defer adaptiveRateLimits.Unlock()

		now := time.Now()
		initialCount := len(adaptiveRateLimits.UserBackoffs)
		cleaned := 0

		for userID, backoff := range adaptiveRateLimits.UserBackoffs {
			var recentHistory []time.Time
			for _, t := range backoff.NotificationHistory {
				if now.Sub(t) <= adaptiveRateLimits.HistoryWindow {
					recentHistory = append(recentHistory, t)
				}
			}

			if len(recentHistory) == 0 {
				backoff.BackoffMultiplier = 1.0
				backoff.ConsecutiveCount = 0
				cleaned++
			}

			backoff.NotificationHistory = recentHistory

			if len(recentHistory) == 0 && now.Sub(backoff.LastSent) > adaptiveRateLimits.HistoryWindow {
				delete(adaptiveRateLimits.UserBackoffs, userID)
				cleaned++
			}
		}

		logger.Log.Infof("Rate limit cleanup completed: processed %d entries, cleaned %d", initialCount, cleaned)
	}
*/
func IsUserRateLimited(userID string) bool {
	if userID == "" {
		logger.Log.Warning("Empty userID passed to IsUserRateLimited")
		return false
	}

	adaptiveRateLimits.RLock()
	defer adaptiveRateLimits.RUnlock()

	backoff, exists := adaptiveRateLimits.UserBackoffs[userID]
	if !exists {
		return false
	}

	if backoff.NotificationHistory == nil {
		logger.Log.Warnf("Nil notification history for user %s", userID)
		return false
	}

	if time.Since(backoff.LastSent) < adaptiveRateLimits.GetBackoffDuration(userID) {
		return true
	}

	if len(backoff.NotificationHistory) >= adaptiveRateLimits.BaseLimit {
		oldest := backoff.NotificationHistory[0]
		if time.Since(oldest) < time.Hour {
			return true
		}
	}

	return false
}

func GetUserRateLimitStatus(userID string) (bool, time.Duration, int) {
	adaptiveRateLimits.RLock()
	defer adaptiveRateLimits.RUnlock()

	backoff, exists := adaptiveRateLimits.UserBackoffs[userID]
	if !exists {
		return false, 0, adaptiveRateLimits.BaseLimit
	}

	isLimited := IsUserRateLimited(userID)
	remainingBackoff := time.Duration(0)
	if isLimited {
		remainingBackoff = adaptiveRateLimits.GetBackoffDuration(userID) - time.Since(backoff.LastSent)
		if remainingBackoff < 0 {
			remainingBackoff = 0
		}
	}

	recentCount := 0
	now := time.Now()
	for _, t := range backoff.NotificationHistory {
		if now.Sub(t) < time.Hour {
			recentCount++
		}
	}
	remainingAllowed := adaptiveRateLimits.BaseLimit - recentCount

	return isLimited, remainingBackoff, remainingAllowed
}
