package integration

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 3 * time.Second,
}

func TestBalancer(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
	if err != nil {
		t.Error(err)
	}
	t.Logf("response from [%s]", resp.Header.Get("lb-from"))

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %d", resp.StatusCode)
	}

	lbFrom := resp.Header.Get("lb-from")
	if lbFrom == "" {
		t.Error("Missing lb-from header")
	}

	resp.Body.Close()
}

func TestClientIPConsistency(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	var servers []string

	for i := 0; i < 5; i++ {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/some-data", baseAddress), nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("X-Forwarded-For", "192.168.1.100")

		resp, err := client.Do(req)
		if err != nil {
			t.Error(err)
			continue
		}

		lbFrom := resp.Header.Get("lb-from")
		servers = append(servers, lbFrom)
		resp.Body.Close()

		t.Logf("Request %d: server %s", i+1, lbFrom)
	}

	if len(servers) > 0 {
		firstServer := servers[0]
		for i, server := range servers {
			if server != firstServer {
				t.Errorf("Request %d went to different server: expected %s, got %s", i+1, firstServer, server)
			}
		}
		t.Logf("All requests consistently routed to: %s", firstServer)
	}
}

func TestMultipleClientsDistribution(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	clientIPs := []string{
		"192.168.1.1",
		"192.168.1.2",
		"192.168.1.3",
		"10.0.0.1",
		"10.0.0.2",
		"172.16.0.1",
		"172.16.0.2",
		"203.0.113.1",
	}

	serverUsage := make(map[string]int)

	for _, clientIP := range clientIPs {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/some-data", baseAddress), nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("X-Forwarded-For", clientIP)

		resp, err := client.Do(req)
		if err != nil {
			t.Error(err)
			continue
		}

		lbFrom := resp.Header.Get("lb-from")
		serverUsage[lbFrom]++
		resp.Body.Close()

		t.Logf("Client %s -> Server %s", clientIP, lbFrom)
	}

	if len(serverUsage) < 2 {
		t.Errorf("Expected distribution across multiple servers, only used: %v", serverUsage)
	}

	t.Logf("Server usage distribution: %v", serverUsage)
}

func TestLoadBalancerHealth(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	for i := 0; i < 10; i++ {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		if err != nil {
			t.Errorf("Request %d failed: %v", i+1, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Request %d returned status %d", i+1, resp.StatusCode)
		}

		lbFrom := resp.Header.Get("lb-from")
		if lbFrom == "" {
			t.Errorf("Request %d missing lb-from header", i+1)
		}

		resp.Body.Close()
	}
}

func BenchmarkBalancer(b *testing.B) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		b.Skip("Integration test is not enabled")
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		if err != nil {
			b.Error(err)
			continue
		}
		resp.Body.Close()
	}
}

func BenchmarkBalancerParallel(b *testing.B) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		b.Skip("Integration test is not enabled")
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
			if err != nil {
				b.Error(err)
				continue
			}
			resp.Body.Close()
		}
	})
}
