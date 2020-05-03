NAME=yggdrasil-http-proxy
PREFIX?=/usr/local

all: build

getDeps:
	go get -v

build:
	go build

install: build
	install -m 0755 $(NAME) $(PREFIX)/bin/$(NAME)

clean:
	rm $(NAME)
