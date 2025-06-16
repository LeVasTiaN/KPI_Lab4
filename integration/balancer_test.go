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

func BenchmarkBalancer(b *testing.B) {
	// TODO: Реалізуйте інтеграційний бенчмарк для балансувальникка.
}
