package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

type AccountValidationResult struct {
	IsValid     bool
	Created     int64
	IsVIP       bool
	ExpiresAt   int64
	ProfileData map[string]interface{}
}

type BanResponse struct {
	Error     string `json:"error"`
	Success   string `json:"success"`
	CanAppeal bool   `json:"canAppeal"`
	Bans      []struct {
		Enforcement string `json:"enforcement"`
		Title       string `json:"title"`
		CanAppeal   bool   `json:"canAppeal"`
	} `json:"bans"`
}

func init() {
	cfg := configuration.Get()
	InitHTTPClients()
	logger.Log.Infof("Initialized endpoints: Profile URL: %s", cfg.API.ProfileEndpoint)
}

func VerifySSOCookie(ssoCookie string) bool {
	cfg := configuration.Get()
	logger.Log.Infof("Starting SSO cookie verification for cookie: %s", ssoCookie)

	profileURL := cfg.API.ProfileEndpoint
	if profileURL == "" {
		logger.Log.Error("PROFILE_ENDPOINT not configured")
		return false
	}

	maxRetries := 3
	var lastError error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("GET", profileURL, nil)
		if err != nil {
			lastError = fmt.Errorf("error creating verification request (attempt %d/%d): %w", attempt, maxRetries, err)
			logger.Log.WithError(err).Error("Error creating verification request")
			continue
		}

		headers := GenerateHeaders(ssoCookie)
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		logger.Log.Infof("Sending verification request to: %s (attempt %d/%d)", profileURL, attempt, maxRetries)
		resp, err := DoRequest(req)
		if err != nil {
			lastError = fmt.Errorf("error sending verification request (attempt %d/%d): %w", attempt, maxRetries, err)
			logger.Log.WithError(err).Error("Error sending verification request")
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		if resp != nil && resp.Body != nil {
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					logger.Log.WithError(err).Error("Failed to close response body")
				}
			}(resp.Body)
		}

		if resp.StatusCode != http.StatusOK {
			lastError = fmt.Errorf("invalid status code (attempt %d/%d): %d", attempt, maxRetries, resp.StatusCode)
			logger.Log.WithFields(logrus.Fields{
				"statusCode": resp.StatusCode,
				"attempt":    attempt,
			}).Error("Invalid status code in SSO verification")
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastError = fmt.Errorf("error reading response body (attempt %d/%d): %w", attempt, maxRetries, err)
			logger.Log.WithError(err).Error("Error reading response body")
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		if len(body) == 0 {
			lastError = fmt.Errorf("empty response body (attempt %d/%d)", attempt, maxRetries)
			logger.Log.Error("Empty response body in SSO verification")
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		logger.Log.Info("SSO cookie verified successfully")
		return true
	}

	logger.Log.WithError(lastError).Error("SSO cookie verification failed after all retries")
	return false
}

func CheckAccount(ssoCookie string, userID string, captchaAPIKey string) (models.Status, error) {
	startTime := time.Now()
	cfg := configuration.Get()
	logger.Log.Info("Starting CheckAccount function")

	var accountID uint = 0
	var captchaProvider string = ""
	var captchaCost float64 = 0.0
	var usedFallback bool = false

	var account models.Account
	if result := database.DB.Where("sso_cookie = ?", ssoCookie).First(&account); result.Error == nil {
		accountID = account.ID
	}

	userSettings, err := GetUserSettings(userID)
	if err != nil {
		return models.StatusUnknown, fmt.Errorf("failed to get user settings: %w", err)
	}

	if err := handleRateLimitCheck(userSettings, "check_endpoint"); err != nil {
		return models.StatusUnknown, err
	}

	if !VerifySSOCookie(ssoCookie) {
		return models.StatusInvalidCookie, nil
	}

	if !IsServiceEnabled("ezcaptcha") &&
		!IsServiceEnabled("capsolver") &&
		!IsServiceEnabled("2captcha") {
		return models.StatusUnknown, fmt.Errorf("no captcha services are currently enabled")
	}

	if !IsServiceEnabled(userSettings.PreferredCaptchaProvider) {
		if IsServiceEnabled("capsolver") {
			userSettings.PreferredCaptchaProvider = "capsolver"
			database.DB.Save(&userSettings)
		} else if IsServiceEnabled("ezcaptcha") {
			userSettings.PreferredCaptchaProvider = "ezcaptcha"
			database.DB.Save(&userSettings)
		} else if IsServiceEnabled("2captcha") {
			userSettings.PreferredCaptchaProvider = "2captcha"
			database.DB.Save(&userSettings)
		} else {
			return models.StatusUnknown, fmt.Errorf("no captcha services are currently enabled")
		}
	}

	isUsingDefaultKey := userSettings.CapSolverAPIKey == "" &&
		userSettings.EZCaptchaAPIKey == "" &&
		userSettings.TwoCaptchaAPIKey == ""

	if isUsingDefaultKey {
		if !validateRateLimit(userID, "check_account", cfg.RateLimits.CheckNow) {
			return models.StatusUnknown, fmt.Errorf("rate limit exceeded for default key users")
		}
	}

	gRecaptchaResponse, usedProvider, err := SolveCaptchaWithFallback(userID, cfg.CaptchaService.RecaptchaSiteKey, cfg.CaptchaService.RecaptchaURL)
	/*
		solver, err := GetCaptchaSolver(userID)
		if err != nil {
			if strings.Contains(err.Error(), "insufficient balance") {
				if err := DisableUserCaptcha(nil, userID, "Insufficient balance"); err != nil {
					logger.Log.WithError(err).Error("Failed to disable user captcha service")
				}
				return models.StatusUnknown, fmt.Errorf("critical error: %w", err)
			}
			return models.StatusUnknown, fmt.Errorf("failed to create captcha solver: %w", err)
		}

		gRecaptchaResponse, err := solver.SolveReCaptchaV2(cfg.CaptchaService.RecaptchaSiteKey, cfg.CaptchaService.RecaptchaURL)
	*/
	if err != nil {
		if strings.Contains(err.Error(), "insufficient balance") {
			if err := DisableUserCaptcha(nil, userID, "Insufficient balance"); err != nil {
				logger.Log.WithError(err).Error("Failed to disable user captcha service")
			}
			return models.StatusUnknown, fmt.Errorf("insufficient captcha balance")
		}
		return models.StatusUnknown, fmt.Errorf("failed to solve reCAPTCHA: %w", err)
	}

	captchaProvider = usedProvider
	usedFallback = (usedProvider != userSettings.PreferredCaptchaProvider)
	logger.Log.Infof("Captcha solved using provider: %s (fallback: %v)", usedProvider, usedFallback)

	if strings.Contains(gRecaptchaResponse, "Invalid") || len(gRecaptchaResponse) < 50 {
		return models.StatusUnknown, fmt.Errorf("invalid captcha response received")
	}

	logger.Log.Info("Successfully received reCAPTCHA response")

	//checkRequest := fmt.Sprintf("%s?locale=en&g-cc=%s", cfg.API.CheckEndpoint, gRecaptchaResponse)
	checkRequest := fmt.Sprintf("%s?locale=en_US&g-cc=%s", cfg.API.CheckEndpoint, gRecaptchaResponse)
	logger.Log.WithField("url", checkRequest).Info("Constructed account check request")

	req, err := http.NewRequest("GET", checkRequest, nil)
	if err != nil {
		return models.StatusUnknown, fmt.Errorf("failed to create request: %w", err)
	}

	headers := GenerateHeaders(ssoCookie)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	logger.Log.WithFields(logrus.Fields{
		"url":          checkRequest,
		"headers":      headers,
		"cookieLength": len(ssoCookie),
	}).Debug("Set request headers")
	var resp *http.Response
	var body []byte
	maxRetries := 3

	backoffDuration := time.Second
	for i := 0; i < maxRetries; i++ {
		logger.Log.Infof("Sending request to check account (attempt %d/%d)", i+1, maxRetries)
		resp, err = DoRequest(req)
		if err != nil {
			if i == maxRetries-1 {
				return models.StatusUnknown, fmt.Errorf("failed to send request after %d attempts: %w", maxRetries, err)
			}
			backoffDuration *= 2
			time.Sleep(backoffDuration)
			continue
		}

		if resp.StatusCode == 429 {
			updateRateLimitBackoff(userSettings, "check_endpoint")
			return models.StatusUnknown, fmt.Errorf("rate limited by Activision API")
		}

		body, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			if i == maxRetries-1 {
				return models.StatusUnknown, fmt.Errorf("failed to read response body after %d attempts: %w", maxRetries, err)
			}
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}

		logger.Log.WithFields(logrus.Fields{
			"statusCode":  resp.StatusCode,
			"bodyLength":  len(body),
			"bodyContent": string(body),
		}).Info("Received API response")

		break
	}

	logger.Log.WithField("body", string(body)).Info("Read response body")

	if resp.ContentLength == 0 {
		logger.Log.Warn("Received empty response (Content-Length: 0)")
	}

	if len(body) == 0 {
		logger.Log.Warn("Empty response body after reading")
	}

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

	var data BanResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return models.StatusUnknown, fmt.Errorf("failed to parse response: %w", err)
	}
	logger.Log.WithField("data", data).Info("Parsed ban data")

	if strings.Contains(string(body), "InvalidCaptchaException") || resp.StatusCode == 400 {
		ReportCapsolverTaskResult(gRecaptchaResponse, false, "Invalid captcha token rejected by Activision API")
		return models.StatusUnknown, fmt.Errorf("invalid captcha response")
	} else if resp.StatusCode == 200 {
		ReportCapsolverTaskResult(gRecaptchaResponse, true, "")
		resetRateLimitBackoff(userSettings, "check_endpoint")
	}

	gameSpecificBans := make(map[string]string)
	overallStatus := models.StatusGood
	isRankLocked := false

	if data.Success == "true" && len(data.Bans) == 0 {
		logger.Log.Info("No bans found, account status is good")
		account.GameSpecificBans = gameSpecificBans
		account.IsRankLocked = false
		database.DB.Save(&account)
		return models.StatusGood, nil
	}

	if err := UpdateCaptchaUsage(userID); err != nil {
		logger.Log.WithError(err).Error("Failed to update captcha usage")
	}

	hasCampaignBan := false
	hasMultiplayerBan := false
	onlyCampaignBanUnderReview := true

	for _, ban := range data.Bans {
		logger.Log.WithField("ban", ban).Info("Processing ban")

		gameSpecificBans[ban.Title] = ban.Enforcement

		if strings.Contains(ban.Title, "SP") || strings.Contains(ban.Title, "CAMPAIGN") {
			hasCampaignBan = true
			if ban.Enforcement == "UNDER_REVIEW" {
				isRankLocked = true
			}
		} else {
			if ban.Enforcement == "UNDER_REVIEW" {
				onlyCampaignBanUnderReview = false
			}
		}

		if strings.Contains(ban.Title, "BO6") && !strings.Contains(ban.Title, "SP") {
			hasMultiplayerBan = true
		}

		switch ban.Enforcement {
		case "PERMANENT":
			overallStatus = models.StatusPermaban
		case "UNDER_REVIEW":
			if overallStatus != models.StatusPermaban {
				overallStatus = models.StatusShadowban
			}
		case "TEMPORARY":
			if overallStatus != models.StatusPermaban && overallStatus != models.StatusShadowban {
				overallStatus = models.StatusTempban
			}
		}
	}

	if hasCampaignBan && overallStatus == models.StatusShadowban {
		if onlyCampaignBanUnderReview {
			logger.Log.Info("Detected BO6 campaign shadowban case - this is a limited matchmaking mode ban")
			overallStatus = models.StatusRankLocked
			isRankLocked = true
			account.IsCampaignOnlyShadowban = true
		} else if hasMultiplayerBan {
			isRankLocked = true
			if len(gameSpecificBans) == 2 {
				for title, enforcement := range gameSpecificBans {
					if enforcement == "UNDER_REVIEW" && strings.Contains(title, "SP") {
						overallStatus = models.StatusRankLocked
						break
					}
				}
			}
		}
	}

	account.GameSpecificBans = gameSpecificBans
	account.IsRankLocked = isRankLocked

	if err := database.DB.Save(&account).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to update account with game-specific bans")
	}

	LogAccountCheck(accountID, userID, "", err == nil,
		captchaProvider, captchaCost, time.Since(startTime).Milliseconds())

	if accountID > 0 {
		LogAccountStatusCheck(accountID, userID, overallStatus, "manual_check", "")
	}

	if usedFallback {
		go notifyUserAboutFallbackUsage(userID, userSettings.PreferredCaptchaProvider, usedProvider)
	}

	logger.Log.Infof("Account status determined: %s, Rank Locked: %v", overallStatus, isRankLocked)
	return overallStatus, nil
}

func handleRateLimitCheck(userSettings models.UserSettings, endpoint string) error {
	userSettings.EnsureMapsInitialized()

	if lastHit, exists := userSettings.LastRateLimitHit[endpoint]; exists {
		backoffMultiplier := userSettings.RateLimitBackoff[endpoint]
		if backoffMultiplier == 0 {
			backoffMultiplier = 1
		}

		backoffDuration := time.Duration(backoffMultiplier) * time.Minute

		if time.Since(lastHit) < backoffDuration {
			remainingTime := backoffDuration - time.Since(lastHit)
			return fmt.Errorf("rate limited - please wait %v before trying again", remainingTime.Round(time.Second))
		}
	}

	return nil
}

func updateRateLimitBackoff(userSettings models.UserSettings, endpoint string) {
	userSettings.EnsureMapsInitialized()

	userSettings.LastRateLimitHit[endpoint] = time.Now()

	currentBackoff := userSettings.RateLimitBackoff[endpoint]
	if currentBackoff == 0 {
		currentBackoff = 1
	} else if currentBackoff < 32 {
		currentBackoff *= 2
	}

	userSettings.RateLimitBackoff[endpoint] = currentBackoff

	if err := database.DB.Save(userSettings).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to update rate limit backoff")
	}
}

func resetRateLimitBackoff(userSettings models.UserSettings, endpoint string) {
	userSettings.EnsureMapsInitialized()

	delete(userSettings.RateLimitBackoff, endpoint)
	delete(userSettings.LastRateLimitHit, endpoint)

	if err := database.DB.Save(userSettings).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to reset rate limit backoff")
	}
}

func UpdateCaptchaUsage(userID string) error {
	settings, err := GetUserSettings(userID)
	if err != nil {
		return err
	}

	if settings.EZCaptchaAPIKey == "" &&
		settings.TwoCaptchaAPIKey == "" &&
		settings.CapSolverAPIKey == "" {
		return nil
	}

	var apiKey string
	var provider string

	switch settings.PreferredCaptchaProvider {
	case "ezcaptcha":
		apiKey = settings.EZCaptchaAPIKey
		provider = "ezcaptcha"
	case "2captcha":
		apiKey = settings.TwoCaptchaAPIKey
		provider = "2captcha"
	default:
		apiKey = settings.CapSolverAPIKey
		provider = "capsolver"
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
	cfg := configuration.Get()

	req, err := http.NewRequest("GET", cfg.API.ProfileEndpoint, nil)
	if err != nil {
		return 0, 0, 0, 0, errors.New("failed to create HTTP request to check account age")
	}
	headers := GenerateHeaders(ssoCookie)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := DoRequest(req)
	if err != nil {
		return 0, 0, 0, 0, errors.New("failed to send HTTP request to check account age")
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Log.WithError(err).Error("Failed to close response body")
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

	createdUTC := created.UTC()
	createdEpoch := createdUTC.Unix()

	now := time.Now().UTC()
	age := now.Sub(createdUTC)
	years := int(age.Hours() / 24 / 365.25)
	months := int(age.Hours()/24/30.44) % 12
	days := int(age.Hours()/24) % 30

	logger.Log.Infof("Account age calculated: %d years, %d months, %d days", years, months, days)
	return years, months, days, createdEpoch, nil
}

func CheckVIPStatus(ssoCookie string) (bool, error) {
	cfg := configuration.Get()
	logger.Log.Info("Checking VIP status")

	req, err := http.NewRequest("GET", cfg.API.CheckVIPEndpoint+ssoCookie, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create HTTP request to check VIP status: %w", err)
	}
	headers := GenerateHeaders(ssoCookie)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := DoRequest(req)
	if err != nil {
		return false, fmt.Errorf("failed to send HTTP request to check VIP status: %w", err)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Log.WithError(err).Error("Failed to close response body")
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

func ValidateAndGetAccountInfo(ssoCookie string) (*AccountValidationResult, error) {
	cfg := configuration.Get()

	req, err := http.NewRequest("GET", cfg.API.ProfileEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile request: %w", err)
	}

	headers := GenerateHeaders(ssoCookie)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := DoRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send profile request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Log.WithError(err).Error("Failed to close response body")
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return &AccountValidationResult{IsValid: false}, nil
	}

	var profileData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&profileData); err != nil {
		return nil, fmt.Errorf("failed to decode profile response: %w", err)
	}

	createdStr, ok := profileData["created"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid creation date format")
	}

	created, err := time.Parse(time.RFC3339, createdStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse creation date: %w", err)
	}

	isVIP, err := CheckVIPStatus(ssoCookie)
	if err != nil {
		logger.Log.WithError(err).Warn("Failed to check VIP status, defaulting to false")
		isVIP = false
	}

	expirationTimestamp, err := DecodeSSOCookie(ssoCookie)
	if err != nil {
		return nil, fmt.Errorf("failed to decode SSO cookie: %w", err)
	}

	return &AccountValidationResult{
		IsValid:     true,
		Created:     created.Unix(),
		IsVIP:       isVIP,
		ExpiresAt:   expirationTimestamp,
		ProfileData: profileData,
	}, nil
}

func notifyUserAboutFallbackUsage(userID, primaryProvider, usedProvider string) {
	settings, err := GetUserSettings(userID)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to get user settings for fallback notification")
		return
	}

	hasCustomKey := settings.CapSolverAPIKey != "" || settings.EZCaptchaAPIKey != "" || settings.TwoCaptchaAPIKey != ""
	if hasCustomKey {
		return
	}

	if settings.HasSeenFallbackNotice {
		return
	}

	lastNotificationKey := fmt.Sprintf("fallback_used_%s", usedProvider)
	settings.EnsureMapsInitialized()

	if lastNotification, exists := settings.LastCommandTimes[lastNotificationKey]; exists {
		if time.Since(lastNotification) < 24*time.Hour {
			return
		}
	}

	settings.LastCommandTimes[lastNotificationKey] = time.Now()
	if err := database.DB.Save(&settings).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to save fallback notification timestamp")
	}

	var account models.Account
	if err := database.DB.Where("user_id = ?", userID).Order("updated_at DESC").First(&account).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to find account for fallback notification")
		return
	}

	providerNames := map[string]string{
		"capsolver": "Capsolver",
		"ezcaptcha": "EZCaptcha",
		"2captcha":  "2Captcha",
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Fallback Captcha Service Used",
		Description: fmt.Sprintf("Your account check was completed using **%s** as a fallback service after **%s** failed.", providerNames[usedProvider], providerNames[primaryProvider]),
		Color:       0xFFA500,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Primary Provider",
				Value:  providerNames[primaryProvider],
				Inline: true,
			},
			{
				Name:   "Fallback Used",
				Value:  providerNames[usedProvider],
				Inline: true,
			},
			{
				Name:   "Why Use Your Own Key?",
				Value:  "• Higher reliability\n• Faster check times\n• Support the bot development\n• Priority processing",
				Inline: false,
			},
			{
				Name:   "Get Your Own Key",
				Value:  fmt.Sprintf("Sign up for %s using our referral link to help keep the bot free for everyone!", providerNames[usedProvider]),
				Inline: false,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if usedProvider == "capsolver" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Sign Up Link",
			Value:  "[Get Capsolver API Key](https://dashboard.capsolver.com/passport/register?inviteCode=6YjROhACQnvP)",
			Inline: false,
		})
	}

	components := []discordgo.MessageComponent{
		discordgo.Button{
			Label:    "Don't Show This Again",
			Style:    discordgo.SecondaryButton,
			CustomID: "dismiss_fallback_notice",
		},
		discordgo.Button{
			Label:    "Set Up My Own Key",
			Style:    discordgo.PrimaryButton,
			CustomID: "set_captcha_from_notice",
		},
	}

	if err := SendNotificationWithComponentsV2(nil, account, embed, "", "fallback_usage_notice", components); err != nil {
		logger.Log.WithError(err).Error("Failed to send fallback usage notification")
	}
}
