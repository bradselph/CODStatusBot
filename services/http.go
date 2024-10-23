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

	"CODStatusBot/logger"
	"CODStatusBot/models"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.LinearBuckets(0, 1, 10),
		},
		[]string{"user_id"},
	)

	httpRequestErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_errors_total",
			Help: "Total number of HTTP request errors",
		},
		[]string{"user_id", "error_type"},
	)

	checkDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "check_duration_seconds",
			Help:    "Total duration of account checks",
			Buckets: prometheus.LinearBuckets(0, 1, 10),
		},
		[]string{"user_id"},
	)
)

var (
	checkURL      = os.Getenv("CHECK_ENDPOINT")       // account status check endpoint.
	profileURL    = os.Getenv("PROFILE_ENDPOINT")     // Endpoint for retrieving profile information
	checkVIP      = os.Getenv("CHECK_VIP_ENDPOINT")   // Endpoint for checking VIP status
	redeemCodeURL = os.Getenv("REDEEM_CODE_ENDPOINT") // Endpoint for redeeming codes
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

// TODO move timeout to when the recaptcha is solved to ensure it doesn't timeout before the request is sent
func CheckAccount(ssoCookie string, userID string, captchaAPIKey string) (models.Status, error) {
	logger.Log.Info("Starting CheckAccount function")

	checkStart := time.Now()
	defer func() {
		checkDuration.WithLabelValues(userID).Observe(time.Since(checkStart).Seconds())
	}()

	solver, err := GetCaptchaSolver(userID)
	if err != nil {
		captchaSolveErrors.WithLabelValues(userID, "solver_init").Inc()
		return models.StatusUnknown, fmt.Errorf("failed to get captcha solver: %w", err)
	}

	solveStart := time.Now()
	gRecaptchaResponse, err := solver.SolveReCaptchaV2(siteKey, pageURL)
	if err != nil {
		captchaSolveErrors.WithLabelValues(userID, "solve_failed").Inc()
		return models.StatusUnknown, fmt.Errorf("failed to solve reCAPTCHA: %w", err)
	}

	captchaSolveTotal.WithLabelValues(userID, "success").Inc()
	captchaSolveDuration.WithLabelValues(userID).Observe(time.Since(solveStart).Seconds())

	logger.Log.Info("Successfully received reCAPTCHA response")

	checkrequest := checkURL + "?locale=en" + "&g-cc=" + gRecaptchaResponse
	logger.Log.WithField("url", checkrequest).Info("Constructed account check request")

	req, err := http.NewRequest("GET", checkrequest, nil)
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

		requestStart := time.Now()
		resp, err = client.Do(req)
		requestDuration := time.Since(requestStart)

		httpRequestDuration.WithLabelValues(userID).Observe(requestDuration.Seconds())

		if err != nil {
			httpRequestErrors.WithLabelValues(userID, "request_failed").Inc()
			if i == maxRetries-1 {
				return models.StatusUnknown, fmt.Errorf("failed to send HTTP request after %d attempts: %w", maxRetries, err)
			}
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
			}
		}(resp.Body)

		logger.Log.WithField("status", resp.Status).Info("Received response")

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			httpRequestErrors.WithLabelValues(userID, "read_failed").Inc()
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
			return models.StatusUnknown, fmt.Errorf("invalid request to new endpoint: %s", errorResponse.Error)
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
