package services

import (
	"reflect"
	"testing"

	"github.com/bradselph/CODStatusBot/models"
	"github.com/bwmarrin/discordgo"
)

func TestCreateAnnouncementEmbed(t *testing.T) {
	tests := []struct {
		name string
		want *discordgo.MessageEmbed
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateAnnouncementEmbed(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateAnnouncementEmbed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetColorForStatus(t *testing.T) {
	type args struct {
		status          models.Status
		isExpiredCookie bool
		isCheckDisabled bool
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetColorForStatus(tt.args.status, tt.args.isExpiredCookie, tt.args.isCheckDisabled); got != tt.want {
				t.Errorf("GetColorForStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmbedTitleFromStatus(t *testing.T) {
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
			if got := EmbedTitleFromStatus(tt.args.status); got != tt.want {
				t.Errorf("EmbedTitleFromStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStatusDescription(t *testing.T) {
	type args struct {
		status       models.Status
		accountTitle string
		ban          models.Ban
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
			if got := GetStatusDescription(tt.args.status, tt.args.accountTitle, tt.args.ban); got != tt.want {
				t.Errorf("GetStatusDescription() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_formatAffectedGames(t *testing.T) {
	type args struct {
		games string
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
			if got := formatAffectedGames(tt.args.games); got != tt.want {
				t.Errorf("formatAffectedGames() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_formatGameTitle(t *testing.T) {
	type args struct {
		title string
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
			if got := formatGameTitle(tt.args.title); got != tt.want {
				t.Errorf("formatGameTitle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_formatEnforcement(t *testing.T) {
	type args struct {
		enforcement string
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
			if got := formatEnforcement(tt.args.enforcement); got != tt.want {
				t.Errorf("formatEnforcement() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateAccountListEmbed(t *testing.T) {
	type args struct {
		accounts   []models.Account
		userID     string
		page       int
		totalPages int
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
			if got := CreateAccountListEmbed(tt.args.accounts, tt.args.userID, tt.args.page, tt.args.totalPages); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateAccountListEmbed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getStatusEmoji(t *testing.T) {
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
			if got := getStatusEmoji(tt.args.status); got != tt.want {
				t.Errorf("getStatusEmoji() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateCheckResultEmbed(t *testing.T) {
	type args struct {
		account      models.Account
		status       models.Status
		userSettings models.UserSettings
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
			if got := CreateCheckResultEmbed(tt.args.account, tt.args.status, tt.args.userSettings); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateCheckResultEmbed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateErrorEmbed(t *testing.T) {
	type args struct {
		title        string
		errorMessage string
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
			if got := CreateErrorEmbed(tt.args.title, tt.args.errorMessage); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateErrorEmbed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateSuccessEmbed(t *testing.T) {
	type args struct {
		title   string
		message string
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
			if got := CreateSuccessEmbed(tt.args.title, tt.args.message); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateSuccessEmbed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateInfoEmbed(t *testing.T) {
	type args struct {
		title   string
		message string
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
			if got := CreateInfoEmbed(tt.args.title, tt.args.message); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateInfoEmbed() = %v, want %v", got, tt.want)
			}
		})
	}
}
