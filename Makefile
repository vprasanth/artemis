VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS  = -X main.version=$(VERSION)
BINARY   = artemis

.PHONY: build run clean tag release

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./main.go

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY) $(BINARY)-*

tag:
ifndef TAG
	$(error Usage: make tag TAG=v0.3.0)
endif
	git tag -a $(TAG) -m "$(TAG)"
	git push origin $(TAG)

release: clean
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-darwin-arm64  ./main.go
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-darwin-amd64  ./main.go
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-linux-amd64   ./main.go
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-linux-arm64   ./main.go
