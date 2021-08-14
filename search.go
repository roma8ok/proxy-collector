package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"

	"github.com/pkg/errors"
)

// makeDDGSearchURL creates a URL to send a query request to DuckDuckGo.
func makeDDGSearchURL(query string) string {
	u := url.URL{
		Scheme: "https",
		Host:   "html.duckduckgo.com",
		Path:   "html",
	}
	q := u.Query()
	q.Set("q", query)
	u.RawQuery = q.Encode()

	return u.String()
}

// findSiteURLsFromDDG returns urls from html.
// findSiteURLsFromDDG only works correctly with DuckDuckGo html pages.
// URL of this page, for example: https://html.duckduckgo.com/html?q=query
func findSiteURLsFromDDG(html []byte) (urls []string) {
	reABody := regexp.MustCompile(`<a class="result__url" href=".+?">`)
	linkBodies := reABody.FindAll(html, -1)

	prefix := []byte(`<a class="result__url" href="`)
	suffix := []byte(`">`)
	for _, linkBody := range linkBodies {
		href := bytes.TrimPrefix(linkBody, prefix)
		href = bytes.TrimSuffix(href, suffix)
		urls = append(urls, string(href))
	}

	return set(urls)
}

// findURLs returns urls from input data.
// findURLs searches URLs with protocol only (http or https).
func findURLs(in []byte) []string {
	re := regexp.MustCompile(`https?://(www\.)?[-a-zA-Z0-9@:%._+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_+.~#?&/=]*)`)
	urls := re.FindAll(in, -1)

	return set(convertBytesToStringSlice(urls))
}

func findProxiesFromHTML(html []byte) []string {
	wholeProxies := findProxiesHostPort(html)
	tableProxy := findProxiesFromTable(html)
	proxies := append(wholeProxies, tableProxy...)

	return set(proxies)
}

// findProxiesHostPort returns proxies from input data.
// findProxiesHostPort searches proxies in format host:port only.
func findProxiesHostPort(in []byte) []string {
	re := regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b:\d{2,5}`)
	return convertBytesToStringSlice(re.FindAll(in, -1))
}

func findProxiesFromTable(in []byte) (proxies []string) {
	reTR := regexp.MustCompile(`<tr>.+?</tr>`)
	trs := reTR.FindAll(in, -1)

	reTDs := regexp.MustCompile(`<td.*?>\b((?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?))\b</td>.*?<td.*?>(\d{2,5})</td>`)
	for _, tr := range trs {
		tds := reTDs.FindSubmatch(tr)

		if len(tds) == 3 {
			proxies = append(proxies, fmt.Sprintf("%s:%s", string(tds[1]), string(tds[2])))
		}
	}

	return
}

// sendGetRequest send GET request through proxy and returns the response body and an error.
// If proxyURL is equal empty string, proxy is not used.
func sendGetRequest(logger Logger, toURL, proxyURL string, headers map[string]string) ([]byte, error) {
	tr := &http.Transport{
		DisableKeepAlives: true,
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
	}

	if proxyURL != "" {
		p, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
		tr.Proxy = http.ProxyURL(p)
	}

	client := &http.Client{
		Timeout:   requestTimeout,
		Transport: tr,
	}

	req, err := http.NewRequest(http.MethodGet, toURL, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			logger.error(errors.WithStack(err))
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, errStatusCodeNotOK
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
