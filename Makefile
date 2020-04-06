NAME=matrix-alertmanager-receiver

all: build

getDeps:
	go get -v

build:
	go build

clean:
	rm $(NAME)
