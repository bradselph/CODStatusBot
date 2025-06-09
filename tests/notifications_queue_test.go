package services

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

func TestAdaptiveRateLimits_GetBackoffDuration(t *testing.T) {
	type args struct {
		userID string
	}
	tests := []struct {
		name string
		a    *AdaptiveRateLimits
		args args
		want time.Duration
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.GetBackoffDuration(tt.args.userID); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AdaptiveRateLimits.GetBackoffDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStartNotificationProcessor(t *testing.T) {
	type args struct {
		discord *discordgo.Session
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			StartNotificationProcessor(tt.args.discord)
		})
	}
}

func TestNotificationQueue_processNextNotification(t *testing.T) {
	type args struct {
		discord *discordgo.Session
	}
	tests := []struct {
		name string
		q    *NotificationQueue
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.q.processNextNotification(tt.args.discord)
		})
	}
}

func Test_sendMessageWithRetry(t *testing.T) {
	type args struct {
		s         *discordgo.Session
		channelID string
		content   string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := sendMessageWithRetry(tt.args.s, tt.args.channelID, tt.args.content); (err != nil) != tt.wantErr {
				t.Errorf("sendMessageWithRetry() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNotificationQueue_AddNotification(t *testing.T) {
	type args struct {
		item NotificationItem
	}
	tests := []struct {
		name string
		q    *NotificationQueue
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.q.AddNotification(tt.args.item)
		})
	}
}

func TestQueueNotification(t *testing.T) {
	type args struct {
		userID  string
		content string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			QueueNotification(tt.args.userID, tt.args.content)
		})
	}
}

func TestNotificationQueue_Shutdown(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		q       *NotificationQueue
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.q.Shutdown(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("NotificationQueue.Shutdown() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsUserRateLimited(t *testing.T) {
	type args struct {
		userID string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUserRateLimited(tt.args.userID); got != tt.want {
				t.Errorf("IsUserRateLimited() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetUserRateLimitStatus(t *testing.T) {
	type args struct {
		userID string
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 time.Duration
		want2 int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2 := GetUserRateLimitStatus(tt.args.userID)
			if got != tt.want {
				t.Errorf("GetUserRateLimitStatus() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("GetUserRateLimitStatus() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("GetUserRateLimitStatus() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}
