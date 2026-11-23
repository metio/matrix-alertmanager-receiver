/*
 * SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package matrix

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/metio/matrix-alertmanager-receiver/internal/config"
	amtemplate "github.com/prometheus/alertmanager/template"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"
)

const (
	resolvedNotificationModeNewEvent = "new-event"
	resolvedNotificationModeEdit     = "edit"

	markerFingerprint = "io.github.metio.matrix-alertmanager-receiver.fingerprint"
	markerStartsAt    = "io.github.metio.matrix-alertmanager-receiver.startsAt"
	markerStatus      = "io.github.metio.matrix-alertmanager-receiver.status"

	historyPageSize = 100
	historyMaxPages = 20
)

var (
	joinRoomSuccessTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "matrix_alertmanager_receiver_join_room_success_total",
		Help: "The total number of successful join room operations",
	}, []string{"room"})
	joinRoomFailureTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "matrix_alertmanager_receiver_join_room_failure_total",
		Help: "The total number of failed join room operations",
	}, []string{"room"})
	sendSuccessTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "matrix_alertmanager_receiver_send_success_total",
		Help: "The total number of successful send operations",
	})
	sendFailureTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "matrix_alertmanager_receiver_send_failure_total",
		Help: "The total number of failed send operations",
	})
	editSuccessTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "matrix_alertmanager_receiver_edit_success_total",
		Help: "The total number of successful in-place edit operations",
	})
	editFailureTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "matrix_alertmanager_receiver_edit_failure_total",
		Help: "The total number of failed in-place edit operations",
	})
	historyFailureTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "matrix_alertmanager_receiver_history_failure_total",
		Help: "The total number of failed room history lookups",
	})
)

type SendingFunc func(alert amtemplate.Alert, htmlText string, roomID string)

type messageClient interface {
	SendMessageEvent(ctx context.Context, roomID id.RoomID, eventType event.Type, contentJSON any, extra ...mautrix.ReqSendEvent) (*mautrix.RespSendEvent, error)
	Messages(ctx context.Context, roomID id.RoomID, from, to string, dir mautrix.Direction, filter *mautrix.FilterPart, limit int) (*mautrix.RespMessages, error)
	JoinedRooms(ctx context.Context) (*mautrix.RespJoinedRooms, error)
	JoinRoomByID(ctx context.Context, roomID id.RoomID) (*mautrix.RespJoinRoom, error)
}

type sender struct {
	client        messageClient
	userID        id.UserID
	roomMapping   map[string]string
	mode          string
	joinedMu      sync.Mutex
	joinedRoomIDs map[id.RoomID]bool
}

func CreatingSendingFunc(ctx context.Context, configuration config.Matrix) SendingFunc {
	matrixClient := createMatrixClient(ctx, configuration)
	sending := &sender{
		client:        matrixClient,
		userID:        id.UserID(configuration.UserID),
		roomMapping:   configuration.RoomMapping,
		mode:          configuration.ResolvedNotificationMode,
		joinedRoomIDs: make(map[id.RoomID]bool),
	}
	if sending.mode == "" {
		sending.mode = resolvedNotificationModeNewEvent
	}
	sending.fetchJoinedRooms(ctx)
	return func(alert amtemplate.Alert, htmlText string, room string) {
		sending.send(ctx, alert, htmlText, room)
	}
}

func (s *sender) send(ctx context.Context, alert amtemplate.Alert, htmlText string, room string) {
	mappedRoom := room
	if mapped, ok := s.roomMapping[room]; ok {
		mappedRoom = mapped
	}
	roomID := id.RoomID(mappedRoom)
	if err := s.joinRoom(ctx, roomID); err != nil {
		joinRoomFailureTotal.WithLabelValues(mappedRoom).Inc()
		slog.ErrorContext(ctx, fmt.Sprintf("Could not join room %s", room), slog.Any("error", err))
		return
	}
	joinRoomSuccessTotal.WithLabelValues(mappedRoom).Inc()

	if s.mode != resolvedNotificationModeEdit {
		s.sendNewEvent(ctx, roomID, format.HTMLToContent(htmlText))
		return
	}

	openEvents, cycleExists, err := s.findOpenEvents(ctx, roomID, alert)
	if err != nil {
		historyFailureTotal.Inc()
		slog.ErrorContext(ctx, "Could not look up previous alert messages, falling back to a new message", slog.Any("error", err))
		openEvents = nil
		cycleExists = false
	}

	if len(openEvents) > 0 {
		for _, original := range openEvents {
			s.sendEdit(ctx, roomID, htmlText, alert, original)
		}
		return
	}

	// A cycle that already has firing messages, all of them resolved, is a
	// retry or restart re-delivering the same resolve: stay silent so the
	// operation stays idempotent.
	if cycleExists {
		slog.DebugContext(ctx, "Alert cycle already resolved, not sending another message",
			slog.String("fingerprint", alert.Fingerprint))
		return
	}

	s.sendNewEvent(ctx, roomID, messageContent(htmlText, alert))
}

func (s *sender) sendNewEvent(ctx context.Context, roomID id.RoomID, content any) {
	if _, err := s.client.SendMessageEvent(ctx, roomID, event.EventMessage, content); err != nil {
		sendFailureTotal.Inc()
		slog.ErrorContext(ctx, "Could not send message to Matrix homeserver", slog.Any("error", err))
		return
	}
	sendSuccessTotal.Inc()
}

func (s *sender) sendEdit(ctx context.Context, roomID id.RoomID, htmlText string, alert amtemplate.Alert, original id.EventID) {
	if _, err := s.client.SendMessageEvent(ctx, roomID, event.EventMessage, editContent(htmlText, alert, original)); err != nil {
		editFailureTotal.Inc()
		slog.ErrorContext(ctx, "Could not edit message on Matrix homeserver",
			slog.String("event-id", original.String()), slog.Any("error", err))
		return
	}
	editSuccessTotal.Inc()
	slog.DebugContext(ctx, "Message edited on Matrix homeserver", slog.String("event-id", original.String()))
}

// findOpenEvents scans the room history backwards for firing messages that
// belong to the same alert cycle (fingerprint + startsAt) and have not been
// resolved yet. The scan stops once it passes the alert's startsAt or hits the
// page cap, so the lookup stays bounded without any persistent state.
//
// The second return value reports whether the cycle has any firing message at
// all, resolved or not, which lets the caller stay idempotent on a repeated
// resolve instead of posting a duplicate.
func (s *sender) findOpenEvents(ctx context.Context, roomID id.RoomID, alert amtemplate.Alert) ([]id.EventID, bool, error) {
	fingerprint := alert.Fingerprint
	startsAt := startsAtMarker(alert.StartsAt)
	var threshold int64
	if !alert.StartsAt.IsZero() {
		threshold = alert.StartsAt.UnixMilli()
	}

	latestStatus := make(map[id.EventID]string)
	var candidates []id.EventID
	from := ""
	for range historyMaxPages {
		response, err := s.client.Messages(ctx, roomID, from, "", mautrix.DirectionBackward,
			&mautrix.FilterPart{Types: []event.Type{event.EventMessage}}, historyPageSize)
		if err != nil {
			return nil, false, err
		}
		reachedStartsAt := false
		for _, timelineEvent := range response.Chunk {
			raw := timelineEvent.Content.Raw
			if raw == nil {
				continue
			}
			if target := replacedEventID(raw); target != "" {
				if _, seen := latestStatus[target]; !seen {
					latestStatus[target] = newContentStatus(raw)
				}
				continue
			}
			if stringField(raw, markerFingerprint) == fingerprint &&
				stringField(raw, markerStartsAt) == startsAt &&
				stringField(raw, markerStatus) == "firing" {
				candidates = append(candidates, timelineEvent.ID)
			}
			if threshold > 0 && timelineEvent.Timestamp > 0 && timelineEvent.Timestamp < threshold {
				reachedStartsAt = true
			}
		}
		if reachedStartsAt || response.End == "" {
			break
		}
		from = response.End
	}

	var open []id.EventID
	for _, candidate := range candidates {
		if status, ok := latestStatus[candidate]; ok && status == "resolved" {
			continue
		}
		open = append(open, candidate)
	}
	return open, len(candidates) > 0, nil
}

func messageContent(htmlText string, alert amtemplate.Alert) map[string]any {
	content := format.HTMLToContent(htmlText)
	message := map[string]any{
		"msgtype":         string(content.MsgType),
		"body":            content.Body,
		markerFingerprint: alert.Fingerprint,
		markerStartsAt:    startsAtMarker(alert.StartsAt),
		markerStatus:      alert.Status,
	}
	if content.Format != "" {
		message["format"] = string(content.Format)
	}
	if content.FormattedBody != "" {
		message["formatted_body"] = content.FormattedBody
	}
	return message
}

func editContent(htmlText string, alert amtemplate.Alert, original id.EventID) map[string]any {
	content := format.HTMLToContent(htmlText)
	message := map[string]any{
		"msgtype":       string(content.MsgType),
		"body":          "* " + content.Body,
		"m.new_content": messageContent(htmlText, alert),
		"m.relates_to": map[string]any{
			"rel_type": string(event.RelReplace),
			"event_id": original.String(),
		},
	}
	if content.Format != "" {
		message["format"] = string(content.Format)
	}
	if content.FormattedBody != "" {
		message["formatted_body"] = content.FormattedBody
	}
	return message
}

func startsAtMarker(startsAt time.Time) string {
	if startsAt.IsZero() {
		return ""
	}
	return startsAt.UTC().Format(time.RFC3339Nano)
}

func replacedEventID(raw map[string]any) id.EventID {
	relatesTo, ok := raw["m.relates_to"].(map[string]any)
	if !ok {
		return ""
	}
	if stringField(relatesTo, "rel_type") != string(event.RelReplace) {
		return ""
	}
	return id.EventID(stringField(relatesTo, "event_id"))
}

func newContentStatus(raw map[string]any) string {
	newContent, ok := raw["m.new_content"].(map[string]any)
	if !ok {
		return ""
	}
	return stringField(newContent, markerStatus)
}

func stringField(raw map[string]any, key string) string {
	value, _ := raw[key].(string)
	return value
}

func createMatrixClient(ctx context.Context, configuration config.Matrix) *mautrix.Client {
	var err error
	var matrixClient *mautrix.Client
	slog.DebugContext(ctx, "Creating Matrix client", slog.Any("configuration", configuration.LogValue()))
	if configuration.Proxy != "" {
		// this is similar to mautrix.NewClient but sets a proxy in the http.Client
		hsURL, err := mautrix.ParseAndNormalizeBaseURL(configuration.HomeServerURL)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to parse/normalize base URL", slog.Any("error", err))
			os.Exit(1)
		}
		proxyUrl, err := url.Parse(configuration.Proxy)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to parse proxy URL", slog.Any("error", err))
			os.Exit(1)
		}
		matrixClient = &mautrix.Client{
			AccessToken:   configuration.AccessToken,
			UserAgent:     mautrix.DefaultUserAgent,
			HomeserverURL: hsURL,
			UserID:        id.UserID(configuration.UserID),
			Client:        &http.Client{Timeout: 180 * time.Second, Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}},
			Syncer:        mautrix.NewDefaultSyncer(),
			Log:           zerolog.Nop(),
			Store:         mautrix.NewMemorySyncStore(),
		}
	} else {
		if matrixClient, err = mautrix.NewClient(configuration.HomeServerURL, id.UserID(configuration.UserID), configuration.AccessToken); err != nil {
			slog.ErrorContext(ctx, "Failed to create matrix client", slog.Any("error", err))
			os.Exit(1)
		}
	}

	slog.DebugContext(ctx, "Created Matrix client")
	return matrixClient
}

func (s *sender) fetchJoinedRooms(ctx context.Context) {
	joinedRooms, err := s.client.JoinedRooms(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Could not fetch Matrix rooms", slog.Any("error", err))
		os.Exit(1)
	}
	s.joinedMu.Lock()
	defer s.joinedMu.Unlock()
	for _, roomID := range joinedRooms.JoinedRooms {
		s.joinedRoomIDs[roomID] = true
	}
}

func (s *sender) joinRoom(ctx context.Context, roomID id.RoomID) error {
	s.joinedMu.Lock()
	joined := s.joinedRoomIDs[roomID]
	s.joinedMu.Unlock()
	if joined {
		return nil
	}
	slog.DebugContext(ctx, "Joining room", slog.String("room", roomID.String()))
	if _, err := s.client.JoinRoomByID(ctx, roomID); err != nil {
		return err
	}
	s.joinedMu.Lock()
	s.joinedRoomIDs[roomID] = true
	s.joinedMu.Unlock()
	return nil
}
