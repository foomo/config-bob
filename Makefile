TAG=`git describe --exact-match --tags $(git log -n1 --pretty='%h') 2>/dev/null || git rev-parse --abbrev-ref HEAD`
#LDFLAGS='-ldflags -X main.Version=$(TAG)'
LDFLAGS=-ldflags "-X main.Version=${TAG}"

all: test build
prepare:
	go get -t -v ./...
test: prepare
	go test -v ./...
build: prepare
	go build $(LDFLAGS) config-bob.go
