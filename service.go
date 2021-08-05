package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

func fillSearchQueries(app App, queueDest *RabbitMQSession, fromProxies bool) error {
	startTime := time.Now()

	var q string
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
		q = randomElementFromArray(proxies)
	} else {
		queries := []string{"proxy list", "proxy", "proxies", "прокси", "список прокси"}
		q = randomElementFromArray(queries)
	}

	if err := queueDest.Push([]byte(q)); err != nil {
		return err
	}

	app.loki.info(fmt.Sprintf(`Done in %s; added "%s"`, formatDuration(time.Now().Sub(startTime)), q))
	return nil
}

func sendSearchBodyFromDDGToQueue(app App, queueSource, queueDest *RabbitMQSession, proxyURL, userAgent string) {
	var handler RabbitMQStreamHandler = func(in []byte) (ack bool) {
		startTime := time.Now()

		searchURL := makeDDGSearchURL(string(in))

		var headers = map[string]string{
			"Content-Type": "text/html; charset=UTF-8",
			"User-Agent":   userAgent,
		}

		resp, err := sendRequest(searchURL, proxyURL, headers)
		if err != nil {
			app.loki.debug(fmt.Sprintf(`sendRequest("%s", "%s", headers); err = %s`, searchURL, proxyURL, errors.WithStack(err)))
			return false
		}
		defer func(Body io.ReadCloser) {
			if err := Body.Close(); err != nil {
				app.loki.error(errors.WithStack(err))
			}
		}(resp.Body)

		if resp.StatusCode != http.StatusOK {
			app.loki.debug(fmt.Sprintf(`sendRequest("%s", "%s", headers); resp.StatusCode = %d`, searchURL, proxyURL, resp.StatusCode))
			return false
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			app.loki.debug(fmt.Sprintf(`sendRequest("%s", "%s", headers); can't read resp.Body`, searchURL, proxyURL))
			return false
		}

		if err := queueDest.Push(body); err != nil {
			app.loki.error(errors.WithStack(err))
			return false
		}

		app.loki.info(fmt.Sprintf(`Done in %s; added body from "%s"`, formatDuration(time.Now().Sub(startTime)), searchURL))
		return true
	}

	queueSource.handleStream(handler)
}

func processSearchBodyFromDDG(app App, queueSource, queueDest *RabbitMQSession) {
	var handler RabbitMQStreamHandler = func(in []byte) (ack bool) {
		startTime := time.Now()

		urls := findSiteURLsFromDDG(string(in))
		addedUrls := 0
		for _, u := range urls {
			changeType, err := app.rdbForSites.set(u)
			if err != nil {
				app.loki.error(errors.WithStack(err))
				return false
			}
			switch changeType {
			case redisChangeAdd, redisChangeUpdate:
				if err := queueDest.Push([]byte(u)); err != nil {
					app.loki.error(errors.WithStack(err))
					return false
				}
				addedUrls += 1
			}
		}

		app.loki.info(fmt.Sprintf(`Done in %s; handled %d urls, added %d urls`, formatDuration(time.Now().Sub(startTime)), len(urls), addedUrls))
		return true
	}

	queueSource.handleStream(handler)
}

func sendHTMLFromProxySourceToQueue(app App, queueSource, queueDest *RabbitMQSession) {
	var handler RabbitMQStreamHandler = func(in []byte) (ack bool) {
		startTime := time.Now()

		var headers = map[string]string{
			"Content-Type": "text/html; charset=UTF-8",
			"User-Agent":   userAgent,
		}

		u := string(in)
		resp, err := sendRequest(u, proxyURL, headers)
		if err != nil {
			app.loki.debug(fmt.Sprintf(`sendRequest("%s", "%s", headers); err = %s`, u, proxyURL, errors.WithStack(err)))
			return false
		}
		defer func(Body io.ReadCloser) {
			if err := Body.Close(); err != nil {
				app.loki.error(errors.WithStack(err))
			}
		}(resp.Body)

		if resp.StatusCode != http.StatusOK {
			app.loki.debug(fmt.Sprintf(`sendRequest("%s", "%s", headers); resp.StatusCode = %d`, u, proxyURL, resp.StatusCode))
			return false
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			app.loki.debug(fmt.Sprintf(`sendRequest("%s", "%s", headers); can't read resp.Body`, u, proxyURL))
			return false
		}
		hPayload := htmlPayload{
			HTML:    string(body),
			FromURL: u,
		}
		hPayloadJSON, err := json.Marshal(hPayload)
		if err != nil {
			app.loki.error(fmt.Errorf(`can't json.Marshal(payload) from url "%s": %s`, u, errors.WithStack(err)))
			return false
		}

		if err := queueDest.Push(hPayloadJSON); err != nil {
			app.loki.error(errors.WithStack(err))
			return false
		}

		app.loki.info(fmt.Sprintf(`Done in %s; added body from "%s"`, formatDuration(time.Now().Sub(startTime)), u))
		return true
	}

	queueSource.handleStream(handler)
}

func processSourceHTML(app App, queueSource, queueDestRawProxies, queueDestProxySources *RabbitMQSession) {
	forbiddenDomains, err := blacklistDomains(app.postgresPool)
	if err != nil {
		app.loki.error(errors.WithStack(err))
	}

	var handler RabbitMQStreamHandler = func(in []byte) (ack bool) {
		startTime := time.Now()

		var hPayload htmlPayload
		if err := json.Unmarshal(in, &hPayload); err != nil {
			app.loki.error(fmt.Errorf(`can't json.Unmarshal(): %s`, errors.WithStack(err)))
			return false
		}

		proxies := findProxiesFromHTML(hPayload.HTML)
		addedProxies := 0
		for _, p := range proxies {
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
					return false
				}
				switch changeType {
				case redisChangeAdd, redisChangeUpdate:
					if err := queueDestProxySources.Push([]byte(u)); err != nil {
						app.loki.error(errors.WithStack(err))
						return false
					}
					addedURLs += 1
				}
			}
		}

		app.loki.info(fmt.Sprintf(`Done in %s; processed "%s"; found/added: %d/%d proxies, %d/%d urls`, formatDuration(time.Now().Sub(startTime)), hPayload.FromURL, len(proxies), addedProxies, len(urls), addedURLs))
		return true
	}

	queueSource.handleStream(handler)
}

func processRawProxy(app App, queueSource *RabbitMQSession) {
	var handler RabbitMQStreamHandler = func(in []byte) (ack bool) {
		startTime := time.Now()

		pURL := string(in)

		pType, anonymous, err := checkProxy(pURL, app.conf.IPAPIURL)
		if err != nil {
			app.loki.info(fmt.Sprintf(`Done in %s; discarded "%s"`, formatDuration(time.Now().Sub(startTime)), pURL))
			return false
		}

		p := Proxy{
			URL:       pURL,
			Active:    true,
			Type:      pType.verbose(),
			Anonymous: anonymous,
			Created:   startTime,
			LastCheck: startTime,
		}

		if err := saveProxyToDB(app.postgresPool, p); err != nil {
			app.loki.error(errors.WithStack(err))
			return false
		}

		app.loki.info(fmt.Sprintf(`Done in %s; added "%s"`, formatDuration(time.Now().Sub(startTime)), pURL))
		return true
	}

	queueSource.handleStream(handler)
}

func fillCheckProxiesQueue(app App, queueDest *RabbitMQSession) error {
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

func processCheckProxy(app App, queueSource *RabbitMQSession) {
	var handler RabbitMQStreamHandler = func(in []byte) (ack bool) {
		startTime := time.Now()

		pURL := string(in)

		pType, anonymous, err := checkProxy(pURL, app.conf.IPAPIURL)
		if err != nil {
			if err := changeProxyToInactive(app.postgresPool, pURL); err != nil {
				app.loki.error(errors.WithStack(err))
				return false
			}
			app.loki.info(fmt.Sprintf(`Done in %s; changed to inactive "%s"`, formatDuration(time.Now().Sub(startTime)), pURL))
			return false
		}

		p := Proxy{
			URL:       pURL,
			Active:    true,
			Type:      pType.verbose(),
			Anonymous: anonymous,
			LastCheck: startTime,
		}

		if err := updateProxyInDB(app.postgresPool, p); err != nil {
			app.loki.error(errors.WithStack(err))
			return false
		}

		app.loki.info(fmt.Sprintf(`Done in %s; updated last check (active) "%s"`, formatDuration(time.Now().Sub(startTime)), pURL))
		return true
	}

	queueSource.handleStream(handler)
}
