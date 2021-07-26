package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/streadway/amqp"
)

func sendSearchBodyFromDDGToQueue(conn *amqp.Connection, searchQuery, proxyURL, userAgent string) error {
	fmt.Println(time.Now().Format("2006-01-02T15:04:05"), "sendSearchBodyFromDDGToQueue - start")
	ts := time.Now()
	defer func(ts time.Time) {
		fmt.Println(time.Now().Format("2006-01-02T15:04:05"), "sendSearchBodyFromDDGToQueue end", time.Now().Sub(ts))
	}(ts)

	searchURL := makeDDGSearchURL(searchQuery)

	var headers = map[string]string{
		"Content-Type": "text/html; charset=UTF-8",
		"User-Agent":   userAgent,
	}

	resp, err := sendRequest(searchURL, proxyURL, headers)
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
		fmt.Println(time.Now().Format("2006-01-02T15:04:05"), "processSearchBodyFromDDG - start")
		ts := time.Now()
		defer func(ts time.Time) {
			fmt.Println(time.Now().Format("2006-01-02T15:04:05"), "processSearchBodyFromDDG end", time.Now().Sub(ts))
		}(ts)

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

func sendHTMLFromProxySourceToQueue(rabbitConn *amqp.Connection) error {
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
		fmt.Println(time.Now().Format("2006-01-02T15:04:05"), "sendHTMLFromProxySourceToQueue - start with", string(in))
		ts := time.Now()
		defer func(ts time.Time) {
			fmt.Println(time.Now().Format("2006-01-02T15:04:05"), "sendHTMLFromProxySourceToQueue end", time.Now().Sub(ts))
		}(ts)

		var headers = map[string]string{
			"Content-Type": "text/html; charset=UTF-8",
			"User-Agent":   userAgent,
		}

		resp, err := sendRequest(string(in), proxyURL, headers)
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
		hPayload := htmlPayload{
			HTML:    string(body),
			FromURL: string(in),
		}
		hPayloadJSON, err := json.Marshal(hPayload)
		if err != nil {
			return err
		}

		if err := publish(chDest, queueProxySourceHTML, hPayloadJSON); err != nil {
			return err
		}
		return nil
	}

	if err := consume(chSource, queueProxySources, handler); err != nil {
		return err
	}

	return nil
}

func processSourceHTML(rabbitConn *amqp.Connection, rdbForSites *redisDB, rdbForProxies *redisDB) error {
	chSource, err := rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer chSource.Close()

	chDestRawProxies, err := rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer chDestRawProxies.Close()

	chDestProxySources, err := rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer chDestProxySources.Close()

	handler := func(in []byte) error {
		fmt.Println(time.Now().Format("2006-01-02T15:04:05"), "processSourceHTML - start")
		ts := time.Now()
		defer func(ts time.Time) {
			fmt.Println(time.Now().Format("2006-01-02T15:04:05"), "processSourceHTML end", time.Now().Sub(ts))
		}(ts)

		var hPayload htmlPayload
		if err := json.Unmarshal(in, &hPayload); err != nil {
			return err
		}

		proxies := findProxiesFromHTML(hPayload.HTML)
		for _, p := range proxies {
			changeType, err := rdbForProxies.set(p)
			if err != nil {
				return err
			}
			switch changeType {
			case redisChangeAdd, redisChangeUpdate:
				if err := publish(chDestRawProxies, queueRawProxies, []byte(p)); err != nil {
				}
			}
		}

		urls := findURLsFromHTML(hPayload.HTML)
		for _, u := range urls {
			if !urlsHaveSameDomain(hPayload.FromURL, u) {
				continue
			}
			if !possibleForProxySourceURL(u) {
				continue
			}

			changeType, err := rdbForSites.set(u)
			if err != nil {
				return err
			}
			switch changeType {
			case redisChangeAdd, redisChangeUpdate:
				if err := publish(chDestProxySources, queueProxySources, []byte(u)); err != nil {
				}
			}
		}

		return nil
	}

	if err := consume(chSource, queueProxySourceHTML, handler); err != nil {
		return err
	}

	return nil
}

func processRawProxy(rabbitConn *amqp.Connection) error {
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
		fmt.Println(time.Now().Format("2006-01-02T15:04:05"), "processRawProxy - start")
		ts := time.Now()
		defer func(ts time.Time) {
			fmt.Println(time.Now().Format("2006-01-02T15:04:05"), "processRawProxy end", time.Now().Sub(ts))
		}(ts)

		pType, anonymous, err := checkProxy(string(in))
		if err != nil {
			return err
		}
		tsNow := time.Now()
		tsNowRFC3339 := tsNow.Format(time.RFC3339)

		p := proxyPayload{
			URL:       string(in),
			Anonymous: anonymous,
			Type:      pType.verbose(),
			TS:        tsNowRFC3339,
		}
		payloadJSON, err := json.Marshal(p)
		if err != nil {
			return err
		}

		if err := publish(chDest, queueProcessedProxies, payloadJSON); err != nil {
		}

		return nil
	}

	if err := consume(chSource, queueRawProxies, handler); err != nil {
		return err
	}

	return nil
}
