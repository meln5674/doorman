.PHONY: all fmt vet test clean

all: fmt vet bin/coverage.html test bin/doorman bin/doorman-arm64 bin/doorman.exe

clean:
	rm -rf bin/

fmt: $(wildcard **/*.go *.go)
	go fmt ./...

vet: $(wildcard **/*.go *.go)
	go vet ./...

bin/coverage.out: $(wildcard **/*.go *.go)
	mkdir -p bin
	go test -coverprofile=bin/coverage.out

bin/coverage.html: bin/coverage.out
	go tool cover -html=bin/coverage.out -o bin/coverage.html

test: $(wildcard **/*.go *.go) bin/coverage.html

bin/doorman: $(wildcard **/*.go *.go)
	CGO_ENABLED=0 GOOS=linux go build -a -tags '-w -extldflags "-static"' -o bin/doorman main.go

bin/doorman-arm64: $(wildcard **/*.go *.go)
	CGO_ENABLED=0 GOARCH=arm64 GOOS=linux go build -a -tags '-w -extldflags "-static"' -o bin/doorman-arm64 main.go

bin/doorman.exe: $(wildcard **/*.go *.go)
	CGO_ENABLED=0 GOOS=windows go build -a -tags '-w -extldflags "-static"' -o bin/doorman.exe main.go
