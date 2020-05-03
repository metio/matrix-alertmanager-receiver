#!/bin/sh
#
# Send a dummy hook to localhost:9088 for testing purposes.

TARGET=http://localhost:9088/alert
PAYLOAD=$(cat << EOF
{"receiver":"matrix","status":"firing","alerts":[{"status":"firing","labels":{"alertname":"instance_down","instance":"example1"},"annotations":{"info":"The instance example1 is down","summary":"instance example1 is down"},"startsAt":"2020-05-03T08:30:06.275828332+02:00","endsAt":"0001-01-01T00:00:00Z","generatorURL":""}],"groupLabels":{"alertname":"instance_down"},"commonLabels":{"alertname":"instance_down","instance":"example1"},"commonAnnotations":{"info":"The instance example1 is down","summary":"instance example1 is down"},"externalURL":"http://control:9093","version":"4","groupKey":"{}:{alertname=\"instance_down\"}"}
EOF
)

curl -X POST -d "$PAYLOAD" "$TARGET"

