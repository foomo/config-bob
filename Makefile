TAG=`git describe --exact-match --tags $(git log -n1 --pretty='%h') 2>/dev/null || git rev-parse --abbrev-ref HEAD`
#LDFLAGS='-ldflags -X main.Version=$(TAG)'
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

release: goreleaser
	goreleaser --rm-dist

goreleaser:
	@go get github.com/goreleaser/goreleaser && go install github.com/goreleaser/goreleaser

