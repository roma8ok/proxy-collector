package main

import "testing"

func TestMakeDDGSearchURL(t *testing.T) {
	rightQueryURL := "https://html.duckduckgo.com/html?q=proxy+list"

	query := makeDDGSearchURL("proxy list")
	if query != rightQueryURL {
		t.Errorf(`makeDDGSearchURL("proxy list") = %s, expected %s`, query, rightQueryURL)
	}
}
