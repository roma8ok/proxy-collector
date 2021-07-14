package main

import (
	"fmt"

	"github.com/streadway/amqp"
)

const (
	rabbitURL                = "amqp://admin:admin@localhost:5672/"
	searchQuery              = "proxy list"
	proxyURL                 = ""
	userAgent                = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36"
	queueSearchBodiesFromDDG = "search_bodies_from_DDG"
)

func main() {
	conn, err := amqp.Dial(rabbitURL)
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
	}(conn)

	if err := sendSearchBodyFromDDGToQueue(conn, searchQuery, proxyURL, userAgent); err != nil {
		fmt.Println(err)
		return
	}
}
