TAG=`git describe --tags`
LDFLAGS=-ldflags "-X main.Version=${TAG}"

all: test build
clean:
	rm -rf bin/config-*
prepare: clean
	go get -t -v ./...
test: prepare
	go test -v ./...
build: prepare
	go build $(LDFLAGS) config-bob.go
build-arch: prepare
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/config-bob-linux-amd64_$(TAG) config-bob.go
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/config-bob-darwin-amd64_$(TAG) config-bob.go

release:
	goreleaser --rm-dist

