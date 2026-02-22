package client

import (
	"net/http"
	"testing"
	"time"
)

func TestNewHTTPClient(t *testing.T) {
	config := &Config{
		Timeout:   30,
		ProxyURL:  "",
		UseHTTP2:  true,
		KeepAlive: true,
		UserAgent: "TestAgent",
	}

	client := NewHTTPClient(config)
	if client == nil {
		t.Fatal("NewHTTPClient returned nil")
	}

	if client.Client == nil {
		t.Error("HTTPClient.Client is nil")
	}

	if client.Config != config {
		t.Error("HTTPClient.Config not set correctly")
	}
}

func TestCreateHTTPClient(t *testing.T) {
	config := &Config{
		Timeout:   60,
		ProxyURL:  "",
		UseHTTP2:  false,
		KeepAlive: true,
		UserAgent: "TestAgent",
	}

	httpClient := CreateHTTPClient(config)
	if httpClient == nil {
		t.Fatal("CreateHTTPClient returned nil")
	}

	expectedTimeout := 60 * time.Second
	if httpClient.Timeout != expectedTimeout {
		t.Errorf("Expected timeout %v, got %v", expectedTimeout, httpClient.Timeout)
	}

	transport, ok := httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Transport is not http.Transport")
	}

	if transport.MaxIdleConns != MaxIdleConns {
		t.Errorf("Expected MaxIdleConns %d, got %d", MaxIdleConns, transport.MaxIdleConns)
	}

	if transport.MaxIdleConnsPerHost != MaxIdleConnsPerHost {
		t.Errorf("Expected MaxIdleConnsPerHost %d, got %d", MaxIdleConnsPerHost, transport.MaxIdleConnsPerHost)
	}
}

func TestCreateHTTPClientWithHTTP2(t *testing.T) {
	config := &Config{
		Timeout:   30,
		UseHTTP2:  true,
		KeepAlive: true,
	}

	httpClient := CreateHTTPClient(config)
	if httpClient == nil {
		t.Fatal("CreateHTTPClient returned nil")
	}

	transport, ok := httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Transport is not http.Transport")
	}

	if !transport.ForceAttemptHTTP2 {
		t.Error("Expected ForceAttemptHTTP2 to be true")
	}
}

func TestCreateHTTPClientWithoutHTTP2(t *testing.T) {
	config := &Config{
		Timeout:   30,
		UseHTTP2:  false,
		KeepAlive: true,
	}

	httpClient := CreateHTTPClient(config)
	if httpClient == nil {
		t.Fatal("CreateHTTPClient returned nil")
	}

	transport, ok := httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Transport is not http.Transport")
	}

	if transport.ForceAttemptHTTP2 {
		t.Error("Expected ForceAttemptHTTP2 to be false")
	}
}

func TestGetClient(t *testing.T) {
	config := &Config{
		Timeout:   30,
		UseHTTP2:  true,
		KeepAlive: true,
	}

	client := NewHTTPClient(config)
	httpClient := client.GetClient()

	if httpClient == nil {
		t.Error("GetClient returned nil")
	}

	if httpClient != client.Client {
		t.Error("GetClient did not return the internal client")
	}
}

func TestConstants(t *testing.T) {
	if MaxIdleConns != 500 {
		t.Errorf("Expected MaxIdleConns 500, got %d", MaxIdleConns)
	}

	if MaxIdleConnsPerHost != 100 {
		t.Errorf("Expected MaxIdleConnsPerHost 100, got %d", MaxIdleConnsPerHost)
	}

	if MaxConnsPerHost != 100 {
		t.Errorf("Expected MaxConnsPerHost 100, got %d", MaxConnsPerHost)
	}

	if IdleConnTimeout != 60*time.Second {
		t.Errorf("Expected IdleConnTimeout 60s, got %v", IdleConnTimeout)
	}

	if DialTimeout != 30*time.Second {
		t.Errorf("Expected DialTimeout 30s, got %v", DialTimeout)
	}

	if KeepAliveDuration != 30*time.Second {
		t.Errorf("Expected KeepAliveDuration 30s, got %v", KeepAliveDuration)
	}

	if TLSHandshakeTimeout != 15*time.Second {
		t.Errorf("Expected TLSHandshakeTimeout 15s, got %v", TLSHandshakeTimeout)
	}
}
