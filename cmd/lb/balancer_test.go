package main

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashClientIP(t *testing.T) {
	ip1 := "192.168.1.1"
	hash1 := hashClientIP(ip1)
	hash2 := hashClientIP(ip1)
	assert.Equal(t, hash1, hash2, "Hash should be consistent for same IP")

	ip2 := "192.168.1.2"
	hash3 := hashClientIP(ip2)
	assert.NotEqual(t, hash1, hash3, "Different IPs should produce different hashes")

	ip3 := "10.0.0.1"
	hash4 := hashClientIP(ip3)
	assert.NotEqual(t, hash1, hash4, "Hash should be different for different IP")
	assert.NotEqual(t, hash3, hash4, "Hash should be different for different IP")
}

func TestGetClientIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100")
	req.RemoteAddr = "10.0.0.1:12345"

	clientIP := getClientIP(req)
	assert.Equal(t, "192.168.1.100", clientIP, "Should extract IP from X-Forwarded-For")

	req.Header.Set("X-Forwarded-For", "192.168.1.100, 10.0.0.1, 172.16.0.1")
	clientIP = getClientIP(req)
	assert.Equal(t, "192.168.1.100", clientIP, "Should take first IP from X-Forwarded-For")

	req.Header.Del("X-Forwarded-For")
	req.Header.Set("X-Real-IP", "192.168.1.200")
	clientIP = getClientIP(req)
	assert.Equal(t, "192.168.1.200", clientIP, "Should extract IP from X-Real-IP")

	req.Header.Del("X-Real-IP")
	req.RemoteAddr = "192.168.1.50:12345"
	clientIP = getClientIP(req)
	assert.Equal(t, "192.168.1.50", clientIP, "Should extract IP from RemoteAddr")
}
