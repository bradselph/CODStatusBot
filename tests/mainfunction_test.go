package services

import (
	"reflect"
	"testing"
	"time"

	"github.com/bradselph/CODStatusBot/models"
	"github.com/bwmarrin/discordgo"
)

func TestInitializeServices(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitializeServices()
		})
	}
}

func TestCheckAccounts(t *testing.T) {
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
			CheckAccounts(tt.args.s)
		})
	}
}

func Test_updateShardStats(t *testing.T) {
	type args struct {
		instanceID     string
		processedUsers int
		durationSec    float64
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
			if err := updateShardStats(tt.args.instanceID, tt.args.processedUsers, tt.args.durationSec); (err != nil) != tt.wantErr {
				t.Errorf("updateShardStats() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandleStatusChange(t *testing.T) {
	type args struct {
		s            *discordgo.Session
		account      models.Account
		newStatus    models.Status
		userSettings *models.UserSettings
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			HandleStatusChange(tt.args.s, tt.args.account, tt.args.newStatus, tt.args.userSettings)
		})
	}
}

func Test_createStatusChangeEmbed(t *testing.T) {
	type args struct {
		account        models.Account
		newStatus      models.Status
		previousStatus models.Status
		ban            models.Ban
	}
	tests := []struct {
		name string
		args args
		want *discordgo.MessageEmbed
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createStatusChangeEmbed(tt.args.account, tt.args.newStatus, tt.args.previousStatus, tt.args.ban); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createStatusChangeEmbed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_handleRankLockedNotification(t *testing.T) {
	type args struct {
		s       *discordgo.Session
		account models.Account
		ban     models.Ban
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handleRankLockedNotification(tt.args.s, tt.args.account, tt.args.ban)
		})
	}
}

func Test_getAffectedGames(t *testing.T) {
	type args struct {
		ssoCookie string
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
			if got := getAffectedGames(tt.args.ssoCookie); got != tt.want {
				t.Errorf("getAffectedGames() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getStatusFields(t *testing.T) {
	type args struct {
		account models.Account
		status  models.Status
		ban     models.Ban
	}
	tests := []struct {
		name string
		args args
		want []*discordgo.MessageEmbedField
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getStatusFields(tt.args.account, tt.args.status, tt.args.ban); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getStatusFields() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_disableAccount(t *testing.T) {
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
			disableAccount(tt.args.s, tt.args.account, tt.args.reason)
		})
	}
}

func Test_handlePermaBanNotification(t *testing.T) {
	type args struct {
		s       *discordgo.Session
		account models.Account
		ban     models.Ban
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlePermaBanNotification(tt.args.s, tt.args.account, tt.args.ban)
		})
	}
}

func Test_handleShadowBanNotification(t *testing.T) {
	type args struct {
		s       *discordgo.Session
		account models.Account
		ban     models.Ban
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handleShadowBanNotification(tt.args.s, tt.args.account, tt.args.ban)
		})
	}
}

func TestScheduleTempBanNotification(t *testing.T) {
	type args struct {
		s        *discordgo.Session
		account  models.Account
		duration string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ScheduleTempBanNotification(tt.args.s, tt.args.account, tt.args.duration)
		})
	}
}

func Test_getChannelForAnnouncement(t *testing.T) {
	type args struct {
		s            *discordgo.Session
		userID       string
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
			got, err := getChannelForAnnouncement(tt.args.s, tt.args.userID, tt.args.userSettings)
			if (err != nil) != tt.wantErr {
				t.Errorf("getChannelForAnnouncement() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getChannelForAnnouncement() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_calculateBanDuration(t *testing.T) {
	type args struct {
		endTime time.Time
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
			if got := calculateBanDuration(tt.args.endTime); got != tt.want {
				t.Errorf("calculateBanDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}
