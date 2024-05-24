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
	docker run --rm -v $(CURR_DIR):/app -w /app golangci/golangci-lint:v1.58.2 golangci-lint run ./...

fmt:
	gofumpt -extra -l -w ${GOFILES_NOVENDOR}
	goimports -local github.com/fleetframework/goga -w ${GOFILES_NOVENDOR}
	gci write --skip-generated --section Standard --section Default --section "Prefix(github.com/fleetframework/sonarqube-prometheus-exporter)" ${GOFILES_NOVENDOR}

test:
	$(GO) test ${GODIRS_NOVENDOR}

build-image: build
	DOCKER_BUILDKIT=1 docker build -t sonarqube-prometheus-exporter .

run:
	realize start

tag:
	git tag ${version}
	git push origin ${version}