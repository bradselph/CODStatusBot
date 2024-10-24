package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
)

var (
	checkURL         = os.Getenv("CHECK_ENDPOINT")       // account status check endpoint.
	profileURL       = os.Getenv("PROFILE_ENDPOINT")     // Endpoint for retrieving profile information
	checkVIP         = os.Getenv("CHECK_VIP_ENDPOINT")   // Endpoint for checking VIP status
	redeemCodeURL    = os.Getenv("REDEEM_CODE_ENDPOINT") // Endpoint for redeeming codes
	recaptchaSiteKey = os.Getenv("RECAPTCHA_SITE_KEY")   // Site key for reCAPTCHA
	recaptchaURL     = os.Getenv("RECAPTCHA_URL")        // URL for reCAPTCHA
)

func VerifySSOCookie(ssoCookie string) bool {
	logger.Log.Infof("Verifying SSO cookie: %s", ssoCookie)
	req, err := http.NewRequest("GET", profileURL, nil)
	if err != nil {
		logger.Log.WithError(err).Error("Error creating verification request")
		return false
	}
	headers := GenerateHeaders(ssoCookie)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.WithError(err).Error("Error sending verification request")
		return false
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Log.WithError(err).Error("Error closing response body")
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		logger.Log.Errorf("Invalid SSOCookie, status code: %d", resp.StatusCode)
		return false
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.WithError(err).Error("Error reading verification response body")
		return false
	}
	if len(body) == 0 { // Check if the response body is empty
		logger.Log.Error("Invalid SSOCookie, response body is empty")
		return false
	}
	logger.Log.Info("SSO cookie verified successfully")
	return true
}

func CheckAccount(ssoCookie string, userID string, captchaAPIKey string) (models.Status, error) {
	logger.Log.Info("Starting CheckAccount function")

	_, err := GetUserSettings(userID)
	if err != nil {
		return models.StatusUnknown, fmt.Errorf("failed to get user settings: %w", err)
	}

	solver, err := GetCaptchaSolver(userID)
	if err != nil {
		return models.StatusUnknown, fmt.Errorf("failed to create captcha solver: %w", err)
	}

	gRecaptchaResponse, err := solver.SolveReCaptchaV2(recaptchaSiteKey, recaptchaURL)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient balance") {
			if err := DisableUserCaptcha(nil, userID, "Insufficient balance"); err != nil {
				logger.Log.WithError(err).Error("Failed to disable user captcha service")
			}
			return models.StatusUnknown, fmt.Errorf("insufficient captcha balance")
		}
		return models.StatusUnknown, fmt.Errorf("failed to solve reCAPTCHA: %w", err)
	}

	logger.Log.Info("Successfully received reCAPTCHA response")

	checkRequest := fmt.Sprintf("%s?locale=en&g-cc=%s", checkURL, gRecaptchaResponse)
	logger.Log.WithField("url", checkRequest).Info("Constructed account check request")

	req, err := http.NewRequest("GET", checkRequest, nil)
	if err != nil {
		return models.StatusUnknown, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	headers := GenerateHeaders(ssoCookie)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	logger.Log.WithField("headers", headers).Info("Set request headers")

	client := &http.Client{
		Timeout: 120 * time.Second,
	}

	var resp *http.Response
	var body []byte
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		logger.Log.Infof("Sending HTTP request to check account (attempt %d/%d)", i+1, maxRetries)
		resp, err = client.Do(req)
		if err != nil {
			if i == maxRetries-1 {
				return models.StatusUnknown, fmt.Errorf("failed to send HTTP request after %d attempts: %w", maxRetries, err)
			}
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}
		defer resp.Body.Close()

		logger.Log.WithField("status", resp.Status).Info("Received response")

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			if i == maxRetries-1 {
				return models.StatusUnknown, fmt.Errorf("failed to read response body after %d attempts: %w", maxRetries, err)
			}
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}
		break
	}

	logger.Log.WithField("body", string(body)).Info("Read response body")

	var errorResponse struct {
		Timestamp string `json:"timestamp"`
		Path      string `json:"path"`
		Status    int    `json:"status"`
		Error     string `json:"error"`
		RequestId string `json:"requestId"`
		Exception string `json:"exception"`
	}

	if err := json.Unmarshal(body, &errorResponse); err == nil {
		logger.Log.WithField("errorResponse", errorResponse).Info("Parsed error response")
		if errorResponse.Status == 400 && errorResponse.Path == "/api/bans/v2/appeal" {
			return models.StatusUnknown, fmt.Errorf("invalid request to endpoint: %s", errorResponse.Error)
		}
	}

	var data struct {
		Error     string `json:"error"`
		Success   string `json:"success"`
		CanAppeal bool   `json:"canAppeal"`
		Bans      []struct {
			Enforcement string `json:"enforcement"`
			Title       string `json:"title"`
			CanAppeal   bool   `json:"canAppeal"`
		} `json:"bans"`
	}

	if string(body) == "" {
		return models.StatusInvalidCookie, nil
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return models.StatusUnknown, fmt.Errorf("failed to decode JSON response: %w", err)
	}
	logger.Log.WithField("data", data).Info("Parsed ban data")

	if err := UpdateCaptchaUsage(userID); err != nil {
		logger.Log.WithError(err).Error("Failed to update captcha usage")
	}

	if len(data.Bans) == 0 {
		logger.Log.Info("No bans found, account status is good")
		return models.StatusGood, nil
	}

	for _, ban := range data.Bans {
		logger.Log.WithField("ban", ban).Info("Processing ban")
		switch ban.Enforcement {
		case "PERMANENT":
			logger.Log.Info("Permanent ban detected")
			return models.StatusPermaban, nil
		case "UNDER_REVIEW":
			logger.Log.Info("Shadowban detected")
			return models.StatusShadowban, nil
		case "TEMPORARY":
			logger.Log.Info("Temporary ban detected")
			return models.StatusTempban, nil
		}
	}

	logger.Log.Info("Unknown account status")
	return models.StatusUnknown, nil
}

func UpdateCaptchaUsage(userID string) error {
	settings, err := GetUserSettings(userID)
	if err != nil {
		return err
	}

	if settings.EZCaptchaAPIKey == "" && settings.TwoCaptchaAPIKey == "" {
		return nil
	}

	apiKey := settings.EZCaptchaAPIKey
	provider := "ezcaptcha"
	if settings.PreferredCaptchaProvider == "2captcha" {
		apiKey = settings.TwoCaptchaAPIKey
		provider = "2captcha"
	}

	isValid, balance, err := ValidateCaptchaKey(apiKey, provider)
	if err != nil {
		return err
	}

	if !isValid {
		return errors.New("invalid captcha key")
	}

	settings.CaptchaBalance = balance
	settings.LastBalanceCheck = time.Now()

	return database.DB.Save(&settings).Error
}

func CheckAccountAge(ssoCookie string) (int, int, int, int64, error) {
	logger.Log.Info("Starting CheckAccountAge function")
	req, err := http.NewRequest("GET", profileURL, nil)
	if err != nil {
		return 0, 0, 0, 0, errors.New("failed to create HTTP request to check account age")
	}
	headers := GenerateHeaders(ssoCookie)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, 0, 0, errors.New("failed to send HTTP request to check account age")
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	var data struct {
		Created string `json:"created"`
	}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return 0, 0, 0, 0, errors.New("failed to decode JSON response from check account age request")
	}

	logger.Log.Infof("Account created date: %s", data.Created)

	created, err := time.Parse(time.RFC3339, data.Created)
	if err != nil {
		return 0, 0, 0, 0, errors.New("failed to parse created date in check account age request")
	}

	// Convert to UTC and get epoch timestamp
	createdUTC := created.UTC()
	createdEpoch := createdUTC.Unix()

	// Calculate age
	now := time.Now().UTC()
	age := now.Sub(createdUTC)
	years := int(age.Hours() / 24 / 365.25)
	months := int(age.Hours()/24/30.44) % 12
	days := int(age.Hours()/24) % 30

	logger.Log.Infof("Account age calculated: %d years, %d months, %d days", years, months, days)
	return years, months, days, createdEpoch, nil
}

func CheckVIPStatus(ssoCookie string) (bool, error) {
	logger.Log.Info("Checking VIP status")
	req, err := http.NewRequest("GET", checkVIP+ssoCookie, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create HTTP request to check VIP status: %w", err)
	}
	headers := GenerateHeaders(ssoCookie)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send HTTP request to check VIP status: %w", err)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Log.Errorf("Failed to close response body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("invalid response status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %w", err)
	}

	var data struct {
		VIP bool `json:"vip"`
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return false, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	logger.Log.Infof("VIP status check complete. Result: %v", data.VIP)
	return data.VIP, nil
}

func RedeemCode(ssoCookie, code string) (string, error) {
	logger.Log.Infof("Attempting to redeem code: %s", code)

	req, err := http.NewRequest("POST", redeemCodeURL, strings.NewReader(fmt.Sprintf("code=%s", code)))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request to redeem code: %w", err)
	}

	headers := GeneratePostHeaders(ssoCookie)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send HTTP request to redeem code: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Log.Errorf("Failed to close response body: %v", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var jsonResponse struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &jsonResponse); err == nil && jsonResponse.Status == "success" {
		logger.Log.Infof("Code redemption successful: %s", jsonResponse.Message)
		return jsonResponse.Message, nil
	}

	bodyStr := string(body)
	if strings.Contains(bodyStr, "redemption-success") {
		start := strings.Index(bodyStr, "Just Unlocked:<br><br><div class=\"accent-highlight mw2\">")
		end := strings.Index(bodyStr, "</div></h4>")
		if start != -1 && end != -1 {
			unlockedItem := strings.TrimSpace(bodyStr[start+len("Just Unlocked:<br><br><div class=\"accent-highlight mw2\">") : end])
			logger.Log.Infof("Successfully claimed reward: %s", unlockedItem)
			return fmt.Sprintf("Successfully claimed reward: %s", unlockedItem), nil
		}
		logger.Log.Info("Successfully claimed reward, but couldn't extract details")
		return "Successfully claimed reward, but couldn't extract details", nil
	}

	logger.Log.Warnf("Unexpected response body: %s", bodyStr)
	return "", fmt.Errorf("failed to redeem code: unexpected response")
}
