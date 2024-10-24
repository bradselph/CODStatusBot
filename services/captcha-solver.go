package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func LoadEnvironmentVariables() error {
	clientKey = os.Getenv("EZCAPTCHA_CLIENT_KEY")
	ezappID = os.Getenv("EZAPPID")
	softID = os.Getenv("SOFT_ID")
	siteAction = os.Getenv("SITE_ACTION")

	if clientKey == "" || ezappID == "" || softID == "" || siteAction == "" {
		return fmt.Errorf("EZCAPTCHA_CLIENT_KEY, EZAPPID, SOFT_ID, or SITE_ACTION is not set in the environment")
	}
	return nil
}

var (
	clientKey  string
	ezappID    string
	softID     string
	siteKey    string
	pageURL    string
	siteAction string
)

type CaptchaSolver interface {
	SolveReCaptchaV2(siteKey, pageURL string) (string, error)
}

type EZCaptchaSolver struct {
	APIKey  string
	EzappID string
}

type TwoCaptchaSolver struct {
	APIKey string
	SoftID string
}

const (
	MaxRetries    = 6
	RetryInterval = 10 * time.Second

	EZCaptchaCreateEndpoint  = "https://api.ez-captcha.com/createTask"
	EZCaptchaResultEndpoint  = "https://api.ez-captcha.com/getTaskResult"
	TwoCaptchaCreateEndpoint = "https://api.2captcha.com/createTask"
	TwoCaptchaResultEndpoint = "https://api.2captcha.com/getTaskResult"
)

func NewCaptchaSolver(apiKey, provider string) (CaptchaSolver, error) {
	switch provider {
	case "ezcaptcha":
		return &EZCaptchaSolver{APIKey: apiKey, EzappID: ezappID}, nil
	case "2captcha":
		return &TwoCaptchaSolver{APIKey: apiKey, SoftID: softID}, nil
	default:
		return nil, errors.New("unsupported captcha provider")
	}
}

func (s *EZCaptchaSolver) SolveReCaptchaV2(siteKey, pageURL string) (string, error) {
	taskID, err := s.createTask(siteKey, pageURL)
	if err != nil {
		return "", fmt.Errorf("failed to create captcha task: %w", err)
	}
	return s.getTaskResult(taskID)
}

func (s *TwoCaptchaSolver) SolveReCaptchaV2(siteKey, pageURL string) (string, error) {
	taskID, err := s.createTask(siteKey, pageURL)
	if err != nil {
		return "", fmt.Errorf("failed to create captcha task: %w", err)
	}
	return s.getTaskResult(taskID)
}

func (s *EZCaptchaSolver) createTask(siteKey, pageURL string) (string, error) {
	payload := map[string]interface{}{
		"clientKey": s.APIKey,
		"appId":     s.EzappID,
		"task": map[string]interface{}{
			"type":        "ReCaptchaV2TaskProxyless",
			"websiteURL":  pageURL,
			"websiteKey":  siteKey,
			"isInvisible": false,
		},
	}

	resp, err := sendRequest(EZCaptchaCreateEndpoint, payload)
	if err != nil {
		return "", err
	}

	var result struct {
		ErrorId          int    `json:"errorId"`
		ErrorCode        string `json:"errorCode"`
		ErrorDescription string `json:"errorDescription"`
		TaskId           string `json:"taskId"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.ErrorId != 0 {
		return "", fmt.Errorf("API error: %s - %s", result.ErrorCode, result.ErrorDescription)
	}

	return result.TaskId, nil
}

func (s *TwoCaptchaSolver) createTask(siteKey, pageURL string) (string, error) {
	payload := map[string]interface{}{
		"clientKey": s.APIKey,
		"softId":    s.SoftID,
		"task": map[string]interface{}{
			"type":       "ReCaptchaV2TaskProxyless",
			"websiteURL": pageURL,
			"websiteKey": siteKey,
		},
	}

	resp, err := sendRequest(TwoCaptchaCreateEndpoint, payload)
	if err != nil {
		return "", err
	}

	var result struct {
		ErrorId int `json:"errorId"`
		TaskId  int `json:"taskId"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.ErrorId != 0 {
		return "", fmt.Errorf("API error creating task")
	}

	return fmt.Sprintf("%d", result.TaskId), nil
}

func (s *EZCaptchaSolver) getTaskResult(taskID string) (string, error) {
	for i := 0; i < MaxRetries; i++ {
		payload := map[string]interface{}{
			"clientKey": s.APIKey,
			"taskId":    taskID,
		}

		resp, err := sendRequest(EZCaptchaResultEndpoint, payload)
		if err != nil {
			return "", err
		}

		var result struct {
			ErrorId          int    `json:"errorId"`
			ErrorCode        string `json:"errorCode"`
			ErrorDescription string `json:"errorDescription"`
			Status           string `json:"status"`
			Solution         struct {
				GRecaptchaResponse string `json:"gRecaptchaResponse"`
			} `json:"solution"`
		}

		if err := json.Unmarshal(resp, &result); err != nil {
			return "", fmt.Errorf("failed to parse response: %w", err)
		}

		if result.ErrorId != 0 {
			return "", fmt.Errorf("API error: %s - %s", result.ErrorCode, result.ErrorDescription)
		}

		if result.Status == "ready" {
			return result.Solution.GRecaptchaResponse, nil
		}

		time.Sleep(RetryInterval)
	}

	return "", errors.New("max retries reached waiting for result")
}

func (s *TwoCaptchaSolver) getTaskResult(taskID string) (string, error) {
	for i := 0; i < MaxRetries; i++ {
		payload := map[string]interface{}{
			"clientKey": s.APIKey,
			"taskId":    taskID,
		}

		resp, err := sendRequest(TwoCaptchaResultEndpoint, payload)
		if err != nil {
			return "", err
		}

		var result struct {
			ErrorId  int    `json:"errorId"`
			Status   string `json:"status"`
			Solution struct {
				GRecaptchaResponse string `json:"gRecaptchaResponse"`
			} `json:"solution"`
		}

		if err := json.Unmarshal(resp, &result); err != nil {
			return "", fmt.Errorf("failed to parse response: %w", err)
		}

		if result.ErrorId != 0 {
			return "", fmt.Errorf("API error getting result")
		}

		if result.Status == "ready" {
			return result.Solution.GRecaptchaResponse, nil
		}

		time.Sleep(RetryInterval)
	}

	return "", errors.New("max retries reached waiting for result")
}

func sendRequest(url string, payload interface{}) ([]byte, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}

func ValidateCaptchaKey(apiKey, provider string) (bool, float64, error) {
	var url string
	switch provider {
	case "ezcaptcha":
		url = "https://api.ez-captcha.com/getBalance"
	case "2captcha":
		url = "https://2captcha.com/getBalance"
	default:
		return false, 0, errors.New("unsupported captcha provider")
	}

	payload := map[string]string{
		"clientKey": apiKey,
		"action":    "getBalance",
	}

	resp, err := sendRequest(url, payload)
	if err != nil {
		return false, 0, err
	}

	var result struct {
		ErrorId int     `json:"errorId"`
		Balance float64 `json:"balance"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return false, 0, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.ErrorId != 0 {
		return false, 0, nil
	}

	return true, result.Balance, nil
}
