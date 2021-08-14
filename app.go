package main

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	proxyURL = "" // Default proxy for requests

	// Default user agent for requests
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36"

	requestTimeout = 10 * time.Second

	// RabbitMQ queue names:
	queueProxySources         = "proxy_sources"
	queueProxySourcesDeferred = "proxy_sources_deferred"
	queueRawProxies           = "raw_proxies"
	queueCheckProxies         = "check_proxies"

	rabbitMQReconnectDelay     = 30 * time.Second // When reconnecting to the server after connection failure
	rabbitMQReInitDelay        = 30 * time.Second // When setting up the channel after a channel exception
	rabbitMQResendDelay        = 10 * time.Second // When resending messages the server didn't confirm
	rabbitMQRecheckStreamDelay = 10 * time.Second

	redisExpiration = time.Hour * 24 // Time after which redis key has expired

	lokiBatchWait = time.Second * 5

	// Service names:
	serviceFindProxySourcesFromDDG      = "findProxySourcesFromDDG"
	serviceProcessProxySources          = "processProxySources"
	serviceTransferDeferredProxySources = "transferDeferredProxySources"
	serviceProcessRawProxies            = "processRawProxies"
	serviceFillCheckProxies             = "fillCheckProxies"
	serviceProcessCheckProxies          = "processCheckProxies"
)

var (
	errStatusCodeNotOK = errors.New("status code is not 200")

	queriesForSearchEngine = []string{
		"proxy list", "proxy lists", "proxy", "proxies", "best proxies", "free proxies",
		"eine Liste von Proxies", "Proxys", "kostenlose Proxys", "offene Proxys",
		"lista de proxy", "proxies abiertos", "proxies gratuitos",
		"список прокси", "прокси", "лучшее прокси", "бесплатные прокси", "открытые прокси",
		"代理名單", "代理列表", "免費代理",
	}
)

// Config is app urls configuration
type Config struct {
	RabbitURL          string `json:"rabbit_url"`
	RedisURLForSites   string `json:"redis_url_for_sites"`
	RedisURLForProxies string `json:"redis_url_for_proxies"`
	PostgresURL        string `json:"postgres_url"`
	IPAPIURL           string `json:"ip_api_url"`
	PromtailURL        string `json:"promtail_url"`
}

type App struct {
	loki          Logger
	postgresPool  *pgxpool.Pool
	rdbForSites   *RedisDB
	rdbForProxies *RedisDB
	conf          Config
	hostExtIP     string // external IP of the host
}
