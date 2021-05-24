.DEFAULT_GOAL := build

COMMIT_HASH = `git rev-parse --short HEAD 2>/dev/null`
BUILD_DATE = `date +%FT%T%z`

GO=go
BINARY_DIR=bin
BINARY_NAME=sonarqube-prometheus-exporter
CURR_DIR = $(shell pwd)
GODIRS_NOVENDOR = $(shell go list ./... | grep -v /vendor/)
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*")
BUILD_INFO_LDFLAGS=-ldflags "-w -extldflags '"-static"' -X main.buildDate=${BUILD_DATE} -X main.version=${COMMIT_HASH} -X main.gitRevision=${COMMIT_HASH}"

build:
	CGO_ENABLED=0 GOOS=linux $(GO) build ${BUILD_INFO_LDFLAGS} -o ${BINARY_DIR}/${BINARY_NAME} ./

.PHONY: build

lint:
	docker run --rm -v $(CURR_DIR):/app -w /app golangci/golangci-lint:v1.32.2 golangci-lint run --enable-all --deadline 10m ./...

fmt:
	gofumpt -extra -l -w -s ${GOFILES_NOVENDOR}
	gofumports -local github.com/fleetframework/sonarqube-prometheus-exporter -l -w ${GOFILES_NOVENDOR}
	gci -local github.com/fleetframework/sonarqube-prometheus-exporter -w ${GOFILES_NOVENDOR}

test:
	$(GO) test ${GODIRS_NOVENDOR}

build-image: build
	DOCKER_BUILDKIT=1 docker build -t sonarqube-prometheus-exporter .

run:
	realize start