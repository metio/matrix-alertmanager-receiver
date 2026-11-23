/*
 * SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package matrix

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	amtemplate "github.com/prometheus/alertmanager/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

const testRoomID = id.RoomID("!room:example.com")

type sentEvent struct {
	roomID  id.RoomID
	content map[string]any
}

type fakeClient struct {
	sent       []sentEvent
	history    []*event.Event
	historyErr error
	nextID     int
}

func (f *fakeClient) SendMessageEvent(_ context.Context, roomID id.RoomID, _ event.Type, contentJSON any, _ ...mautrix.ReqSendEvent) (*mautrix.RespSendEvent, error) {
	encoded, err := json.Marshal(contentJSON)
	if err != nil {
		return nil, err
	}
	content := map[string]any{}
	if err := json.Unmarshal(encoded, &content); err != nil {
		return nil, err
	}
	f.sent = append(f.sent, sentEvent{roomID: roomID, content: content})
	f.nextID++
	return &mautrix.RespSendEvent{EventID: id.EventID("$sent" + string(rune('a'+f.nextID)))}, nil
}

func (f *fakeClient) Messages(_ context.Context, _ id.RoomID, _, _ string, _ mautrix.Direction, _ *mautrix.FilterPart, _ int) (*mautrix.RespMessages, error) {
	if f.historyErr != nil {
		return nil, f.historyErr
	}
	return &mautrix.RespMessages{Chunk: f.history}, nil
}

func (f *fakeClient) JoinedRooms(_ context.Context) (*mautrix.RespJoinedRooms, error) {
	return &mautrix.RespJoinedRooms{JoinedRooms: []id.RoomID{testRoomID}}, nil
}

func (f *fakeClient) JoinRoomByID(_ context.Context, roomID id.RoomID) (*mautrix.RespJoinRoom, error) {
	return &mautrix.RespJoinRoom{RoomID: roomID}, nil
}

func newTestSender(client messageClient, mode string) *sender {
	return &sender{
		client:        client,
		userID:        "@alerts:example.com",
		roomMapping:   map[string]string{},
		mode:          mode,
		joinedRoomIDs: map[id.RoomID]bool{testRoomID: true},
	}
}

func testAlert(status string, startsAt time.Time) amtemplate.Alert {
	return amtemplate.Alert{
		Status:      status,
		Fingerprint: "abc123",
		StartsAt:    startsAt,
		Labels:      amtemplate.KV{"alertname": "SomethingBroke"},
	}
}

func historyEvent(eventID id.EventID, timestamp int64, content map[string]any) *event.Event {
	return &event.Event{
		ID:        eventID,
		Type:      event.EventMessage,
		Timestamp: timestamp,
		Content:   event.Content{Raw: content},
	}
}

func firingContent(alert amtemplate.Alert) map[string]any {
	return messageContent("<p>firing</p>", alert)
}

func editContentFor(alert amtemplate.Alert, original id.EventID) map[string]any {
	return editContent("<p>resolved</p>", alert, original)
}

func TestNewEventModeSendsEveryAlertAsNewMessage(t *testing.T) {
	client := &fakeClient{}
	sending := newTestSender(client, resolvedNotificationModeNewEvent)
	startsAt := time.Now()

	sending.send(context.Background(), testAlert("firing", startsAt), "<p>firing</p>", "homelab")
	sending.send(context.Background(), testAlert("resolved", startsAt), "<p>resolved</p>", "homelab")

	require.Len(t, client.sent, 2)
	for _, sent := range client.sent {
		assert.NotContains(t, sent.content, "m.relates_to")
	}
}

func TestEditModeSendsFiringMessageWithMarkers(t *testing.T) {
	client := &fakeClient{}
	sending := newTestSender(client, resolvedNotificationModeEdit)
	startsAt := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)

	sending.send(context.Background(), testAlert("firing", startsAt), "<p>firing</p>", "homelab")

	require.Len(t, client.sent, 1)
	content := client.sent[0].content
	assert.Equal(t, "abc123", content[markerFingerprint])
	assert.Equal(t, startsAt.Format(time.RFC3339Nano), content[markerStartsAt])
	assert.Equal(t, "firing", content[markerStatus])
	assert.NotContains(t, content, "m.relates_to")
}

func TestEditModeResolvedEditsMatchingFiringMessageInPlace(t *testing.T) {
	startsAt := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	original := id.EventID("$original")
	client := &fakeClient{
		history: []*event.Event{
			historyEvent(original, startsAt.UnixMilli()+1000, firingContent(testAlert("firing", startsAt))),
		},
	}
	sending := newTestSender(client, resolvedNotificationModeEdit)

	sending.send(context.Background(), testAlert("resolved", startsAt), "<p>resolved</p>", "homelab")

	require.Len(t, client.sent, 1)
	content := client.sent[0].content
	relatesTo, ok := content["m.relates_to"].(map[string]any)
	require.True(t, ok, "expected an m.relates_to relation")
	assert.Equal(t, string(event.RelReplace), relatesTo["rel_type"])
	assert.Equal(t, original.String(), relatesTo["event_id"])
	newContent, ok := content["m.new_content"].(map[string]any)
	require.True(t, ok, "expected m.new_content")
	assert.Equal(t, "resolved", newContent[markerStatus])
}

func TestEditModeResolvedFallsBackToNewMessageWhenNoFiringMessageExists(t *testing.T) {
	startsAt := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	client := &fakeClient{}
	sending := newTestSender(client, resolvedNotificationModeEdit)

	sending.send(context.Background(), testAlert("resolved", startsAt), "<p>resolved</p>", "homelab")

	require.Len(t, client.sent, 1)
	assert.NotContains(t, client.sent[0].content, "m.relates_to")
	assert.Equal(t, "resolved", client.sent[0].content[markerStatus])
}

func TestEditModeRepeatEditsExistingMessageInsteadOfSendingNewOne(t *testing.T) {
	startsAt := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	original := id.EventID("$original")
	client := &fakeClient{
		history: []*event.Event{
			historyEvent(original, startsAt.UnixMilli()+1000, firingContent(testAlert("firing", startsAt))),
		},
	}
	sending := newTestSender(client, resolvedNotificationModeEdit)

	sending.send(context.Background(), testAlert("firing", startsAt), "<p>firing again</p>", "homelab")

	require.Len(t, client.sent, 1)
	relatesTo, ok := client.sent[0].content["m.relates_to"].(map[string]any)
	require.True(t, ok, "expected an m.relates_to relation")
	assert.Equal(t, original.String(), relatesTo["event_id"])
}

func TestEditModeRefireAfterResolveSendsNewMessage(t *testing.T) {
	firstCycle := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	secondCycle := firstCycle.Add(2 * time.Hour)
	original := id.EventID("$original")
	resolved := testAlert("resolved", firstCycle)
	client := &fakeClient{
		history: []*event.Event{
			historyEvent("$edit", firstCycle.UnixMilli()+2000, editContentFor(resolved, original)),
			historyEvent(original, firstCycle.UnixMilli()+1000, firingContent(testAlert("firing", firstCycle))),
		},
	}
	sending := newTestSender(client, resolvedNotificationModeEdit)

	sending.send(context.Background(), testAlert("firing", secondCycle), "<p>firing again</p>", "homelab")

	require.Len(t, client.sent, 1)
	assert.NotContains(t, client.sent[0].content, "m.relates_to")
}

func TestEditModeAlreadyResolvedMessageIsNotEditedTwice(t *testing.T) {
	startsAt := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	original := id.EventID("$original")
	resolved := testAlert("resolved", startsAt)
	client := &fakeClient{
		history: []*event.Event{
			historyEvent("$edit", startsAt.UnixMilli()+2000, editContentFor(resolved, original)),
			historyEvent(original, startsAt.UnixMilli()+1000, firingContent(testAlert("firing", startsAt))),
		},
	}
	sending := newTestSender(client, resolvedNotificationModeEdit)

	sending.send(context.Background(), resolved, "<p>resolved</p>", "homelab")

	assert.Empty(t, client.sent)
}

func TestEditModeHistoryFailureFallsBackToNewResolvedMessage(t *testing.T) {
	startsAt := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	client := &fakeClient{historyErr: errors.New("boom")}
	sending := newTestSender(client, resolvedNotificationModeEdit)

	sending.send(context.Background(), testAlert("resolved", startsAt), "<p>resolved</p>", "homelab")

	require.Len(t, client.sent, 1)
	assert.NotContains(t, client.sent[0].content, "m.relates_to")
}
