package admin

import (
	"encoding/json"
	"github.com/joho/godotenv"
	"html/template"
	"net/http"
	"os"
	"sync"
	"time"

	"CODStatusBot/database"
	"CODStatusBot/logger"
	"CODStatusBot/models"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/gorilla/sessions"
)

var (
	statsLimiter      *limiter.Limiter
	cachedStats       Stats
	cachedStatsLock   sync.RWMutex
	cacheInterval     = 15 * time.Minute
	NotificationMutex sync.Mutex
	store             *sessions.CookieStore
)

type Stats struct {
	TotalAccounts            int
	ActiveAccounts           int
	PermaBannedAccounts      int
	ShadowBannedAccounts     int
	TotalUsers               int
	ChecksLastHour           int
	ChecksLast24Hours        int
	TotalBans                int
	RecentBans               int
	AverageChecksPerDay      float64
	TotalNotifications       int
	RecentNotifications      int
	UsersWithCustomAPIKey    int
	AverageAccountsPerUser   float64
	OldestAccount            time.Time
	NewestAccount            time.Time
	TotalShadowbans          int
	TotalTempbans            int
	BanDates                 []string         `json:"banDates"`
	BanCounts                []int            `json:"banCounts"`
	AccountStatsHistory      []HistoricalData `json:"accountStatsHistory"`
	UserStatsHistory         []HistoricalData `json:"userStatsHistory"`
	CheckStatsHistory        []HistoricalData `json:"checkStatsHistory"`
	BanStatsHistory          []HistoricalData `json:"banStatsHistory"`
	NotificationStatsHistory []HistoricalData `json:"notificationStatsHistory"`
}

type HistoricalData struct {
	Date  string `json:"date"`
	Value int    `json:"value"`
}

func init() {
	if err := godotenv.Load(); err != nil {
		logger.Log.WithError(err).Error("Error loading .env file")
	}

	statsLimiter = tollbooth.NewLimiter(1, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour})
	sessionKey := os.Getenv("SESSION_KEY")
	if sessionKey == "" {
		logger.Log.Error("SESSION_KEY not set in environment variables")
	}
	store = sessions.NewCookieStore([]byte(sessionKey))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   false,
	}
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "admin-session")
		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
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

	cachedStatsLock.Lock()
	cachedStats = stats
	cachedStatsLock.Unlock()
}

func GetCachedStats() Stats {
	cachedStatsLock.RLock()
	defer cachedStatsLock.RUnlock()
	return cachedStats
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
	err := json.NewEncoder(w).Encode(stats)
	if err != nil {
		return
	}
}

func TermsHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/terms.html")
	if err != nil {
		logger.Log.WithError(err).Error("Failed to parse terms template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func PolicyHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/policy.html")
	if err != nil {
		logger.Log.WithError(err).Error("Failed to parse policy template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		logger.Log.WithError(err).Error("Failed to parse index template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func HelpHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/help.html")
	if err != nil {
		logger.Log.WithError(err).Error("Failed to parse help template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "admin-session")
	session.Values["authenticated"] = false
	session.Save(r, w)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/login.html")
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
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}
		tmpl.Execute(w, map[string]string{"Error": "Invalid credentials"})
		return
	}
	tmpl.Execute(w, nil)
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
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
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

	tmpl, err := template.ParseFiles("templates/dashboard.html")
	if err != nil {
		logger.Log.WithError(err).Error("Failed to parse dashboard template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, stats)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to execute dashboard template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	logger.Log.Info("Dashboard rendered successfully")
}

func getStats() (Stats, error) {
	var stats Stats

	stats.TotalAccounts, _ = getTotalAccounts()
	stats.ActiveAccounts, _ = getActiveAccounts()
	stats.PermaBannedAccounts, _ = getPermaBannedAccounts()
	stats.ShadowBannedAccounts, _ = getShadowBannedAccounts()
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

// TODO ensure TotalBans is not counting duplicates.
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
	err := database.DB.Model(&models.Ban{}).Where("status = ?", banType).Count(&count).Error
	return int(count), err
}

func getBanDataForChart() ([]string, []int, error) {
	var banData []struct {
		Date  time.Time
		Count int
	}
	err := database.DB.Model(&models.Ban{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at > ?", time.Now().AddDate(0, 0, -30)).
		Group("DATE(created_at)").
		Order("date").
		Scan(&banData).Error
	if err != nil {
		return nil, nil, err
	}

	var dates []string
	var counts []int
	for _, data := range banData {
		dates = append(dates, data.Date.Format("2006-01-02"))
		counts = append(counts, data.Count)
	}

	return dates, counts, nil
}

func getHistoricalData(statType string) ([]HistoricalData, error) {
	var data []HistoricalData
	var err error

	switch statType {
	case "accounts":
		err = database.DB.Raw(`
			SELECT DATE(FROM_UNIXTIME(created)) as date, COUNT(*) as value
			FROM accounts
			WHERE created > ?
			GROUP BY date
			ORDER BY date ASC
		`, time.Now().AddDate(0, 0, -30).Unix()).Scan(&data).Error

	case "users":
		err = database.DB.Raw(`
			SELECT DATE(created_at) as date, COUNT(*) as value
			FROM user_settings
			WHERE created_at > ?
			GROUP BY date
			ORDER BY date ASC
		`, time.Now().AddDate(0, 0, -30)).Scan(&data).Error

	case "checks":
		err = database.DB.Raw(`
			SELECT DATE(FROM_UNIXTIME(last_check)) as date, COUNT(*) as value
			FROM accounts
			WHERE last_check > ?
			GROUP BY date
			ORDER BY date ASC
		`, time.Now().AddDate(0, 0, -30).Unix()).Scan(&data).Error

	case "bans":
		err = database.DB.Raw(`
			SELECT DATE(created_at) as date, COUNT(*) as value
			FROM bans
			WHERE created_at > ?
			GROUP BY date
			ORDER BY date ASC
		`, time.Now().AddDate(0, 0, -30)).Scan(&data).Error

	case "notifications":
		err = database.DB.Raw(`
			SELECT DATE(FROM_UNIXTIME(last_notification)) as date, COUNT(*) as value
			FROM accounts
			WHERE last_notification > ?
			GROUP BY date
			ORDER BY date ASC
		`, time.Now().AddDate(0, 0, -30).Unix()).Scan(&data).Error

	default:
		return nil, nil
	}

	return data, err
}
