# select image
FROM alpine:3.13
RUN apk add --no-cache tzdata ca-certificates
WORKDIR /app
COPY sonarqube-prometheus-exporter /app/

ENTRYPOINT ["/app/sonarqube-prometheus-exporter"]