package services

import (
	"testing"
	"time"

	"github.com/bradselph/CODStatusBot/models"
	"github.com/bwmarrin/discordgo"
)

func Test_validateRateLimit(t *testing.T) {
	type args struct {
		userID   string
		action   string
		duration time.Duration
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
			if got := validateRateLimit(tt.args.userID, tt.args.action, tt.args.duration); got != tt.want {
				t.Errorf("validateRateLimit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkActionRateLimit(t *testing.T) {
	type args struct {
		userID   string
		action   string
		duration time.Duration
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
			if got := checkActionRateLimit(tt.args.userID, tt.args.action, tt.args.duration); got != tt.want {
				t.Errorf("checkActionRateLimit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getActionLimit(t *testing.T) {
	type args struct {
		action string
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
			if got := getActionLimit(tt.args.action); got != tt.want {
				t.Errorf("getActionLimit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processUserAccounts(t *testing.T) {
	type args struct {
		s        *discordgo.Session
		userID   string
		accounts []models.Account
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processUserAccounts(tt.args.s, tt.args.userID, tt.args.accounts)
		})
	}
}

func Test_notifyUserOfServiceIssue(t *testing.T) {
	type args struct {
		s      *discordgo.Session
		userID string
		err    error
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifyUserOfServiceIssue(tt.args.s, tt.args.userID, tt.args.err)
		})
	}
}

func Test_shouldCheckAccount(t *testing.T) {
	type args struct {
		account  models.Account
		settings models.UserSettings
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
			if got := shouldCheckAccount(tt.args.account, tt.args.settings); got != tt.want {
				t.Errorf("shouldCheckAccount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_hasStatusChanged(t *testing.T) {
	type args struct {
		account   models.Account
		newStatus models.Status
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
			if got := hasStatusChanged(tt.args.account, tt.args.newStatus); got != tt.want {
				t.Errorf("hasStatusChanged() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_handleCheckError(t *testing.T) {
	type args struct {
		s       *discordgo.Session
		account *models.Account
		err     error
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handleCheckError(tt.args.s, tt.args.account, tt.args.err)
		})
	}
}

func Test_processNotifications(t *testing.T) {
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
			processNotifications(tt.args.s, tt.args.accounts, tt.args.userSettings)
		})
	}
}

func Test_isComingFromBannedState(t *testing.T) {
	type args struct {
		account models.Account
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
			if got := isComingFromBannedState(tt.args.account); got != tt.want {
				t.Errorf("isComingFromBannedState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_shouldCheckExpiration(t *testing.T) {
	type args struct {
		account models.Account
		now     time.Time
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
			if got := shouldCheckExpiration(tt.args.account, tt.args.now); got != tt.want {
				t.Errorf("shouldCheckExpiration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validateUserCaptchaService(t *testing.T) {
	type args struct {
		userID       string
		userSettings models.UserSettings
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
			if err := validateUserCaptchaService(tt.args.userID, tt.args.userSettings); (err != nil) != tt.wantErr {
				t.Errorf("validateUserCaptchaService() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateDefaultCapsolverConfig(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateDefaultCapsolverConfig(); (err != nil) != tt.wantErr {
				t.Errorf("ValidateDefaultCapsolverConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetCheckStatus(t *testing.T) {
	type args struct {
		isCheckDisabled bool
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
			if got := GetCheckStatus(tt.args.isCheckDisabled); got != tt.want {
				t.Errorf("GetCheckStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}
