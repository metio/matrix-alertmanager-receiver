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
	"html"
	"log/slog"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
	"os"
	"regexp"
)

var (
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

// An HTMLMessage is the contents of a Matrix HTML formated message event.
type HTMLMessage struct {
	Body          string `json:"body"`
	MsgType       string `json:"msgtype"`
	Format        string `json:"format"`
	FormattedBody string `json:"formatted_body"`
}

var htmlRegex = regexp.MustCompile("<[^<]+?>")

// GetHTMLMessage returns an HTMLMessage with the body set to a stripped version of the provided HTML, in addition
// to the provided HTML.
func GetHTMLMessage(msgtype, htmlText string) HTMLMessage {
	return HTMLMessage{
		Body:          html.UnescapeString(htmlRegex.ReplaceAllLiteralString(htmlText, "")),
		MsgType:       msgtype,
		Format:        "org.matrix.custom.html",
		FormattedBody: htmlText,
	}
}

func CreatingSendingFunc(ctx context.Context, configuration config.Matrix) SendingFunc {
	matrixClient := createMatrixClient(ctx, configuration)
	return func(htmlText string, room string) {
		if respSendEvent, err := matrixClient.SendMessageEvent(ctx, id.RoomID(room), event.NewEventType("m.room.message"), GetHTMLMessage("m.text", htmlText)); err != nil {
			sendFailureTotal.Inc()
			slog.ErrorContext(ctx, "Could not send message to Matrix homeserver", slog.Any("error", err))
		} else {
			sendSuccessTotal.Inc()
			slog.DebugContext(ctx, fmt.Sprintf("Message %s sent to Matrix homeserver", respSendEvent.EventID))
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
	} else {
		slog.DebugContext(ctx, "Created Matrix client")
	}
	return matrixClient
}
