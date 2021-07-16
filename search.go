package main

import (
	"crypto/tls"
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

	for k, v := range headers {
		req.Header.Add(k, v)
	}

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
