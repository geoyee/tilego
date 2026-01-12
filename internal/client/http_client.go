// Package client provides HTTP client functionality for tile downloads.
package client

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/http2"
)

const (
	MaxIdleConns        = 500
	MaxIdleConnsPerHost = 100
	MaxConnsPerHost     = 100
	IdleConnTimeout     = 60 * time.Second
	DialTimeout         = 30 * time.Second
	KeepAliveDuration   = 30 * time.Second
	TLSHandshakeTimeout = 15 * time.Second
)

type HTTPClient struct {
	client *http.Client
	config *Config
}

type Config struct {
	Timeout   int
	ProxyURL  string
	UseHTTP2  bool
	KeepAlive bool
	UserAgent string
}

func NewHTTPClient(config *Config) *HTTPClient {
	return &HTTPClient{
		config: config,
		client: createHTTPClient(config),
	}
}

func createHTTPClient(config *Config) *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   DialTimeout,
			KeepAlive: KeepAliveDuration,
		}).DialContext,
		ForceAttemptHTTP2:   config.UseHTTP2,
		MaxIdleConns:        MaxIdleConns,
		MaxIdleConnsPerHost: MaxIdleConnsPerHost,
		MaxConnsPerHost:     MaxConnsPerHost,
		IdleConnTimeout:     IdleConnTimeout,
		TLSHandshakeTimeout: TLSHandshakeTimeout,
		DisableCompression:  true,
	}

	if config.ProxyURL != "" {
		log.Printf("Configuring proxy: %s", config.ProxyURL)
		proxyURL, err := url.Parse(config.ProxyURL)
		if err != nil {
			log.Printf("Failed to parse proxy URL: %v", err)
		} else {
			transport.Proxy = http.ProxyURL(proxyURL)
			log.Printf("Proxy configured successfully: %s", proxyURL.Host)
		}
	}

	if config.UseHTTP2 {
		if err := http2.ConfigureTransport(transport); err != nil {
			log.Printf("Failed to configure HTTP/2: %v", err)
		}
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
	}

	transport.TLSClientConfig = tlsConfig

	return &http.Client{
		Transport: transport,
		Timeout:   time.Duration(config.Timeout) * time.Second,
	}
}

func (c *HTTPClient) TestProxyConnection() error {
	if c.config.ProxyURL == "" {
		return nil
	}

	testURL := "http://httpbin.org/ip"

	transport := &http.Transport{}
	proxyURL, err := url.Parse(c.config.ProxyURL)
	if err != nil {
		return fmt.Errorf("failed to parse proxy URL: %w", err)
	}
	transport.Proxy = http.ProxyURL(proxyURL)

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("proxy connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("proxy returned HTTP %d", resp.StatusCode)
	}

	return nil
}

func (c *HTTPClient) GetClient() *http.Client {
	return c.client
}

func SafeCloseResponse(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
