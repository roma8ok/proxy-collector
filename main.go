package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/streadway/amqp"
)

const (
	searchQuery = "proxy list"
	proxyURL    = ""
	userAgent   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36"

	requestTimeout = 5 * time.Second

	rabbitURL                = "amqp://admin:admin@localhost:5672/"
	queueSearchBodiesFromDDG = "search_bodies_from_DDG"
	queueProxySources        = "proxy_sources"
	queueProxySourceHTML     = "proxy_source_html"
	queueRawProxies          = "raw_proxies"

	redisURL        = "redis://localhost:6379"
	redisExpiration = time.Hour * 24
)

func main() {
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)

	rabbitConn, err := amqp.Dial(rabbitURL)
	if err != nil {
		panic(err)
	}
	defer func(conn *amqp.Connection) {
		err := conn.Close()
		if err != nil {
			panic(err)
		}
	}(rabbitConn)
	if err := initQueues(rabbitConn, []string{queueSearchBodiesFromDDG, queueProxySources, queueProxySourceHTML, queueRawProxies}); err != nil {
		panic(err)
	}

	rdb, err := newRedisDB(redisURL)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			if err := sendSearchBodyFromDDGToQueue(rabbitConn, searchQuery, proxyURL, userAgent); err != nil {
				panic(err)
			}
			time.Sleep(time.Minute)
		}
	}()

	go func() {
		if err := processSearchBodyFromDDG(rabbitConn, rdb); err != nil {
			panic(err)
		}
	}()

	go func() {
		if err := sendHTMLFromProxySourceToQueue(rabbitConn); err != nil {
			panic(err)
		}
	}()

	go func() {
		if err := processSourceHTML(rabbitConn); err != nil {
			panic(err)
		}
	}()

	<-exit
}
