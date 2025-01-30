package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var backends []string

type BackendResponse struct {
	Code    int
	Backend string
}

func forwardRequestToBackend(wg *sync.WaitGroup, client *http.Client, backend string, req *http.Request, responses chan<- BackendResponse) {
	defer wg.Done()

	/* Recreate the request with same Method, path,query and Body, but to the specified backend */

	forwardReqStr := fmt.Sprintf("http://%s%s", backend, req.RequestURI)
	forwardReq, err := http.NewRequest(req.Method, forwardReqStr, req.Body)
	if err != nil {
		log.Printf("Error creating request for %s: %v", backend, err)

		responses <- BackendResponse{500, backend} // Internal server error
		return
	}

	// Override Host with original Host
	forwardReq.Host = req.Host
	// Add original headers
	forwardReq.Header = req.Header.Clone()

	// Send request
	resp, err := client.Do(forwardReq)
	if err != nil {
		log.Printf("Error forwarding to %s: %v", backend, err)
		if os.IsTimeout(err) {
			responses <- BackendResponse{504, backend}
		}
		responses <- BackendResponse{503, backend} // Service unavailable. Probably backend is down.
		return
	}
	defer resp.Body.Close()

	// Read response body
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response from %s: %v", backend, err)
		responses <- BackendResponse{502, backend}
		return
	}

	responses <- BackendResponse{resp.StatusCode, backend}
}

/*
* Take incoming request and make a new request for each backend in a new goroutine.
* In response to origin request send a report sucess or failure with HTTP code.
 */
func handler(w http.ResponseWriter, req *http.Request) {
	client := &http.Client{}
	var wg sync.WaitGroup
	responses := make(chan BackendResponse, len(backends))

	for _, backend := range backends {
		wg.Add(1)
		go forwardRequestToBackend(&wg, client, backend, req, responses)
	}

	wg.Wait()
	close(responses)

	statusCode := http.StatusOK
	for response := range responses {
		if response.Code >= 400 && response.Code != 503 {
			// 503 = backend down: don't catch as an error
			statusCode = response.Code
		}
	}
	w.WriteHeader(statusCode)

	for response := range responses {
		w.Header().Add("X-Multicaster-Backend-Response", fmt.Sprintf("%s=%d", response.Backend, response.Code))
	}
}

/*
* Just print incoming request elements
 */
func debugHandler(w http.ResponseWriter, req *http.Request) {
	backend := "127.0.0.1:8080"
	/* Recreate the request with same Method, path,query and Body, but to the specified backend */
	forwardReqStr := fmt.Sprintf("http://%s%s", backend, req.RequestURI)
	forwardReq, err := http.NewRequest(req.Method, forwardReqStr, req.Body)
	if err != nil {
		log.Printf("Error creating request for %s: %v", backend, err)
		return
	}
	forwardReq.Host = req.Host
	forwardReq.Header = req.Header.Clone()

	fmt.Printf("%#v\n", req)
	fmt.Printf("%#v\n", forwardReq)

}
func main() {
	backendsStr := os.Getenv("BACKENDS")
	listenAddress := os.Getenv("LISTEN")
	if len(listenAddress) == 0 {
		listenAddress = ":8080"
	}
	clientTimeOutStr := os.Getenv("HTTP_CLIENT_TIMEOUT")
	// Default value for http client timeout 10s
	http.DefaultClient.Timeout = 10 * time.Second
	if len(clientTimeOutStr) > 0 {
		clientTimeOut, err := strconv.Atoi(clientTimeOutStr)
		if err == nil {
			http.DefaultClient.Timeout = time.Duration(clientTimeOut) * time.Millisecond
		}
	}
	http.DefaultClient.Timeout = 10 * time.Second

	if len(backendsStr) == 0 {
		log.Println("BACKENDS environment var not defined or empty (BACKENDS=IP:PORT,IP:PORT)")
		log.Println("Running as debugger")
		http.HandleFunc("/", debugHandler)
	} else {
		backends = strings.Split(backendsStr, ",")
		http.HandleFunc("/", handler)
	}
	fmt.Printf("Starting server on %s\n", listenAddress)
	if err := http.ListenAndServe(listenAddress, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
