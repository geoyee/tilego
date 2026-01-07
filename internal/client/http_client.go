// Package client 提供HTTP客户端相关功能
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
	// MaxIdleConns 最大空闲连接数
	MaxIdleConns = 200
	// MaxIdleConnsPerHost 每个主机的最大空闲连接数
	MaxIdleConnsPerHost = 50
	// MaxConnsPerHost 每个主机的最大连接数
	MaxConnsPerHost = 50
	// IdleConnTimeout 空闲连接超时时间
	IdleConnTimeout = 30 * time.Second
)

// HTTPClient HTTP客户端封装
type HTTPClient struct {
	client *http.Client
	config *Config
}

// Config HTTP客户端配置
type Config struct {
	Timeout   int
	ProxyURL  string
	UseHTTP2  bool
	KeepAlive bool
	UserAgent string
}

// NewHTTPClient 创建新的HTTP客户端
func NewHTTPClient(config *Config) *HTTPClient {
	return &HTTPClient{
		config: config,
		client: createHTTPClient(config),
	}
}

// createHTTPClient 创建HTTP客户端
func createHTTPClient(config *Config) *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     config.UseHTTP2,
		MaxIdleConns:          MaxIdleConns,
		MaxIdleConnsPerHost:   MaxIdleConnsPerHost,
		MaxConnsPerHost:       MaxConnsPerHost,
		IdleConnTimeout:       IdleConnTimeout,
		TLSHandshakeTimeout:   15 * time.Second,
		ExpectContinueTimeout: 5 * time.Second,
		DisableCompression:    true,
	}

	// 设置代理
	if config.ProxyURL != "" {
		log.Printf("正在配置代理: %s", config.ProxyURL)
		proxyURL, err := url.Parse(config.ProxyURL)
		if err != nil {
			log.Printf("代理URL解析失败: %v", err)
		} else {
			transport.Proxy = http.ProxyURL(proxyURL)
			log.Printf("代理设置成功: %s", proxyURL.Host)
		}
	}

	if config.UseHTTP2 {
		http2.ConfigureTransport(transport)
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

// TestProxyConnection 测试代理连接
func (c *HTTPClient) TestProxyConnection() error {
	if c.config.ProxyURL == "" {
		return nil
	}

	testURL := "http://httpbin.org/ip"

	transport := &http.Transport{}
	if c.config.ProxyURL != "" {
		proxyURL, err := url.Parse(c.config.ProxyURL)
		if err != nil {
			return err
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

// GetClient 获取HTTP客户端
func (c *HTTPClient) GetClient() *http.Client {
	return c.client
}

// SafeCloseResponse 安全关闭响应体
func SafeCloseResponse(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
