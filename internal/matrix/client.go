/*
 * SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package matrix

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sebhoss/matrix-alertmanager-receiver/internal/config"
	"log/slog"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"
	"os"
	"slices"
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
)

type SendingFunc func(htmlText string, roomID string)

var joinedRoomIDs []string

func CreatingSendingFunc(ctx context.Context, configuration config.Matrix) SendingFunc {
	matrixClient := createMatrixClient(ctx, configuration)
	fetchJoinedRooms(ctx, matrixClient)
	return func(htmlText string, room string) {
		mappedRoom := room
		if mapped, ok := configuration.RoomMapping[room]; ok {
			mappedRoom = mapped
		}
		if err := joinRoom(ctx, matrixClient, mappedRoom); err != nil {
			joinRoomFailureTotal.WithLabelValues(mappedRoom).Inc()
			slog.ErrorContext(ctx, fmt.Sprintf("Could not join room %s", room), slog.Any("error", err))
		} else {
			joinRoomSuccessTotal.WithLabelValues(mappedRoom).Inc()
			if respSendEvent, err := matrixClient.SendMessageEvent(ctx, id.RoomID(mappedRoom), event.NewEventType("m.room.message"), format.HTMLToContent(htmlText)); err != nil {
				sendFailureTotal.Inc()
				slog.ErrorContext(ctx, "Could not send message to Matrix homeserver", slog.Any("error", err))
			} else {
				sendSuccessTotal.Inc()
				slog.DebugContext(ctx, fmt.Sprintf("Message %s sent to Matrix homeserver", respSendEvent.EventID))
			}
		}
	}
}

func createMatrixClient(ctx context.Context, configuration config.Matrix) *mautrix.Client {
	var err error
	var matrixClient *mautrix.Client
	slog.DebugContext(ctx, "Creating Matrix client", slog.Any("configuration", configuration.LogValue()))
	if matrixClient, err = mautrix.NewClient(configuration.HomeServerURL, id.UserID(configuration.UserID), configuration.AccessToken); err != nil {
		slog.ErrorContext(ctx, "Failed to create matrix client", slog.Any("error", err))
		os.Exit(1)
	}
	slog.DebugContext(ctx, "Created Matrix client")
	return matrixClient
}

func fetchJoinedRooms(ctx context.Context, client *mautrix.Client) {
	joinedRooms, err := client.JoinedRooms(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Could not fetch Matrix rooms", slog.Any("error", err))
		os.Exit(1)
	}
	for _, roomID := range joinedRooms.JoinedRooms {
		joinedRoomIDs = append(joinedRoomIDs, roomID.String())
	}
}

func joinRoom(ctx context.Context, client *mautrix.Client, roomToJoin string) error {
	if !slices.Contains(joinedRoomIDs, roomToJoin) {
		slog.DebugContext(ctx, "Joining room", slog.String("room", roomToJoin))
		_, err := client.JoinRoomByID(ctx, id.RoomID(roomToJoin))
		if err != nil {
			return err
		}
		joinedRoomIDs = append(joinedRoomIDs, roomToJoin)
	}
	return nil
}
