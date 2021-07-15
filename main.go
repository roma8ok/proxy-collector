package main

import (
	"fmt"
	"time"

	"github.com/streadway/amqp"
)

const (
	searchQuery = "proxy list"
	proxyURL    = ""
	userAgent   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36"

	rabbitURL                = "amqp://admin:admin@localhost:5672/"
	queueSearchBodiesFromDDG = "search_bodies_from_DDG"
	queueProxySources        = "proxy_sources"

	redisURL        = "redis://localhost:6379"
	redisExpiration = time.Hour * 24
)

func main() {
	rabbitConn, err := amqp.Dial(rabbitURL)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func(conn *amqp.Connection) {
		err := conn.Close()
		if err != nil {
			fmt.Println(err)
			return
		}
	}(rabbitConn)

	rdb, err := newRedisDB(redisURL)
	if err != nil {
		fmt.Println(err)
		return
	}

	if err := sendSearchBodyFromDDGToQueue(rabbitConn, searchQuery, proxyURL, userAgent); err != nil {
		fmt.Println(err)
		return
	}

	if err := processSearchBodyFromDDG(rabbitConn, rdb); err != nil {
		fmt.Println(err)
		return
	}
}
