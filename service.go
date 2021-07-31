package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/streadway/amqp"
)

func fillSearchQueries(rabbitConn *amqp.Connection, postgresPool *pgxpool.Pool, fromProxies bool) error {
	ch, err := rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	fmt.Println(time.Now().Format("2006-01-02T15:04:05"), "fillSearchQueries - start")
	ts := time.Now()
	defer func(ts time.Time) {
		fmt.Println(time.Now().Format("2006-01-02T15:04:05"), "fillSearchQueries end", time.Now().Sub(ts))
	}(ts)

	var q string
	proxies := make([]string, 0)
	if fromProxies {
		freshProxyList, err := freshProxies(postgresPool, time.Minute*60)
		if err == nil {
			for _, fProxy := range freshProxyList {
				proxies = append(proxies, fProxy.URL)
			}
		}
	}

	if len(proxies) > 0 {
		q = randomElementFromArray(proxies)
	} else {
		queries := []string{"proxy list", "proxy", "proxies", "прокси", "список прокси"}
		q = randomElementFromArray(queries)
	}

	if err := publish(ch, queueSearchQueries, []byte(q)); err != nil {
		return err
	}

	return nil
}

func sendSearchBodyFromDDGToQueue(rabbitConn *amqp.Connection, proxyURL, userAgent string) error {
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
		fmt.Println(time.Now().Format("2006-01-02T15:04:05"), "sendSearchBodyFromDDGToQueue - start")
		ts := time.Now()
		defer func(ts time.Time) {
			fmt.Println(time.Now().Format("2006-01-02T15:04:05"), "sendSearchBodyFromDDGToQueue end", time.Now().Sub(ts))
		}(ts)

		searchURL := makeDDGSearchURL(string(in))

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

		if err := publish(chDest, queueSearchBodiesFromDDG, body); err != nil {
			return err
		}

		return nil
	}

	if err := consume(chSource, queueSearchQueries, handler); err != nil {
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

func processSourceHTML(rabbitConn *amqp.Connection, rdbForSites *redisDB, rdbForProxies *redisDB, postgresPool *pgxpool.Pool) error {
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

	forbiddenDomains, err := blacklistDomains(postgresPool)
	if err != nil {
		return err
	}

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
			if !possibleForProxySourceURL(u, forbiddenDomains) {
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

func processRawProxy(rabbitConn *amqp.Connection, postgresPool *pgxpool.Pool) error {
	ch, err := rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

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

		p := proxy{
			URL:       string(in),
			Active:    true,
			Type:      pType.verbose(),
			Anonymous: anonymous,
			Created:   ts,
			LastCheck: ts,
		}

		if err := saveProxyToDB(postgresPool, p); err != nil {
			return err
		}

		return nil
	}

	if err := consume(ch, queueRawProxies, handler); err != nil {
		return err
	}

	return nil
}

func fillCheckProxiesQueue(rabbitConn *amqp.Connection, postgresPool *pgxpool.Pool) error {
	ch, err := rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	fmt.Println(time.Now().Format("2006-01-02T15:04:05"), "fillCheckProxiesQueue - start")
	ts := time.Now()
	defer func(ts time.Time) {
		fmt.Println(time.Now().Format("2006-01-02T15:04:05"), "fillCheckProxiesQueue end", time.Now().Sub(ts))
	}(ts)

	proxies, err := freshProxies(postgresPool, time.Hour*24)
	if err != nil {
		return err
	}
	for _, p := range proxies {
		if err := publish(ch, queueCheckProxies, []byte(p.URL)); err != nil {
			return err
		}
	}

	return nil
}

func processCheckProxy(rabbitConn *amqp.Connection, postgresPool *pgxpool.Pool) error {
	ch, err := rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	handler := func(in []byte) error {
		fmt.Println(time.Now().Format("2006-01-02T15:04:05"), "processCheckProxy - start")
		ts := time.Now()
		defer func(ts time.Time) {
			fmt.Println(time.Now().Format("2006-01-02T15:04:05"), "processCheckProxy end", time.Now().Sub(ts))
		}(ts)

		proxyURL := string(in)

		pType, anonymous, err := checkProxy(proxyURL)
		if err != nil {
			if err := changeProxyToInactive(postgresPool, proxyURL); err != nil {
			}
			return err
		}

		p := proxy{
			URL:       proxyURL,
			Active:    true,
			Type:      pType.verbose(),
			Anonymous: anonymous,
			LastCheck: ts,
		}

		if err := updateProxyInDB(postgresPool, p); err != nil {
			return err
		}

		return nil
	}

	if err := consume(ch, queueCheckProxies, handler); err != nil {
		return err
	}

	return nil
}
