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

	"CODStatusBot/logger"
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

type createTaskRequest struct {
	ClientKey string `json:"clientKey"`
	softId    string `json:"softId"`
	EzAppID   string `json:"appId"`
	Task      task   `json:"task"`
}

type task struct {
	Type        string `json:"type"`
	WebsiteURL  string `json:"websiteURL"`
	WebsiteKey  string `json:"websiteKey"`
	IsInvisible bool   `json:"isInvisible"`
	Action      string `json:"action,omitempty"`
}

type createTaskResponse struct {
	ErrorID          int    `json:"errorId"`
	ErrorCode        string `json:"errorCode"`
	ErrorDescription string `json:"errorDescription"`
	TaskID           string `json:"taskId"`
}

type getTaskResultRequest struct {
	ClientKey string `json:"clientKey"`
	TaskID    string `json:"taskId"`
}

type getTaskResultResponse struct {
	ErrorID          int    `json:"errorId"`
	ErrorCode        string `json:"errorCode"`
	ErrorDescription string `json:"errorDescription"`
	Status           string `json:"status"`
	Solution         struct {
		GRecaptchaResponse string `json:"gRecaptchaResponse"`
	} `json:"solution"`
}

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
		"EzAppID":   s.EzappID,
		"task": map[string]string{
			"type":       "ReCaptchaV2TaskProxyless",
			"websiteURL": pageURL,
			"websiteKey": siteKey,
		},
	}

	return sendRequest("https://api.ez-captcha.com/createTask", payload)
}

func (s *TwoCaptchaSolver) createTask(siteKey, pageURL string) (string, error) {
	payload := map[string]interface{}{
		"clientKey": s.APIKey,
		"softId":    s.SoftID,
		"task": map[string]string{
			"type":       "ReCaptchaV2TaskProxyless",
			"websiteURL": pageURL,
			"websiteKey": siteKey,
		},
	}

	return sendRequest("https://2captcha.com/createTask", payload)
}

func (s *EZCaptchaSolver) getTaskResult(taskID string) (string, error) {
	return pollForResult("https://api.ez-captcha.com/getTaskResult", s.APIKey, taskID)
}

func (s *TwoCaptchaSolver) getTaskResult(taskID string) (string, error) {
	return pollForResult("https://2captcha.com/getTaskResult", s.APIKey, taskID)
}

func sendRequest(url string, payload interface{}) (string, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON payload: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var result struct {
		TaskID string `json:"taskId"`
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return result.TaskID, nil
}

func pollForResult(url, apiKey, taskID string) (string, error) {
	for i := 0; i < MaxRetries; i++ {
		payload := map[string]string{
			"clientKey": apiKey,
			"taskId":    taskID,
		}

		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON payload: %w", err)
		}

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
		if err != nil {
			return "", fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read response body: %w", err)
		}

		var result struct {
			Status   string `json:"status"`
			Solution struct {
				GRecaptchaResponse string `json:"gRecaptchaResponse"`
			} `json:"solution"`
		}
		err = json.Unmarshal(body, &result)
		if err != nil {
			return "", fmt.Errorf("failed to parse JSON response: %w", err)
		}

		if result.Status == "ready" {
			return result.Solution.GRecaptchaResponse, nil
		}

		logger.Log.Infof("Waiting for captcha solution. Attempt %d/%d", i+1, MaxRetries)
		time.Sleep(RetryInterval)
	}

	return "", errors.New("max retries reached, captcha solving timed out")
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

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return false, 0, fmt.Errorf("failed to marshal JSON payload: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return false, 0, fmt.Errorf("failed to send getBalance request: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Log.Errorf("Failed to close response body: %v", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, 0, fmt.Errorf("failed to read response body: %v", err)
	}

	var result struct {
		ErrorId int     `json:"errorId"`
		Balance float64 `json:"balance"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return false, 0, fmt.Errorf("failed to parse JSON response: %v", err)
	}

	if result.ErrorId != 0 {
		return false, 0, nil
	}

	return true, result.Balance, nil
}
