package main

import (
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	proxyURL  = ""
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36"

	requestTimeout = 5 * time.Second

	queueSearchQueries       = "1_search_queries"
	queueSearchBodiesFromDDG = "2_search_bodies_from_DDG"
	queueProxySources        = "3_proxy_sources"
	queueProxySourceHTML     = "4_proxy_source_html"
	queueRawProxies          = "5_raw_proxies"
	queueCheckProxies        = "6_check_proxies"

	rabbitMQReconnectDelay     = 30 * time.Second // When reconnecting to the server after connection failure
	rabbitMQReInitDelay        = 30 * time.Second // When setting up the channel after a channel exception
	rabbitMQResendDelay        = 10 * time.Second // When resending messages the server didn't confirm
	rabbitMQRecheckStreamDelay = 10 * time.Second

	redisExpiration = time.Hour * 24

	lokiBatchWait = time.Second * 5

	serviceFillSearchQueries              = "fillSearchQueries"
	serviceSendSearchBodyFromDDGToQueue   = "sendSearchBodyFromDDGToQueue"
	serviceProcessSearchBodyFromDDG       = "processSearchBodyFromDDG"
	serviceSendHTMLFromProxySourceToQueue = "sendHTMLFromProxySourceToQueue"
	serviceProcessSourceHTML              = "processSourceHTML"
	serviceProcessRawProxy                = "processRawProxy"
	serviceFillCheckProxiesQueue          = "fillCheckProxiesQueue"
	serviceProcessCheckProxies            = "processCheckProxies"
)

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
	rdbForSites   *redisDB
	rdbForProxies *redisDB
	conf          Config
}
