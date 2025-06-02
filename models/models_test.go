package models

import (
	"testing"

	"gorm.io/gorm"
)

func TestUserSettings_EnsureMapsInitialized(t *testing.T) {
	tests := []struct {
		name string
		u    *UserSettings
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.u.EnsureMapsInitialized()
		})
	}
}

func TestUserSettings_BeforeCreate(t *testing.T) {
	type args struct {
		tx *gorm.DB
	}
	tests := []struct {
		name    string
		u       *UserSettings
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.u.BeforeCreate(tt.args.tx); (err != nil) != tt.wantErr {
				t.Errorf("UserSettings.BeforeCreate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserSettings_AfterFind(t *testing.T) {
	type args struct {
		tx *gorm.DB
	}
	tests := []struct {
		name    string
		u       *UserSettings
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.u.AfterFind(tt.args.tx); (err != nil) != tt.wantErr {
				t.Errorf("UserSettings.AfterFind() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserSettings_UpdateInteractionContext(t *testing.T) {
	type args struct {
		guildID string
	}
	tests := []struct {
		name string
		u    *UserSettings
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.u.UpdateInteractionContext(tt.args.guildID)
		})
	}
}

func TestBan_BeforeCreate(t *testing.T) {
	type args struct {
		tx *gorm.DB
	}
	tests := []struct {
		name    string
		b       *Ban
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.b.BeforeCreate(tt.args.tx); (err != nil) != tt.wantErr {
				t.Errorf("Ban.BeforeCreate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAccount_BeforeCreate(t *testing.T) {
	type args struct {
		tx *gorm.DB
	}
	tests := []struct {
		name    string
		a       *Account
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.a.BeforeCreate(tt.args.tx); (err != nil) != tt.wantErr {
				t.Errorf("Account.BeforeCreate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAccount_BeforeSave(t *testing.T) {
	type args struct {
		tx *gorm.DB
	}
	tests := []struct {
		name    string
		a       *Account
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.a.BeforeSave(tt.args.tx); (err != nil) != tt.wantErr {
				t.Errorf("Account.BeforeSave() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAccount_AfterFind(t *testing.T) {
	type args struct {
		tx *gorm.DB
	}
	tests := []struct {
		name    string
		a       *Account
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.a.AfterFind(tt.args.tx); (err != nil) != tt.wantErr {
				t.Errorf("Account.AfterFind() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
