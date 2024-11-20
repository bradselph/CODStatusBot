package webserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bradselph/CODStatusBot/services"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"

	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/gorilla/sessions"
)

const (
	discordAPIBase = "https://discord.com/api/v10"
)

var (
	isDevelopment          bool
	statsLimiter           *limiter.Limiter
	cachedStats            Stats
	cachedStatsLock        sync.RWMutex
	cacheInterval          = 15 * time.Minute
	NotificationMutex      sync.Mutex
	store                  *sessions.CookieStore
	cachedDiscordStats     DiscordStats
	cachedDiscordStatsLock sync.RWMutex
	templatesDir           string
)

func initRateLimiter() {
	var rate = 1.0
	rateStr := os.Getenv("STATS_RATE_LIMIT")
	if rateStr != "" {
		if parsedRate, err := strconv.ParseFloat(rateStr, 64); err == nil {
			rate = parsedRate
		}
	}
	statsLimiter = tollbooth.NewLimiter(rate, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour})
}

type Stats struct {
	TotalAccounts            int     // Total accounts in the database
	ActiveAccounts           int     // Accounts that have checked in the last 24 hours
	BannedAccounts           int     // Total of all ban types
	PermaBannedAccounts      int     // Permanent bans
	ShadowBannedAccounts     int     // Shadow bans
	TempBannedAccounts       int     // Temporary bans
	TotalUsers               int     // Total users in the database
	ChecksLastHour           int     // Checks in the last hour
	ChecksLast24Hours        int     // Checks in the last 24 hours
	TotalBans                int     // Total bans in the database
	RecentBans               int     // Recent bans in the last 24 hours
	AverageChecksPerDay      float64 // Average checks per day
	TotalNotifications       int
	RecentNotifications      int              // Recent notifications in the last 24 hours
	UsersWithCustomAPIKey    int              // Users with a custom API key
	AverageAccountsPerUser   float64          // Average accounts per user
	OldestAccount            time.Time        // Oldest account in the database
	NewestAccount            time.Time        // Newest account in the database
	TotalShadowbans          int              // Total shadow bans in the database
	TotalTempbans            int              // Total temporary bans in the database
	BanDates                 []string         `json:"banDates"`                 // Dates of bans
	BanCounts                []int            `json:"banCounts"`                // Counts of bans on each date
	AccountStatsHistory      []HistoricalData `json:"accountStatsHistory"`      // Historical data for accounts
	UserStatsHistory         []HistoricalData `json:"userStatsHistory"`         // Historical data for users
	CheckStatsHistory        []HistoricalData `json:"checkStatsHistory"`        //	Historical data for checks
	BanStatsHistory          []HistoricalData `json:"banStatsHistory"`          // Historical data for bans
	NotificationStatsHistory []HistoricalData `json:"notificationStatsHistory"` // Historical data for notifications
}

type HistoricalData struct {
	Date  string `json:"date"`
	Value int    `json:"value"`
}

type DiscordStats struct {
	ServerCount int `json:"server_count"`
}

//nolint:gochecknoinits
func init() {
	if err := godotenv.Load(); err != nil {
		logger.Log.WithError(err).Error("Error loading .env file")
	}

	isDevelopment = os.Getenv("ENVIRONMENT") == "development"

	initRateLimiter()
	sessionKey := os.Getenv("SESSION_KEY")
	if sessionKey == "" {
		logger.Log.Error("SESSION_KEY not set in environment variables")
	}

	store = sessions.NewCookieStore([]byte(sessionKey))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   !isDevelopment,
		SameSite: http.SameSiteLaxMode,
	}

	templatesDir = os.Getenv("TEMPLATES_DIR")
	if templatesDir == "" {
		templatesDir = "templates"
	}

}

func StartAdminDashboard() *http.Server {
	r := mux.NewRouter().StrictSlash(true)

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isDevelopment {
				if proto := r.Header.Get("X-Forwarded-Proto"); proto != "https" {
					url := "https://" + r.Host + r.URL.Path
					if r.URL.RawQuery != "" {
						url += "?" + r.URL.RawQuery
					}
					http.Redirect(w, r, url, http.StatusPermanentRedirect)
					return
				}

				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			if !strings.HasPrefix(r.URL.Path, "/static/") {
				w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
				w.Header().Set("Pragma", "no-cache")
			}

			startTime := time.Now()
			logger.Log.WithFields(logrus.Fields{
				"method":     r.Method,
				"path":       r.URL.Path,
				"remote_ip":  r.RemoteAddr,
				"user_agent": r.UserAgent(),
			}).Info("Incoming request")

			next.ServeHTTP(w, r)

			duration := time.Since(startTime)
			logger.Log.WithFields(logrus.Fields{
				"method":      r.Method,
				"path":        r.URL.Path,
				"duration_ms": duration.Milliseconds(),
			}).Info("Request completed")
		})
	})

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/static/") && !strings.Contains(r.URL.Path, ".") {
				if r.URL.Path != "/" && r.URL.Path[len(r.URL.Path)-1] != '/' {
					http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	})

	r.HandleFunc("/health/", HealthCheckHandler).Methods("GET")

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".css") ||
			strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Cache-Control", "public, max-age=31536000")
		}

		staticDir := os.Getenv("STATIC_DIR")
		if staticDir == "" {
			staticDir = "static"
		}

		http.FileServer(http.Dir(staticDir)).ServeHTTP(w, r)
	})))

	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/help/", HelpHandler)
	r.HandleFunc("/terms/", TermsHandler)
	r.HandleFunc("/policy/", PolicyHandler)
	r.HandleFunc("/admin/login/", LoginHandler)
	r.HandleFunc("/admin/logout/", LogoutHandler)
	r.HandleFunc("/admin/", AuthMiddleware(DashboardHandler))
	r.HandleFunc("/admin/stats/", AuthMiddleware(StatsHandler))
	r.HandleFunc("/api/server-count/", ServerCountHandler)

	port := os.Getenv("ADMIN_PORT")

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 20 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		logger.Log.Infof("Admin dashboard starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.WithError(err).Fatal("Failed to start admin dashboard")
		}
	}()

	return server
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	dbHealthy := true
	if err := database.DB.Exec("SELECT 1").Error; err != nil {
		dbHealthy = false
	}

	var accounts int64
	database.DB.Model(&models.Account{}).Count(&accounts)

	health := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "3.10.0",
		"services": map[string]interface{}{
			"ezcaptcha_enabled": services.IsServiceEnabled("ezcaptcha"),
			"2captcha_enabled":  services.IsServiceEnabled("2captcha"),
			"database": map[string]interface{}{
				"status":   dbHealthy,
				"accounts": accounts,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "admin-session")
		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			http.Redirect(w, r, "/admin/login/", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func StartStatsCaching() {
	go func() {
		for {
			updateCachedStats()
			time.Sleep(cacheInterval)
		}
	}()
}

func updateCachedStats() {
	stats, err := getStats()
	if err != nil {
		logger.Log.WithError(err).Error("Error updating cached stats")
		return
	}

	go func() {
		var accountStats []HistoricalData
		accountStats, err = getHistoricalData("accounts")
		if err == nil {
			cachedStatsLock.Lock()
			cachedStats.AccountStatsHistory = accountStats
			cachedStatsLock.Unlock()
		}
	}()

	go func() {
		var userStats []HistoricalData
		userStats, err = getHistoricalData("users")
		if err == nil {
			cachedStatsLock.Lock()
			cachedStats.UserStatsHistory = userStats
			cachedStatsLock.Unlock()
		}
	}()

	go func() {
		var checkStats []HistoricalData
		checkStats, err = getHistoricalData("checks")
		if err == nil {
			cachedStatsLock.Lock()
			cachedStats.CheckStatsHistory = checkStats
			cachedStatsLock.Unlock()
		}
	}()

	go func() {
		var banStats []HistoricalData
		banStats, err = getHistoricalData("bans")
		if err == nil {
			cachedStatsLock.Lock()
			cachedStats.BanStatsHistory = banStats
			cachedStatsLock.Unlock()
		}
	}()

	go func() {
		var notificationStats []HistoricalData
		notificationStats, err = getHistoricalData("notifications")
		if err == nil {
			cachedStatsLock.Lock()
			cachedStats.NotificationStatsHistory = notificationStats
			cachedStatsLock.Unlock()
		}
	}()

	var botstats DiscordStats
	botstats, err = fetchDiscordStats()
	if err != nil {
		logger.Log.WithError(err).Error("Failed to fetch Discord botstats")
		return
	}

	cachedStatsLock.Lock()
	cachedDiscordStatsLock.Lock()
	cachedStats = stats
	cachedDiscordStats = botstats
	cachedStatsLock.Unlock()
	cachedDiscordStatsLock.Unlock()
}

func GetCachedStats() Stats {
	cachedStatsLock.RLock()
	defer cachedStatsLock.RUnlock()
	return cachedStats
}

func ServerCountHandler(w http.ResponseWriter, _ *http.Request) {
	cachedDiscordStatsLock.RLock()
	botstats := cachedDiscordStats
	cachedDiscordStatsLock.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(botstats); err != nil {
		logger.Log.WithError(err).Error("Error encoding server count response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func StatsHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	httpError := tollbooth.LimitByRequest(statsLimiter, w, r)
	if httpError != nil {
		http.Error(w, httpError.Message, httpError.StatusCode)
		return
	}

	stats := GetCachedStats()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		logger.Log.WithError(err).Error("Error encoding stats")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func TermsHandler(w http.ResponseWriter, _ *http.Request) {
	tmpl, err := template.ParseFiles(filepath.Join(templatesDir, "terms.html"))
	if err != nil {
		logger.Log.WithError(err).Error("Failed to parse terms template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, nil); err != nil {
		logger.Log.WithError(err).Error("Failed to execute template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func PolicyHandler(w http.ResponseWriter, _ *http.Request) {
	tmpl, err := template.ParseFiles(filepath.Join(templatesDir, "policy.html"))
	if err != nil {
		logger.Log.WithError(err).Error("Failed to parse policy template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, nil); err != nil {
		logger.Log.WithError(err).Error("Failed to execute template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func HomeHandler(w http.ResponseWriter, _ *http.Request) {
	tmpl, err := template.ParseFiles(filepath.Join(templatesDir, "index.html"))
	if err != nil {
		logger.Log.WithError(err).Error("Failed to parse index template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, nil); err != nil {
		logger.Log.WithError(err).Error("Failed to execute template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func HelpHandler(w http.ResponseWriter, _ *http.Request) {
	tmpl, err := template.ParseFiles(filepath.Join(templatesDir, "help.html"))
	if err != nil {
		logger.Log.WithError(err).Error("Failed to parse help template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, nil); err != nil {
		logger.Log.WithError(err).Error("Failed to execute template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "admin-session")
	session.Values["authenticated"] = false
	err := session.Save(r, w)
	if err != nil {
		return
	}
	http.Redirect(w, r, "/admin/login/", http.StatusSeeOther)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(filepath.Join(templatesDir, "login.html"))
	if err != nil {
		logger.Log.WithError(err).Error("Failed to parse login template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")
		if checkCredentials(username, password) {
			session, err := store.Get(r, "admin-session")
			if err != nil {
				logger.Log.WithError(err).Error("Error getting session")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			session.Values["authenticated"] = true
			err = session.Save(r, w)
			if err != nil {
				logger.Log.WithError(err).Error("Error saving session")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/admin/", http.StatusSeeOther)
			return
		}
		err := tmpl.Execute(w, map[string]string{"Error": "Invalid credentials"})
		if err != nil {
			return
		}
		return
	}
	err = tmpl.Execute(w, nil)
	if err != nil {
		return
	}
}

func isAuthenticated(r *http.Request) bool {
	session, err := store.Get(r, "admin-session")
	if err != nil {
		logger.Log.WithError(err).Error("Error retrieving session")
		return false
	}
	auth, ok := session.Values["authenticated"].(bool)
	return ok && auth
}

func checkCredentials(username, password string) bool {
	expectedUsername := os.Getenv("ADMIN_USERNAME")
	expectedPassword := os.Getenv("ADMIN_PASSWORD")
	return username == expectedUsername && password == expectedPassword
}

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "admin-session")
	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		http.Redirect(w, r, "/admin/login/", http.StatusSeeOther)
		return
	}

	logger.Log.Info("Dashboard handler called")

	httpError := tollbooth.LimitByRequest(statsLimiter, w, r)
	if httpError != nil {
		logger.Log.WithError(httpError).Error("Rate limit exceeded")
		http.Error(w, httpError.Message, httpError.StatusCode)
		return
	}

	stats := GetCachedStats()

	logger.Log.WithField("stats", stats).Info("Retrieved cached stats")

	tmpl, err := template.ParseFiles(filepath.Join(templatesDir, "dashboard.html"))
	if err != nil {
		logger.Log.WithError(err).Error("Failed to parse dashboard template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, stats)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to execute dashboard template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Log.Info("Dashboard rendered successfully")
}

func getStats() (Stats, error) {
	var stats Stats

	stats.TotalAccounts, _ = getTotalAccounts()
	stats.ActiveAccounts, _ = getActiveAccounts()
	stats.PermaBannedAccounts, _ = getPermaBannedAccounts()
	stats.ShadowBannedAccounts, _ = getShadowBannedAccounts()
	stats.TempBannedAccounts, _ = getTempBannedAccounts()
	stats.BannedAccounts = stats.PermaBannedAccounts + stats.ShadowBannedAccounts + stats.TempBannedAccounts
	stats.TotalUsers, _ = getTotalUsers()
	stats.ChecksLastHour, _ = getChecksInTimeRange(1 * time.Hour)
	stats.ChecksLast24Hours, _ = getChecksInTimeRange(24 * time.Hour)
	stats.TotalBans, _ = getTotalBans()
	stats.RecentBans, _ = getRecentBans(24 * time.Hour)
	stats.AverageChecksPerDay, _ = getAverageChecksPerDay()
	stats.TotalNotifications, _ = getTotalNotifications()
	stats.RecentNotifications, _ = getRecentNotifications(24 * time.Hour)
	stats.UsersWithCustomAPIKey, _ = getUsersWithCustomAPIKey()
	stats.AverageAccountsPerUser, _ = getAverageAccountsPerUser()
	stats.OldestAccount, stats.NewestAccount, _ = getAccountAgeRange()
	stats.TotalShadowbans, _ = getTotalBansByType(models.StatusShadowban)
	stats.TotalTempbans, _ = getTotalBansByType(models.StatusTempban)
	stats.BanDates, stats.BanCounts, _ = getBanDataForChart()
	stats.AccountStatsHistory, _ = getHistoricalData("accounts")
	stats.UserStatsHistory, _ = getHistoricalData("users")
	stats.CheckStatsHistory, _ = getHistoricalData("checks")
	stats.BanStatsHistory, _ = getHistoricalData("bans")
	stats.NotificationStatsHistory, _ = getHistoricalData("notifications")

	return stats, nil
}

func getTotalAccounts() (int, error) {
	var count int64
	err := database.DB.Model(&models.Account{}).Count(&count).Error
	return int(count), err
}

func getActiveAccounts() (int, error) {
	var count int64
	err := database.DB.Model(&models.Account{}).Where("is_permabanned = ? AND is_expired_cookie = ?", false, false).Count(&count).Error
	return int(count), err
}

func getPermaBannedAccounts() (int, error) {
	var count int64
	err := database.DB.Model(&models.Account{}).Where("is_permabanned = ?", true).Count(&count).Error
	return int(count), err
}

func getShadowBannedAccounts() (int, error) {
	var count int64
	err := database.DB.Model(&models.Account{}).Where("is_shadowbanned = ?", true).Count(&count).Error
	return int(count), err
}

func getTempBannedAccounts() (int, error) {
	var count int64
	err := database.DB.Model(&models.Account{}).Where("status = ?", models.StatusTempban).Count(&count).Error
	return int(count), err
}

func getTotalUsers() (int, error) {
	var count int64
	err := database.DB.Model(&models.UserSettings{}).Count(&count).Error
	return int(count), err
}

func getChecksInTimeRange(duration time.Duration) (int, error) {
	var count int64
	timeThreshold := time.Now().Add(-duration).Unix()
	err := database.DB.Model(&models.Account{}).Where("last_check > ?", timeThreshold).Count(&count).Error
	return int(count), err
}

func getTotalBans() (int, error) {
	var count int64
	err := database.DB.Model(&models.Ban{}).Count(&count).Error
	return int(count), err
}

func getRecentBans(duration time.Duration) (int, error) {
	var count int64
	err := database.DB.Model(&models.Ban{}).Where("created_at > ?", time.Now().Add(-duration)).Count(&count).Error
	return int(count), err
}

func getAverageChecksPerDay() (float64, error) {
	var result struct {
		AvgChecks float64
	}
	err := database.DB.Raw(`
        SELECT AVG(checks_per_day) AS avg_checks
        FROM (
            SELECT DATE(FROM_UNIXTIME(last_check)) AS date, COUNT(*) AS checks_per_day
            FROM accounts
            GROUP BY date
        ) subquery;
    `).Scan(&result).Error
	return result.AvgChecks, err
}

func getTotalNotifications() (int, error) {
	var count int64
	err := database.DB.Model(&models.Account{}).Where("last_notification > 0").Count(&count).Error
	return int(count), err
}

func getRecentNotifications(duration time.Duration) (int, error) {
	var count int64
	err := database.DB.Model(&models.Account{}).Where("last_notification > ?", time.Now().Add(-duration).Unix()).Count(&count).Error
	return int(count), err
}

func getUsersWithCustomAPIKey() (int, error) {
	var count int64
	err := database.DB.Model(&models.UserSettings{}).Where("captcha_api_key != ''").Count(&count).Error
	return int(count), err
}

func getAverageAccountsPerUser() (float64, error) {
	var result struct {
		AvgAccounts float64
	}
	err := database.DB.Raw(`
        SELECT AVG(accounts_per_user) AS avg_accounts
        FROM (
            SELECT user_id, COUNT(*) AS accounts_per_user
            FROM accounts
            GROUP BY user_id
        ) subquery;
    `).Scan(&result).Error
	return result.AvgAccounts, err
}

func getAccountAgeRange() (time.Time, time.Time, error) {
	var oldestAccount, newestAccount models.Account
	cutoffDate := time.Date(2003, 1, 1, 0, 0, 0, 0, time.UTC)
	err := database.DB.Where("created >= ?", cutoffDate).
		Order("created ASC").First(&oldestAccount).Error
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	err = database.DB.Where("created >= ?", cutoffDate).
		Order("created DESC").First(&newestAccount).Error
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return time.Unix(oldestAccount.Created, 0), time.Unix(newestAccount.Created, 0), err
}

func getTotalBansByType(banType models.Status) (int, error) {
	var count int64
	if banType == models.StatusShadowban {
		var shadowbanCount int64
		var currentlyShadowbannedCount int64

		if err := database.DB.Model(&models.Ban{}).Where("status = ?", banType).Count(&shadowbanCount).Error; err != nil {
			return 0, err
		}

		if err := database.DB.Model(&models.Account{}).Where("is_shadowbanned = ?", true).Count(&currentlyShadowbannedCount).Error; err != nil {
			return 0, err
		}

		return int(shadowbanCount), nil
	}

	err := database.DB.Model(&models.Ban{}).Where("status = ?", banType).Count(&count).Error
	return int(count), err
}

func getBanDataForChart() ([]string, []int, error) {
	var banData []struct {
		Date  time.Time
		Count int
		Type  string
	}

	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	err := database.DB.Model(&models.Ban{}).
		Select("DATE(created_at) as date, COUNT(*) as count, status as type").
		Where("created_at > ?", thirtyDaysAgo).
		Group("DATE(created_at), status").
		Order("date").
		Scan(&banData).Error

	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch ban data: %w", err)
	}

	dates := make(map[string]bool)
	shadowbanCounts := make(map[string]int)
	permaBanCounts := make(map[string]int)
	tempBanCounts := make(map[string]int)

	for _, data := range banData {
		dateStr := data.Date.Format("2006-01-02")
		dates[dateStr] = true
		switch data.Type {
		case string(models.StatusShadowban):
			shadowbanCounts[dateStr] += data.Count
		case string(models.StatusPermaban):
			permaBanCounts[dateStr] += data.Count
		case string(models.StatusTempban):
			tempBanCounts[dateStr] += data.Count
		}
	}

	var datesList []string
	var countsList []int

	for date := range dates {
		datesList = append(datesList, date)
		totalCount := shadowbanCounts[date] + permaBanCounts[date] + tempBanCounts[date]
		countsList = append(countsList, totalCount)
	}

	sort.Strings(datesList)

	return datesList, countsList, nil
}

func getHistoricalData(statType string) ([]HistoricalData, error) {
	var data []HistoricalData
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	switch statType {
	case "accounts":
		err := database.DB.Raw(`
            SELECT DATE(FROM_UNIXTIME(created)) as date,
                   COUNT(*) as value
            FROM accounts
            WHERE created > ?
            GROUP BY date
            ORDER BY date
        `, thirtyDaysAgo.Unix()).Scan(&data).Error
		return data, err

	case "users":
		err := database.DB.Raw(`
            SELECT DATE(created_at) as date,
                   COUNT(*) as value
            FROM user_settings
            WHERE created_at > ?
            GROUP BY date
            ORDER BY date
        `, thirtyDaysAgo).Scan(&data).Error
		return data, err

	case "checks":
		err := database.DB.Raw(`
            SELECT DATE(FROM_UNIXTIME(last_check)) as date,
                   COUNT(*) as value
            FROM accounts
            WHERE last_check > ?
            GROUP BY date
            ORDER BY date
        `, thirtyDaysAgo.Unix()).Scan(&data).Error
		return data, err

	case "bans":
		err := database.DB.Raw(`
            SELECT DATE(created_at) as date,
                   COUNT(*) as value
            FROM bans
            WHERE created_at > ?
            GROUP BY date
            ORDER BY date
        `, thirtyDaysAgo).Scan(&data).Error
		return data, err

	case "notifications":
		err := database.DB.Raw(`
            SELECT DATE(FROM_UNIXTIME(last_notification)) as date,
                   COUNT(*) as value
            FROM accounts
            WHERE last_notification > ?
            GROUP BY date
            ORDER BY date
        `, thirtyDaysAgo.Unix()).Scan(&data).Error
		return data, err

	default:
		return nil, fmt.Errorf("unknown stat type: %s", statType)
	}
}

func fetchDiscordStats() (DiscordStats, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", discordAPIBase+"/users/@me/guilds", nil)
	if err != nil {
		return DiscordStats{}, fmt.Errorf("failed to create request: %w", err)
	}

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		return DiscordStats{}, fmt.Errorf("DISCORD_TOKEN not set in environment")
	}

	req.Header.Set("Authorization", "Bot "+token)
	req.Header.Set("User-Agent", "CodStatusBot, v3.5)")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return DiscordStats{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Log.WithError(err).Error("Failed to close response body")
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return DiscordStats{}, fmt.Errorf("discord API returned status %d: %s", resp.StatusCode, string(body))
	}

	var guilds []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&guilds); err != nil {
		return DiscordStats{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return DiscordStats{
		ServerCount: len(guilds),
	}, nil
}
