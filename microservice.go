package main

import (
	"io/ioutil"
	"net/http"

	"github.com/streadway/amqp"
)

func sendSearchBodyFromDDGToQueue(conn *amqp.Connection, searchQuery, proxyURL, userAgent string) error {
	searchURL := makeDDGSearchURL(searchQuery)
	resp, err := sendSearchRequest(searchURL, proxyURL, userAgent)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	if err := publish(ch, queueSearchBodiesFromDDG, body); err != nil {
		return err
	}

	return nil
}

func processSearchBodyFromDDG(rabbitConn *amqp.Connection, rdb *redisDB) error {
	chSource, err := rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer chSource.Close()

	chDest, err := rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer chDest.Close()

	handler := func(in []byte) error {
		urls := findSiteURLsFromDDG(string(in))
		for _, u := range urls {
			changeType, err := rdb.set(u)
			if err != nil {
				return err
			}
			switch changeType {
			case redisChangeAdd, redisChangeUpdate:
				if err := publish(chDest, queueProxySources, []byte(u)); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := consume(chSource, queueSearchBodiesFromDDG, handler); err != nil {
		return err
	}

	return nil
}
