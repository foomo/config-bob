all: test build
test:
	go test ./...
build:
	go build -o config-bob .
# renders example/source-vault with the test vault, keys as in config-bob_test.go
vault-example:
	@out=$$(mktemp -d) && CFB_KEYS=Zd5U+W/WK3cPgH4Mth5seweXjnloLmU+6exo7bzSAdE= CFB_TOKEN=config-bob-test \
		go run . build --vault-dir example/vault example/source-vault $$out > /dev/null && cat $$out/app.yaml; rm -rf $$out
