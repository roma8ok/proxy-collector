package main

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
)

type ProxyType int

const (
	proxyTypeHTTP ProxyType = iota
	proxyTypeHTTPS
	proxyTypeSOCKS
)

type CheckPayload struct {
	success bool
	pType   ProxyType
	ip      string
}

// possibleForProxySourceURL checks proxy url for being a proxy.
func possibleForProxySourceURL(u string) bool {
	uParsed, err := url.Parse(u)
	if err != nil {
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

// checkTCPAddress tries to connect to tcp address.
// checkTCPAddress returns an error if the attempt is failed.
func checkTCPAddress(p string) error {
	conn, err := net.DialTimeout("tcp", p, requestTimeout)
	if err == nil {
		conn.Close()
	}
	return err
}

// sendProxyCheckRequest sends GET HTTP request to ipAPIURL through a proxy.
// sendProxyCheckRequest returns IP from ipAPIURL and an error.
func sendProxyCheckRequest(proxy, ipAPIURL string) (ip string, err error) {
	proxyURL, err := url.Parse(proxy)
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

// externalIP sends GET HTTP request to ipAPIURL.
// externalIP returns IP from ipAPIURL and an error.
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

// checkProxy sends some HTTP requests to find out if the pHostPort is a proxy.
// checkProxy returns Proxy and an error if pHostPort is not proxy.
func checkProxy(pHostPort, hostExtIP, ipAPIURL string) (p Proxy, err error) {
	if err := checkTCPAddress(pHostPort); err != nil {
		return p, err
	}

	ch := make(chan CheckPayload)

	go func() {
		ip, err := sendProxyCheckRequest("http://"+pHostPort, ipAPIURL)
		if err != nil {
			ch <- CheckPayload{
				success: false,
				pType:   proxyTypeHTTP,
				ip:      "",
			}
		} else {
			ch <- CheckPayload{
				success: true,
				pType:   proxyTypeHTTP,
				ip:      ip,
			}
		}
	}()

	go func() {
		ip, err := sendProxyCheckRequest("https://"+pHostPort, ipAPIURL)
		if err != nil {
			ch <- CheckPayload{
				success: false,
				pType:   proxyTypeHTTPS,
				ip:      "",
			}
		} else {
			ch <- CheckPayload{
				success: true,
				pType:   proxyTypeHTTPS,
				ip:      ip,
			}
		}
	}()

	go func() {
		ip, err := sendProxyCheckRequest("socks5://"+pHostPort, ipAPIURL)
		if err != nil {
			ch <- CheckPayload{
				success: false,
				pType:   proxyTypeSOCKS,
				ip:      "",
			}
		} else {
			ch <- CheckPayload{
				success: true,
				pType:   proxyTypeSOCKS,
				ip:      ip,
			}
		}
	}()

	p1, p2, p3 := <-ch, <-ch, <-ch

	successCounter := 0
	anonymousCounter := 0
	for _, cPayload := range []CheckPayload{p1, p2, p3} {
		if !cPayload.success {
			continue
		}
		successCounter += 1

		switch cPayload.pType {
		case proxyTypeHTTP:
			p.HTTP = true
		case proxyTypeHTTPS:
			p.HTTPS = true
		case proxyTypeSOCKS:
			p.SOCKS5 = true
		}

		if hostExtIP != cPayload.ip {
			anonymousCounter += 1
		}
	}

	p.URL = pHostPort
	if successCounter > 0 && successCounter == anonymousCounter {
		p.Anonymous = true
	}
	if successCounter > 0 {
		p.Active = true
		return p, nil
	}

	return p, errors.New("checkProxy error")
}
