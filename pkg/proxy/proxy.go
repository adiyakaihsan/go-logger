package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type Proxy struct {
	backends []string
}

func (p *Proxy) proxyRequest(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// copy r.Body because it is closed after first proxy request done.
	var bodyBytes []byte
	var err error
	if r.Body != nil {
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
	}

	var responses []*http.Response
	// iterate backends, and make requests to each backend then append the resp to responses
	for _, backend := range p.backends {
		targetUrl := fmt.Sprintf("%s%s", backend, r.URL)

		proxyReq, err := http.NewRequest(r.Method, targetUrl, bytes.NewReader(bodyBytes))
		if err != nil {
			return
		}

		for header, values := range r.Header {
			for _, value := range values {
				proxyReq.Header.Add(header, value)
			}
		}

		// Add X-Forwarded headers
		proxyReq.Header.Set("X-Forwarded-Host", r.Host)
		proxyReq.Header.Set("X-Forwarded-Proto", r.URL.Scheme)

		client := &http.Client{}
		resp, err := client.Do(proxyReq)
		if err != nil {
			http.Error(w, "Error sending proxy request", http.StatusBadGateway)
			log.Printf("Error1: %v", err)
			return
		}
		responses = append(responses, resp)
	}
	// write response
	w.WriteHeader(http.StatusOK)
	for _, response := range responses {
		// Copy response body
		_, err := io.Copy(w, response.Body)
		if err != nil {
			log.Printf("Error copying response: %v", err)
		}
		defer response.Body.Close()
	}

}

func Run() {
	router := httprouter.New()

// register backend
	p := &Proxy{
		backends: []string{"http://localhost:8082", "http://localhost:8083"},
	}

	router.POST("/api/v1/log/search", p.proxyRequest)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", "8256"),
		Handler: router,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Printf("HTTP server error: %v", err)
	}
}
