package main

import (
	"fmt"
	"io"

	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

var backends []string
func forwardRequestToBackend(wg *sync.WaitGroup, client *http.Client, backend string, req *http.Request, responses chan<- int) {
	defer wg.Done()

	/* Recreate the request with same Method, path,query and Body, but to the specified backend */

  forwardReqStr := fmt.Sprintf("http://%s%s",backend,req.RequestURI)
	forwardReq, err := http.NewRequest(req.Method, forwardReqStr, req.Body)
	if err != nil {
		log.Printf("Error creating request for %s: %v", backend, err)

		responses <- 500 // Internal server error
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
		responses <- 502 // bad gateway
		return
	}
	defer resp.Body.Close()


	// Read response body
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response from %s: %v", backend, err)
		responses <- 502
		return
	}

  if resp.StatusCode >= 400 {
	  responses <- resp.StatusCode
    return
  } 
	// Ajouter la réponse dans le canal
	responses <- 200
}

/*
* Take incoming request and make a new request for each backend in a new goroutine.
* In response to origin request send a report sucess or failure with HTTP code.
 */
func handler(w http.ResponseWriter, req *http.Request) {
	client := &http.Client{}
	var wg sync.WaitGroup
	responses := make(chan int, len(backends))

	for _, backend := range backends {
		wg.Add(1)
		go forwardRequestToBackend(&wg, client, backend, req, responses)
	}

	wg.Wait()
	close(responses)

  statusCode := http.StatusInternalServerError
	for response := range responses {
    if response == 200 {
      statusCode = http.StatusOK
    }
  } 
  w.WriteHeader(statusCode)
	for response := range responses {
		fmt.Fprintf(w, "%d\n", response)
	}
}

/*
* Just print incoming request elements
 */
func debugHandler(w http.ResponseWriter, req *http.Request) {
	backend := "127.0.0.1:8080"
	/* Recreate the request with same Method, path,query and Body, but to the specified backend */
  forwardReqStr := fmt.Sprintf("http://%s%s",backend,req.RequestURI)
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
	if len(backendsStr) == 0 {
		log.Println("BACKENDS environment var not defined or empty (BACKENDS=IP:PORT,IP:PORT)")
		log.Println("Running as debugger")
		http.HandleFunc("/", debugHandler)
	} else {
		backends = strings.Split(backendsStr, ",")
		http.HandleFunc("/", handler)
	}
	fmt.Printf("Starting server on %s\n",listenAddress)
	if err := http.ListenAndServe(listenAddress, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
