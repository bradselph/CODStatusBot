package services

import (
	"net/http"
	"sync"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/logger"
)

var (
	defaultClient      *http.Client
	longTimeoutClient  *http.Client
	clientMutex        sync.RWMutex
	clientsInitialized bool
)

func InitHTTPClients() {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	if clientsInitialized {
		return
	}

	cfg := configuration.Get()
	proxyManager := GetProxyManager()

	if cfg.Proxy.Enabled && len(cfg.Proxy.Proxies) > 0 {
		logger.Log.Infof("Using proxies for HTTP clients, %d proxies available", len(cfg.Proxy.Proxies))
		defaultClient = proxyManager.GetClient()

		longTimeoutClient = &http.Client{
			Timeout:   60 * time.Second,
			Transport: defaultClient.Transport,
		}
	} else {
		logger.Log.Info("Using direct HTTP clients without proxies")

		transport := &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   false,
			ForceAttemptHTTP2:   true,
			MaxConnsPerHost:     0,
			TLSHandshakeTimeout: 10 * time.Second,
		}

		defaultClient = &http.Client{
			Timeout:   30 * time.Second,
			Transport: NewHeaderTransport(transport, cfg.Proxy.UserAgents),
		}

		longTimeoutClient = &http.Client{
			Timeout:   60 * time.Second,
			Transport: NewHeaderTransport(transport, cfg.Proxy.UserAgents),
		}
	}

	clientsInitialized = true
}

func GetDefaultHTTPClient() *http.Client {
	clientMutex.RLock()

	if !clientsInitialized {
		clientMutex.RUnlock()
		InitHTTPClients()
		clientMutex.RLock()
	}

	cfg := configuration.Get()
	if cfg.Proxy.Enabled {
		clientMutex.RUnlock()
		proxyManager := GetProxyManager()
		return proxyManager.GetClient()
	}

	defer clientMutex.RUnlock()
	return defaultClient
}

func GetLongTimeoutHTTPClient() *http.Client {
	clientMutex.RLock()

	if !clientsInitialized {
		clientMutex.RUnlock()
		InitHTTPClients()
		clientMutex.RLock()
	}

	cfg := configuration.Get()
	if cfg.Proxy.Enabled {
		clientMutex.RUnlock()
		proxyManager := GetProxyManager()

		client := proxyManager.GetClient()
		return &http.Client{
			Timeout:   60 * time.Second,
			Transport: client.Transport,
		}
	}

	defer clientMutex.RUnlock()
	return longTimeoutClient
}
