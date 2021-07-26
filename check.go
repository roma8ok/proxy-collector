package main

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path/filepath"

	"golang.org/x/net/proxy"
)

type proxyType int

const (
	proxyTypeHTTP proxyType = iota
	proxyTypeHTTPS
	proxyTypeSOCKS
)

func (t *proxyType) verbose() string {
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

func possibleForProxySourceURL(u string) bool {
	uParsed, err := url.Parse(u)
	if err != nil {
		return false
	}

	forbiddenDomains := []string{
		"dictionary.cambridge.org", "cambridge.org", "www.cambridge.org",
		"microsoft.com", "www.microsoft.com",
		"theunfolder.com", "www.theunfolder.com",
		"zen.yandex.ru",
		"avast.com", "www.avast.com",
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

func sendHTTPCheckRequest(p string) (ip string, err error) {
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

func sendSOCKSCheckRequest(p string) (ip string, err error) {
	d := net.Dialer{
		Timeout:   requestTimeout,
		KeepAlive: requestTimeout,
	}

	dialer, err := proxy.SOCKS5("tcp", p, nil, &d)
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

func externalIP() (string, error) {
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

func checkProxy(p string) (pType proxyType, anonymous bool, err error) {
	if err := connectToProxy(p); err != nil {
		return 0, false, err
	}

	extIP, err := externalIP()
	if err != nil {
		return 0, false, err
	}

	if ip, err := sendHTTPCheckRequest("http://" + p); err == nil {
		return proxyTypeHTTP, extIP != ip, nil
	}
	if ip, err := sendHTTPCheckRequest("https://" + p); err == nil {
		return proxyTypeHTTPS, extIP != ip, nil
	}
	if ip, err := sendSOCKSCheckRequest("socks://" + p); err == nil {
		return proxyTypeSOCKS, extIP != ip, nil
	}

	return 0, false, errors.New("checkProxy error")
}
