package main

import (
	"os"
	"flag"
	"fmt"
	"log"
	"net/http"
	"github.com/matrix-org/gomatrix"
)

func main() {
	// Initialize logger.
	var logger *log.Logger = log.New(os.Stdout, "", log.Flags())

	// Handle command-line arguments.
	var homeserver = flag.String("homeserver", "https://matrix.org", "Address of Matrix homeserver")
	var user = flag.String("user", "", "Full MXID (e.g. @example.domain.tld) of Matrix user")
	var token = flag.String("token", "", "Access Token of Matrix user")
	var target = flag.String("target-room", "", "Matrix room to be notified of alerts.")
	var port = flag.Int("port", 9088, "HTTP port to listen on (incoming alertmanager webhooks)")
	flag.Parse()

	if *user == "" {
		logger.Fatal("Matrix user is required. See --help for usage.")
	}
	if *token == "" {
		logger.Fatal("Matrix access token is required. See --help for usage.")
	}
	if *target== "" {
		logger.Fatal("Matrix target room is required. See --help for usage.")
	}

	// Initialize Matrix client.
	matrixClient, err := gomatrix.NewClient(*homeserver, *user, *token)
	if err != nil {
		logger.Fatalf("Could not log in to Matrix (%v): %v", *homeserver, err)
	}

	/*
	logger.Printf("Syncing with Matrix homserver (%v)", *homeserver)
	err = matrixClient.Sync()
	if err != nil {
		logger.Fatalf("Could not sync with Matrix homeserver (%v): %v", *homeserver, err)
	}
	*/
	_ = matrixClient

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
