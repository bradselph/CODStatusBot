package services

import (
	"reflect"
	"testing"
	"time"

	"github.com/bradselph/CODStatusBot/models"
	"github.com/bwmarrin/discordgo"
)

func Test_formatVIPStatus(t *testing.T) {
	type args struct {
		isVIP bool
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatVIPStatus(tt.args.isVIP); got != tt.want {
				t.Errorf("formatVIPStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_formatCheckStatus(t *testing.T) {
	type args struct {
		isDisabled bool
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatCheckStatus(tt.args.isDisabled); got != tt.want {
				t.Errorf("formatCheckStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getNotificationType(t *testing.T) {
	type args struct {
		status models.Status
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getNotificationType(tt.args.status); got != tt.want {
				t.Errorf("getNotificationType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getDefaultCooldown(t *testing.T) {
	tests := []struct {
		name string
		want time.Duration
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDefaultCooldown(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getDefaultCooldown() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getMaxNotificationsPerHour(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getMaxNotificationsPerHour(); got != tt.want {
				t.Errorf("getMaxNotificationsPerHour() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getMaxNotificationsPerDay(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getMaxNotificationsPerDay(); got != tt.want {
				t.Errorf("getMaxNotificationsPerDay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getMinNotificationInterval(t *testing.T) {
	tests := []struct {
		name string
		want time.Duration
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getMinNotificationInterval(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getMinNotificationInterval() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewNotificationLimiter(t *testing.T) {
	tests := []struct {
		name string
		want *NotificationLimiter
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewNotificationLimiter(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewNotificationLimiter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotifyAdmin(t *testing.T) {
	type args struct {
		s       *discordgo.Session
		message string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NotifyAdmin(tt.args.s, tt.args.message)
		})
	}
}

func TestGetCooldownDuration(t *testing.T) {
	type args struct {
		userSettings     models.UserSettings
		notificationType string
		defaultCooldown  time.Duration
	}
	tests := []struct {
		name string
		args args
		want time.Duration
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCooldownDuration(tt.args.userSettings, tt.args.notificationType, tt.args.defaultCooldown); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCooldownDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetNotificationChannel(t *testing.T) {
	type args struct {
		s            *discordgo.Session
		account      models.Account
		userSettings models.UserSettings
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetNotificationChannel(tt.args.s, tt.args.account, tt.args.userSettings)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNotificationChannel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetNotificationChannel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	type args struct {
		d time.Duration
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatDuration(tt.args.d); got != tt.want {
				t.Errorf("FormatDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckAndNotifyBalance(t *testing.T) {
	type args struct {
		s       *discordgo.Session
		userID  string
		balance float64
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CheckAndNotifyBalance(tt.args.s, tt.args.userID, tt.args.balance)
		})
	}
}

func TestBalanceWarningFields(t *testing.T) {
	tests := []struct {
		name string
		want []*discordgo.MessageEmbedField
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BalanceWarningFields(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BalanceWarningFields() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScheduleBalanceChecks(t *testing.T) {
	type args struct {
		s *discordgo.Session
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ScheduleBalanceChecks(tt.args.s)
		})
	}
}

func TestDisableUserCaptcha(t *testing.T) {
	type args struct {
		s      *discordgo.Session
		userID string
		reason string
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
			if err := DisableUserCaptcha(tt.args.s, tt.args.userID, tt.args.reason); (err != nil) != tt.wantErr {
				t.Errorf("DisableUserCaptcha() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getEnabledServicesString(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getEnabledServicesString(); got != tt.want {
				t.Errorf("getEnabledServicesString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotificationLimiter_CanSendNotification(t *testing.T) {
	type args struct {
		userID           string
		notificationType string
	}
	tests := []struct {
		name string
		nl   *NotificationLimiter
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.nl.CanSendNotification(tt.args.userID, tt.args.notificationType); got != tt.want {
				t.Errorf("NotificationLimiter.CanSendNotification() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSendNotification(t *testing.T) {
	type args struct {
		s                *discordgo.Session
		account          models.Account
		embed            *discordgo.MessageEmbed
		content          string
		notificationType string
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
			if err := SendNotification(tt.args.s, tt.args.account, tt.args.embed, tt.args.content, tt.args.notificationType); (err != nil) != tt.wantErr {
				t.Errorf("SendNotification() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_storeSuppressedNotification(t *testing.T) {
	type args struct {
		userID           string
		notificationType string
		embed            *discordgo.MessageEmbed
		content          string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storeSuppressedNotification(tt.args.userID, tt.args.notificationType, tt.args.embed, tt.args.content)
		})
	}
}

func TestNotifyAdminWithCooldown(t *testing.T) {
	type args struct {
		s                *discordgo.Session
		message          string
		cooldownDuration time.Duration
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NotifyAdminWithCooldown(tt.args.s, tt.args.message, tt.args.cooldownDuration)
		})
	}
}

func TestSendGlobalAnnouncement(t *testing.T) {
	type args struct {
		s      *discordgo.Session
		userID string
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
			if err := SendGlobalAnnouncement(tt.args.s, tt.args.userID); (err != nil) != tt.wantErr {
				t.Errorf("SendGlobalAnnouncement() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSendAnnouncementToAllUsers(t *testing.T) {
	type args struct {
		s *discordgo.Session
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
			if err := SendAnnouncementToAllUsers(tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("SendAnnouncementToAllUsers() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNotifyUserAboutDisabledAccount(t *testing.T) {
	type args struct {
		s       *discordgo.Session
		account models.Account
		reason  string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NotifyUserAboutDisabledAccount(tt.args.s, tt.args.account, tt.args.reason)
		})
	}
}

func TestNotifyCookieExpiringSoon(t *testing.T) {
	type args struct {
		s        *discordgo.Session
		accounts []models.Account
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
			if err := NotifyCookieExpiringSoon(tt.args.s, tt.args.accounts); (err != nil) != tt.wantErr {
				t.Errorf("NotifyCookieExpiringSoon() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSendConsolidatedDailyUpdate(t *testing.T) {
	type args struct {
		s            *discordgo.Session
		userID       string
		userSettings models.UserSettings
		accounts     []models.Account
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SendConsolidatedDailyUpdate(tt.args.s, tt.args.userID, tt.args.userSettings, tt.args.accounts)
		})
	}
}

func TestGetStatusIcon(t *testing.T) {
	type args struct {
		status models.Status
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetStatusIcon(tt.args.status); got != tt.want {
				t.Errorf("GetStatusIcon() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_formatAccountStatus(t *testing.T) {
	type args struct {
		account             models.Account
		status              models.Status
		timeUntilExpiration time.Duration
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatAccountStatus(tt.args.account, tt.args.status, tt.args.timeUntilExpiration); got != tt.want {
				t.Errorf("formatAccountStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkAccountsNeedingAttention(t *testing.T) {
	type args struct {
		s            *discordgo.Session
		accounts     []models.Account
		userSettings models.UserSettings
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkAccountsNeedingAttention(tt.args.s, tt.args.accounts, tt.args.userSettings)
		})
	}
}

func Test_notifyAccountErrors(t *testing.T) {
	type args struct {
		s             *discordgo.Session
		errorAccounts []models.Account
		userSettings  models.UserSettings
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifyAccountErrors(tt.args.s, tt.args.errorAccounts, tt.args.userSettings)
		})
	}
}

func TestTrackMessageFailure(t *testing.T) {
	type args struct {
		userID       string
		errorMessage string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			TrackMessageFailure(tt.args.userID, tt.args.errorMessage)
		})
	}
}
