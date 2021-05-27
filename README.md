# Prometheus Exporter for Sonarqube

![Build Status](https://github.com/fleetframework/sonarqube-prometheus-exporter/actions/workflows/build.yml/badge.svg)
![Release](https://img.shields.io/github/v/release/fleetframework/sonarqube-prometheus-exporter)

## Usage

```
Usage of bin/sonarqube-prometheus-exporter:
  -help
        Show help
  -version
        Show version
  -metrics-ns string
        Prometheus metrics namespace. Default: sonar (default "sonarxxx")
  -url string
        Required. Sonarqube URL
  -user string
        Required. Sonarqube User
  -password string
        Required. Sonarqube Password
  -port string
        Exporter port (default "8080")
  -scrape-timeout string
        Metrics scraper timeout. Default: 1m (default "1m")
  -label-separator string
        Label Separator. For instance, for Sonar with Label 'key#value', Prometheus attribute {project="my-project-name"} (default "#")
  -log string
        Logging level, e.g. debug,info. Default: debug (default "info")

```

## Run As Docker Container

```sh
  docker run -p 8080:8080 ghcr.io/fleetframework/sonarqube-prometheus-exporter:v0.0.2 -port 8080 -url <sonar-url> -user <sonar-user> -password <sonar-password>
```

or with environment variables

```sh
  docker run -p 8080:8080 -e PORT=8080 -e SONAR_URL=<sonar-url> -e SONAR_USER=<sonar-user> -e SONAR_PASSWORD=<sonar-password> ghcr.io/fleetframework/sonarqube-prometheus-exporter:v0.0.2
```
