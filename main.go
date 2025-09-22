package main

import (
	"encoding/json"
	"io"
	"net/http"
)

type RequestPayload struct {
	URL string `json:"url"`
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	// Parse input JSON
	var payload RequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if payload.URL == "" {
		http.Error(w, "missing url", http.StatusBadRequest)
		return
	}

	// Create an HTTP client that ignores proxy settings
	client := &http.Client{
		Transport: &http.Transport{Proxy: nil},
	}

	// Make the request
	resp, err := client.Get(payload.URL)
	if err != nil {
		http.Error(w, "failed to fetch url: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy headers and status code
	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	io.Copy(w, resp.Body)
}

func main() {
	http.HandleFunc("/proxy", proxyHandler)
	http.ListenAndServe(":8080", nil)
}
