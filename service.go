package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

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
		queries := []string{"proxy list", "proxy", "proxies", "прокси", "список прокси"}
		query = randomElementFromSlice(queries)
	}

	searchURL := makeDDGSearchURL(query)

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
		return errors.New(fmt.Sprintf(`sendRequest("%s", "%s", headers); resp.StatusCode = %d`, searchURL, proxyURL, resp.StatusCode))
	}
	body, err := ioutil.ReadAll(resp.Body)
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

	app.loki.info(fmt.Sprintf(`Done in %s; found/added: %d/%d urls`, formatDuration(time.Now().Sub(startTime)), len(urls), addedUrls))

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
		resp, err := sendRequest(proxySource, proxyURL, headers)
		if err != nil {
			app.loki.debug(fmt.Sprintf(`sendRequest("%s", "%s", headers); err = %s`, proxySource, proxyURL, errors.WithStack(err)))
			return false
		}
		defer func(Body io.ReadCloser) {
			if err := Body.Close(); err != nil {
				app.loki.error(errors.WithStack(err))
			}
		}(resp.Body)

		if resp.StatusCode != http.StatusOK {
			app.loki.debug(fmt.Sprintf(`sendRequest("%s", "%s", headers); resp.StatusCode = %d`, proxySource, proxyURL, resp.StatusCode))
			return false
		}
		html, err := ioutil.ReadAll(resp.Body)
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

		urlsFromHTML := findURLsFromHTML(html)
		addedURLs := 0
		if len(proxiesFromHTML) > 0 {
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
			formatDuration(time.Now().Sub(startTime)), len(proxiesFromHTML), addedProxies, len(urlsFromHTML), addedURLs, proxySource))

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

func processRawProxy(app App, queueSource *RabbitMQSession) {
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
