package verdansk

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bradselph/CODStatusBot/services"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

var verdanskRateLimits = struct {
	sync.RWMutex
	userLimits     map[string]userRateLimit
	globalSettings verdanskLimitSettings
}{
	userLimits: make(map[string]userRateLimit),
	globalSettings: verdanskLimitSettings{
		CommandCooldown:    time.Hour * 1,    // 1 hour between commands per user
		MaxRequestsPerDay:  3,                // Max 3 requests per day per user
		CleanupTime:        time.Minute * 30, // 30 minutes before cleanup
		MaxDownloadsPerDay: 10,               // Maximum stats downloads per day across all users
		MaxConcurrentJobs:  3,                // Max concurrent jobs for a single user
	},
}

type userRateLimit struct {
	lastRequest     time.Time
	todayRequests   int
	todayDownloads  int
	resetTime       time.Time
	downloadHistory []time.Time
}

type verdanskLimitSettings struct {
	CommandCooldown    time.Duration
	MaxRequestsPerDay  int
	CleanupTime        time.Duration
	MaxDownloadsPerDay int
	MaxConcurrentJobs  int
}

func canUserRunVerdanskCommand(userID string) (bool, string) {
	verdanskRateLimits.RLock()
	defer verdanskRateLimits.RUnlock()

	now := time.Now()
	settings := verdanskRateLimits.globalSettings
	limit, exists := verdanskRateLimits.userLimits[userID]

	if !exists {
		return true, ""
	}

	if now.Sub(limit.lastRequest) < settings.CommandCooldown {
		remaining := limit.lastRequest.Add(settings.CommandCooldown).Sub(now).Round(time.Second)
		return false, fmt.Sprintf("Command cooldown in effect. Please try again in %s", remaining)
	}

	if now.Day() == limit.lastRequest.Day() &&
		now.Month() == limit.lastRequest.Month() &&
		now.Year() == limit.lastRequest.Year() {
		if limit.todayRequests >= settings.MaxRequestsPerDay {
			nextDay := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
			return false, fmt.Sprintf("Daily limit reached (%d). Reset at midnight (%s remaining)",
				settings.MaxRequestsPerDay, time.Until(nextDay).Round(time.Minute))
		}
	}

	return true, ""
}

func updateUserRateLimit(userID string) {
	verdanskRateLimits.Lock()
	defer verdanskRateLimits.Unlock()

	now := time.Now()
	limit, exists := verdanskRateLimits.userLimits[userID]

	if !exists {
		verdanskRateLimits.userLimits[userID] = userRateLimit{
			lastRequest:   now,
			todayRequests: 1,
			resetTime:     time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location()),
		}
		return
	}

	// Reset counters if it's a new day
	if now.Day() != limit.lastRequest.Day() ||
		now.Month() != limit.lastRequest.Month() ||
		now.Year() != limit.lastRequest.Year() {
		limit.todayRequests = 0
		limit.todayDownloads = 0
		limit.resetTime = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	}

	limit.lastRequest = now
	limit.todayRequests++

	verdanskRateLimits.userLimits[userID] = limit
}

func trackVerdanskDownload(userID string) {
	verdanskRateLimits.Lock()
	defer verdanskRateLimits.Unlock()

	now := time.Now()
	limit, exists := verdanskRateLimits.userLimits[userID]

	if !exists {
		verdanskRateLimits.userLimits[userID] = userRateLimit{
			lastRequest:     now,
			todayRequests:   1,
			todayDownloads:  1,
			downloadHistory: []time.Time{now},
		}
		return
	}

	var recentDownloads []time.Time
	for _, t := range limit.downloadHistory {
		if now.Sub(t) < 24*time.Hour {
			recentDownloads = append(recentDownloads, t)
		}
	}

	limit.todayDownloads++
	limit.downloadHistory = append(recentDownloads, now)

	verdanskRateLimits.userLimits[userID] = limit
}

func getTotalDownloadsToday() int {
	verdanskRateLimits.RLock()
	defer verdanskRateLimits.RUnlock()

	total := 0
	now := time.Now()

	for _, limit := range verdanskRateLimits.userLimits {
		if now.Day() == limit.lastRequest.Day() &&
			now.Month() == limit.lastRequest.Month() &&
			now.Year() == limit.lastRequest.Year() {
			total += limit.todayDownloads
		}
	}

	return total
}

type PlayerPreferences struct {
	Visible bool `json:"visible"`
}

type StatValue struct {
	OrderValue  *int   `json:"order_value"`
	StringValue string `json:"string_value"`
}

type ImageDownload struct {
	Name string
	URL  string
	Data []byte
	Err  error
}

type VerdanskStats map[string]StatValue

var (
	tempZipFiles = struct {
		sync.RWMutex
		files map[string]time.Time
	}{
		files: make(map[string]time.Time),
	}
	X_APIKey string
)

const (
	userAgentString = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36"
	originURL       = "https://www.callofduty.com"
)

func CommandVerdansk(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cfg := configuration.Get()
	X_APIKey = cfg.Verdansk.APIKey

	log := logger.Log.WithFields(logrus.Fields{
		"command": "verdansk",
		"action":  "start",
	})

	log.Info("Verdansk command initiated")

	userID, err := services.GetUserID(i)
	if err != nil {
		log.WithError(err).Error("Failed to get user ID")
		respondToInteraction(s, i, "An error occurred while processing your request.")
		return
	}

	log = log.WithField("userID", userID)

	canRun, message := canUserRunVerdanskCommand(userID)
	if !canRun {
		log.Infof("Rate limit applied: %s", message)
		respondToInteraction(s, i, fmt.Sprintf("⏳ **Rate Limited**: %s", message))
		return
	}

	if getTotalDownloadsToday() >= verdanskRateLimits.globalSettings.MaxDownloadsPerDay {
		log.Warn("Global download limit reached")
		respondToInteraction(s, i, "🛑 **Service Unavailable**: Verdansk stats service has reached its daily limit. Please try again tomorrow.")
		return
	}

	updateUserRateLimit(userID)

	var accounts []models.Account
	result := database.DB.Where("user_id = ?", userID).Find(&accounts)
	if result.Error != nil {
		log.WithError(result.Error).Error("Error fetching user accounts")
		respondToInteraction(s, i, "Error fetching your accounts. Please try again.")
		return
	}

	log.WithField("accountCount", len(accounts)).Info("Found user accounts")

	components := []discordgo.MessageComponent{
		discordgo.Button{
			Label:    "Provide Activision ID",
			Style:    discordgo.PrimaryButton,
			CustomID: "verdansk_provide_id",
		},
	}

	if len(accounts) > 0 {
		components = append(components, discordgo.Button{
			Label:    "Select Account",
			Style:    discordgo.SuccessButton,
			CustomID: "verdansk_select_account",
		})
	}

	verdanskComponents := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: components,
		},
	}

	err = services.RespondWithPreferenceAndComponents(s, i, "How would you like to check Verdansk Replay stats?", nil, verdanskComponents, false)
	if err != nil {
		log.WithError(err).Error("Error responding with method selection")
	} else {
		log.Info("Successfully displayed method selection options")
	}

	services.LogCommandExecution("verdansk", userID, i.GuildID, err == nil, time.Now().UnixMilli(), "")
}

func HandleMethodSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	userID, _ := services.GetUserID(i)
	log := logger.Log.WithFields(logrus.Fields{
		"command":  "verdansk",
		"action":   "method_selection",
		"userID":   userID,
		"customID": customID,
	})

	log.Info("Processing method selection")

	switch customID {
	case "verdansk_provide_id":
		log.Info("User chose to provide Activision ID")
		showActivisionIDModal(s, i)
	case "verdansk_select_account":
		log.Info("User chose to select an account")
		showAccountSelection(s, i)
	default:
		log.Warn("Invalid selection")
		respondToInteraction(s, i, "Invalid selection.")
	}
}

func showActivisionIDModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID, _ := services.GetUserID(i)
	log := logger.Log.WithFields(logrus.Fields{
		"command": "verdansk",
		"action":  "show_activision_id_modal",
		"userID":  userID,
	})

	log.Info("Showing Activision ID entry modal")

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "verdansk_activision_id_modal",
			Title:    "Enter Activision ID",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "activision_id",
							Label:       "Activision ID (e.g. Username#1234)",
							Style:       discordgo.TextInputShort,
							Placeholder: "Enter Activision ID",
							Required:    true,
							MinLength:   3,
							MaxLength:   32,
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.WithError(err).Error("Error showing Activision ID modal")
	} else {
		log.Info("Successfully displayed Activision ID modal")
	}
}

func showAccountSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID, err := services.GetUserID(i)
	log := logger.Log.WithFields(logrus.Fields{
		"command": "verdansk",
		"action":  "show_account_selection",
		"userID":  userID,
	})

	log.Info("Preparing account selection")

	if err != nil {
		log.WithError(err).Error("Failed to get user ID")
		respondToInteraction(s, i, "An error occurred while processing your request.")
		return
	}

	var accounts []models.Account
	result := database.DB.Where("user_id = ?", userID).Find(&accounts)
	if result.Error != nil {
		log.WithError(result.Error).Error("Error fetching user accounts")
		respondToInteraction(s, i, "Error fetching your accounts. Please try again.")
		return
	}

	if len(accounts) == 0 {
		log.Warn("No accounts found")
		respondToInteraction(s, i, "You don't have any monitored accounts.")
		return
	}

	log.WithField("accountCount", len(accounts)).Info("Found accounts for selection")

	var (
		components []discordgo.MessageComponent
		currentRow []discordgo.MessageComponent
	)

	var ogVerdanskAccounts int
	for _, account := range accounts {
		buttonStyle := discordgo.PrimaryButton
		if account.IsOGVerdansk {
			buttonStyle = discordgo.SuccessButton
			ogVerdanskAccounts++
		}

		currentRow = append(currentRow, discordgo.Button{
			Label:    account.Title,
			Style:    buttonStyle,
			CustomID: fmt.Sprintf("verdansk_account_%d", account.ID),
		})

		if len(currentRow) == 5 {
			components = append(components, discordgo.ActionsRow{Components: currentRow})
			currentRow = []discordgo.MessageComponent{}
		}
	}

	if len(currentRow) > 0 {
		components = append(components, discordgo.ActionsRow{Components: currentRow})
	}

	log.WithField("ogVerdanskAccounts", ogVerdanskAccounts).Info("Displaying account selection buttons")

	err = services.UpdateMessageWithPreference(s, i, "Select an account to check Verdansk Replay stats:\n(Green buttons indicate accounts with confirmed Verdansk stats)", nil, components)
	if err != nil {
		log.WithError(err).Error("Error responding with account selection")
	} else {
		log.Info("Successfully displayed account selection options")
	}
}

func HandleAccountSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	userID, _ := services.GetUserID(i)

	log := logger.Log.WithFields(logrus.Fields{
		"command":  "verdansk",
		"action":   "handle_account_selection",
		"userID":   userID,
		"customID": customID,
	})

	log.Info("Processing account selection")

	if !strings.HasPrefix(customID, "verdansk_account_") {
		log.Warn("Invalid custom ID format")
		return
	}

	accountID := strings.TrimPrefix(customID, "verdansk_account_")
	log = log.WithField("accountID", accountID)

	var account models.Account
	if err := database.DB.First(&account, accountID).Error; err != nil {
		log.WithError(err).Error("Error fetching account")
		respondToInteraction(s, i, "Error: Account not found or you don't have permission to check its Verdansk stats.")
		return
	}

	log.WithField("accountTitle", account.Title).Info("Account found, deferring response")

	err := services.DeferWithPreference(s, i, false)
	if err != nil {
		log.WithError(err).Error("Error sending deferred response")
		return
	}

	log.Info("Retrieving Activision ID from account")
	activisionID, err := getActivisionIDFromAccount(account)
	if err != nil {
		log.WithError(err).Error("Error getting Activision ID from account")
		sendFollowupMessage(s, i, "Error: Could not determine Activision ID from account. Please ensure your cookie is valid.")
		return
	}

	log.WithField("activisionID", activisionID).Info("Successfully retrieved Activision ID, processing stats")
	processVerdanskStats(s, i, activisionID, &account)
}

func HandleActivisionIDModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	userID, _ := services.GetUserID(i)

	log := logger.Log.WithFields(logrus.Fields{
		"command": "verdansk",
		"action":  "handle_activision_id_modal",
		"userID":  userID,
	})

	log.Info("Processing Activision ID modal submission")

	var activisionID string
	for _, comp := range data.Components {
		if row, ok := comp.(*discordgo.ActionsRow); ok {
			for _, rowComp := range row.Components {
				if textInput, ok := rowComp.(*discordgo.TextInput); ok && textInput.CustomID == "activision_id" {
					activisionID = strings.TrimSpace(textInput.Value)
				}
			}
		}
	}

	log = log.WithField("activisionID", activisionID)

	if activisionID == "" {
		log.Warn("Empty Activision ID provided")
		respondToInteraction(s, i, "Error: Activision ID is required.")
		return
	}

	log.Info("Deferring response")
	err := services.DeferWithPreference(s, i, false)
	if err != nil {
		log.WithError(err).Error("Error sending deferred response")
		return
	}

	log.Info("Processing Verdansk stats for provided ID")
	processVerdanskStats(s, i, activisionID, nil)
}

func getActivisionIDFromAccount(account models.Account) (string, error) {
	log := logger.Log.WithFields(logrus.Fields{
		"function":     "getActivisionIDFromAccount",
		"accountID":    account.ID,
		"accountTitle": account.Title,
	})

	log.Info("Retrieving Activision ID from account")

	if account.ActivisionID != "" {
		log.WithField("activisionID", account.ActivisionID).Info("Using stored Activision ID")
		return account.ActivisionID, nil
	}

	cfg := configuration.Get()
	if !services.VerifySSOCookie(account.SSOCookie) {
		log.Error("Invalid SSO cookie")
		return "", fmt.Errorf("invalid SSO cookie")
	}

	client := services.GetDefaultHTTPClient()
	req, err := http.NewRequest("GET", cfg.API.ProfileEndpoint, nil)
	if err != nil {
		log.WithError(err).Error("Error creating request")
		return "", fmt.Errorf("error creating request: %w", err)
	}

	headers := services.GenerateHeaders(account.SSOCookie)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	log.Debug("Sending profile request")
	resp, err := client.Do(req)
	if err != nil {
		log.WithError(err).Error("Error sending request")
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.WithField("statusCode", resp.StatusCode).Error("API returned non-OK status")
		return "", fmt.Errorf("API returned status code %d", resp.StatusCode)
	}

	var profileData struct {
		Username string `json:"username"`
		Accounts []struct {
			Username string `json:"username"`
			Provider string `json:"provider"`
		} `json:"accounts"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&profileData); err != nil {
		log.WithError(err).Error("Error decoding response")
		return "", fmt.Errorf("error decoding response: %w", err)
	}

	for _, acc := range profileData.Accounts {
		if acc.Provider == "uno" {
			log.WithField("activisionID", acc.Username).Info("Found Activision ID (UNO provider)")

			account.ActivisionID = acc.Username
			if err := database.DB.Save(&account).Error; err != nil {
				log.WithError(err).Warn("Failed to save Activision ID to account")
			} else {
				log.Info("Activision ID saved to account")
			}
			return acc.Username, nil
		}
	}

	log.Error("No Activision ID found in profile")
	return "", fmt.Errorf("no Activision ID found")
}

func processVerdanskStats(s *discordgo.Session, i *discordgo.InteractionCreate, activisionID string, account *models.Account) {
	userID, _ := services.GetUserID(i)
	log := logger.Log.WithFields(logrus.Fields{
		"function":     "processVerdanskStats",
		"userID":       userID,
		"activisionID": activisionID,
	})

	log.Info("Processing Verdansk stats")
	trackVerdanskDownload(userID)

	cfg := configuration.Get()
	tempDir := cfg.Verdansk.TempDir
	if tempDir == "" {
		tempDir = "verdansk_temp"
	}

	cleanupTime := cfg.Verdansk.CleanupTime
	if cleanupTime == 0 {
		cleanupTime = verdanskRateLimits.globalSettings.CleanupTime
	}

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.WithError(err).Error("Error creating temp directory")
		sendFollowupMessage(s, i, "Error: Failed to create temporary directory for stats.")
		return
	}

	encodedID := url.QueryEscape(activisionID)
	log.WithField("encodedID", encodedID).Debug("Encoded Activision ID")

	client := services.GetDefaultHTTPClient()

	log.Info("Fetching player preferences")
	preferences, err := fetchPlayerPreferences(client, encodedID)
	if err != nil {
		log.WithError(err).Error("Error fetching player preferences")

		errorReason := "An unknown error occurred."
		remedyMessage := "Please try again later."

		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			errorReason = "Account not found in Verdansk records."
			remedyMessage = "This account either did not play during the Verdansk era or was created after Verdansk ended."
		} else if strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "forbidden") {
			errorReason = "Access to this account's Verdansk data is restricted."
			remedyMessage = "Check your privacy settings at https://profile.callofduty.com/cod/login and ensure game data is set to 'visible'."
		} else if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "too many requests") {
			errorReason = "Rate limit reached for Activision's API."
			remedyMessage = "The service is experiencing high traffic. Please try again in 15-30 minutes."
		} else if strings.Contains(err.Error(), "500") || strings.Contains(err.Error(), "503") {
			errorReason = "Activision's API is currently experiencing issues."
			remedyMessage = "The Verdansk Replay service may be temporarily unavailable. Please try again later."
		}

		errorEmbed := &discordgo.MessageEmbed{
			Title:       "Verdansk Stats Unavailable",
			Description: fmt.Sprintf("Unable to retrieve Verdansk Replay stats for **%s**", activisionID),
			Color:       0xFF0000,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Error Reason",
					Value: errorReason,
				},
				{
					Name:  "What You Can Do",
					Value: remedyMessage,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "COD Status Bot - Verdansk Replay",
			},
		}

		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{errorEmbed},
			Flags:  discordgo.MessageFlagsEphemeral,
		})

		if err != nil {
			sendFollowupMessage(s, i, fmt.Sprintf("Error: Failed to fetch preferences for %s. %v", activisionID, err))
		}
		return
	}

	if !preferences.Visible {
		log.Info("Verdansk stats not available for this account")

		unavailableEmbed := &discordgo.MessageEmbed{
			Title:       "Verdansk Stats Not Available",
			Description: fmt.Sprintf("Unfortunately, Verdansk Replay stats are not available for **%s**.", activisionID),
			Color:       0xFF9900,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name: "Possible Reasons",
					Value: "• Not enough Verdansk gameplay (at least 5 matches required)\n" +
						"• Your Game Data settings are set to private\n" +
						"• The account was created after Verdansk ended",
				},
				{
					Name: "How to Fix",
					Value: "1. Go to [callofduty.com/profile](https://profile.callofduty.com/cod/login)\n" +
						"2. Log in with your Activision account\n" +
						"3. Go to Privacy & Security settings\n" +
						"4. Make sure Game Data is set to visible\n" +
						"5. Try again in 24 hours as changes may take time to apply",
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "COD Status Bot - Verdansk Replay",
			},
		}

		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{unavailableEmbed},
			Flags:  discordgo.MessageFlagsEphemeral,
		})

		if err != nil {
			sendFollowupMessage(s, i, fmt.Sprintf("Verdansk stats for %s are not available. This could be because:\n- You haven't played enough in Verdansk (at least 5 deployments required)\n- Your Game Player Data settings need to be updated at https://profile.callofduty.com/cod/login", activisionID))
		}
		return
	}

	if account != nil && !account.IsOGVerdansk {
		log.Info("Setting OGVerdansk flag for account")
		account.IsOGVerdansk = true
		if err := database.DB.Save(account).Error; err != nil {
			log.WithError(err).Error("Failed to update OGVerdansk flag")
		}
	}

	log.Info("Fetching player stats")
	stats, err := fetchPlayerStats(client, encodedID)
	if err != nil {
		log.WithError(err).Error("Error fetching player stats")
		sendFollowupMessage(s, i, fmt.Sprintf("Error: Failed to fetch stats for %s. %v", activisionID, err))
		return
	}

	if len(stats) == 0 {
		log.Warn("No stats found for player")
		sendFollowupMessage(s, i, fmt.Sprintf("No Verdansk stats found for %s.", activisionID))
		return
	}

	log.WithField("statCount", len(stats)).Info("Successfully retrieved stats")

	uniqueID := fmt.Sprintf("%s_%d", strings.ReplaceAll(activisionID, "#", "_"), time.Now().Unix())
	outputDir := filepath.Join(tempDir, uniqueID)
	zipFilename := filepath.Join(tempDir, uniqueID+".zip")

	log.WithFields(logrus.Fields{
		"outputDir":   outputDir,
		"zipFilename": zipFilename,
	}).Info("Downloading stat images")

	images, err := downloadImages(client, stats, outputDir, 3)
	if err != nil {
		log.WithError(err).Error("Error downloading stat images")
		sendFollowupMessage(s, i, fmt.Sprintf("Error: Failed to download stat images for %s. %v", activisionID, err))
		return
	}

	log.WithField("imageCount", len(images)).Info("Downloaded images successfully")

	images = enrichVerdanskFilenames(images)

	log.Info("Creating zip file")
	if err := createZip(images, zipFilename); err != nil {
		log.WithError(err).Error("Error creating zip file")
		sendFollowupMessage(s, i, fmt.Sprintf("Error: Failed to create zip file for %s. %v", activisionID, err))
		return
	}

	tempZipFiles.Lock()
	tempZipFiles.files[zipFilename] = time.Now().Add(cleanupTime)
	tempZipFiles.Unlock()

	maxImages := 9
	log.Info("Preparing embeds and files for Discord")
	var embeds []*discordgo.MessageEmbed
	var files []*discordgo.File

	zipFile, err := os.Open(zipFilename)
	if err != nil {
		log.WithError(err).Error("Error opening zip file")
		sendFollowupMessage(s, i, fmt.Sprintf("Error: Failed to prepare zip file for %s.", activisionID))
		return
	}
	defer zipFile.Close()
	files = append(files, &discordgo.File{
		Name:   fmt.Sprintf("%s_verdansk_stats.zip", activisionID),
		Reader: zipFile,
	})

	log.WithField("imageCount", len(images)).Info("Downloaded images successfully")
	images = enrichVerdanskFilenames(images)
	summaryEmbed := generateVerdanskStatsEmbed(activisionID, stats)
	embeds = append(embeds, summaryEmbed)

	var displayedImages int
	for _, img := range images {
		if displayedImages >= maxImages {
			break
		}

		formattedName := formatStatName(img.Name)
		embed := &discordgo.MessageEmbed{
			Title:       formattedName,
			Description: fmt.Sprintf("Verdansk Replay Stat %d/%d", displayedImages+1, len(images)),
			Image: &discordgo.MessageEmbedImage{
				URL: fmt.Sprintf("attachment://%s.jpg", img.Name),
			},
			Color: 0x00BFFF,
		}
		embeds = append(embeds, embed)
		files = append(files, &discordgo.File{
			Name:   fmt.Sprintf("%s.jpg", img.Name),
			Reader: bytes.NewReader(img.Data),
		})
		displayedImages++
	}

	contentMsg := fmt.Sprintf("Verdansk Replay Stats for %s", activisionID)
	if len(images) > maxImages {
		contentMsg += fmt.Sprintf(" (Showing %d/%d images)\n\nDownload the zip file for all %d images. The data will expire after 30 minutes.",
			maxImages, len(images), len(images))
	} else {
		contentMsg += "\n\nThe images will expire after 30 minutes. Download the zip file for permanent access."
	}

	if account != nil {
		contentMsg += "\n\nThese stats have been associated with your account and marked with the Verdansk flag."
	}

	if account != nil {
		if err := storeVerdanskStatsWithAccount(account, stats); err != nil {
			log.WithError(err).Warn("Failed to store Verdansk stats with account")
		} else {
			contentMsg += "\n\nYour Verdansk stats have been saved to your account for future reference."
		}
	}

	log.Info("Sending stats to Discord")
	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: contentMsg,
		Embeds:  embeds,
		Files:   files,
		Flags:   discordgo.MessageFlagsEphemeral,
	})

	if err != nil {
		log.WithError(err).Error("Error sending stat images")
		if strings.Contains(err.Error(), "Maximum number of allowed attachments") {
			sendFollowupMessage(s, i, fmt.Sprintf("Error: Discord's attachment limit prevents showing all images. Please download the zip file for the complete set of %d images.", len(images)))
		} else {
			sendFollowupMessage(s, i, fmt.Sprintf("Error: Failed to send stat images for %s. Please try again later.", activisionID))
		}
		return
	}
	log.Info("Successfully sent Verdansk stats to user")

	go func() {
		time.Sleep(cleanupTime)
		cleanupTempFiles(uniqueID)
	}()

	services.LogCommandExecution("verdansk_stats_retrieved", userID, i.GuildID, true,
		time.Now().UnixMilli(), fmt.Sprintf("Activision ID: %s, Images: %d", activisionID, len(images)))
}

func fetchPlayerPreferences(client *http.Client, encodedGamerTag string) (*PlayerPreferences, error) {
	log := logger.Log.WithFields(logrus.Fields{
		"function":        "fetchPlayerPreferences",
		"encodedGamerTag": encodedGamerTag,
	})

	cfg := configuration.Get()
	preferencesEndpoint := cfg.Verdansk.PreferencesEndpoint
	apiKey := cfg.Verdansk.APIKey

	if preferencesEndpoint == "" {
		preferencesEndpoint = "https://pd.callofduty.com/api/x/v1/campaign/warzonewrapped/preferences/gamer/%s"
	}

	if apiKey == "" {
		apiKey = "a855a770-cf8a-4ae8-9f30-b787d676e608"
	}

	url := strings.Replace(preferencesEndpoint, "{encodedGamerTag}", encodedGamerTag, 1)
	log.WithField("url", url).Debug("Preferences URL")

	log.Info("Sending preflight request")
	if err := doPreflightRequest(client, url); err != nil {
		log.WithError(err).Error("Preflight request failed")
		return nil, fmt.Errorf("preflight request failed: %w", err)
	}

	time.Sleep(time.Duration(300+rand.Intn(500)) * time.Millisecond)

	log.Info("Creating preferences request")
	req, err := createVerdanskAPIRequest("GET", url)
	if err != nil {
		log.WithError(err).Error("Error creating request")
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	log.Debug("Sending preferences API request")
	resp, err := client.Do(req)
	if err != nil {
		log.WithError(err).Error("Error sending request")
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	log.WithField("statusCode", resp.StatusCode).Debug("Received API response")
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.WithFields(logrus.Fields{
			"statusCode": resp.StatusCode,
			"body":       string(body),
		}).Error("API returned non-OK status")
		return nil, fmt.Errorf("API returned status code %d: %s", resp.StatusCode, string(body))
	}

	body, err := readResponseBody(resp)
	if err != nil {
		log.WithError(err).Error("Error reading response body")
		return nil, err
	}

	log.WithField("body", string(body)).Debug("Received preferences response")

	var result PlayerPreferences
	if err := json.Unmarshal(body, &result); err != nil {
		log.WithError(err).Error("Error decoding response")
		return nil, fmt.Errorf("error decoding response: %w, body: %s", err, string(body))
	}

	log.WithField("visible", result.Visible).Info("Successfully fetched player preferences")
	return &result, nil
}

func fetchPlayerStats(client *http.Client, encodedGamerTag string) (map[string]StatValue, error) {
	log := logger.Log.WithFields(logrus.Fields{
		"function":        "fetchPlayerStats",
		"encodedGamerTag": encodedGamerTag,
	})

	cfg := configuration.Get()
	statsEndpoint := cfg.Verdansk.StatsEndpoint

	if statsEndpoint == "" {
		statsEndpoint = "https://pd.callofduty.com/api/x/v1/campaign/warzonewrapped/stats/gamer/%s"
	}

	url := strings.Replace(statsEndpoint, "{encodedGamerTag}", encodedGamerTag, 1)
	log.WithField("url", url).Debug("Stats URL")

	log.Info("Sending preflight request")
	if err := doPreflightRequest(client, url); err != nil {
		log.WithError(err).Error("Preflight request failed")
		return nil, fmt.Errorf("preflight request failed: %w", err)
	}

	time.Sleep(time.Duration(300+rand.Intn(500)) * time.Millisecond)

	log.Info("Creating stats request")
	req, err := createVerdanskAPIRequest("GET", url)
	if err != nil {
		log.WithError(err).Error("Error creating request")
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	log.Debug("Sending stats API request")
	resp, err := client.Do(req)
	if err != nil {
		log.WithError(err).Error("Error sending request")
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	log.WithField("statusCode", resp.StatusCode).Debug("Received API response")
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.WithFields(logrus.Fields{
			"statusCode": resp.StatusCode,
			"body":       string(body),
		}).Error("API returned non-OK status")
		return nil, fmt.Errorf("API returned status code %d: %s", resp.StatusCode, string(body))
	}

	body, err := readResponseBody(resp)
	if err != nil {
		log.WithError(err).Error("Error reading response body")
		return nil, err
	}

	log.WithField("bodyLength", len(body)).Debug("Received stats response")

	var stats map[string]StatValue
	if err := json.Unmarshal(body, &stats); err != nil {
		log.WithError(err).Error("Error decoding response")
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	stats = enrichStatData(stats)

	log.WithField("statCount", len(stats)).Info("Successfully fetched player stats")
	return stats, nil
}

func downloadImages(client *http.Client, stats map[string]StatValue, outputDir string, concurrency int) ([]ImageDownload, error) {
	log := logger.Log.WithFields(logrus.Fields{
		"function":    "downloadImages",
		"outputDir":   outputDir,
		"concurrency": concurrency,
	})

	log.Info("Starting image downloads")

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.WithError(err).Error("Error creating directory")
		return nil, fmt.Errorf("error creating directory: %w", err)
	}

	var downloads []ImageDownload
	for statName, stat := range stats {
		if stat.StringValue != "" && strings.HasPrefix(stat.StringValue, "http") {
			downloads = append(downloads, ImageDownload{
				Name: statName,
				URL:  stat.StringValue,
			})
		}
	}

	if len(downloads) == 0 {
		log.Warn("No images found to download")
		return nil, fmt.Errorf("no images found to download")
	}

	log.WithField("downloadCount", len(downloads)).Info("Preparing to download images")

	results := make(chan ImageDownload, len(downloads))
	var wg sync.WaitGroup
	limiter := make(chan struct{}, concurrency)

	for _, download := range downloads {
		wg.Add(1)
		go func(dl ImageDownload) {
			defer wg.Done()

			dlLog := log.WithFields(logrus.Fields{
				"imageName": dl.Name,
				"imageURL":  dl.URL,
			})

			dlLog.Debug("Starting image download")

			limiter <- struct{}{}
			defer func() { <-limiter }()

			time.Sleep(time.Duration(200+rand.Intn(300)) * time.Millisecond)

			req, err := http.NewRequest("GET", dl.URL, nil)
			if err != nil {
				dlLog.WithError(err).Error("Error creating request")
				dl.Err = fmt.Errorf("error creating request: %w", err)
				results <- dl
				return
			}

			req.Header.Set("User-Agent", userAgentString)
			req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
			req.Header.Set("Accept-Encoding", "gzip, deflate, br")
			req.Header.Set("Referer", originURL)
			req.Header.Set("Origin", originURL)

			dlLog.Debug("Sending image request")
			resp, err := client.Do(req)
			if err != nil {
				dlLog.WithError(err).Error("Error downloading image")
				dl.Err = fmt.Errorf("error downloading: %w", err)
				results <- dl
				return
			}
			defer resp.Body.Close()

			dlLog.WithField("statusCode", resp.StatusCode).Debug("Received image response")
			if resp.StatusCode != http.StatusOK {
				dlLog.WithField("statusCode", resp.StatusCode).Error("Non-OK status code")
				dl.Err = fmt.Errorf("status code %d", resp.StatusCode)
				results <- dl
				return
			}

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				dlLog.WithError(err).Error("Error reading image data")
				dl.Err = fmt.Errorf("error reading data: %w", err)
				results <- dl
				return
			}

			filePath := filepath.Join(outputDir, dl.Name+".jpg")
			dlLog.WithField("filePath", filePath).Debug("Saving image to file")

			if len(data) > 100000 {
				optimized, err := optimizeJPEG(data, 85)
				if err == nil && len(optimized) < len(data) {
					data = optimized
					dlLog.Info("Successfully optimized image")
				}
			}

			if err := os.WriteFile(filePath, data, 0644); err != nil {
				dlLog.WithError(err).Error("Error saving file")
				dl.Err = fmt.Errorf("error saving file: %w", err)
				results <- dl
				return
			}

			dl.Data = data
			dlLog.Info("Successfully downloaded image")
			results <- dl
		}(download)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var downloadedImages []ImageDownload
	var failedCount int

	for result := range results {
		if result.Err == nil {
			downloadedImages = append(downloadedImages, result)
		} else {
			failedCount++
			log.WithFields(logrus.Fields{
				"imageName": result.Name,
				"error":     result.Err,
			}).Error("Failed to download image")
		}
	}

	log.WithFields(logrus.Fields{
		"successCount": len(downloadedImages),
		"failedCount":  failedCount,
	}).Info("Download results")

	downloadedImages = sortStatImages(downloadedImages, stats)

	return downloadedImages, nil
}

func createZip(images []ImageDownload, zipName string) error {
	log := logger.Log.WithFields(logrus.Fields{
		"function": "createZip",
		"zipName":  zipName,
		"images":   len(images),
	})

	log.Info("Creating ZIP archive")

	zipFile, err := os.Create(zipName)
	if err != nil {
		log.WithError(err).Error("Error creating zip file")
		return fmt.Errorf("error creating zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, img := range images {
		imgLog := log.WithField("imageName", img.Name)
		imgLog.Debug("Adding image to ZIP")

		zipEntry, err := zipWriter.Create(img.Name + ".jpg")
		if err != nil {
			imgLog.WithError(err).Error("Error creating zip entry")
			return fmt.Errorf("error creating zip entry for %s: %w", img.Name, err)
		}

		if _, err := io.Copy(zipEntry, bytes.NewReader(img.Data)); err != nil {
			imgLog.WithError(err).Error("Error writing image to zip")
			return fmt.Errorf("error writing image to zip: %w", err)
		}
	}

	log.Info("Successfully created ZIP archive")
	return nil
}

func cleanupTempFiles(uniqueID string) {
	log := logger.Log.WithFields(logrus.Fields{
		"function": "cleanupTempFiles",
		"uniqueID": uniqueID,
	})

	log.Info("Cleaning up temporary files")

	cfg := configuration.Get()
	tempDir := cfg.Verdansk.TempDir
	if tempDir == "" {
		tempDir = "verdansk_temp"
	}

	tempZipFiles.Lock()
	defer tempZipFiles.Unlock()

	zipFilename := filepath.Join(tempDir, uniqueID+".zip")
	delete(tempZipFiles.files, zipFilename)

	outputDir := filepath.Join(tempDir, uniqueID)
	if err := os.RemoveAll(outputDir); err != nil {
		log.WithError(err).Errorf("Failed to remove temp directory %s", outputDir)
	} else {
		log.WithField("outputDir", outputDir).Info("Removed temp directory")
	}

	if err := os.Remove(zipFilename); err != nil {
		log.WithError(err).Errorf("Failed to remove temp zip file %s", zipFilename)
	} else {
		log.WithField("zipFilename", zipFilename).Info("Removed zip file")
	}
}

func InitCleanupRoutine() {
	log := logger.Log.WithField("function", "InitCleanupRoutine")
	log.Info("Starting cleanup routine for Verdansk temporary files")

	cleanupAllTempFiles()

	go func() {
		for {
			time.Sleep(10 * time.Minute)
			cleanupOldZipFiles()
		}
	}()
}

func cleanupAllTempFiles() {
	log := logger.Log.WithField("function", "cleanupAllTempFiles")
	log.Info("Performing full cleanup of Verdansk temp files")

	cfg := configuration.Get()
	tempDir := cfg.Verdansk.TempDir
	if tempDir == "" {
		tempDir = "verdansk_temp"
	}

	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			log.WithError(err).Error("Failed to create Verdansk temp directory")
			return
		}
	}

	files, err := os.ReadDir(tempDir)
	if err != nil {
		log.WithError(err).Error("Error reading Verdansk temp directory")
		return
	}

	cleanedCount := 0
	for _, file := range files {
		if file.Name() == ".gitkeep" || file.Name() == ".git" {
			continue
		}

		filePath := filepath.Join(tempDir, file.Name())
		if err := os.RemoveAll(filePath); err != nil {
			log.WithError(err).Errorf("Failed to remove %s", filePath)
		} else {
			cleanedCount++
		}
	}

	log.Infof("Full cleanup complete, removed %d files/directories", cleanedCount)
}

func cleanupOldZipFiles() {
	log := logger.Log.WithField("function", "cleanupOldZipFiles")
	log.Debug("Running scheduled cleanup of old zip files")

	cfg := configuration.Get()
	tempDir := cfg.Verdansk.TempDir
	if tempDir == "" {
		tempDir = "verdansk_temp"
	}

	tempZipFiles.Lock()
	defer tempZipFiles.Unlock()

	now := time.Now()
	deletedCount := 0
	errorCount := 0

	for zipFile, expireTime := range tempZipFiles.files {
		if now.After(expireTime) {
			baseName := filepath.Base(zipFile)
			uniqueID := strings.TrimSuffix(baseName, ".zip")

			outputDir := filepath.Join(tempDir, uniqueID)
			if err := os.RemoveAll(outputDir); err != nil {
				log.WithError(err).Errorf("Failed to remove temp directory %s", outputDir)
				errorCount++
			}

			if err := os.Remove(zipFile); err != nil {
				log.WithError(err).Errorf("Failed to remove temp zip file %s", zipFile)
				errorCount++
			} else {
				deletedCount++
			}

			delete(tempZipFiles.files, zipFile)
		}
	}

	if deletedCount > 0 || errorCount > 0 {
		log.WithFields(logrus.Fields{
			"deletedCount": deletedCount,
			"errorCount":   errorCount,
		}).Info("Cleaned up expired zip files")
	}

	files, err := os.ReadDir(tempDir)
	if err != nil {
		log.WithError(err).Error("Error reading Verdansk temp directory")
		return
	}

	oldFileThreshold := now.Add(-24 * time.Hour)
	oldFilesRemoved := 0

	for _, file := range files {
		if file.Name() == ".gitkeep" || file.Name() == ".git" {
			continue
		}

		filePath := filepath.Join(tempDir, file.Name())
		info, err := os.Stat(filePath)
		if err != nil {
			log.WithError(err).Errorf("Error getting file info for %s", filePath)
			continue
		}

		if info.ModTime().Before(oldFileThreshold) {
			if err := os.RemoveAll(filePath); err != nil {
				log.WithError(err).Errorf("Failed to remove old file %s", filePath)
			} else {
				oldFilesRemoved++
			}
		}
	}

	if oldFilesRemoved > 0 {
		log.Infof("Removed %d untracked old files during cleanup", oldFilesRemoved)
	}
}
func doPreflightRequest(client *http.Client, targetURL string) error {
	log := logger.Log.WithFields(logrus.Fields{
		"function":  "doPreflightRequest",
		"targetURL": targetURL,
	})

	log.Debug("Setting up preflight request")

	req, err := http.NewRequest("OPTIONS", targetURL, nil)
	if err != nil {
		log.WithError(err).Error("Error creating preflight request")
		return fmt.Errorf("error creating preflight request: %w", err)
	}

	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-encoding", "gzip, deflate, br")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("access-control-request-headers", "x-api-key")
	req.Header.Set("access-control-request-method", "GET")
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("origin", originURL)
	req.Header.Set("pragma", "no-cache")
	req.Header.Set("referer", originURL+"/")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-site")
	req.Header.Set("user-agent", userAgentString)

	log.Debug("Sending preflight request")
	resp, err := client.Do(req)
	if err != nil {
		log.WithError(err).Error("Error sending preflight request")
		return fmt.Errorf("error sending preflight request: %w", err)
	}
	defer resp.Body.Close()

	log.WithField("statusCode", resp.StatusCode).Debug("Received preflight response")
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.WithFields(logrus.Fields{
			"statusCode": resp.StatusCode,
			"body":       string(body),
		}).Error("Preflight request failed")
		return fmt.Errorf("preflight request failed with status code %d: %s", resp.StatusCode, string(body))
	}

	log.Debug("Preflight request successful")
	return nil
}

func readResponseBody(resp *http.Response) ([]byte, error) {
	log := logger.Log.WithFields(logrus.Fields{
		"function":    "readResponseBody",
		"contentType": resp.Header.Get("Content-Type"),
		"encoding":    resp.Header.Get("Content-Encoding"),
	})

	log.Debug("Reading response body")

	var reader io.ReadCloser
	var err error

	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		log.Debug("Using gzip reader for compressed response")
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			log.WithError(err).Error("Error creating gzip reader")
			return nil, fmt.Errorf("error creating gzip reader: %w", err)
		}
		defer reader.Close()
	default:
		log.Debug("Using standard reader for response")
		reader = resp.Body
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		log.WithError(err).Error("Error reading response body")
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	log.WithField("bodyLength", len(body)).Debug("Successfully read response body")
	return body, nil
}

func formatStatName(name string) string {
	words := strings.Split(name, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[0:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

func respondToInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := services.RespondWithPreference(s, i, message, nil, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error responding to interaction")
	}
}

func sendFollowupMessage(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	_, err := services.FollowupWithPreference(s, i, message, nil, nil, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error sending followup message")
	}
}

func storeVerdanskStatsWithAccount(account *models.Account, stats map[string]StatValue) error {
	if account == nil {
		return fmt.Errorf("no account provided")
	}

	log := logger.Log.WithFields(logrus.Fields{
		"function":  "storeVerdanskStatsWithAccount",
		"accountID": account.ID,
	})

	account.IsOGVerdansk = true
	if err := database.DB.Save(account).Error; err != nil {
		log.WithError(err).Error("Failed to update account with Verdansk flag")
		return err
	}

	log.Info("Successfully stored Verdansk stats with account")
	return nil
}

func createVerdanskAPIRequest(method, url string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-encoding", "gzip, deflate, br")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("dnt", "1")
	req.Header.Set("origin", originURL)
	req.Header.Set("pragma", "no-cache")
	req.Header.Set("referer", originURL+"/")
	req.Header.Set("sec-ch-ua", "\"Chromium\";v=\"134\", \"Not:A-Brand\";v=\"24\", \"Google Chrome\";v=\"134\"")
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", "\"Windows\"")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-site")
	req.Header.Set("user-agent", userAgentString)

	if method != "OPTIONS" {
		req.Header.Set("x-api-key", X_APIKey)
	}

	return req, nil
}

func validateVerdanskAPIResponse(body []byte, expectedType string) error {
	switch expectedType {
	case "preferences":
		var prefs PlayerPreferences
		if err := json.Unmarshal(body, &prefs); err != nil {
			return fmt.Errorf("invalid preferences data: %w", err)
		}
		return nil
	case "stats":
		var stats map[string]StatValue
		if err := json.Unmarshal(body, &stats); err != nil {
			return fmt.Errorf("invalid stats data: %w", err)
		}
		if len(stats) == 0 {
			return fmt.Errorf("empty stats data received")
		}
		return nil
	default:
		return fmt.Errorf("unknown validation type: %s", expectedType)
	}
}

func sortStatImages(images []ImageDownload, stats map[string]StatValue) []ImageDownload {
	sortedImages := make([]ImageDownload, len(images))
	copy(sortedImages, images)

	sort.SliceStable(sortedImages, func(i, j int) bool {
		nameI := sortedImages[i].Name
		nameJ := sortedImages[j].Name

		if stats[nameI].OrderValue != nil && stats[nameJ].OrderValue != nil {
			return *stats[nameI].OrderValue < *stats[nameJ].OrderValue
		}

		if stats[nameI].OrderValue != nil {
			return true
		}
		if stats[nameJ].OrderValue != nil {
			return false
		}

		return nameI < nameJ
	})

	return sortedImages
}

func getImageCategory(statName string) string {
	lowerName := strings.ToLower(statName)

	if strings.Contains(lowerName, "kill") ||
		strings.Contains(lowerName, "death") ||
		strings.Contains(lowerName, "kd") {
		return "Combat"
	}

	if strings.Contains(lowerName, "win") ||
		strings.Contains(lowerName, "placement") ||
		strings.Contains(lowerName, "medal") {
		return "Performance"
	}

	if strings.Contains(lowerName, "time") ||
		strings.Contains(lowerName, "session") ||
		strings.Contains(lowerName, "played") {
		return "Engagement"
	}

	if strings.Contains(lowerName, "weapon") ||
		strings.Contains(lowerName, "loadout") ||
		strings.Contains(lowerName, "equipment") {
		return "Loadout"
	}

	if strings.Contains(lowerName, "friend") ||
		strings.Contains(lowerName, "team") ||
		strings.Contains(lowerName, "squad") {
		return "Social"
	}

	return "Miscellaneous"
}

func organizeImagesForDisplay(images []ImageDownload, stats map[string]StatValue, maxImages int) []ImageDownload {
	if len(images) <= maxImages {
		return sortStatImages(images, stats)
	}

	categorizedImages := make(map[string][]ImageDownload)

	for _, img := range images {
		category := getImageCategory(img.Name)
		categorizedImages[category] = append(categorizedImages[category], img)
	}

	categoryPriority := []string{
		"Performance", "Combat", "Loadout", "Engagement", "Social", "Miscellaneous",
	}

	var selectedImages []ImageDownload
	remainingSlots := maxImages

	for _, category := range categoryPriority {
		if catImages, exists := categorizedImages[category]; exists && len(catImages) > 0 {
			sortedCatImages := sortStatImages(catImages, stats)
			selectedImages = append(selectedImages, sortedCatImages[0])
			categorizedImages[category] = sortedCatImages[1:]
			remainingSlots--

			if remainingSlots <= 0 {
				break
			}
		}
	}

	if remainingSlots > 0 {
		for remainingSlots > 0 {
			for _, category := range categoryPriority {
				if catImages, exists := categorizedImages[category]; exists && len(catImages) > 0 {
					selectedImages = append(selectedImages, catImages[0])
					categorizedImages[category] = catImages[1:]
					remainingSlots--

					if remainingSlots <= 0 {
						break
					}
				}
			}

			allEmpty := true
			for _, catImages := range categorizedImages {
				if len(catImages) > 0 {
					allEmpty = false
					break
				}
			}

			if allEmpty {
				break
			}
		}
	}

	return selectedImages
}

func generateVerdanskStatsEmbed(activisionID string, stats map[string]StatValue) *discordgo.MessageEmbed {
	var fields []*discordgo.MessageEmbedField

	keyStats := map[string]string{
		"total_kills":            "Total Kills",
		"total_deaths":           "Total Deaths",
		"kd_ratio":               "K/D Ratio",
		"matches_played":         "Matches Played",
		"total_wins":             "Total Wins",
		"win_percentage":         "Win Percentage",
		"hours_played":           "Hours Played",
		"favorite_weapon":        "Favorite Weapon",
		"favorite_drop_location": "Favorite Drop Location",
	}

	for statKey, displayName := range keyStats {
		possibleKeys := []string{
			statKey,
			strings.ReplaceAll(statKey, "_", ""),
			"verdansk_" + statKey,
			"warzone_" + statKey,
		}

		for _, key := range possibleKeys {
			if stat, exists := stats[key]; exists && stat.StringValue != "" {
				fields = append(fields, &discordgo.MessageEmbedField{
					Name:   displayName,
					Value:  stat.StringValue,
					Inline: true,
				})
				break
			}
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Verdansk Replay Stats Summary for %s", activisionID),
		Description: "Here's a summary of your Verdansk career stats. Download the images or zip file to see detailed visualizations.",
		Color:       0x00BFFF,
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Data provided by Call of Duty Verdansk Replay",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	return embed
}

func retryVerdanskRequest(client *http.Client, method, url string, maxRetries int) (*http.Response, error) {
	log := logger.Log.WithFields(logrus.Fields{
		"function": "retryVerdanskRequest",
		"method":   method,
		"url":      url,
	})

	var resp *http.Response
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := createVerdanskAPIRequest(method, url)
		if err != nil {
			return nil, err
		}

		log.Infof("Sending request (attempt %d/%d)", attempt, maxRetries)
		resp, err = client.Do(req)

		if err != nil {
			if strings.Contains(err.Error(), "timeout") ||
				strings.Contains(err.Error(), "connection reset") ||
				strings.Contains(err.Error(), "connection refused") {

				log.WithError(err).Warnf("Temporary error on attempt %d, retrying", attempt)
				backoffDuration := time.Duration(100*attempt*attempt) * time.Millisecond
				time.Sleep(backoffDuration)
				continue
			}

			return nil, err
		}

		if resp.StatusCode >= 500 && resp.StatusCode < 600 {
			log.Warnf("Server error %d on attempt %d, retrying", resp.StatusCode, attempt)
			resp.Body.Close()

			backoffDuration := time.Duration(100*attempt*attempt) * time.Millisecond
			time.Sleep(backoffDuration)
			continue
		}

		return resp, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, err)
	}

	return resp, nil
}

var monitorActiveDownloads = struct {
	sync.RWMutex
	active     map[string]int
	maxPerUser int
}{
	active:     make(map[string]int),
	maxPerUser: 2,
}

func canStartNewDownload(userID string) bool {
	monitorActiveDownloads.RLock()
	defer monitorActiveDownloads.RUnlock()

	count := monitorActiveDownloads.active[userID]
	return count < monitorActiveDownloads.maxPerUser
}

func incrementActiveDownloads(userID string) {
	monitorActiveDownloads.Lock()
	defer monitorActiveDownloads.Unlock()

	monitorActiveDownloads.active[userID]++
}

func decrementActiveDownloads(userID string) {
	monitorActiveDownloads.Lock()
	defer monitorActiveDownloads.Unlock()

	monitorActiveDownloads.active[userID]--
	if monitorActiveDownloads.active[userID] < 0 {
		monitorActiveDownloads.active[userID] = 0
	}
}

func markAccountWithVerdanskStats(userID, activisionID string) {
	log := logger.Log.WithFields(logrus.Fields{
		"function":     "markAccountWithVerdanskStats",
		"userID":       userID,
		"activisionID": activisionID,
	})

	var accounts []models.Account
	result := database.DB.Where("user_id = ? AND activision_id = ?", userID, activisionID).Find(&accounts)

	if result.Error != nil {
		log.WithError(result.Error).Error("Error fetching accounts")
		return
	}

	if len(accounts) == 0 {
		log.Info("No accounts found with matching Activision ID")
		return
	}

	for _, account := range accounts {
		if !account.IsOGVerdansk {
			account.IsOGVerdansk = true
			if err := database.DB.Save(&account).Error; err != nil {
				log.WithError(err).Errorf("Failed to update OGVerdansk flag for account %d", account.ID)
			} else {
				log.Infof("Successfully marked account %d as having Verdansk stats", account.ID)
			}
		}
	}
}

func loadVerdanskConfiguration() {
	cfg := configuration.Get()

	if cfg.Verdansk.APIKey != "" {
		X_APIKey = cfg.Verdansk.APIKey
	} else {
		X_APIKey = "a855a770-cf8a-4ae8-9f30-b787d676e608"
		logger.Log.Warn("Using default X-API-Key for Verdansk.")
	}

	tempDir := cfg.Verdansk.TempDir
	if tempDir == "" {
		tempDir = "verdansk_temp"
	}

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		logger.Log.WithError(err).Error("Failed to create Verdansk temp directory")
	}

	cleanupTime := cfg.Verdansk.CleanupTime
	if cleanupTime == 0 {
		cleanupTime = 30 * time.Minute
		logger.Log.Info("Using default cleanup time of 30 minutes for Verdansk files")
	}

	logger.Log.WithFields(logrus.Fields{
		"apiKey":      X_APIKey[:8] + "...",
		"tempDir":     tempDir,
		"cleanupTime": cleanupTime,
	}).Info("Verdansk configuration loaded")
}

func StartVerdanskCleanupRoutine() {
	loadVerdanskConfiguration()

	InitCleanupRoutine()

	go func() {
		cfg := configuration.Get()
		tempDir := cfg.Verdansk.TempDir
		if tempDir == "" {
			tempDir = "verdansk_temp"
		}

		logger.Log.Info("Performing initial cleanup of Verdansk temp files")

		files, err := os.ReadDir(tempDir)
		if err != nil {
			logger.Log.WithError(err).Error("Error reading Verdansk temp directory")
			return
		}

		cleanedFiles := 0
		for _, file := range files {
			filePath := filepath.Join(tempDir, file.Name())

			if file.Name() == ".gitkeep" || file.Name() == ".git" {
				continue
			}

			if err := os.RemoveAll(filePath); err != nil {
				logger.Log.WithError(err).Errorf("Failed to remove %s", filePath)
			} else {
				cleanedFiles++
			}
		}

		logger.Log.Infof("Initial cleanup complete, removed %d files/directories", cleanedFiles)
	}()
}

type VerdanskSession struct {
	UserID           string
	ActivisionID     string
	AccountID        uint
	StartTime        time.Time
	DownloadedStats  bool
	ImagesDownloaded int
	TempDir          string
	ZipFile          string
	State            string
	ErrorMessage     string
	ConcurrentJobs   int
}

var sessionManager = struct {
	sync.RWMutex
	sessions map[string]*VerdanskSession
}{
	sessions: make(map[string]*VerdanskSession),
}

func createVerdanskSession(userID, activisionID string, accountID uint) *VerdanskSession {
	sessionManager.Lock()
	defer sessionManager.Unlock()

	session := &VerdanskSession{
		UserID:         userID,
		ActivisionID:   activisionID,
		AccountID:      accountID,
		StartTime:      time.Now(),
		State:          "starting",
		ConcurrentJobs: 3,
	}

	sessionManager.sessions[userID] = session
	return session
}

func updateSessionState(userID, newState string, errorMsg string) {
	sessionManager.Lock()
	defer sessionManager.Unlock()

	session, exists := sessionManager.sessions[userID]
	if !exists {
		return
	}

	oldState := session.State
	session.State = newState

	if errorMsg != "" {
		session.ErrorMessage = errorMsg
	}

	logger.Log.WithFields(logrus.Fields{
		"userID":       userID,
		"activisionID": session.ActivisionID,
		"transition":   fmt.Sprintf("%s -> %s", oldState, newState),
		"error":        errorMsg,
		"duration":     time.Since(session.StartTime).String(),
	}).Info("Verdansk session state updated")
}

func logVerdanskJobCompletion(userID, operationType string, success bool, duration time.Duration, details string) {
	services.LogCommandExecution(
		fmt.Sprintf("verdansk_%s", operationType),
		userID,
		"",
		success,
		duration.Milliseconds(),
		details,
	)

	logger.Log.WithFields(logrus.Fields{
		"userID":        userID,
		"operationType": operationType,
		"success":       success,
		"duration":      duration.String(),
		"details":       details,
	}).Info("Verdansk operation completed")
}

func cleanupSession(userID string) {
	sessionManager.Lock()
	defer sessionManager.Unlock()

	session, exists := sessionManager.sessions[userID]
	if !exists {
		return
	}

	if session.TempDir != "" && session.ZipFile != "" {
		tempDir := session.TempDir
		zipFile := session.ZipFile

		go func() {
			time.Sleep(30 * time.Minute)

			if err := os.RemoveAll(tempDir); err != nil {
				logger.Log.WithError(err).Errorf("Failed to remove Verdansk temp directory %s", tempDir)
			}

			if err := os.Remove(zipFile); err != nil {
				logger.Log.WithError(err).Errorf("Failed to remove Verdansk zip file %s", zipFile)
			}

			logger.Log.Infof("Cleaned up Verdansk session files for user %s", userID)
		}()
	}

	delete(sessionManager.sessions, userID)
}

func optimizeJPEG(imageData []byte, quality int) ([]byte, error) {
	if len(imageData) == 0 {
		return nil, fmt.Errorf("empty image data")
	}

	img, err := jpeg.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode JPEG: %w", err)
	}

	var buf bytes.Buffer
	err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
	if err != nil {
		return nil, fmt.Errorf("failed to re-encode JPEG: %w", err)
	}

	return buf.Bytes(), nil
}

func processImageBatch(images []ImageDownload) ([]ImageDownload, error) {
	var processed []ImageDownload
	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, len(images))

	for _, img := range images {
		wg.Add(1)
		go func(image ImageDownload) {
			defer wg.Done()

			if image.Err != nil || len(image.Data) == 0 {
				mu.Lock()
				processed = append(processed, image)
				mu.Unlock()
				return
			}

			optimized, err := optimizeJPEG(image.Data, 80)
			if err != nil {
				image.Err = err
				errChan <- fmt.Errorf("failed to optimize image %s: %w", image.Name, err)
				mu.Lock()
				processed = append(processed, image)
				mu.Unlock()
				return
			}

			if len(optimized) < len(image.Data) {
				image.Data = optimized
			}

			mu.Lock()
			processed = append(processed, image)
			mu.Unlock()
		}(img)
	}

	wg.Wait()
	close(errChan)

	var errors []string
	for err := range errChan {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		return processed, fmt.Errorf("errors processing images: %s", strings.Join(errors, "; "))
	}

	return processed, nil
}

func sendProgressUpdate(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	if i.Interaction.Token == "" {
		return
	}

	_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: fmt.Sprintf("⏳ **Processing Update**: %s", message),
		Flags:   discordgo.MessageFlagsEphemeral,
	})

	if err != nil {
		logger.Log.WithError(err).Warn("Failed to send progress update")
	}
}

func createVerdanskProgressBar(current, total int, barLength int) string {
	if total <= 0 {
		return "[--------------------] (Unknown)"
	}

	percent := float64(current) / float64(total)
	filledLength := int(math.Round(float64(barLength) * percent))

	bar := "["
	for i := 0; i < barLength; i++ {
		if i < filledLength {
			bar += "="
		} else if i == filledLength {
			bar += ">"
		} else {
			bar += "-"
		}
	}
	bar += fmt.Sprintf("] (%d/%d, %.0f%%)", current, total, percent*100)

	return bar
}

func generateVerdanskSummary(stats map[string]StatValue) string {
	summary := "**Verdansk Career Summary**\n\n"

	statGroups := []struct {
		Title string
		Keys  []string
	}{
		{
			Title: "Combat Performance",
			Keys:  []string{"kills", "deaths", "kd_ratio", "headshots", "accuracy"},
		},
		{
			Title: "Match Statistics",
			Keys:  []string{"games_played", "wins", "win_percentage", "avg_placement"},
		},
		{
			Title: "Gameplay",
			Keys:  []string{"time_played", "favorite_drop", "favorite_weapon", "most_kills"},
		},
	}

	for _, group := range statGroups {
		groupContent := ""

		for _, key := range group.Keys {
			for statKey, value := range stats {
				normalizedKey := strings.ToLower(strings.ReplaceAll(statKey, "_", ""))
				if strings.Contains(normalizedKey, key) && value.StringValue != "" {
					displayName := formatStatName(statKey)
					groupContent += fmt.Sprintf("• **%s**: %s\n", displayName, value.StringValue)
					break
				}
			}
		}

		if groupContent != "" {
			summary += fmt.Sprintf("**%s**\n%s\n", group.Title, groupContent)
		}
	}

	return summary
}

func verifyVerdanskAvailability(activisionID string) (bool, error) {
	client := services.GetDefaultHTTPClient()
	encodedID := url.QueryEscape(activisionID)

	preferences, err := fetchPlayerPreferences(client, encodedID)
	if err != nil {
		return false, err
	}

	return preferences.Visible, nil
}

func getActivisionAccountInfo(client *http.Client, activisionID string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"activisionID": activisionID,
		"platform":     "unknown",
	}, nil
}

func sendVerdanskErrorEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, title, description, errorDetail string) {
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0xFF0000,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Error Details",
				Value: errorDetail,
			},
			{
				Name: "Next Steps",
				Value: "You can try the following:\n" +
					"• Make sure your profile is set to visible in Activision settings\n" +
					"• Try again with a different account\n" +
					"• Ensure your Activision ID is correct",
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "COD Status Bot - Verdansk Replay",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
		Flags:  discordgo.MessageFlagsEphemeral,
	})

	if err != nil {
		logger.Log.WithError(err).Error("Failed to send error embed")
	}
}

func isVerdanskAPIAvailable() bool {
	client := services.GetDefaultHTTPClient()

	req, err := createVerdanskAPIRequest("OPTIONS", "https://pd.callofduty.com/api/x/v1/campaign/warzonewrapped/preferences/gamer/test")
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode < 500
}

type VerdanskConfig struct {
	EnabledInGuilds  bool
	MaxConcurrentDLs int
	MaxImageQuality  int
	EnableOptimize   bool
	APIRateLimit     time.Duration
}

var verdanskConfig = VerdanskConfig{
	EnabledInGuilds:  true,
	MaxConcurrentDLs: 3,
	MaxImageQuality:  85,
	EnableOptimize:   true,
	APIRateLimit:     500 * time.Millisecond,
}

var verdanskRateLimiter = struct {
	sync.Mutex
	lastRequest time.Time
}{
	lastRequest: time.Now().Add(-24 * time.Hour),
}

func waitForRateLimit() {
	verdanskRateLimiter.Mutex.Lock()
	defer verdanskRateLimiter.Mutex.Unlock()

	elapsed := time.Since(verdanskRateLimiter.lastRequest)
	if elapsed < verdanskConfig.APIRateLimit {
		time.Sleep(verdanskConfig.APIRateLimit - elapsed)
	}

	verdanskRateLimiter.lastRequest = time.Now()
}

func enrichStatData(stats map[string]StatValue) map[string]StatValue {
	enriched := make(map[string]StatValue, len(stats))
	for k, v := range stats {
		enriched[k] = v
	}

	if kdStr, exists := getStatStringValue(stats, "kd_ratio"); exists {
		if kdVal, err := strconv.ParseFloat(kdStr, 64); err == nil {
			enriched["kd_ratio"] = StatValue{
				OrderValue:  stats["kd_ratio"].OrderValue,
				StringValue: fmt.Sprintf("%.2f", kdVal),
			}
		}
	}

	wins := getStatIntValue(stats, "wins")
	matches := getStatIntValue(stats, "matches_played")
	if wins > 0 && matches > 0 {
		winPct := float64(wins) / float64(matches) * 100
		enriched["calculated_win_percentage"] = StatValue{
			OrderValue:  new(int),
			StringValue: fmt.Sprintf("%.1f%%", winPct),
		}
	}

	return enriched
}

func getStatStringValue(stats map[string]StatValue, keys ...string) (string, bool) {
	for _, key := range keys {
		if stat, exists := stats[key]; exists && stat.StringValue != "" {
			return stat.StringValue, true
		}

		if stat, exists := stats["verdansk_"+key]; exists && stat.StringValue != "" {
			return stat.StringValue, true
		}
	}
	return "", false
}

func getStatIntValue(stats map[string]StatValue, keys ...string) int {
	for _, key := range keys {
		if strVal, exists := getStatStringValue(stats, key); exists {
			if val, err := strconv.Atoi(strVal); err == nil {
				return val
			}
		}
	}
	return 0
}

func HandleVerdanskCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !isVerdanskAPIAvailable() {
		respondToInteraction(s, i, "The Verdansk Replay API is currently unavailable. Please try again later.")
		return
	}

	cfg := configuration.Get()

	verdanskRateLimits.Lock()
	verdanskRateLimits.globalSettings.CommandCooldown = cfg.Verdansk.CommandCooldown
	verdanskRateLimits.globalSettings.MaxRequestsPerDay = cfg.Verdansk.MaxRequestsPerDay
	verdanskRateLimits.globalSettings.CleanupTime = cfg.Verdansk.CleanupTime
	verdanskRateLimits.Unlock()

	CommandVerdansk(s, i)
}

func notifyUnsupportedMessage(s *discordgo.Session, i *discordgo.InteractionCreate, activisionID string) {
	embed := &discordgo.MessageEmbed{
		Title:       "Verdansk Stats Not Available",
		Description: fmt.Sprintf("Unfortunately, Verdansk Replay stats are not available for **%s**.", activisionID),
		Color:       0xFF9900,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name: "Possible Reasons",
				Value: "• Not enough Verdansk gameplay (at least 5 matches required)\n" +
					"• Your Game Data settings are set to private\n" +
					"• The account was created after Verdansk ended",
			},
			{
				Name: "How to Fix",
				Value: "1. Go to [callofduty.com/profile](https://profile.callofduty.com/cod/login)\n" +
					"2. Log in with your Activision account\n" +
					"3. Go to Privacy & Security settings\n" +
					"4. Make sure Game Data is set to visible\n" +
					"5. Try again in 24 hours as changes may take time to apply",
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "COD Status Bot - Verdansk Replay",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
		Flags:  discordgo.MessageFlagsEphemeral,
	})
}

func getVerdanskFeatureEmbed() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "Verdansk Replay Stats Feature",
		Description: "The Verdansk Replay feature allows you to view your Warzone statistics from the original Verdansk map (March 2020 - December 2021).",
		Color:       0x00BFFF,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name: "Requirements",
				Value: "• An Activision account that played Warzone during the Verdansk era\n" +
					"• At least 5 matches played in Verdansk\n" +
					"• Game Data visibility enabled in your Activision profile",
				Inline: false,
			},
			{
				Name: "How to Use",
				Value: "1. Use the `/verdansk` command\n" +
					"2. Choose to either select one of your monitored accounts or enter your Activision ID\n" +
					"3. Wait while the bot retrieves and processes your Verdansk stats\n" +
					"4. View the results and download the ZIP file for permanent access",
				Inline: false,
			},
			{
				Name:   "Data Privacy",
				Value:  "This command only retrieves publicly available data from Call of Duty's official APIs. Your data is temporarily stored for processing and automatically deleted after 30 minutes.",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Note: Verdansk stats are provided by Call of Duty and may not be available for all accounts.",
		},
	}
}

func enrichVerdanskFilenames(images []ImageDownload) []ImageDownload {
	nameMap := map[string]string{
		"total_kills":            "01_Total_Kills",
		"kd_ratio":               "02_KD_Ratio",
		"favorite_weapon":        "03_Favorite_Weapon",
		"favorite_drop_location": "04_Favorite_Drop",
		"total_wins":             "05_Total_Wins",
		"matches_played":         "06_Matches_Played",
		"hours_played":           "07_Hours_Played",
		"best_game":              "08_Best_Game",
		"verdansk_map":           "09_Verdansk_Map",
	}

	result := make([]ImageDownload, len(images))
	for i, img := range images {
		newName := img.Name

		if betterName, exists := nameMap[img.Name]; exists {
			newName = betterName
		} else {
			formattedName := strings.ReplaceAll(img.Name, "_", " ")
			words := strings.Split(formattedName, " ")
			for i, word := range words {
				if len(word) > 0 {
					words[i] = strings.ToUpper(word[0:1]) + strings.ToLower(word[1:])
				}
			}
			newName = strings.Join(words, "_")
		}

		result[i] = ImageDownload{
			Name: newName,
			URL:  img.URL,
			Data: img.Data,
			Err:  img.Err,
		}
	}

	return result
}

func groupImagesByCategory(images []ImageDownload) map[string][]ImageDownload {
	categories := map[string][]ImageDownload{
		"Combat":        {},
		"Performance":   {},
		"Gameplay":      {},
		"Weapons":       {},
		"Locations":     {},
		"Miscellaneous": {},
	}

	categoryKeywords := map[string][]string{
		"Combat":      {"kill", "death", "kd", "headshot", "accuracy"},
		"Performance": {"win", "score", "placement", "match", "game"},
		"Gameplay":    {"time", "hour", "day", "played", "session"},
		"Weapons":     {"weapon", "gun", "loadout", "equipment", "favorite_weapon"},
		"Locations":   {"location", "drop", "zone", "map", "area"},
	}

	for _, img := range images {
		lowerName := strings.ToLower(img.Name)
		assigned := false

		for category, keywords := range categoryKeywords {
			for _, keyword := range keywords {
				if strings.Contains(lowerName, keyword) {
					categories[category] = append(categories[category], img)
					assigned = true
					break
				}
			}
			if assigned {
				break
			}
		}

		if !assigned {
			categories["Miscellaneous"] = append(categories["Miscellaneous"], img)
		}
	}

	return categories
}
