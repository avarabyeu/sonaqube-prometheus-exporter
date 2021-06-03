# Prometheus Exporter for Sonarqube

![Build Status](https://github.com/fleetframework/sonarqube-prometheus-exporter/actions/workflows/build.yml/badge.svg)
![Release](https://img.shields.io/github/v/release/fleetframework/sonarqube-prometheus-exporter)

## Usage

### CLI arguments

```
Usage of bin/sonarqube-prometheus-exporter:
  -help
        Show help
  -labels string
        Static labels to be added to all metrics. In form 'label1=labelvalue,label2=labelValue'
  -log string
        Logging level, e.g. debug,info. Default: debug (default "info")
  -metrics-ns string
        Prometheus metrics namespace. Default: sonar (default "sonar")
  -password string
        Required. Sonarqube Password
  -port string
        Exporter port. Default 8080 (default "8080")
  -scrape-timeout string
        Metrics scraper timeout. Default: 1m (default "1m")
  -tag-export-empty string
        Export projects that does not have tags defined in 'TAG_KEYS'. Prometheus labels will be set to empty in this case (default "TRUE")
  -tag-keys string
        List of tag keys to be used as metric labels
  -tag-separator string
        Tag Separator. For instance, for Sonar project with tag 'key#value', Prometheus will have label {project="my-project-name"} if defined in TAG_KEYS list (default "#")
  -url string
        Required. Sonarqube URL
  -user string
        Required. Sonarqube User
  -version
        Show version
```

### Environment Variables
| Variable Name        | Default Value | Description                                                                                                        |
|----------------------|---------------|--------------------------------------------------------------------------------------------------------------------|
| SONAR_URL            |               | Sonarqube URL                                                                                                      |
| SONAR_USER           |               | Sonarqube User                                                                                                     |
| SONAR_PASSWORD       |               | Sonarqube Password                                                                                                 |
| PORT                 | 8080          | Exporter port                                                                                                      |
| SONAR_SCRAPE_TIMEOUT | 1m            | Metrics scraper timeout                                                                                            |
| TAG_SEPARATOR        | #             | Tag Separator. For instance, for Sonar with tag 'key#value', Prometheus label {project="my-project-name"}          |
| TAG_KEYS             |               | List of tag keys to be used as metric labels. For instance, 'module,product'                                       |
| TAG_EXPORT_EMPTY     | TRUE          | Export projects that does not have tags defined in 'TAG_KEYS'. Prometheus labels will be set to empty in this case |
| LABELS               |               | Static labels to be added to all metrics. In form 'label1=labelvalue,label2=labelValue'                            |
| METRICS_NAMESPACE    | sonar         | Prometheus metrics namespace                                                                                       |
| LOGGING_LEVEL        | info          | Logging level, e.g. debug,info                                                                                     |

## Run As Docker Container

```sh
  docker run -p 8080:8080 ghcr.io/fleetframework/sonarqube-prometheus-exporter:v0.0.5 -port 8080 -url <sonar-url> -user <sonar-user> -password <sonar-password>
```

or with environment variables

```sh
  docker run -p 8080:8080 -e PORT=8080 -e SONAR_URL=<sonar-url> \
  -e SONAR_USER=<sonar-user> \
  -e SONAR_PASSWORD=<sonar-password> \
  ghcr.io/fleetframework/sonarqube-prometheus-exporter:v0.0.5
```
