package services

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
)

type ProxyManager struct {
	sync.RWMutex
	Proxies            []string        // List of available proxies in format http://user:pass@host:port
	ActiveProxies      map[string]bool // Map of active proxies
	ProxyFailures      map[string]int  // Count of consecutive failures per proxy
	ProxyLastUsed      map[string]time.Time
	ProxyRateLimits    map[string]time.Time
	DefaultClient      *http.Client
	ProxyClients       map[string]*http.Client
	ProxyEnabled       bool
	RotationStrategy   string // "round-robin", "random", "least-used"
	CurrentProxyIndex  int
	MaxFailures        int
	CooldownPeriod     time.Duration
	ProxyRefreshPeriod time.Duration
	LastRefresh        time.Time
	UserAgents         []string
}

var (
	proxyManager      *ProxyManager
	proxyManagerMutex sync.RWMutex
)

func GetProxyManager() *ProxyManager {
	proxyManagerMutex.Lock()
	defer proxyManagerMutex.Unlock()

	if proxyManager == nil {
		proxyManager = initializeProxyManager()
	}

	return proxyManager
}

func initializeProxyManager() *ProxyManager {
	cfg := configuration.Get()

	pm := &ProxyManager{
		ActiveProxies:      make(map[string]bool),
		ProxyFailures:      make(map[string]int),
		ProxyLastUsed:      make(map[string]time.Time),
		ProxyRateLimits:    make(map[string]time.Time),
		ProxyClients:       make(map[string]*http.Client),
		ProxyEnabled:       cfg.Proxy.Enabled,
		RotationStrategy:   cfg.Proxy.RotationStrategy,
		CurrentProxyIndex:  0,
		MaxFailures:        cfg.Proxy.MaxFailures,
		CooldownPeriod:     cfg.Proxy.CooldownPeriod,
		ProxyRefreshPeriod: cfg.Proxy.RefreshPeriod,
		LastRefresh:        time.Now(),
		UserAgents:         cfg.Proxy.UserAgents,
	}

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
		ForceAttemptHTTP2:   true,
		MaxConnsPerHost:     0,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	pm.DefaultClient = &http.Client{
		Timeout:   30 * time.Second,
		Transport: NewHeaderTransport(transport, pm.UserAgents),
	}

	LoadProxyConfiguration(pm)

	return pm
}

func LoadProxyConfiguration(pm *ProxyManager) {
	cfg := configuration.Get()

	if len(cfg.Proxy.Proxies) > 0 {
		pm.Proxies = cfg.Proxy.Proxies
		for _, proxy := range pm.Proxies {
			pm.ActiveProxies[proxy] = true
		}
		pm.ProxyEnabled = true
		logger.Log.Infof("Loaded %d proxies from configuration", len(pm.Proxies))
	} else {
		pm.ProxyEnabled = false
		logger.Log.Info("No proxies configured, using direct connection")
	}

	if len(cfg.Proxy.UserAgents) > 0 {
		pm.UserAgents = cfg.Proxy.UserAgents
		logger.Log.Infof("Loaded %d user agents", len(pm.UserAgents))
	}

	for _, proxy := range pm.Proxies {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			logger.Log.WithError(err).Errorf("Invalid proxy URL: %s", proxy)
			continue
		}

		proxyTransport := &http.Transport{
			Proxy:               http.ProxyURL(proxyURL),
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   false,
			ForceAttemptHTTP2:   true,
			MaxConnsPerHost:     0,
			TLSHandshakeTimeout: 10 * time.Second,
		}

		pm.ProxyClients[proxy] = &http.Client{
			Timeout:   30 * time.Second,
			Transport: NewHeaderTransport(proxyTransport, pm.UserAgents),
		}
	}

	if pm.ProxyEnabled {
		go pm.initializeProxyStats()
	}
}

func (pm *ProxyManager) initializeProxyStats() {
	for proxy := range pm.ActiveProxies {
		maskedProxy := maskProxyUrl(proxy)

		var proxyStats models.ProxyStats
		result := database.DB.Where("proxy_url = ?", maskedProxy).FirstOrCreate(&proxyStats)
		if result.Error != nil {
			logger.Log.WithError(result.Error).Errorf("Failed to register proxy: %s", maskedProxy)
			continue
		}

		if result.RowsAffected > 0 {
			proxyStats.Status = "active"
			proxyStats.LastCheck = time.Now()
			if err := database.DB.Save(&proxyStats).Error; err != nil {
				logger.Log.WithError(err).Errorf("Failed to update proxy status: %s", maskedProxy)
			}
		}
	}
}

func maskProxyUrl(proxyUrl string) string {
	parsedUrl, err := url.Parse(proxyUrl)
	if err != nil {
		return "invalid-proxy-url"
	}

	if parsedUrl.User != nil {
		return fmt.Sprintf("%s://*****:****@%s", parsedUrl.Scheme, parsedUrl.Host)
	}

	return fmt.Sprintf("%s://%s", parsedUrl.Scheme, parsedUrl.Host)
}

func (pm *ProxyManager) GetClient() *http.Client {
	pm.RLock()
	defer pm.RUnlock()

	if !pm.ProxyEnabled || len(pm.ActiveProxies) == 0 {
		return pm.DefaultClient
	}

	proxy := pm.getNextProxy()
	if proxy == "" {
		logger.Log.Warn("No active proxies available, using default client")
		return pm.DefaultClient
	}

	client, ok := pm.ProxyClients[proxy]
	if !ok {
		logger.Log.Warnf("No client found for proxy %s, using default client", maskProxyUrl(proxy))
		return pm.DefaultClient
	}

	return client
}

func (pm *ProxyManager) getNextProxy() string {
	if len(pm.Proxies) == 0 {
		return ""
	}

	now := time.Now()
	var selectedProxy string

	switch pm.RotationStrategy {
	case "round-robin":
		for i := 0; i < len(pm.Proxies); i++ {
			pm.CurrentProxyIndex = (pm.CurrentProxyIndex + 1) % len(pm.Proxies)
			proxy := pm.Proxies[pm.CurrentProxyIndex]

			if rateLimitedUntil, ok := pm.ProxyRateLimits[proxy]; ok && now.Before(rateLimitedUntil) {
				continue
			}
			if !pm.ActiveProxies[proxy] {
				continue
			}

			selectedProxy = proxy
			break
		}

	case "random":
		var activeProxies []string
		for proxy := range pm.ActiveProxies {
			if pm.ActiveProxies[proxy] {
				if rateLimitedUntil, ok := pm.ProxyRateLimits[proxy]; !ok || now.After(rateLimitedUntil) {
					activeProxies = append(activeProxies, proxy)
				}
			}
		}

		if len(activeProxies) > 0 {
			selectedProxy = activeProxies[rand.Intn(len(activeProxies))]
		}

	case "least-used":
		var leastRecentProxy string
		var leastRecentTime time.Time

		for proxy := range pm.ActiveProxies {
			if !pm.ActiveProxies[proxy] {
				continue
			}

			if rateLimitedUntil, ok := pm.ProxyRateLimits[proxy]; ok && now.Before(rateLimitedUntil) {
				continue
			}

			lastUsed, ok := pm.ProxyLastUsed[proxy]
			if !ok || (leastRecentProxy == "" || lastUsed.Before(leastRecentTime)) {
				leastRecentProxy = proxy
				leastRecentTime = lastUsed
			}
		}

		selectedProxy = leastRecentProxy
	}

	if selectedProxy != "" {
		pm.ProxyLastUsed[selectedProxy] = now
	}

	return selectedProxy
}

func (pm *ProxyManager) MarkProxySuccess(proxyURL string) {
	if proxyURL == "" {
		return
	}

	pm.Lock()
	defer pm.Unlock()

	// Reset failure count
	pm.ProxyFailures[proxyURL] = 0

	go pm.updateProxyStats(proxyURL, true, "")
}

func (pm *ProxyManager) MarkProxyFailure(proxyURL string, reason string) {
	if proxyURL == "" {
		return
	}

	pm.Lock()
	defer pm.Unlock()

	pm.ProxyFailures[proxyURL]++

	if pm.ProxyFailures[proxyURL] >= pm.MaxFailures {
		pm.ActiveProxies[proxyURL] = false
		pm.ProxyRateLimits[proxyURL] = time.Now().Add(pm.CooldownPeriod)
		logger.Log.Warnf("Disabled proxy %s due to %d consecutive failures. Will retry after %s",
			maskProxyUrl(proxyURL), pm.ProxyFailures[proxyURL], pm.CooldownPeriod)
	}

	go pm.updateProxyStats(proxyURL, false, reason)
}

func (pm *ProxyManager) MarkProxyRateLimited(proxyURL string, duration time.Duration) {
	if proxyURL == "" {
		return
	}

	pm.Lock()
	defer pm.Unlock()

	pm.ProxyRateLimits[proxyURL] = time.Now().Add(duration)

	logger.Log.Warnf("Proxy %s rate limited for %s", maskProxyUrl(proxyURL), duration)

	go pm.updateProxyStats(proxyURL, false, "rate_limited")
}

func (pm *ProxyManager) updateProxyStats(proxyURL string, success bool, errorReason string) {
	maskedProxy := maskProxyUrl(proxyURL)

	var proxyStats models.ProxyStats
	if err := database.DB.Where("proxy_url = ?", maskedProxy).First(&proxyStats).Error; err != nil {
		proxyStats = models.ProxyStats{
			ProxyURL:     maskedProxy,
			Status:       "active",
			LastCheck:    time.Now(),
			SuccessCount: 0,
			FailureCount: 0,
		}
	}

	if success {
		proxyStats.SuccessCount++
		proxyStats.ConsecutiveFailures = 0
		proxyStats.Status = "active"
	} else {
		proxyStats.FailureCount++
		proxyStats.ConsecutiveFailures++
		proxyStats.LastError = errorReason

		if proxyStats.ConsecutiveFailures >= pm.MaxFailures {
			proxyStats.Status = "suspended"
			expTime := time.Now().Add(pm.CooldownPeriod)
			proxyStats.RateLimitedUntil = &expTime
		}
	}

	proxyStats.LastCheck = time.Now()

	if err := database.DB.Save(&proxyStats).Error; err != nil {
		logger.Log.WithError(err).Errorf("Failed to update proxy stats for %s", maskedProxy)
	}
}

func (pm *ProxyManager) RefreshProxies() {
	pm.Lock()
	defer pm.Unlock()

	if time.Since(pm.LastRefresh) < pm.ProxyRefreshPeriod {
		return
	}

	now := time.Now()

	for proxy, rateLimitedUntil := range pm.ProxyRateLimits {
		if now.After(rateLimitedUntil) {
			pm.ActiveProxies[proxy] = true
			delete(pm.ProxyRateLimits, proxy)
			pm.ProxyFailures[proxy] = 0
			logger.Log.Infof("Reactivated proxy %s after cooldown period", maskProxyUrl(proxy))

			go func(p string) {
				maskedProxy := maskProxyUrl(p)
				if err := database.DB.Model(&models.ProxyStats{}).
					Where("proxy_url = ?", maskedProxy).
					Updates(map[string]interface{}{
						"status":               "active",
						"consecutive_failures": 0,
						"rate_limited_until":   nil,
					}).Error; err != nil {
					logger.Log.WithError(err).Errorf("Failed to update reactivated proxy status for %s", maskedProxy)
				}
			}(proxy)
		}
	}

	cfg := configuration.Get()
	if len(cfg.Proxy.Proxies) > 0 {
		newProxies := cfg.Proxy.Proxies

		if !sameStringSlice(pm.Proxies, newProxies) {
			logger.Log.Infof("Proxy list changed, refreshing proxies. Old count: %d, New count: %d",
				len(pm.Proxies), len(newProxies))

			pm.Proxies = newProxies

			for _, proxy := range pm.Proxies {
				if _, exists := pm.ProxyClients[proxy]; !exists {
					proxyURL, err := url.Parse(proxy)
					if err != nil {
						logger.Log.WithError(err).Errorf("Invalid proxy URL: %s", maskProxyUrl(proxy))
						continue
					}

					proxyTransport := &http.Transport{
						Proxy:               http.ProxyURL(proxyURL),
						MaxIdleConns:        100,
						MaxIdleConnsPerHost: 20,
						IdleConnTimeout:     90 * time.Second,
						DisableKeepAlives:   false,
						ForceAttemptHTTP2:   true,
						MaxConnsPerHost:     0,
						TLSHandshakeTimeout: 10 * time.Second,
					}

					pm.ProxyClients[proxy] = &http.Client{
						Timeout:   30 * time.Second,
						Transport: NewHeaderTransport(proxyTransport, pm.UserAgents),
					}
				}

				pm.ActiveProxies[proxy] = true
			}

			go pm.initializeProxyStats()
		}
	}

	pm.LastRefresh = now
	pm.ProxyEnabled = len(pm.ActiveProxies) > 0

	logger.Log.Infof("Proxy refresh complete. Active proxies: %d/%d", countActiveProxies(pm.ActiveProxies), len(pm.Proxies))
}

func countActiveProxies(activeProxies map[string]bool) int {
	count := 0
	for _, active := range activeProxies {
		if active {
			count++
		}
	}
	return count
}

func sameStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aMap := make(map[string]bool)
	for _, v := range a {
		aMap[v] = true
	}

	for _, v := range b {
		if !aMap[v] {
			return false
		}
	}

	return true
}

func NewHeaderTransport(base http.RoundTripper, userAgents []string) http.RoundTripper {
	return &customHeaderTransport{
		base:       base,
		userAgents: userAgents,
	}
}

type customHeaderTransport struct {
	base       http.RoundTripper
	userAgents []string
}

func (t *customHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" && len(t.userAgents) > 0 {
		userAgent := t.userAgents[rand.Intn(len(t.userAgents))]
		req.Header.Set("User-Agent", userAgent)
	} else if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36")
	}

	return t.base.RoundTrip(req)
}

func GetSharedClient() *http.Client {
	pm := GetProxyManager()
	pm.RefreshProxies()
	return pm.GetClient()
}

func DoRequest(req *http.Request) (*http.Response, error) {
	pm := GetProxyManager()
	pm.RefreshProxies()

	if !pm.ProxyEnabled {
		return pm.DefaultClient.Do(req)
	}

	client := pm.GetClient()

	var proxyURL string
	if transport, ok := client.Transport.(*customHeaderTransport); ok {
		if httpTransport, ok := transport.base.(*http.Transport); ok {
			if httpTransport.Proxy != nil {
				if proxy, err := httpTransport.Proxy(req.WithContext(context.Background())); err == nil && proxy != nil {
					proxyURL = proxy.String()
				}
			}
		}
	}

	resp, err := client.Do(req)

	if err != nil {
		if proxyURL != "" {
			pm.MarkProxyFailure(proxyURL, err.Error())
		}
		return nil, err
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		if proxyURL != "" {
			pm.MarkProxyRateLimited(proxyURL, 10*time.Minute)
		}
	} else if resp.StatusCode >= 400 {
		if proxyURL != "" {
			pm.MarkProxyFailure(proxyURL, fmt.Sprintf("HTTP %d", resp.StatusCode))
		}
	} else {
		if proxyURL != "" {
			pm.MarkProxySuccess(proxyURL)
		}
	}

	return resp, nil
}
