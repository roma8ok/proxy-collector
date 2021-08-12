package main

import (
	"fmt"
	"math/rand"
	"net/url"
	"time"
)

// set gets a slice of strings and returns a slice of strings in which each element is not repeated.
func set(sl []string) (uniq []string) {
	s := make(map[string]struct{})
	for _, val := range sl {
		if _, ok := s[val]; !ok {
			s[val] = struct{}{}
			uniq = append(uniq, val)
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

func formatDuration(d time.Duration) string {
	min := int64(d.Minutes())
	sec := int64(d.Seconds()) - min*60
	ms := d.Milliseconds() - min*60*1000 - sec*1000
	return fmt.Sprintf("%02dm:%02ds:%03dms", min, sec, ms)
}

func fillString(in string, maxLen int, filling string) (out string) {
	letters := []rune(in)
	if len(letters) > maxLen {
		out += string(letters[:maxLen])
	} else {
		for l := 0; l < maxLen; l++ {
			if l < len(letters) {
				out += string(letters[l])
			} else {
				out += filling
			}
		}
	}
	return
}
