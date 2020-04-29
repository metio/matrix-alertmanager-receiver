package main

import (
	"os"
	"flag"
	"fmt"
	"log"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"github.com/matrix-org/gomatrix"
)

type Alert struct {

}

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

	joinedRooms, err := matrixClient.JoinedRooms()
	if err != nil {
		logger.Fatalf("Could not fetch joined rooms: %v", err)
	}

	alreadyJoinedTarget := false
	for _, roomID := range joinedRooms.JoinedRooms {
		// FIXME: will only work if target is a roomID, not an alias.
		if *target == roomID {
			alreadyJoinedTarget = true
		}
	}

	if !alreadyJoinedTarget {
		logger.Printf("Trying to join %v...", *target)
		_, err := matrixClient.JoinRoom(*target, "", nil)
		if err != nil {
			logger.Fatalf("Failed to join %v: %v", *target, err)
		}
	}

	// Initialize HTTP serve (= listen for incoming requests).
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `Hi! I receive prometheus-alertmanager webhooks on /alert and forward them to Matrix.

You will find more details on: http://git.sr.ht/~fnux/matrix-prometheus-alertmanager`)
	})

	http.HandleFunc("/alert", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var alert Alert
		reqBody, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(reqBody, &alert)

		// Check validity

		logger.Printf("Sending message")
		_, err := matrixClient.SendText(*target, "spouik spouik spouik")
		if err != nil {
			logger.Fatalf("Failed to send message: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	})

	var listenAddr = fmt.Sprintf(":%v", *port)
	logger.Printf("Listening for HTTP requests (webhooks) on %v", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
