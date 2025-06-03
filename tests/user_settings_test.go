package services

import (
	"reflect"
	"testing"

	"github.com/bradselph/CODStatusBot/models"
)

func Test_initDefaultSettings(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initDefaultSettings()
		})
	}
}

func TestGetUserSettings(t *testing.T) {
	type args struct {
		userID string
	}
	tests := []struct {
		name    string
		args    args
		want    models.UserSettings
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetUserSettings(tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserSettings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetUserSettings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetUserCaptchaKey(t *testing.T) {
	type args struct {
		userID string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   float64
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := GetUserCaptchaKey(tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserCaptchaKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetUserCaptchaKey() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetUserCaptchaKey() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestGetCaptchaSolver(t *testing.T) {
	type args struct {
		userID string
	}
	tests := []struct {
		name    string
		args    args
		want    CaptchaSolver
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCaptchaSolver(tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCaptchaSolver() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCaptchaSolver() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDefaultSettings(t *testing.T) {
	tests := []struct {
		name    string
		want    models.UserSettings
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetDefaultSettings()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDefaultSettings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDefaultSettings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoveCaptchaKey(t *testing.T) {
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
			if err := RemoveCaptchaKey(tt.args.userID); (err != nil) != tt.wantErr {
				t.Errorf("RemoveCaptchaKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
