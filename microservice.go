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
	if err := sendToQueue(conn, queueSearchBodiesFromDDG, body); err != nil {
		return err
	}

	return nil
}
