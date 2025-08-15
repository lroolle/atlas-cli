BINARY_NAME=atl
VERSION=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

.PHONY: build
build:
	go build -o ${BINARY_NAME} ${LDFLAGS} .

.PHONY: clean
clean:
	go clean
	rm -f ${BINARY_NAME}

.PHONY: test
test:
	go test -v ./...

.PHONY: run
run:
	go run main.go

.PHONY: install
install: build
	@if [ -z "$(GOPATH)" ]; then \
		echo "GOPATH not set, installing to ~/go/bin"; \
		mkdir -p ~/go/bin; \
		cp ${BINARY_NAME} ~/go/bin/${BINARY_NAME}; \
	else \
		cp ${BINARY_NAME} ${GOPATH}/bin/${BINARY_NAME}; \
	fi
	@echo "Installed as 'atl'"

# Easter egg for those who appreciate irony
.PHONY: ass
ass:
	@echo "Building for Atlassian Server Stuff..."
	@go build -o ass ${LDFLAGS} .
	@echo "Built as 'ass' - because we know what you're thinking 😏"

.PHONY: cross
cross:
	GOOS=linux GOARCH=amd64 go build -o ${BINARY_NAME}-linux-amd64 ${LDFLAGS} .
	GOOS=darwin GOARCH=amd64 go build -o ${BINARY_NAME}-darwin-amd64 ${LDFLAGS} .
	GOOS=darwin GOARCH=arm64 go build -o ${BINARY_NAME}-darwin-arm64 ${LDFLAGS} .
	GOOS=windows GOARCH=amd64 go build -o ${BINARY_NAME}-windows-amd64.exe ${LDFLAGS} .

.DEFAULT_GOAL := build