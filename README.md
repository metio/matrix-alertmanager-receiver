# matrix-alertmanager-receiver

[![builds.sr.ht status](https://builds.sr.ht/~fnux/matrix-alertmanager-receiver.svg)](https://builds.sr.ht/~fnux/matrix-alertmanager-receiver?)

Simple daemon - less than 150 lines of Go - forwarding
[prometheus-alertmanager](https://duckduckgo.com/?q=prometheus+alertmanagaer&ia=software)
events to matrix room. While there already is a
[matrix-alertmanager](https://git.feneas.org/jaywink/matrix-alertmanager)
project out there, use of the JavaScript ecosystem made it rather painful to
use from my point of view.

Feel free to directly send [me](https://fnux.ch/) patches and questions by email.

## Build

Make sure you have [Go](https://golang.org/) installed (`golang-bin` package on
Fedora).

```
go build -v
```

## Usage

Note: you are supposed to expose this service via a proxy such as Nginx,
providing basic HTTP authentication.

```
I (master|✚1) ~/W/f/matrix-alertmanager-receiver » ./matrix-alertmanager-receiver --help
Usage of ./matrix-alertmanager-receiver:
  -config string
    	Path to configuration file (default "/etc/matrix-alertmanager-receiver.toml")
I [2] (master|✚1) ~/W/f/matrix-alertmanager-receiver » cat config.toml
Homeserver = "https://staging.matrix.ungleich.cloud"
TargetRoomID = "!jHFKHemgIAaDJekoxN:matrix-staging.ungleich.ch"
MXID = "@fnux:matrix-staging.ungleich.ch"
MXToken = "secretsecretsecret"
HTTPPort = 9088
HTTPAddress = ""
I (master|✚1) ~/W/f/matrix-alertmanager-receiver » ./matrix-alertmanager-receiver -config config.toml
2020/05/03 10:50:47 Reading configuration from config.toml.
2020/05/03 10:50:47 Connecting to Matrix Homserver https://staging.matrix.ungleich.cloud as @fnux:matrix-staging.ungleich.ch.
2020/05/03 10:50:47 @fnux:matrix-staging.ungleich.ch is already part of !jHFKHemgIAaDJekoxN:matrix-staging.ungleich.ch.
2020/05/03 10:50:47 Listening for HTTP requests (webhooks) on :9088
2020/05/03 10:50:55 Received valid hook from [::1]:44886
2020/05/03 10:50:55 > FIRING instance example1 is down
```
