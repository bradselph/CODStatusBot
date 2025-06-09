package logger

import "testing"

func TestInitializeLogger(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := InitializeLogger(); (err != nil) != tt.wantErr {
				t.Errorf("InitializeLogger() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_rotateLog(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rotateLog()
		})
	}
}

func Test_checkRotation(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkRotation()
		})
	}
}
