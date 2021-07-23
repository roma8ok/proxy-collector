package main

import "strings"

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
	suffixes := []string{".css", ".ico", ".png", ".jpg", ".jpeg", ".webp", ".gif", ".js", ".ts", ".woff", ".woff2"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(u, suffix) {
			return false
		}
	}
	return true
}
