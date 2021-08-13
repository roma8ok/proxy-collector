package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMakeDDGSearchURL(t *testing.T) {
	rightQueryURL := "https://html.duckduckgo.com/html?q=proxy+list"

	query := makeDDGSearchURL("proxy list")
	if query != rightQueryURL {
		t.Errorf(`makeDDGSearchURL("proxy list") = %s, expected %s`, query, rightQueryURL)
	}
}

func TestSendGetRequest_Success(t *testing.T) {
	const successBody = "OK"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, successBody)
	}))
	defer server.Close()

	logger, _ := newLogger("", "")
	headers := make(map[string]string)

	body, err := sendGetRequest(logger, server.URL, "", headers)
	if err != nil {
		t.Errorf(`sendGetRequest(logger, %s, "", %s) error: %s`, server.URL, headers, err)
	}
	if string(body) != successBody {
		t.Errorf(`sendGetRequest(logger, %s, "", %s) = "%s", expected "%s"`, server.URL, headers, string(body), successBody)
	}
}

func TestSendGetRequest_Headers(t *testing.T) {
	const (
		successBody = "OK"
		failureBody = "wrong_headers"
		headerName  = "rightHeader"
		headerValue = "right_value"
	)
	rightHeaders := map[string]string{
		headerName: headerValue,
	}
	wrongHeaders := map[string]string{
		"wrongHeader": "wrong_value",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(headerName) == headerValue {
			_, _ = io.WriteString(w, successBody)
		} else {
			_, _ = io.WriteString(w, failureBody)
		}
	}))
	defer server.Close()

	logger, _ := newLogger("", "")

	body, err := sendGetRequest(logger, server.URL, "", rightHeaders)
	if err != nil {
		t.Errorf(`sendGetRequest(logger, %s, "", %s) error: %s`, server.URL, rightHeaders, err)
	}
	if string(body) != successBody {
		t.Errorf(`sendGetRequest(logger, %s, "", %s) = "%s", expected "%s"`, server.URL, rightHeaders, string(body), successBody)
	}

	body, err = sendGetRequest(logger, server.URL, "", wrongHeaders)
	if err != nil {
		t.Errorf(`sendGetRequest(logger, %s, "", %s) error: %s`, server.URL, wrongHeaders, err)
	}
	if string(body) != failureBody {
		t.Errorf(`sendGetRequest(logger, %s, "", %s) = "%s", expected "%s"`, server.URL, wrongHeaders, string(body), failureBody)
	}
}

func TestSendGetRequest_StatusCode(t *testing.T) {
	const (
		successBody = "OK"
		failureBody = "wrong_headers"
		headerName  = "rightHeader"
		headerValue = "right_value"
	)
	rightHeaders := map[string]string{
		headerName: headerValue,
	}
	wrongHeaders := map[string]string{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(headerName) == headerValue {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, successBody)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, failureBody)
		}
	}))
	defer server.Close()

	logger, _ := newLogger("", "")

	body, err := sendGetRequest(logger, server.URL, "", rightHeaders)
	if err != nil {
		t.Errorf(`sendGetRequest(logger, %s, "", %s) error: %s`, server.URL, rightHeaders, err)
	}
	if string(body) != successBody {
		t.Errorf(`sendGetRequest(logger, %s, "", %s) = "%s", expected "%s"`, server.URL, rightHeaders, string(body), successBody)
	}

	body, err = sendGetRequest(logger, server.URL, "", wrongHeaders)
	if err != errStatusCodeNotOK {
		t.Errorf(`sendGetRequest(logger, %s, "", %s); expected error: %s, got: %s`, server.URL, wrongHeaders, errStatusCodeNotOK, err)
	}
}
