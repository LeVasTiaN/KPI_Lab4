package main

import (
	"context"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/roman-mazur/architecture-practice-4-template/httptools"
	"github.com/roman-mazur/architecture-practice-4-template/signal"
)

var (
	port       = flag.Int("port", 8090, "load balancer port")
	timeoutSec = flag.Int("timeout-sec", 3, "request timeout time in seconds")
	https      = flag.Bool("https", false, "whether backends support HTTPs")

	traceEnabled = flag.Bool("trace", false, "whether to include tracing information into responses")
)

//test
var (
	timeout     = time.Duration(*timeoutSec) * time.Second
	serversPool = []string{
		"server1:8080",
		"server2:8080",
		"server3:8080",
	}
	healthyServers []string
	serversMutex   sync.RWMutex
)

func scheme() string {
	if *https {
		return "https"
	}
	return "http"
}

func health(dst string) bool {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s://%s/health", scheme(), dst), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	if resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

func hashClientIP(clientIP string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(clientIP))
	return h.Sum32()
}

func getClientIP(r *http.Request) string {
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		ips := strings.Split(forwardedFor, ",")
		return strings.TrimSpace(ips[0])
	}

	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	clientIP := r.RemoteAddr
	if idx := strings.LastIndex(clientIP, ":"); idx != -1 {
		clientIP = clientIP[:idx]
	}
	return clientIP
}

func selectServer(clientIP string) (string, error) {
	serversMutex.RLock()
	defer serversMutex.RUnlock()

	if len(healthyServers) == 0 {
		return "", fmt.Errorf("no healthy servers available")
	}

	serverIndex := int(hashClientIP(clientIP)) % len(healthyServers)
	return healthyServers[serverIndex], nil
}

func updateHealthyServers() {
	var healthy []string
	for _, server := range serversPool {
		if health(server) {
			healthy = append(healthy, server)
		}
	}

	serversMutex.Lock()
	healthyServers = healthy
	serversMutex.Unlock()

	log.Printf("Healthy servers: %v", healthy)
}

func forward(dst string, rw http.ResponseWriter, r *http.Request) error {
	ctx, _ := context.WithTimeout(r.Context(), timeout)
	fwdRequest := r.Clone(ctx)
	fwdRequest.RequestURI = ""
	fwdRequest.URL.Host = dst
	fwdRequest.URL.Scheme = scheme()
	fwdRequest.Host = dst

	resp, err := http.DefaultClient.Do(fwdRequest)
	if err == nil {
		for k, values := range resp.Header {
			for _, value := range values {
				rw.Header().Add(k, value)
			}
		}
		if *traceEnabled {
			rw.Header().Set("lb-from", dst)
		}
		log.Println("fwd", resp.StatusCode, resp.Request.URL)
		rw.WriteHeader(resp.StatusCode)
		defer resp.Body.Close()
		_, err := io.Copy(rw, resp.Body)
		if err != nil {
			log.Printf("Failed to write response: %s", err)
		}
		return nil
	} else {
		log.Printf("Failed to get response from %s: %s", dst, err)
		rw.WriteHeader(http.StatusServiceUnavailable)
		return err
	}
}

func main() {
	flag.Parse()

	updateHealthyServers()

	for _, server := range serversPool {
		server := server
		go func() {
			for range time.Tick(10 * time.Second) {
				isHealthy := health(server)
				log.Println(server, "healthy:", isHealthy)
				updateHealthyServers()
			}
		}()
	}

	frontend := httptools.CreateServer(*port, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)

		selectedServer, err := selectServer(clientIP)
		if err != nil {
			log.Printf("Error selecting server: %s", err)
			rw.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		log.Printf("Client %s -> Server %s", clientIP, selectedServer)
		forward(selectedServer, rw, r)
	}))

	log.Println("Starting load balancer...")
	log.Printf("Tracing support enabled: %t", *traceEnabled)
	frontend.Start()
	signal.WaitForTerminationSignal()
}
