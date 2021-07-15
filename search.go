package main

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
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

func sendSearchRequest(searchURL, proxyURL, userAgent string) (*http.Response, error) {
	tr := &http.Transport{
		IdleConnTimeout: 5 * time.Second,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if proxyURL != "" {
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
		tr.Proxy = http.ProxyURL(proxy)
	}

	client := &http.Client{Transport: tr}

	req, err := http.NewRequest(http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "text/html; charset=UTF-8")
	req.Header.Add("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func findSiteURLsFromDDG(htmlBody string) (urls []string) {
	reABody := regexp.MustCompile(`<a class="result__url" href=".+?">`)
	linkBodies := reABody.FindAllString(htmlBody, -1)

	for _, linkBody := range linkBodies {
		href := strings.TrimPrefix(linkBody, `<a class="result__url" href="`)
		href = strings.TrimSuffix(href, `">`)
		urls = append(urls, href)
	}

	return uniqueArrayOfValues(urls)
}
