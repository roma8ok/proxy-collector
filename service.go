package main

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

/*
findProxySourcesFromDDG sends proxy sources found in DuckDuckGo to the destination rabbitMQ queue.
The "proxy source" is the url.

The query is retrieved from queriesForSearchEngine or from recently checked proxies.

findProxySourcesFromDDG only sends the proxy source to the destination queue if the proxy source is new or
hasn't been checked for a long time. This is verified with redis.
*/
func findProxySourcesFromDDG(app App, queueDest *RabbitMQSession, fromProxies bool) error {
	startTime := time.Now()

	var query string
	proxies := make([]string, 0)
	if fromProxies {
		freshProxyList, err := freshProxies(app.postgresPool, time.Minute*60)
		if err == nil {
			for _, fProxy := range freshProxyList {
				proxies = append(proxies, fProxy.URL)
			}
		} else {
			return err
		}
	}

	if len(proxies) > 0 {
		query = randomElementFromSlice(proxies)
	} else {
		query = randomElementFromSlice(queriesForSearchEngine)
	}

	searchURL := makeDDGSearchURL(query)

	var headers = map[string]string{
		"Content-Type": "text/html; charset=UTF-8",
		"User-Agent":   userAgent,
	}

	body, err := sendGetRequest(app.loki, searchURL, proxyURL, headers)
	if err != nil {
		return err
	}

	urls := findSiteURLsFromDDG(body)
	addedUrls := 0
	for _, u := range urls {
		changeType, err := app.rdbForSites.set(u)
		if err != nil {
			return err
		}
		switch changeType {
		case redisChangeAdd, redisChangeUpdate:
			if err := queueDest.Push([]byte(u)); err != nil {
				return err
			}
			addedUrls += 1
		}
	}

	app.loki.info(fmt.Sprintf(`Done in %s; found/added: %d/%d urls; query: "%s"`,
		formatDuration(time.Now().Sub(startTime)), len(urls), addedUrls, query))

	return nil
}

func processProxySources(app App, queueSource, queueDestRawProxies, queueDestProxySourcesDeferred *RabbitMQSession) {
	var handler RabbitMQStreamHandler = func(in []byte) (ack bool) {
		startTime := time.Now()

		var headers = map[string]string{
			"Content-Type": "text/html; charset=UTF-8",
			"User-Agent":   userAgent,
		}

		proxySource := string(in)
		html, err := sendGetRequest(app.loki, proxySource, proxyURL, headers)

		if err != nil {
			app.loki.debug(fmt.Sprintf(`sendRequest("%s", "%s", headers); can't read resp.Body`, proxySource, proxyURL))
			return false
		}

		proxiesFromHTML := findProxiesFromHTML(html)
		addedProxies := 0
		for _, p := range proxiesFromHTML {
			changeType, err := app.rdbForProxies.set(p)
			if err != nil {
				app.loki.error(errors.WithStack(err))
				return false
			}
			switch changeType {
			case redisChangeAdd, redisChangeUpdate:
				if err := queueDestRawProxies.Push([]byte(p)); err != nil {
					app.loki.error(errors.WithStack(err))
					return false
				}
				addedProxies += 1
			}
		}

		foundURLs := 0
		addedURLs := 0
		if len(proxiesFromHTML) > 0 {
			urlsFromHTML := findURLs(html)
			foundURLs = len(urlsFromHTML)
			for _, urlFromHTML := range urlsFromHTML {
				if !urlsHaveSameDomain(proxySource, urlFromHTML) {
					continue
				}
				if !possibleForProxySourceURL(urlFromHTML) {
					continue
				}

				changeType, err := app.rdbForSites.set(urlFromHTML)
				if err != nil {
					app.loki.error(errors.WithStack(err))
					return false
				}
				switch changeType {
				case redisChangeAdd, redisChangeUpdate:
					if err := queueDestProxySourcesDeferred.Push([]byte(urlFromHTML)); err != nil {
						app.loki.error(errors.WithStack(err))
						return false
					}
					addedURLs += 1
				}
			}
		}

		app.loki.info(fmt.Sprintf(`Done in %s; found/added: %d/%d proxies, %d/%d urls; processed "%s"`,
			formatDuration(time.Now().Sub(startTime)), len(proxiesFromHTML), addedProxies, foundURLs, addedURLs, proxySource))

		return true
	}

	queueSource.handleStream(handler)
}

func transferDeferredProxySources(app App, queueSource, queueDest *RabbitMQSession) {
	var handler RabbitMQStreamHandler = func(in []byte) (ack bool) {
		startTime := time.Now()

		if err := queueDest.Push(in); err != nil {
			app.loki.error(errors.WithStack(err))
			return false
		}

		app.loki.info(fmt.Sprintf(`Done in %s; transfer "%s"`, formatDuration(time.Now().Sub(startTime)), string(in)))
		return true
	}

	queueSource.handleStream(handler)
}

func processRawProxies(app App, queueSource *RabbitMQSession) {
	var handler RabbitMQStreamHandler = func(in []byte) (ack bool) {
		startTime := time.Now()

		pURL := string(in)

		p, err := checkProxy(pURL, app.hostExtIP, app.conf.IPAPIURL)
		if err != nil {
			app.loki.info(fmt.Sprintf(`Done in %s; discarded "%s"`, formatDuration(time.Now().Sub(startTime)), pURL))
			return false
		}
		p.Created = startTime
		p.LastCheck = startTime

		if err := saveProxyToDB(app.postgresPool, p); err != nil {
			app.loki.error(errors.WithStack(err))
			return false
		}

		app.loki.info(fmt.Sprintf(`Done in %s; added "%s"`, formatDuration(time.Now().Sub(startTime)), pURL))
		return true
	}

	queueSource.handleStream(handler)
}

func fillCheckProxies(app App, queueDest *RabbitMQSession) error {
	startTime := time.Now()

	proxies, err := freshProxies(app.postgresPool, time.Hour*24*30) // 30 days
	if err != nil {
		return err
	}
	for _, p := range proxies {
		if err := queueDest.Push([]byte(p.URL)); err != nil {
			return err
		}
	}

	app.loki.info(fmt.Sprintf(`Done in %s; added %d proxies`, formatDuration(time.Now().Sub(startTime)), len(proxies)))
	return nil
}

func processCheckProxies(app App, queueSource *RabbitMQSession) {
	var handler RabbitMQStreamHandler = func(in []byte) (ack bool) {
		startTime := time.Now()

		pURL := string(in)

		p, err := checkProxy(pURL, app.hostExtIP, app.conf.IPAPIURL)
		if err != nil {
			if err := changeProxyToInactive(app.postgresPool, pURL); err != nil {
				app.loki.error(errors.WithStack(err))
				return false
			}
			app.loki.info(fmt.Sprintf(`Done in %s; proxy is inactive "%s"`, formatDuration(time.Now().Sub(startTime)), pURL))
			return false
		}

		p.LastCheck = startTime

		if err := updateProxyInDB(app.postgresPool, p); err != nil {
			app.loki.error(errors.WithStack(err))
			return false
		}

		app.loki.info(fmt.Sprintf(`Done in %s; proxy is active "%s"`, formatDuration(time.Now().Sub(startTime)), pURL))
		return true
	}

	queueSource.handleStream(handler)
}
