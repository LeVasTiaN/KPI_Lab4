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
