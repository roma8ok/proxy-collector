package main

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path/filepath"

	netProxy "golang.org/x/net/proxy"
)

type ProxyType int

const (
	proxyTypeHTTP ProxyType = iota
	proxyTypeHTTPS
	proxyTypeSOCKS
)

// verbose converts ProxyType int to verbose string.
func (t *ProxyType) verbose() string {
	switch *t {
	case proxyTypeHTTP:
		return "http"
	case proxyTypeHTTPS:
		return "https"
	case proxyTypeSOCKS:
		return "socks"
	}
	return ""
}

// possibleForProxySourceURL checks proxy url for being a proxy.
func possibleForProxySourceURL(u string, forbiddenDomains []string) bool {
	uParsed, err := url.Parse(u)
	if err != nil {
		return false
	}

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

func connectToProxy(p string) error {
	conn, err := net.DialTimeout("tcp", p, requestTimeout)
	if err == nil {
		conn.Close()
	}
	return err
}

func sendHTTPCheckRequest(p, ipAPIURL string) (ip string, err error) {
	proxyURL, err := url.Parse(p)
	if err != nil {
		return
	}

	tr := &http.Transport{
		DisableKeepAlives: true,
		Proxy:             http.ProxyURL(proxyURL),
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Timeout:   requestTimeout,
		Transport: tr,
	}

	res, err := client.Get(ipAPIURL)
	if err != nil {
		return
	}

	defer res.Body.Close()

	ipBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	ip = string(ipBytes)
	return
}

func sendSOCKSCheckRequest(ipAPIURL, p string) (ip string, err error) {
	d := net.Dialer{
		Timeout:   requestTimeout,
		KeepAlive: requestTimeout,
	}

	dialer, err := netProxy.SOCKS5("tcp", p, nil, &d)
	if err != nil {
		return
	}

	client := &http.Client{
		Timeout: requestTimeout,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			Dial:              dialer.Dial,
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		},
	}
	res, err := client.Get(ipAPIURL)
	if err != nil {
		return
	}

	defer res.Body.Close()

	ipBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	ip = string(ipBytes)
	return
}

func externalIP(ipAPIURL string) (string, error) {
	tr := &http.Transport{
		DisableKeepAlives: true,
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Timeout:   requestTimeout,
		Transport: tr,
	}

	res, err := client.Get(ipAPIURL)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	ip, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(ip), nil
}

func checkProxy(p, ipAPIURL string) (pType ProxyType, anonymous bool, err error) {
	if err := connectToProxy(p); err != nil {
		return 0, false, err
	}

	extIP, err := externalIP(ipAPIURL)
	if err != nil {
		return 0, false, err
	}

	if ip, err := sendHTTPCheckRequest("http://"+p, ipAPIURL); err == nil {
		return proxyTypeHTTP, extIP != ip, nil
	}
	if ip, err := sendHTTPCheckRequest("https://"+p, ipAPIURL); err == nil {
		return proxyTypeHTTPS, extIP != ip, nil
	}
	if ip, err := sendSOCKSCheckRequest("socks://"+p, ipAPIURL); err == nil {
		return proxyTypeSOCKS, extIP != ip, nil
	}

	return 0, false, errors.New("checkProxy error")
}
