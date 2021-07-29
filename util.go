package main

import (
	"math/rand"
	"net/url"
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

func randomElementFromArray(arr []string) string {
	if len(arr) == 0 {
		return ""
	}
	return arr[rand.Intn(len(arr))]
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
