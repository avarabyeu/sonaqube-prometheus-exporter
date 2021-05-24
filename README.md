# Prometheus Exporter for Sonarqube

## Usage

```
  -help
        Show help
  -label-separator string
        Label Separator. For instance, for Sonar with Label 'key#value', Prometheus attribute {project="my-project-name"} (default "#")
  -password string
        Sonarqube Password
  -port int
        Exporter port (default 8080)
  -scrape-timeout duration
        Metrics scraper timeout (default 1m0s)
  -url string
        Sonarqube URL
  -user string
        Sonarqube User
  -version
        Show version

```

## Run As Docker Container

```sh
  docker run -p 8080:8080 ghcr.io/avarabyeu/sonarqube-prometheus-exporter:v0.0.1 -port 8080 -url <sonar-url> -user <sonar-user> -password <sonar-password>
```