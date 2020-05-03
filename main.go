package main

import (
	"os"
	"flag"
	"fmt"
	"log"
	"io/ioutil"
	"net/http"
	"encoding/json"
	"github.com/prometheus/alertmanager/template"
	"github.com/BurntSushi/toml"
	"github.com/matrix-org/gomatrix"
)

var logger *log.Logger

type Configuration struct {
	Homeserver string
	TargetRoomID string

	MXID string
	MXToken string

	HTTPPort int
	HTTPToken string
}

func generateMatrixMessageBody(alert template.Alert) string {
	return alert.Status + " // " + alert.Annotations["summary"]
}

func getMatrixClient(homeserver string, user string, token string, targetRoomID string) *gomatrix.Client {
	logger.Printf("Connecting to Matrix Homserver %v as %v.", homeserver, user)
	matrixClient, err := gomatrix.NewClient(homeserver, user, token)
	if err != nil {
		logger.Fatalf("Could not log in to Matrix Homeserver (%v): %v", homeserver, err)
	}

	joinedRooms, err := matrixClient.JoinedRooms()
	if err != nil {
		logger.Fatalf("Could not fetch Matrix rooms: %v", err)
	}

	alreadyJoinedTarget := false
	for _, roomID := range joinedRooms.JoinedRooms {
		if targetRoomID == roomID {
			alreadyJoinedTarget = true
		}
	}

	if alreadyJoinedTarget {
		logger.Printf("%v is already part of %v.", user, targetRoomID,)
	} else {
		logger.Printf("Joining %v.", targetRoomID)
		_, err := matrixClient.JoinRoom(targetRoomID, "", nil)
		if err != nil {
			logger.Fatalf("Failed to join %v: %v", targetRoomID, err)
		}
	}

	return matrixClient
}

func handleIncomingHooks( w http.ResponseWriter, r *http.Request,
	matrixClient *gomatrix.Client, targetRoomID string) {

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	payload := template.Data{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	logger.Printf("Received valid hook from %v", r.RemoteAddr)

	for _, alert := range payload.Alerts {
		body := generateMatrixMessageBody(alert)
		logger.Printf("> %v", body)
		_, err := matrixClient.SendText(targetRoomID, body)
		if err != nil {
			logger.Printf(">> Could not forward to Matrix: %v", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func main() {
	// Initialize logger.
	logger = log.New(os.Stdout, "", log.Flags())

	// We use a configuration file since we need to specify secrets, and read
	// everything else from it to keep things simple.
	var configPath = flag.String("config", "/etc/matrix-alertmanager-receiver.toml", "Path to configuration file")
	flag.Parse()

	logger.Printf("Reading configuration from %v.", *configPath)
	raw, err := ioutil.ReadFile(*configPath)
	if err != nil {
		logger.Fatalf("Could not read configuration file (%v): %v", *configPath, err)
	}

	var config Configuration
	md, err := toml.Decode(string(raw), &config)
	if err != nil {
		logger.Fatalf("Could not parse configuration file (%v): %v", *configPath, err)
	}

	for _, field := range []string{"Homeserver", "MXID", "MXToken", "TargetRoomID", "HTTPPort"} {
		if ! md.IsDefined(field) {
			logger.Fatalf("Field %v is not set in config. Exiting.", field)
		}
	}

	// Initialize Matrix client.
	matrixClient := getMatrixClient(
		config.Homeserver, config.MXID, config.MXToken, config.TargetRoomID)

	// Initialize HTTP server.
	http.HandleFunc("/alert", func(w http.ResponseWriter, r *http.Request) {
		handleIncomingHooks(w, r, matrixClient, config.TargetRoomID)
	})

	var listenAddr = fmt.Sprintf(":%v", config.HTTPPort)
	logger.Printf("Listening for HTTP requests (webhooks) on %v", listenAddr)
	logger.Fatal(http.ListenAndServe(listenAddr, nil))
}
