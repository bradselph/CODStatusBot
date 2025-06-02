package services

import (
	"reflect"
	"testing"
)

func Test_cleanupStaleTasks(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupStaleTasks()
		})
	}
}

func TestStoreCapsolverTaskInfo(t *testing.T) {
	type args struct {
		taskID string
		token  string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			StoreCapsolverTaskInfo(tt.args.taskID, tt.args.token)
		})
	}
}

func TestReportCapsolverTaskResult(t *testing.T) {
	type args struct {
		token        string
		isValid      bool
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
			ReportCapsolverTaskResult(tt.args.token, tt.args.isValid, tt.args.errorMessage)
		})
	}
}

func Test_reportCapsolverTaskResult(t *testing.T) {
	type args struct {
		apiKey       string
		appID        string
		taskID       string
		isValid      bool
		errorCode    int
		errorMessage string
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
			if err := reportCapsolverTaskResult(tt.args.apiKey, tt.args.appID, tt.args.taskID, tt.args.isValid, tt.args.errorCode, tt.args.errorMessage); (err != nil) != tt.wantErr {
				t.Errorf("reportCapsolverTaskResult() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsServiceEnabled(t *testing.T) {
	type args struct {
		provider string
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
			if got := IsServiceEnabled(tt.args.provider); got != tt.want {
				t.Errorf("IsServiceEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerifyEZCaptchaConfig(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VerifyEZCaptchaConfig(); got != tt.want {
				t.Errorf("VerifyEZCaptchaConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewCaptchaSolver(t *testing.T) {
	type args struct {
		apiKey   string
		provider string
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
			got, err := NewCaptchaSolver(tt.args.apiKey, tt.args.provider)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCaptchaSolver() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCaptchaSolver() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCapsolverSolver_SolveReCaptchaV2(t *testing.T) {
	type args struct {
		siteKey string
		pageURL string
	}
	tests := []struct {
		name    string
		s       *CapsolverSolver
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.SolveReCaptchaV2(tt.args.siteKey, tt.args.pageURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("CapsolverSolver.SolveReCaptchaV2() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CapsolverSolver.SolveReCaptchaV2() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEZCaptchaSolver_SolveReCaptchaV2(t *testing.T) {
	type args struct {
		siteKey string
		pageURL string
	}
	tests := []struct {
		name    string
		s       *EZCaptchaSolver
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.SolveReCaptchaV2(tt.args.siteKey, tt.args.pageURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("EZCaptchaSolver.SolveReCaptchaV2() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EZCaptchaSolver.SolveReCaptchaV2() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTwoCaptchaSolver_SolveReCaptchaV2(t *testing.T) {
	type args struct {
		siteKey string
		pageURL string
	}
	tests := []struct {
		name    string
		s       *TwoCaptchaSolver
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.SolveReCaptchaV2(tt.args.siteKey, tt.args.pageURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("TwoCaptchaSolver.SolveReCaptchaV2() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("TwoCaptchaSolver.SolveReCaptchaV2() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCapsolverSolver_createTask(t *testing.T) {
	type args struct {
		siteKey string
		pageURL string
	}
	tests := []struct {
		name    string
		s       *CapsolverSolver
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.createTask(tt.args.siteKey, tt.args.pageURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("CapsolverSolver.createTask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CapsolverSolver.createTask() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEZCaptchaSolver_createTask(t *testing.T) {
	type args struct {
		siteKey string
		pageURL string
	}
	tests := []struct {
		name    string
		s       *EZCaptchaSolver
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.createTask(tt.args.siteKey, tt.args.pageURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("EZCaptchaSolver.createTask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EZCaptchaSolver.createTask() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTwoCaptchaSolver_createTask(t *testing.T) {
	type args struct {
		siteKey string
		pageURL string
	}
	tests := []struct {
		name    string
		s       *TwoCaptchaSolver
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.createTask(tt.args.siteKey, tt.args.pageURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("TwoCaptchaSolver.createTask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("TwoCaptchaSolver.createTask() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCapsolverSolver_getTaskResult(t *testing.T) {
	type args struct {
		taskID string
	}
	tests := []struct {
		name    string
		s       *CapsolverSolver
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.getTaskResult(tt.args.taskID)
			if (err != nil) != tt.wantErr {
				t.Errorf("CapsolverSolver.getTaskResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CapsolverSolver.getTaskResult() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEZCaptchaSolver_getTaskResult(t *testing.T) {
	type args struct {
		taskID string
	}
	tests := []struct {
		name    string
		s       *EZCaptchaSolver
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.getTaskResult(tt.args.taskID)
			if (err != nil) != tt.wantErr {
				t.Errorf("EZCaptchaSolver.getTaskResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EZCaptchaSolver.getTaskResult() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTwoCaptchaSolver_getTaskResult(t *testing.T) {
	type args struct {
		taskID string
	}
	tests := []struct {
		name    string
		s       *TwoCaptchaSolver
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.getTaskResult(tt.args.taskID)
			if (err != nil) != tt.wantErr {
				t.Errorf("TwoCaptchaSolver.getTaskResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("TwoCaptchaSolver.getTaskResult() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sendRequest(t *testing.T) {
	type args struct {
		url     string
		payload interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sendRequest(tt.args.url, tt.args.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("sendRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sendRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getBalanceThreshold(t *testing.T) {
	type args struct {
		provider string
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getBalanceThreshold(tt.args.provider); got != tt.want {
				t.Errorf("getBalanceThreshold() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateCaptchaKey(t *testing.T) {
	type args struct {
		apiKey   string
		provider string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		want1   float64
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := ValidateCaptchaKey(tt.args.apiKey, tt.args.provider)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCaptchaKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ValidateCaptchaKey() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("ValidateCaptchaKey() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_validateCapsolverKey(t *testing.T) {
	type args struct {
		apiKey string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		want1   float64
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := validateCapsolverKey(tt.args.apiKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCapsolverKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("validateCapsolverKey() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("validateCapsolverKey() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_validateEZCaptchaKey(t *testing.T) {
	type args struct {
		apiKey string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		want1   float64
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := validateEZCaptchaKey(tt.args.apiKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEZCaptchaKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("validateEZCaptchaKey() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("validateEZCaptchaKey() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_validate2CaptchaKey(t *testing.T) {
	type args struct {
		apiKey string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		want1   float64
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := validate2CaptchaKey(tt.args.apiKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("validate2CaptchaKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("validate2CaptchaKey() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("validate2CaptchaKey() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
