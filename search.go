package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"

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

func findSiteURLsFromDDG(html []byte) (urls []string) {
	reABody := regexp.MustCompile(`<a class="result__url" href=".+?">`)
	linkBodies := reABody.FindAll(html, -1)

	for _, linkBody := range linkBodies {
		href := strings.TrimPrefix(string(linkBody), `<a class="result__url" href="`)
		href = strings.TrimSuffix(href, `">`)
		urls = append(urls, href)
	}

	return set(urls)
}

func findURLsFromHTML(html []byte) []string {
	re := regexp.MustCompile(`https?://(www\.)?[-a-zA-Z0-9@:%._+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_+.~#?&/=]*)`)
	urls := re.FindAll(html, -1)

	return set(convertBytesToStringSlice(urls))
}

func findProxiesFromHTML(html []byte) []string {
	wholeProxies := findWholeProxies(html)
	tableProxy := findProxiesFromTable(html)
	proxies := append(wholeProxies, tableProxy...)

	return set(proxies)
}

func findWholeProxies(in []byte) []string {
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
