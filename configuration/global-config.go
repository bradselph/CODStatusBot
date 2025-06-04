package configuration

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bradselph/CODStatusBot/logger"
)

type Config struct {
	// Admin API Endpoints
	Admin struct {
		Port           int
		APIKey         string
		Enabled        bool
		BasePath       string
		StatsRateLimit float64
		RetentionDays  int
	}

	Sharding struct {
		Enabled      bool
		ShardID      int
		TotalShards  int
		HeartbeatSec int
	}

	Performance struct {
		DbMaxIdleConns int `json:"db_max_idle_conns"`
		DbMaxOpenConns int `json:"db_max_open_conns"`
	}

	// Environment
	Environment string
	LogDir      string

	// Database Settings
	Database struct {
		User     string
		Password string
		Name     string
		Host     string
		Port     string
		Var      string
	}

	// Discord Settings
	Discord struct {
		Token       string
		DeveloperID string
		ClientID    string
		PublicKey   string
	}

	// Proxy Configuration
	Proxy struct {
		Enabled          bool
		Proxies          []string
		MaxFailures      int
		CooldownPeriod   time.Duration
		RefreshPeriod    time.Duration
		RotationStrategy string
		UserAgents       []string
	}

	// Rate Limiting Settings
	RateLimit struct {
		GlobalWindow time.Duration
		APIWindow    time.Duration
	}

	// Captcha Service Settings
	CaptchaService struct {
		Capsolver struct {
			Enabled       bool
			ClientKey     string
			AppID         string
			BalanceMin    float64
			MaxRetries    int
			RetryInterval time.Duration
		}
		EZCaptcha struct {
			Enabled    bool
			ClientKey  string
			AppID      string
			BalanceMin float64
		}
		TwoCaptcha struct {
			Enabled    bool
			ClientKey  string
			SoftID     string
			BalanceMin float64
		}
		RecaptchaSiteKey string
		RecaptchaURL     string
		MaxRetries       int
	}

	CaptchaEndpoints struct {
		Capsolver struct {
			Create   string
			Result   string
			Feedback string
		}
		EZCaptcha struct {
			Create string
			Result string
		}
		TwoCaptcha struct {
			Create string
			Result string
		}
		MaxRetries    int
		RetryInterval time.Duration
	}

	// API Endpoints
	API struct {
		CheckEndpoint           string
		ProfileEndpoint         string
		CheckVIPEndpoint        string
		DetailedProfileEndpoint string
		RedeemCodeEndpoint      string
	}

	// Rate Limits and Intervals
	RateLimits struct {
		CheckNow           time.Duration
		Default            time.Duration
		DefaultMaxAccounts int
		PremiumMaxAccounts int
	}

	// Intervals
	Intervals struct {
		Check              int
		Notification       float64
		Cooldown           float64
		Sleep              int
		PermaBanCheck      float64
		StatusChange       float64
		GlobalNotification float64
		CookieExpiration   float64
		TempBanUpdate      float64
	}

	// User Management Settings
	Users struct {
		MaxMessageFailures     int
		InactiveUserPeriod     time.Duration
		UnreachableResetPeriod time.Duration
		CleanupInterval        time.Duration
	}

	// Verdansk Stats Settings
	Verdansk struct {
		PreferencesEndpoint string
		StatsEndpoint       string
		APIKey              string
		TempDir             string
		CleanupTime         time.Duration
		CommandCooldown     time.Duration
		MaxRequestsPerDay   int
	}

	// Notification Settings
	Notifications struct {
		DefaultCooldown      time.Duration
		MaxPerHour           int
		MaxPerDay            int
		MinInterval          time.Duration
		BackoffBaseInterval  time.Duration
		BackoffMaxMultiplier float64
		BackoffHistoryWindow time.Duration
	}

	// Emoji Settings
	Emojis struct {
		CheckCircle    string
		BanCircle      string
		InfoCircle     string
		StopWatch      string
		QuestionCircle string
	}

	//TODO: Remove completely
	// Sentry Settings
	Sentry struct {
		DSN              string
		TracesSampleRate float64
		Debug            bool
	}

	// Components v2 Settings
	ComponentsV2 struct {
		Enabled bool
		Flag    int //TODO: Remove if possible
	}

	// Error Handling Settings
	ErrorHandling struct {
		MaxConsecutiveErrors           int
		CookieExpirationWarningHours   int
		AccountErrorThreshold          int
		ErrorNotificationCooldownHours int
	}

	// Startup Settings
	Startup struct {
		TimeoutSeconds      int
		ShutdownTimeout     time.Duration
		HealthCheckInterval time.Duration
	}

	// Message Settings
	Message struct {
		MaxLength        int
		MaxEmbedFields   int
		MaxComponentRows int
		EmbedDescLimit   int
	}

	// Fallback Settings
	Fallback struct {
		TimeoutSeconds    int
		RetryAttempts     int
		BackoffMultiplier float64
	}
}

var AppConfig Config

func Load() error {
	logger.Log.Info("Loading configuration...")

	AppConfig.Environment = getEnvWithDefault("ENVIRONMENT", "development")
	AppConfig.LogDir = getEnvWithDefault("LOG_DIR", "logs")

	requiredDbFields := map[string]*string{
		"DB_USER":     &AppConfig.Database.User,
		"DB_PASSWORD": &AppConfig.Database.Password,
		"DB_NAME":     &AppConfig.Database.Name,
		"DB_HOST":     &AppConfig.Database.Host,
		"DB_PORT":     &AppConfig.Database.Port,
		"DB_VAR":      &AppConfig.Database.Var,
	}

	for env, field := range requiredDbFields {
		*field = os.Getenv(env)
	}

	AppConfig.Discord.Token = os.Getenv("DISCORD_TOKEN")
	AppConfig.Discord.DeveloperID = os.Getenv("DEVELOPER_ID")
	AppConfig.Discord.ClientID = os.Getenv("DISCORD_CLIENT_ID")
	AppConfig.Discord.PublicKey = os.Getenv("DISCORD_PUBLIC_KEY")

	loadStartupConfig()
	loadAdminConfig()
	loadCaptchaConfig()
	loadAPIEndpoints()
	loadUserSettings()
	loadNotificationSettings()
	loadRateLimits()
	loadIntervals()
	loadEmojiConfig()
	loadPerformanceConfig()
	loadVerdanskConfig()
	loadShardingConfig()
	loadProxyConfig()
	loadRateLimitConfig()
	loadSentryConfig() //TODO: Remove if possible
	loadComponentsV2Config()
	loadErrorHandlingConfig()
	loadMessageConfig()
	loadFallbackConfig()

	if err := validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	logConfigurationValues()

	return nil
}

func loadAdminConfig() {
	AppConfig.Admin.Enabled = getEnvAsBool("ADMIN_API_ENABLED", true)
	AppConfig.Admin.Port = getEnvAsInt("ADMIN_PORT", 8080)
	AppConfig.Admin.APIKey = os.Getenv("ADMIN_API_KEY")
	AppConfig.Admin.BasePath = getEnvWithDefault("ADMIN_API_BASE_PATH", "/api")
	AppConfig.Admin.StatsRateLimit = getEnvAsFloat("ADMIN_STATS_RATE_LIMIT", 25.0)
	AppConfig.Admin.RetentionDays = getEnvAsInt("ANALYTICS_RETENTION_DAYS", 90)
}

func loadUserSettings() {
	AppConfig.Users.MaxMessageFailures = getEnvAsInt("MAX_MESSAGE_FAILURES", 25)
	inactiveDays := getEnvAsInt("INACTIVE_USER_DAYS", 90)
	AppConfig.Users.InactiveUserPeriod = time.Duration(inactiveDays) * 24 * time.Hour
	unreachableDays := getEnvAsInt("UNREACHABLE_RESET_DAYS", 30)
	AppConfig.Users.UnreachableResetPeriod = time.Duration(unreachableDays) * 24 * time.Hour
	AppConfig.Users.CleanupInterval = time.Duration(getEnvAsInt("USER_CLEANUP_INTERVAL_HOURS", 24)) * time.Hour
}

func loadNotificationSettings() {
	cooldownMinutes := getEnvAsInt("NOTIFICATION_DEFAULT_COOLDOWN_MINUTES", 60)
	AppConfig.Notifications.DefaultCooldown = time.Duration(cooldownMinutes) * time.Minute
	AppConfig.Notifications.MaxPerHour = getEnvAsInt("NOTIFICATION_MAX_PER_HOUR", 4)
	AppConfig.Notifications.MaxPerDay = getEnvAsInt("NOTIFICATION_MAX_PER_DAY", 10)
	minIntervalMinutes := getEnvAsInt("NOTIFICATION_MIN_INTERVAL_MINUTES", 5)
	AppConfig.Notifications.MinInterval = time.Duration(minIntervalMinutes) * time.Minute
	baseIntervalMinutes := getEnvAsInt("NOTIFICATION_BACKOFF_BASE_MINUTES", 5)
	AppConfig.Notifications.BackoffBaseInterval = time.Duration(baseIntervalMinutes) * time.Minute
	AppConfig.Notifications.BackoffMaxMultiplier = getEnvAsFloat("NOTIFICATION_BACKOFF_MAX_MULTIPLIER", 6.0)
	historyHours := getEnvAsInt("NOTIFICATION_HISTORY_WINDOW_HOURS", 24)
	AppConfig.Notifications.BackoffHistoryWindow = time.Duration(historyHours) * time.Hour
}

func loadCaptchaConfig() {
	// Capsolver
	AppConfig.CaptchaService.Capsolver.Enabled = os.Getenv("CAPSOLVER_ENABLED") == "true"
	AppConfig.CaptchaService.Capsolver.ClientKey = os.Getenv("CAPSOLVER_CLIENT_KEY")
	AppConfig.CaptchaService.Capsolver.AppID = os.Getenv("CAPSOLVER_APP_ID")
	AppConfig.CaptchaService.Capsolver.BalanceMin = getEnvAsFloat("CAPSOLVER_BALANCE_MIN", 0.10)
	AppConfig.CaptchaService.Capsolver.MaxRetries = getEnvAsInt("CAPSOLVER_MAX_RETRIES", 6)
	AppConfig.CaptchaService.Capsolver.RetryInterval = time.Duration(getEnvAsInt("CAPSOLVER_RETRY_INTERVAL", 10)) * time.Second

	// EZCaptcha
	AppConfig.CaptchaService.EZCaptcha.Enabled = os.Getenv("EZCAPTCHA_ENABLED") == "true"
	AppConfig.CaptchaService.EZCaptcha.ClientKey = os.Getenv("EZCAPTCHA_CLIENT_KEY")
	AppConfig.CaptchaService.EZCaptcha.AppID = os.Getenv("EZAPPID")
	AppConfig.CaptchaService.EZCaptcha.BalanceMin = getEnvAsFloat("EZCAPBALMIN", 50)

	// 2Captcha
	AppConfig.CaptchaService.TwoCaptcha.Enabled = os.Getenv("TWOCAPTCHA_ENABLED") == "true"
	AppConfig.CaptchaService.TwoCaptcha.SoftID = os.Getenv("SOFT_ID")
	AppConfig.CaptchaService.TwoCaptcha.BalanceMin = getEnvAsFloat("TWOCAPBALMIN", 0.10)

	// Common Captcha Settings
	AppConfig.CaptchaService.RecaptchaSiteKey = os.Getenv("RECAPTCHA_SITE_KEY")
	AppConfig.CaptchaService.RecaptchaURL = os.Getenv("RECAPTCHA_URL")
	AppConfig.CaptchaService.MaxRetries = getEnvAsInt("MAX_RETRIES", 3)

	// Captcha Endpoints
	AppConfig.CaptchaEndpoints.Capsolver.Create = getEnvWithDefault("CAPSOLVER_CREATE_ENDPOINT", "https://api.capsolver.com/createTask")
	AppConfig.CaptchaEndpoints.Capsolver.Result = getEnvWithDefault("CAPSOLVER_RESULT_ENDPOINT", "https://api.capsolver.com/getTaskResult")
	AppConfig.CaptchaEndpoints.Capsolver.Feedback = getEnvWithDefault("CAPSOLVER_FEEDBACK_ENDPOINT", "https://api.capsolver.com/feedbackTask")

	AppConfig.CaptchaEndpoints.EZCaptcha.Create = getEnvWithDefault("EZCAPTCHA_CREATE_ENDPOINT", "https://api.ez-captcha.com/createTask")
	AppConfig.CaptchaEndpoints.EZCaptcha.Result = getEnvWithDefault("EZCAPTCHA_RESULT_ENDPOINT", "https://api.ez-captcha.com/getTaskResult")

	AppConfig.CaptchaEndpoints.TwoCaptcha.Create = getEnvWithDefault("TWOCAPTCHA_CREATE_ENDPOINT", "https://api.2captcha.com/createTask")
	AppConfig.CaptchaEndpoints.TwoCaptcha.Result = getEnvWithDefault("TWOCAPTCHA_RESULT_ENDPOINT", "https://api.2captcha.com/getTaskResult")

	AppConfig.CaptchaEndpoints.MaxRetries = getEnvAsInt("CAPTCHA_ENDPOINT_MAX_RETRIES", 6)
	AppConfig.CaptchaEndpoints.RetryInterval = time.Duration(getEnvAsInt("CAPTCHA_ENDPOINT_RETRY_INTERVAL", 10)) * time.Second
}

func loadAPIEndpoints() {
	AppConfig.API.CheckEndpoint = os.Getenv("CHECK_ENDPOINT")
	AppConfig.API.ProfileEndpoint = os.Getenv("PROFILE_ENDPOINT")
	AppConfig.API.CheckVIPEndpoint = os.Getenv("CHECK_VIP_ENDPOINT")
	AppConfig.API.DetailedProfileEndpoint = os.Getenv("DETAILED_PROFILE_ENDPOINT")
	AppConfig.API.RedeemCodeEndpoint = os.Getenv("REDEEM_CODE_ENDPOINT")
}

func loadRateLimits() {
	AppConfig.RateLimits.CheckNow = time.Duration(getEnvAsInt("CHECK_NOW_RATE_LIMIT", 3600)) * time.Second
	AppConfig.RateLimits.Default = time.Duration(getEnvAsInt("DEFAULT_RATE_LIMIT", 180)) * time.Minute
	AppConfig.RateLimits.DefaultMaxAccounts = getEnvAsInt("DEFAULT_USER_MAXACCOUNTS", 3)
	AppConfig.RateLimits.PremiumMaxAccounts = getEnvAsInt("PREM_USER_MAXACCOUNTS", 15)
}

func loadIntervals() {
	AppConfig.Intervals.Check = getEnvAsInt("CHECK_INTERVAL", 15)
	AppConfig.Intervals.Notification = getEnvAsFloat("NOTIFICATION_INTERVAL", 24)
	AppConfig.Intervals.Cooldown = getEnvAsFloat("COOLDOWN_DURATION", 6)
	AppConfig.Intervals.Sleep = getEnvAsInt("SLEEP_DURATION", 1)
	AppConfig.Intervals.PermaBanCheck = getEnvAsFloat("COOKIE_CHECK_INTERVAL_PERMABAN", 24)
	AppConfig.Intervals.StatusChange = getEnvAsFloat("STATUS_CHANGE_COOLDOWN", 1)
	AppConfig.Intervals.GlobalNotification = getEnvAsFloat("GLOBAL_NOTIFICATION_COOLDOWN", 2)
	AppConfig.Intervals.CookieExpiration = getEnvAsFloat("COOKIE_EXPIRATION_WARNING", 24)
	AppConfig.Intervals.TempBanUpdate = getEnvAsFloat("TEMP_BAN_UPDATE_INTERVAL", 24)
}

func loadEmojiConfig() {
	AppConfig.Emojis.CheckCircle = os.Getenv("CHECKCIRCLE")
	AppConfig.Emojis.BanCircle = os.Getenv("BANCIRCLE")
	AppConfig.Emojis.InfoCircle = os.Getenv("INFOCIRCLE")
	AppConfig.Emojis.StopWatch = os.Getenv("STOPWATCH")
	AppConfig.Emojis.QuestionCircle = os.Getenv("QUESTIONCIRCLE")
}

func loadPerformanceConfig() {
	AppConfig.Performance.DbMaxIdleConns = getEnvAsInt("DB_MAX_IDLE_CONNS", 10)
	AppConfig.Performance.DbMaxOpenConns = getEnvAsInt("DB_MAX_OPEN_CONNS", 100)
}

func loadProxyConfig() {
	proxyList := os.Getenv("HTTP_PROXIES")
	if proxyList != "" {
		AppConfig.Proxy.Proxies = strings.Split(proxyList, ",")
		for i, proxy := range AppConfig.Proxy.Proxies {
			AppConfig.Proxy.Proxies[i] = strings.TrimSpace(proxy)
		}
		AppConfig.Proxy.Enabled = true
	} else {
		AppConfig.Proxy.Enabled = false
	}

	AppConfig.Proxy.MaxFailures = getEnvAsInt("PROXY_MAX_FAILURES", 3)
	AppConfig.Proxy.CooldownPeriod = time.Duration(getEnvAsInt("PROXY_COOLDOWN_MINUTES", 10)) * time.Minute
	AppConfig.Proxy.RefreshPeriod = time.Duration(getEnvAsInt("PROXY_REFRESH_MINUTES", 30)) * time.Minute
	AppConfig.Proxy.RotationStrategy = getEnvWithDefault("PROXY_ROTATION_STRATEGY", "least-used")

	userAgentList := os.Getenv("USER_AGENTS")
	if userAgentList != "" {
		AppConfig.Proxy.UserAgents = strings.Split(userAgentList, ",")
		for i, ua := range AppConfig.Proxy.UserAgents {
			AppConfig.Proxy.UserAgents[i] = strings.TrimSpace(ua)
		}
	} else {
		AppConfig.Proxy.UserAgents = []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36 Edg/133.0.2623.71",
		}
	}
}

func loadRateLimitConfig() {
	AppConfig.RateLimit.GlobalWindow = time.Duration(getEnvAsInt("GLOBAL_RATE_LIMIT_WINDOW", 60)) * time.Second
	AppConfig.RateLimit.APIWindow = time.Duration(getEnvAsInt("API_RATE_LIMIT_WINDOW", 10)) * time.Second
}

func loadSentryConfig() {
	AppConfig.Sentry.DSN = os.Getenv("SENTRY_DSN")
	AppConfig.Sentry.TracesSampleRate = getEnvAsFloat("SENTRY_TRACES_SAMPLE_RATE", 1.0)
	AppConfig.Sentry.Debug = getEnvAsBool("SENTRY_DEBUG", false)
}

func validate() error {
	var missingVars []string

	requiredVars := map[string]string{
		"DISCORD_TOKEN":      AppConfig.Discord.Token,
		"DEVELOPER_ID":       AppConfig.Discord.DeveloperID,
		"DB_USER":            AppConfig.Database.User,
		"DB_PASSWORD":        AppConfig.Database.Password,
		"DB_NAME":            AppConfig.Database.Name,
		"DB_HOST":            AppConfig.Database.Host,
		"DB_PORT":            AppConfig.Database.Port,
		"DB_VAR":             AppConfig.Database.Var,
		"PROFILE_ENDPOINT":   AppConfig.API.ProfileEndpoint,
		"CHECK_VIP_ENDPOINT": AppConfig.API.CheckVIPEndpoint,
		"CHECK_ENDPOINT":     AppConfig.API.CheckEndpoint,
	}

	for key, value := range requiredVars {
		if value == "" {
			missingVars = append(missingVars, key)
		}
	}

	if len(missingVars) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missingVars)
	}

	if AppConfig.Discord.ClientID == "" {
		logger.Log.Warn("DISCORD_CLIENT_ID not set")
	}

	if !AppConfig.CaptchaService.Capsolver.Enabled &&
		!AppConfig.CaptchaService.EZCaptcha.Enabled &&
		!AppConfig.CaptchaService.TwoCaptcha.Enabled {
		logger.Log.Warn("No captcha services are enabled - functionality will be limited")
	}

	if AppConfig.CaptchaService.Capsolver.Enabled && AppConfig.CaptchaService.Capsolver.ClientKey == "" {
		return fmt.Errorf("capsolver is enabled but no client key provided")
	}
	if AppConfig.CaptchaService.EZCaptcha.Enabled && AppConfig.CaptchaService.EZCaptcha.ClientKey == "" {
		return fmt.Errorf("EZCaptcha is enabled but no client key provided")
	}
	if AppConfig.CaptchaService.TwoCaptcha.Enabled && AppConfig.CaptchaService.TwoCaptcha.ClientKey == "" {
		return fmt.Errorf("2Captcha is enabled but no client key provided")
	}
	return nil
}

func logConfigurationValues() {
	logger.Log.Infof("Loaded configuration - Environment: %s", AppConfig.Environment)
	logger.Log.Infof("Loaded rate limits and intervals: CHECK_INTERVAL=%d, NOTIFICATION_INTERVAL=%.2f, "+
		"COOLDOWN_DURATION=%.2f, SLEEP_DURATION=%d, COOKIE_CHECK_INTERVAL_PERMABAN=%.2f, "+
		"STATUS_CHANGE_COOLDOWN=%.2f, GLOBAL_NOTIFICATION_COOLDOWN=%.2f, COOKIE_EXPIRATION_WARNING=%.2f, "+
		"TEMP_BAN_UPDATE_INTERVAL=%.2f, CHECK_NOW_RATE_LIMIT=%v, DEFAULT_RATE_LIMIT=%v",
		AppConfig.Intervals.Check,
		AppConfig.Intervals.Notification,
		AppConfig.Intervals.Cooldown,
		AppConfig.Intervals.Sleep,
		AppConfig.Intervals.PermaBanCheck,
		AppConfig.Intervals.StatusChange,
		AppConfig.Intervals.GlobalNotification,
		AppConfig.Intervals.CookieExpiration,
		AppConfig.Intervals.TempBanUpdate,
		AppConfig.RateLimits.CheckNow,
		AppConfig.RateLimits.Default)

	// Log user management settings
	logger.Log.Infof("Loaded user management settings: MAX_MESSAGE_FAILURES=%d, INACTIVE_USER_DAYS=%.2f, "+
		"UNREACHABLE_RESET_DAYS=%.2f",
		AppConfig.Users.MaxMessageFailures,
		AppConfig.Users.InactiveUserPeriod.Hours()/24,
		AppConfig.Users.UnreachableResetPeriod.Hours()/24)

	// Log notification settings
	logger.Log.Infof("Loaded notification settings: DEFAULT_COOLDOWN=%v, MAX_PER_HOUR=%d, "+
		"MAX_PER_DAY=%d, MIN_INTERVAL=%v",
		AppConfig.Notifications.DefaultCooldown,
		AppConfig.Notifications.MaxPerHour,
		AppConfig.Notifications.MaxPerDay,
		AppConfig.Notifications.MinInterval)

	// Log admin API settings
	logger.Log.Infof("Loaded admin API settings: ENABLED=%v, PORT=%d, STATS_RATE_LIMIT=%.2f, "+
		"RETENTION_DAYS=%d",
		AppConfig.Admin.Enabled,
		AppConfig.Admin.Port,
		AppConfig.Admin.StatsRateLimit,
		AppConfig.Admin.RetentionDays)

	// Log proxy settings
	if AppConfig.Proxy.Enabled {
		logger.Log.Infof("Loaded proxy settings: ENABLED=%v, PROXIES=%d, STRATEGY=%s, MAX_FAILURES=%d",
			AppConfig.Proxy.Enabled,
			len(AppConfig.Proxy.Proxies),
			AppConfig.Proxy.RotationStrategy,
			AppConfig.Proxy.MaxFailures)
	} else {
		logger.Log.Info("Proxy configuration: DISABLED")
	}

	// Log enabled captcha services
	var enabledServices []string
	if AppConfig.CaptchaService.Capsolver.Enabled {
		enabledServices = append(enabledServices, "Capsolver")
	}
	if AppConfig.CaptchaService.EZCaptcha.Enabled {
		enabledServices = append(enabledServices, "EZCaptcha")
	}
	if AppConfig.CaptchaService.TwoCaptcha.Enabled {
		enabledServices = append(enabledServices, "2Captcha")
	}

	if len(enabledServices) > 0 {
		logger.Log.Infof("Enabled captcha services: %s", strings.Join(enabledServices, ", "))
	} else {
		logger.Log.Warn("No captcha services are enabled")
	}

	if AppConfig.Discord.ClientID != "" {
		logger.Log.Info("OAuth2 configuration loaded successfully")
	}

	if AppConfig.Sharding.Enabled {
		logger.Log.Infof("Sharding configuration: ENABLED - Shard %d of %d", AppConfig.Sharding.ShardID, AppConfig.Sharding.TotalShards)
	} else {
		logger.Log.Info("Sharding configuration: DISABLED")
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
		logger.Log.WithField("key", key).WithField("default", defaultValue).
			Error("Failed to parse integer from environment variable")
	}
	return defaultValue
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
		logger.Log.WithField("key", key).WithField("default", defaultValue).
			Error("Failed to parse float from environment variable")
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultValue
}

func Get() *Config {
	return &AppConfig
}

func GetDefaultSettings() struct {
	CheckInterval        int
	NotificationInterval float64
	CooldownDuration     float64
	StatusChangeCooldown float64
} {
	return struct {
		CheckInterval        int
		NotificationInterval float64
		CooldownDuration     float64
		StatusChangeCooldown float64
	}{
		CheckInterval:        AppConfig.Intervals.Check,
		NotificationInterval: AppConfig.Intervals.Notification,
		CooldownDuration:     AppConfig.Intervals.Cooldown,
		StatusChangeCooldown: AppConfig.Intervals.StatusChange,
	}
}

func loadVerdanskConfig() {
	AppConfig.Verdansk.PreferencesEndpoint = getEnvWithDefault("VERDANSK_PREFERENCES", "https://pd.callofduty.com/api/x/v1/campaign/warzonewrapped/preferences/gamer/{encodedGamerTag}")
	AppConfig.Verdansk.StatsEndpoint = getEnvWithDefault("VERDANSK_STATS", "https://pd.callofduty.com/api/x/v1/campaign/warzonewrapped/stats/gamer/{encodedGamerTag}")
	AppConfig.Verdansk.APIKey = getEnvWithDefault("X_API_KEY", "a855a770-cf8a-4ae8-9f30-b787d676e608")
	AppConfig.Verdansk.TempDir = getEnvWithDefault("VERDANSK_TEMP_DIR", "verdansk_temp")
	AppConfig.Verdansk.CleanupTime = time.Duration(getEnvAsInt("VERDANSK_CLEANUP_MINUTES", 30)) * time.Minute
	AppConfig.Verdansk.CommandCooldown = time.Duration(getEnvAsInt("VERDANSK_COMMAND_COOLDOWN_MINUTES", 60)) * time.Minute
	AppConfig.Verdansk.MaxRequestsPerDay = getEnvAsInt("VERDANSK_MAX_REQUESTS_PER_DAY", 3)
}

func loadShardingConfig() {
	AppConfig.Sharding.Enabled = getEnvAsBool("SHARDING_ENABLED", false)
	AppConfig.Sharding.ShardID = getEnvAsInt("SHARD_ID", 0)
	AppConfig.Sharding.TotalShards = getEnvAsInt("TOTAL_SHARDS", 1)
	AppConfig.Sharding.HeartbeatSec = getEnvAsInt("SHARD_HEARTBEAT_SEC", 30)

	if AppConfig.Sharding.Enabled {
		if AppConfig.Sharding.TotalShards < 1 {
			logger.Log.Warn("Invalid TOTAL_SHARDS value, must be at least 1. Setting to 1.")
			AppConfig.Sharding.TotalShards = 1
		}

		if AppConfig.Sharding.ShardID < 0 || AppConfig.Sharding.ShardID >= AppConfig.Sharding.TotalShards {
			logger.Log.Warnf("Invalid SHARD_ID %d for TOTAL_SHARDS %d. Setting to 0.",
				AppConfig.Sharding.ShardID, AppConfig.Sharding.TotalShards)
			AppConfig.Sharding.ShardID = 0
		}

		logger.Log.Infof("Sharding enabled: This is shard %d of %d",
			AppConfig.Sharding.ShardID, AppConfig.Sharding.TotalShards)
	} else {
		logger.Log.Info("Sharding disabled: Running in single instance mode")
	}
}

func loadComponentsV2Config() {
	AppConfig.ComponentsV2.Enabled = getEnvAsBool("COMPONENTS_V2_ENABLED", true)
	AppConfig.ComponentsV2.Flag = getEnvAsInt("COMPONENTS_V2_FLAG", 32768) //TODO: Remove if possible
}

func loadErrorHandlingConfig() {
	AppConfig.ErrorHandling.MaxConsecutiveErrors = getEnvAsInt("MAX_CONSECUTIVE_ERRORS", 25)
	AppConfig.ErrorHandling.CookieExpirationWarningHours = getEnvAsInt("COOKIE_EXPIRATION_WARNING_HOURS", 24)
	AppConfig.ErrorHandling.AccountErrorThreshold = getEnvAsInt("ACCOUNT_ERROR_THRESHOLD", 50)
	AppConfig.ErrorHandling.ErrorNotificationCooldownHours = getEnvAsInt("ERROR_NOTIFICATION_COOLDOWN_HOURS", 6)
}

func loadStartupConfig() {
	AppConfig.Startup.TimeoutSeconds = getEnvAsInt("STARTUP_TIMEOUT_SECONDS", 30)
	shutdownTimeout := getEnvAsInt("SHUTDOWN_TIMEOUT_SECONDS", 15)
	AppConfig.Startup.ShutdownTimeout = time.Duration(shutdownTimeout) * time.Second
	healthCheckInterval := getEnvAsInt("HEALTH_CHECK_INTERVAL_SECONDS", 60)
	AppConfig.Startup.HealthCheckInterval = time.Duration(healthCheckInterval) * time.Second
}

func loadMessageConfig() {
	AppConfig.Message.MaxLength = getEnvAsInt("MAX_MESSAGE_LENGTH", 2000)
	AppConfig.Message.MaxEmbedFields = getEnvAsInt("MAX_EMBED_FIELDS", 25)
	AppConfig.Message.MaxComponentRows = getEnvAsInt("MAX_COMPONENT_ROWS", 5)
	AppConfig.Message.EmbedDescLimit = getEnvAsInt("EMBED_DESCRIPTION_LIMIT", 4096)
}

func loadFallbackConfig() {
	AppConfig.Fallback.TimeoutSeconds = getEnvAsInt("FALLBACK_TIMEOUT_SECONDS", 30)
	AppConfig.Fallback.RetryAttempts = getEnvAsInt("FALLBACK_RETRY_ATTEMPTS", 3)
	AppConfig.Fallback.BackoffMultiplier = getEnvAsFloat("FALLBACK_BACKOFF_MULTIPLIER", 2.0)
}
