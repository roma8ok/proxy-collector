package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

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

func sendRequest(searchURL, proxyURL string, headers map[string]string) (*http.Response, error) {
	tr := &http.Transport{
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		IdleConnTimeout:       requestTimeout,
		ResponseHeaderTimeout: requestTimeout,
		ExpectContinueTimeout: requestTimeout,
	}

	if proxyURL != "" {
		p, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
		tr.Proxy = http.ProxyURL(p)
	}

	client := &http.Client{Transport: tr}

	req, err := http.NewRequest(http.MethodGet, searchURL, nil)
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

	return resp, nil
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
