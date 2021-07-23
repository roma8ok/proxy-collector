package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/streadway/amqp"
)

const (
	proxyURL  = ""
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36"

	requestTimeout = 5 * time.Second

	rabbitURL                = "amqp://admin:admin@localhost:5672/"
	queueSearchBodiesFromDDG = "search_bodies_from_DDG"
	queueProxySources        = "proxy_sources"
	queueProxySourceHTML     = "proxy_source_html"
	queueRawProxies          = "raw_proxies"

	redisURL        = "redis://localhost:6379"
	redisExpiration = time.Hour * 24

	serviceSendSearchBodyFromDDGToQueue   = "sendSearchBodyFromDDGToQueue"
	serviceProcessSearchBodyFromDDG       = "processSearchBodyFromDDG"
	serviceSendHTMLFromProxySourceToQueue = "sendHTMLFromProxySourceToQueue"
	serviceProcessSourceHTML              = "processSourceHTML"
)

func main() {
	var sFlags arrayFlags
	flag.Var(&sFlags, "service", fmt.Sprintf("Select one of more services for activate.\nAvailable: %s, %s, %s, %s.", serviceSendSearchBodyFromDDGToQueue, serviceProcessSearchBodyFromDDG, serviceSendHTMLFromProxySourceToQueue, serviceProcessSourceHTML))
	flag.Parse()
	if len(sFlags) == 0 {
		panic("Select one of more services for activate (flag -service)")
	}

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

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)

	if isExist(serviceSendSearchBodyFromDDGToQueue, sFlags) {
		fmt.Printf("Start worker for %s\n", serviceSendSearchBodyFromDDGToQueue)
		go func() {
			for {
				for _, query := range []string{"proxy list", "proxy", "proxies", "прокси", "список прокси"} {
					if err := sendSearchBodyFromDDGToQueue(rabbitConn, query, proxyURL, userAgent); err != nil {
						panic(err)
					}
					time.Sleep(time.Minute)
				}
			}
		}()
	}

	if isExist(serviceProcessSearchBodyFromDDG, sFlags) {
		fmt.Printf("Start worker for %s\n", serviceProcessSearchBodyFromDDG)
		go func() {
			if err := processSearchBodyFromDDG(rabbitConn, rdb); err != nil {
				panic(err)
			}
		}()
	}

	if isExist(serviceSendHTMLFromProxySourceToQueue, sFlags) {
		fmt.Printf("Start worker for %s\n", serviceSendHTMLFromProxySourceToQueue)
		go func() {
			if err := sendHTMLFromProxySourceToQueue(rabbitConn); err != nil {
				panic(err)
			}
		}()
	}

	if isExist(serviceProcessSourceHTML, sFlags) {
		fmt.Printf("Start worker for %s\n", serviceProcessSourceHTML)
		go func() {
			if err := processSourceHTML(rabbitConn, rdb); err != nil {
				panic(err)
			}
		}()
	}

	<-exit
}
