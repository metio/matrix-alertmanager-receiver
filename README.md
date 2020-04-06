# matrix-alertmanager-receiver

Simple daemon forwarding
[prometheus-alertmanager](https://duckduckgo.com/?q=prometheus+alertmanagaer&ia=software)
events to matrix room. While there already is a
[matrix-alertmanager](https://git.feneas.org/jaywink/matrix-alertmanager)
project out there, use of the JavaScript ecosystem (and lack of logging) made
it rather painful to use from my point of view.

See `matrix-alermanager-receiver.scd` for usage. [Go](http://golang.org/) is
required to build the `matrix-alertmanager-receiver` binary, `make` and `scdoc`
(manpage generation) are optional but convenient.
