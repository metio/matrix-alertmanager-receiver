/*
 * SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package matrix

import (
	"context"
	"github.com/matrix-org/gomatrix"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sebhoss/matrix-alertmanager-receiver/config"
	"log/slog"
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
		msg := gomatrix.GetHTMLMessage("m.text", htmlText)
		mappedRoom := room
		if mapped, ok := configuration.RoomMapping[room]; ok {
			mappedRoom = mapped
		}
		err := joinRoom(ctx, matrixClient, mappedRoom)
		if err != nil {
			joinRoomFailureTotal.WithLabelValues(mappedRoom).Inc()
			slog.ErrorContext(ctx, "Failed to join room", slog.Any("error", err))
		} else {
			joinRoomSuccessTotal.WithLabelValues(mappedRoom).Inc()
			_, err = matrixClient.SendMessageEvent(mappedRoom, "m.room.message", msg)
			if err != nil {
				sendFailureTotal.Inc()
				slog.ErrorContext(ctx, "Could not send message to Matrix homeserver", slog.Any("error", err))
			} else {
				sendSuccessTotal.Inc()
				slog.DebugContext(ctx, "Message sent to Matrix homeserver")
			}
		}
	}
}

func createMatrixClient(ctx context.Context, configuration config.Matrix) *gomatrix.Client {
	slog.DebugContext(ctx, "Creating Matrix client", slog.Any("configuration", configuration.LogValue()))
	matrixClient, err := gomatrix.NewClient(configuration.HomeServerURL, configuration.UserID, configuration.AccessToken)
	if err != nil {
		slog.ErrorContext(ctx, "Could not log in to Matrix homeserver", slog.Any("error", err))
		os.Exit(1)
	}
	slog.DebugContext(ctx, "Created Matrix client")
	return matrixClient
}

func fetchJoinedRooms(ctx context.Context, client *gomatrix.Client) {
	joinedRooms, err := client.JoinedRooms()
	if err != nil {
		slog.ErrorContext(ctx, "Could not fetch Matrix rooms", slog.Any("error", err))
		os.Exit(1)
	}
	for _, roomID := range joinedRooms.JoinedRooms {
		joinedRoomIDs = append(joinedRoomIDs, roomID)
	}
}

func joinRoom(ctx context.Context, client *gomatrix.Client, roomToJoin string) error {
	if !slices.Contains(joinedRoomIDs, roomToJoin) {
		slog.DebugContext(ctx, "Joining room", slog.String("room", roomToJoin))
		_, err := client.JoinRoom(roomToJoin, "", nil)
		if err != nil {
			return err
		}
		joinedRoomIDs = append(joinedRoomIDs, roomToJoin)
	}
	return nil
}
