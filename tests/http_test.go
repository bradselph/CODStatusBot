package services

import (
	"reflect"
	"testing"

	"github.com/bradselph/CODStatusBot/models"
)

func TestVerifySSOCookie(t *testing.T) {
	type args struct {
		ssoCookie string
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
			if got := VerifySSOCookie(tt.args.ssoCookie); got != tt.want {
				t.Errorf("VerifySSOCookie() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckAccount(t *testing.T) {
	type args struct {
		ssoCookie     string
		userID        string
		captchaAPIKey string
	}
	tests := []struct {
		name    string
		args    args
		want    models.Status
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckAccount(tt.args.ssoCookie, tt.args.userID, tt.args.captchaAPIKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckAccount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckAccount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_handleRateLimitCheck(t *testing.T) {
	type args struct {
		userSettings models.UserSettings
		endpoint     string
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
			if err := handleRateLimitCheck(tt.args.userSettings, tt.args.endpoint); (err != nil) != tt.wantErr {
				t.Errorf("handleRateLimitCheck() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_updateRateLimitBackoff(t *testing.T) {
	type args struct {
		userSettings models.UserSettings
		endpoint     string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateRateLimitBackoff(tt.args.userSettings, tt.args.endpoint)
		})
	}
}

func Test_resetRateLimitBackoff(t *testing.T) {
	type args struct {
		userSettings models.UserSettings
		endpoint     string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetRateLimitBackoff(tt.args.userSettings, tt.args.endpoint)
		})
	}
}

func TestUpdateCaptchaUsage(t *testing.T) {
	type args struct {
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
			if err := UpdateCaptchaUsage(tt.args.userID); (err != nil) != tt.wantErr {
				t.Errorf("UpdateCaptchaUsage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckAccountAge(t *testing.T) {
	type args struct {
		ssoCookie string
	}
	tests := []struct {
		name    string
		args    args
		want    int
		want1   int
		want2   int
		want3   int64
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2, got3, err := CheckAccountAge(tt.args.ssoCookie)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckAccountAge() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CheckAccountAge() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("CheckAccountAge() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("CheckAccountAge() got2 = %v, want %v", got2, tt.want2)
			}
			if got3 != tt.want3 {
				t.Errorf("CheckAccountAge() got3 = %v, want %v", got3, tt.want3)
			}
		})
	}
}

func TestCheckVIPStatus(t *testing.T) {
	type args struct {
		ssoCookie string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckVIPStatus(tt.args.ssoCookie)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckVIPStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CheckVIPStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateAndGetAccountInfo(t *testing.T) {
	type args struct {
		ssoCookie string
	}
	tests := []struct {
		name    string
		args    args
		want    *AccountValidationResult
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateAndGetAccountInfo(tt.args.ssoCookie)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAndGetAccountInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValidateAndGetAccountInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_notifyUserAboutFallbackUsage(t *testing.T) {
	type args struct {
		userID          string
		primaryProvider string
		usedProvider    string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifyUserAboutFallbackUsage(tt.args.userID, tt.args.primaryProvider, tt.args.usedProvider)
		})
	}
}
