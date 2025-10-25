package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"
)

type RequestPayload struct {
	URL string `json:"url"`
}

const (
	maxResponseSize = 2 * 1024 * 1024 // 2MB
	serverAddr      = ":8080"
)

// --- Utility functions ---

func writeJSONError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func isAllowedURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	host := parsed.Hostname()
	ips, err := net.LookupIP(host)
	if err != nil {
		return false
	}

	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() {
			return false
		}
	}
	return true
}

// --- Core handler ---

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		writeJSONError(w, "content-type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	var payload RequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if payload.URL == "" {
		writeJSONError(w, "missing url", http.StatusBadRequest)
		return
	}

	if !isAllowedURL(payload.URL) {
		writeJSONError(w, "access to private/internal URLs not allowed", http.StatusForbidden)
		return
	}

	client := &http.Client{
		Transport: &http.Transport{Proxy: nil},
		Timeout:   10 * time.Second,
	}

	req, err := http.NewRequest(r.Method, payload.URL, nil)
	if err != nil {
		writeJSONError(w, "failed to create request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Proxying request to: %s", payload.URL)
	resp, err := client.Do(req)
	if err != nil {
		writeJSONError(w, "failed to fetch url: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy headers and status
	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.Header().Set("Access-Control-Allow-Origin", "*") // Optional CORS
	w.WriteHeader(resp.StatusCode)

	// Limit response size for safety
	io.CopyN(w, resp.Body, maxResponseSize)
}

// --- Entry point ---

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/proxy", proxyHandler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	log.Printf("✅ Proxy server started on %s", serverAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("❌ Server error: %v", err)
	}
}
