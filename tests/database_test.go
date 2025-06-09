package database

import "testing"

func TestDatabaselogin(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Databaselogin(); (err != nil) != tt.wantErr {
				t.Errorf("Databaselogin() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCloseConnection(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CloseConnection(); (err != nil) != tt.wantErr {
				t.Errorf("CloseConnection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
