package main

import (
	"fmt"
	"time"
)

const (
	proxyURL  = ""
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36"

	requestTimeout = 5 * time.Second

	queueSearchQueries       = "search_queries"
	queueSearchBodiesFromDDG = "search_bodies_from_DDG"
	queueProxySources        = "proxy_sources"
	queueProxySourceHTML     = "proxy_source_html"
	queueRawProxies          = "raw_proxies"

	ipAPIURL = "https://api64.ipify.org"

	redisExpiration = time.Hour * 24

	serviceFillSearchQueries              = "fillSearchQueries"
	serviceSendSearchBodyFromDDGToQueue   = "sendSearchBodyFromDDGToQueue"
	serviceProcessSearchBodyFromDDG       = "processSearchBodyFromDDG"
	serviceSendHTMLFromProxySourceToQueue = "sendHTMLFromProxySourceToQueue"
	serviceProcessSourceHTML              = "processSourceHTML"
	serviceProcessRawProxy                = "processRawProxy"
)

type config struct {
	RabbitURL          string `json:"rabbit_url"`
	RedisURLForSites   string `json:"redis_url_for_sites"`
	RedisURLForProxies string `json:"redis_url_for_proxies"`
	PostgresURL        string `json:"postgres_url"`
}

type arrayFlags []string

func (i *arrayFlags) String() string {
	return fmt.Sprint(*i)
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}
