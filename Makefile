

all: test build
prepare:
	go get -t -v ./...
test: prepare
	go test -v ./...
build: prepare
	go build config-bob.go
