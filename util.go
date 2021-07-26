package main

import (
	"net/url"
	"path/filepath"
)

func uniqueArrayOfValues(arr []string) (unique []string) {
	set := make(map[string]struct{})
	for _, val := range arr {
		if _, ok := set[val]; !ok {
			set[val] = struct{}{}
			unique = append(unique, val)
		}
	}
	return
}

func isExist(val string, arr []string) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}

func possibleForProxySourceURL(u string) bool {
	uParsed, err := url.Parse(u)
	if err != nil {
		return false
	}

	forbiddenDomains := []string{
		"dictionary.cambridge.org", "cambridge.org", "www.cambridge.org",
		"microsoft.com", "www.microsoft.com",
		"theunfolder.com", "www.theunfolder.com",
		"zen.yandex.ru"}
	if isExist(uParsed.Hostname(), forbiddenDomains) {
		return false
	}

	uParsed.RawQuery = ""
	uParsed.RawFragment = ""
	forbiddenExts := []string{".css", ".ico", ".png", ".jpg", ".jpeg", ".webp", ".gif", ".js", ".ts", ".woff", ".woff2"}
	if isExist(filepath.Ext(uParsed.String()), forbiddenExts) {
		return false
	}

	return true
}

func urlsHaveSameDomain(u1, u2 string) bool {
	u1Parsed, err := url.Parse(u1)
	if err != nil {
		return false
	}
	u2Parsed, err := url.Parse(u2)
	if err != nil {
		return false
	}
	if u1Parsed.Hostname() == u2Parsed.Hostname() {
		return true
	}
	return false
}
