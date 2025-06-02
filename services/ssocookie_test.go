package services

import (
	"reflect"
	"testing"
	"time"
)

func TestDecodeSSOCookie(t *testing.T) {
	type args struct {
		encodedStr string
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeSSOCookie(tt.args.encodedStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeSSOCookie() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DecodeSSOCookie() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckSSOCookieExpiration(t *testing.T) {
	type args struct {
		expirationTimestamp int64
	}
	tests := []struct {
		name    string
		args    args
		want    time.Duration
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckSSOCookieExpiration(tt.args.expirationTimestamp)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckSSOCookieExpiration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckSSOCookieExpiration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatExpirationTime(t *testing.T) {
	type args struct {
		expirationTimestamp int64
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
			if got := FormatExpirationTime(tt.args.expirationTimestamp); got != tt.want {
				t.Errorf("FormatExpirationTime() = %v, want %v", got, tt.want)
			}
		})
	}
}
