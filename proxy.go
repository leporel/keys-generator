package main

import (
	"context"
	"fmt"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type DialContext func(ctx context.Context, network, address string) (net.Conn, error)

var re = regexp.MustCompile(`(?m)^(?:(\w+)(?::(\w+))?@)?((?:\d{1,3})(?:\.\d{1,3}){3})(?::(\d{1,5}))?$`)

// NewProxyClient return proxy SOCKS5 client
// if addr empty, will return default http client
// proxyHost example:
//		addr=user:pass@ip_addr:port
//		addr=ip_addr:port
//		addr=host
func NewProxyClient(addr string) (*http.Client, error) {
	baseDialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	var dialContext DialContext

	host := addr
	var auth *proxy.Auth

	if strings.Contains(addr, "@") {
		str := re.FindAllStringSubmatch(addr, -1)

		if len(str) != 1 {
			return nil, fmt.Errorf("proxy addr string is not valid")
		}

		host = str[0][3] + ":" + str[0][4]

		auth = &proxy.Auth{
			User:     str[0][1],
			Password: str[0][2],
		}
	}

	if host != "" {
		dialSocksProxy, err := proxy.SOCKS5("tcp", host, auth, baseDialer)
		if err != nil {
			return nil, fmt.Errorf("error creating SOCKS5 proxy: %s", err)
		}
		if contextDialer, ok := dialSocksProxy.(proxy.ContextDialer); ok {
			dialContext = contextDialer.DialContext
		} else {
			return nil, fmt.Errorf("failed type assertion to DialContext")
		}
	} else {
		dialContext = (baseDialer).DialContext
	}

	httpClient := newClient(dialContext, host)
	return httpClient, nil
}

func newClient(dialContext DialContext, host string) *http.Client {
	var proxyT func(*http.Request) (*url.URL, error)
	if host != "" {
		proxyUrl, _ := url.Parse(host)
		proxyT = http.ProxyURL(proxyUrl)
	}
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy:                 proxyT,
			DialContext:           dialContext,
			MaxIdleConns:          10,
			IdleConnTimeout:       60 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 5 * time.Second,
		},
	}
}
