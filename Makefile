all: build

.PHONY: build
build: test
	go build -v ./...

.PHONY: test
test:
	go test -v ./...
