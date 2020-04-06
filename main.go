package main

import (
	"os"
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Initialize logger.
	var logger *log.Logger = log.New(os.Stdout, "", log.Flags())

	// Handle command-line arguments.
	var port = flag.Int("port", 9088, "HTTP port to listen on (incoming alertmanager webhooks)")
	flag.Parse()

	// Initialize Matrix client.
	// TODO

	// Initialize HTTP serve (= listen for incoming requests).
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `Hi! I receive prometheus-alertmanager webhooks on /alert and forward them to Matrix.

You will find more details on: http://git.sr.ht/~fnux/matrix-prometheus-alertmanager`)
	})

	http.HandleFunc("/alert", func(w http.ResponseWriter, r *http.Request) {
	})

	var listenAddr = fmt.Sprintf(":%v", *port)
	logger.Printf("Listening for HTTP requests (webhooks) on %v", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
