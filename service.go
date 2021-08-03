package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

func fillSearchQueries(app App, fromProxies bool) error {
	startTime := time.Now()

	ch, err := app.rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer func(ch *amqp.Channel) {
		if err := ch.Close(); err != nil {
			app.loki.error(errors.WithStack(err))
		}
	}(ch)

	var q string
	proxies := make([]string, 0)
	if fromProxies {
		freshProxyList, err := freshProxies(app.postgresPool, time.Minute*60)
		if err == nil {
			for _, fProxy := range freshProxyList {
				proxies = append(proxies, fProxy.URL)
			}
		} else {
			app.loki.error(errors.WithStack(err))
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

	app.loki.info(fmt.Sprintf(`done in %s; added "%s"`, formatDuration(time.Now().Sub(startTime)), q))
	return nil
}

func sendSearchBodyFromDDGToQueue(app App, proxyURL, userAgent string) error {
	chSource, err := app.rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer func(chSource *amqp.Channel) {
		if err := chSource.Close(); err != nil {
			app.loki.error(errors.WithStack(err))
		}
	}(chSource)

	chDest, err := app.rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer func(chDest *amqp.Channel) {
		if err := chDest.Close(); err != nil {
			app.loki.error(errors.WithStack(err))
		}
	}(chDest)

	handler := func(in []byte) error {
		startTime := time.Now()

		searchURL := makeDDGSearchURL(string(in))

		var headers = map[string]string{
			"Content-Type": "text/html; charset=UTF-8",
			"User-Agent":   userAgent,
		}

		resp, err := sendRequest(searchURL, proxyURL, headers)
		if err != nil {
			return err
		}
		defer func(Body io.ReadCloser) {
			if err := Body.Close(); err != nil {
				app.loki.error(errors.WithStack(err))
			}
		}(resp.Body)

		if resp.StatusCode != http.StatusOK {
			return err
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if err := publish(chDest, queueSearchBodiesFromDDG, body); err != nil {
			app.loki.error(errors.WithStack(err))
			return err
		}

		app.loki.info(fmt.Sprintf(`done in %s; added body from "%s"`, formatDuration(time.Now().Sub(startTime)), searchURL))
		return nil
	}

	if err := consume(chSource, queueSearchQueries, handler); err != nil {
		return err
	}

	return nil
}

func processSearchBodyFromDDG(app App) error {
	chSource, err := app.rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer func(chSource *amqp.Channel) {
		if err := chSource.Close(); err != nil {
			app.loki.error(errors.WithStack(err))
		}
	}(chSource)

	chDest, err := app.rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer func(chDest *amqp.Channel) {
		if err := chDest.Close(); err != nil {
			app.loki.error(errors.WithStack(err))
		}
	}(chDest)

	handler := func(in []byte) error {
		startTime := time.Now()

		urls := findSiteURLsFromDDG(string(in))
		addedUrls := 0
		for _, u := range urls {
			changeType, err := app.rdbForSites.set(u)
			if err != nil {
				app.loki.error(errors.WithStack(err))
				return err
			}
			switch changeType {
			case redisChangeAdd, redisChangeUpdate:
				if err := publish(chDest, queueProxySources, []byte(u)); err != nil {
					app.loki.error(errors.WithStack(err))
					return err
				}
				addedUrls += 1
			}
		}

		app.loki.info(fmt.Sprintf(`done in %s; handled %d urls, added %d urls`, formatDuration(time.Now().Sub(startTime)), len(urls), addedUrls))
		return nil
	}

	if err := consume(chSource, queueSearchBodiesFromDDG, handler); err != nil {
		return err
	}

	return nil
}

func sendHTMLFromProxySourceToQueue(app App) error {
	chSource, err := app.rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer func(chSource *amqp.Channel) {
		if err := chSource.Close(); err != nil {
			app.loki.error(errors.WithStack(err))
		}
	}(chSource)

	chDest, err := app.rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer func(chDest *amqp.Channel) {
		if err := chDest.Close(); err != nil {
			app.loki.error(errors.WithStack(err))
		}
	}(chDest)

	handler := func(in []byte) error {
		startTime := time.Now()

		var headers = map[string]string{
			"Content-Type": "text/html; charset=UTF-8",
			"User-Agent":   userAgent,
		}

		u := string(in)
		resp, err := sendRequest(u, proxyURL, headers)
		if err != nil {
			return err
		}
		defer func(Body io.ReadCloser) {
			if err := Body.Close(); err != nil {
				app.loki.error(errors.WithStack(err))
			}
		}(resp.Body)

		if resp.StatusCode != http.StatusOK {
			return err
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		hPayload := htmlPayload{
			HTML:    string(body),
			FromURL: u,
		}
		hPayloadJSON, err := json.Marshal(hPayload)
		if err != nil {
			return err
		}

		if err := publish(chDest, queueProxySourceHTML, hPayloadJSON); err != nil {
			app.loki.error(errors.WithStack(err))
		}

		app.loki.info(fmt.Sprintf(`done in %s; added body from "%s"`, formatDuration(time.Now().Sub(startTime)), u))
		return nil
	}

	if err := consume(chSource, queueProxySources, handler); err != nil {
		return err
	}

	return nil
}

func processSourceHTML(app App) error {
	chSource, err := app.rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer func(chSource *amqp.Channel) {
		if err := chSource.Close(); err != nil {
			app.loki.error(errors.WithStack(err))
		}
	}(chSource)

	chDestRawProxies, err := app.rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer func(chDestRawProxies *amqp.Channel) {
		if err := chDestRawProxies.Close(); err != nil {
			app.loki.error(errors.WithStack(err))
		}
	}(chDestRawProxies)

	chDestProxySources, err := app.rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer func(chDestProxySources *amqp.Channel) {
		if err := chDestProxySources.Close(); err != nil {
			app.loki.error(errors.WithStack(err))
		}
	}(chDestProxySources)

	forbiddenDomains, err := blacklistDomains(app.postgresPool)
	if err != nil {
		return err
	}

	handler := func(in []byte) error {
		startTime := time.Now()

		var hPayload htmlPayload
		if err := json.Unmarshal(in, &hPayload); err != nil {
			app.loki.error(errors.WithStack(err))
			return err
		}

		proxies := findProxiesFromHTML(hPayload.HTML)
		addedProxies := 0
		for _, p := range proxies {
			changeType, err := app.rdbForProxies.set(p)
			if err != nil {
				app.loki.error(errors.WithStack(err))
				return err
			}
			switch changeType {
			case redisChangeAdd, redisChangeUpdate:
				if err := publish(chDestRawProxies, queueRawProxies, []byte(p)); err != nil {
					app.loki.error(errors.WithStack(err))
					return err
				}
				addedProxies += 1
			}
		}

		urls := findURLsFromHTML(hPayload.HTML)
		addedURLs := 0
		if len(proxies) > 0 {
			for _, u := range urls {
				if !urlsHaveSameDomain(hPayload.FromURL, u) {
					continue
				}
				if !possibleForProxySourceURL(u, forbiddenDomains) {
					continue
				}

				changeType, err := app.rdbForSites.set(u)
				if err != nil {
					app.loki.error(errors.WithStack(err))
					return err
				}
				switch changeType {
				case redisChangeAdd, redisChangeUpdate:
					if err := publish(chDestProxySources, queueProxySources, []byte(u)); err != nil {
						app.loki.error(errors.WithStack(err))
						return err
					}
					addedURLs += 1
				}
			}
		}

		app.loki.info(fmt.Sprintf(`done in %s; processed "%s"; found/added: %d/%d proxies, %d/%d urls`, formatDuration(time.Now().Sub(startTime)), hPayload.FromURL, len(proxies), addedProxies, len(urls), addedURLs))
		return nil
	}

	if err := consume(chSource, queueProxySourceHTML, handler); err != nil {
		return err
	}

	return nil
}

func processRawProxy(app App) error {
	ch, err := app.rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer func(ch *amqp.Channel) {
		if err := ch.Close(); err != nil {
			app.loki.error(errors.WithStack(err))
		}
	}(ch)

	handler := func(in []byte) error {
		startTime := time.Now()

		pURL := string(in)

		pType, anonymous, err := checkProxy(pURL, app.conf.IPAPIURL)
		if err != nil {
			app.loki.info(fmt.Sprintf(`done in %s; discarded "%s"`, formatDuration(time.Now().Sub(startTime)), pURL))
			return err
		}

		p := proxy{
			URL:       pURL,
			Active:    true,
			Type:      pType.verbose(),
			Anonymous: anonymous,
			Created:   startTime,
			LastCheck: startTime,
		}

		if err := saveProxyToDB(app.postgresPool, p); err != nil {
			app.loki.error(errors.WithStack(err))
			return err
		}

		app.loki.info(fmt.Sprintf(`done in %s; added "%s"`, formatDuration(time.Now().Sub(startTime)), pURL))
		return nil
	}

	if err := consume(ch, queueRawProxies, handler); err != nil {
		return err
	}

	return nil
}

func fillCheckProxiesQueue(app App) error {
	ch, err := app.rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer func(ch *amqp.Channel) {
		if err := ch.Close(); err != nil {
			app.loki.error(errors.WithStack(err))
		}
	}(ch)

	startTime := time.Now()

	proxies, err := freshProxies(app.postgresPool, time.Hour*24)
	if err != nil {
		return err
	}
	for _, p := range proxies {
		if err := publish(ch, queueCheckProxies, []byte(p.URL)); err != nil {
			return err
		}
	}

	app.loki.info(fmt.Sprintf(`done in %s; added %d proxies`, formatDuration(time.Now().Sub(startTime)), len(proxies)))
	return nil
}

func processCheckProxy(app App) error {
	ch, err := app.rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer func(ch *amqp.Channel) {
		if err := ch.Close(); err != nil {
			app.loki.error(errors.WithStack(err))
		}
	}(ch)

	handler := func(in []byte) error {
		startTime := time.Now()

		pURL := string(in)

		pType, anonymous, err := checkProxy(pURL, app.conf.IPAPIURL)
		if err != nil {
			if err := changeProxyToInactive(app.postgresPool, pURL); err != nil {
				app.loki.error(errors.WithStack(err))
				return err
			}
			app.loki.info(fmt.Sprintf(`done in %s; changed to inactive "%s"`, formatDuration(time.Now().Sub(startTime)), pURL))
			return err
		}

		p := proxy{
			URL:       pURL,
			Active:    true,
			Type:      pType.verbose(),
			Anonymous: anonymous,
			LastCheck: startTime,
		}

		if err := updateProxyInDB(app.postgresPool, p); err != nil {
			app.loki.error(errors.WithStack(err))
			return err
		}

		app.loki.info(fmt.Sprintf(`done in %s; updated last check (active) "%s"`, formatDuration(time.Now().Sub(startTime)), pURL))
		return nil
	}

	if err := consume(ch, queueCheckProxies, handler); err != nil {
		return err
	}

	return nil
}
