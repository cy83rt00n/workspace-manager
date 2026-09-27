VERSION ?= dev
GOOS ?= linux
GOARCH ?= amd64
CGO_ENABLED ?= 0

LDFLAGS = -X github.com/cy83rt00n/workspace-manager/internal/version.Version=$(VERSION) -s -w
BINARY  = wsm

.PHONY: fmt vet test coverage build clean

fmt:
	gofmt -l .

vet:
	go vet ./...

test:
	go test ./...

coverage:
	go test -cover -coverprofile=coverage.out ./...

build:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags "$(LDFLAGS)" ./...

clean:
	rm -rf bin coverage.out
