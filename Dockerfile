## This one is based on Debian
FROM golang:1.16-alpine AS builder
WORKDIR /app

# install dependencies
RUN apk update && apk add --no-cache build-base tzdata


# copy deps files
COPY ./Makefile ./go.mod ./go.sum ./
# cache dependencies
RUN go mod download

COPY *.go/ ./

# build an application
RUN make build

# select image
FROM alpine:3.13
RUN apk add --no-cache tzdata ca-certificates
WORKDIR /app
COPY --from=builder /app/bin/ /app/

CMD ["/app/sonarqube-prometheus-exporter"]