package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var (
	port           int
	scrapeTimeout  time.Duration
	sonarURL       string
	sonarUser      string
	sonarPassword  string
	labelSeparator string
)

var (
	gitRevision = "HEAD"
	buildDate   = "unknown"

	versionCmd bool
)

// nolint:gochecknoinits
func init() {
	flag.IntVar(&port, "port", 8080, "Exporter port")
	flag.DurationVar(&scrapeTimeout, "scrape-timeout", 1*time.Minute, "Metrics scraper timeout")
	flag.StringVar(&sonarURL, "url", "", "Sonarqube URL")
	flag.StringVar(&sonarUser, "user", "", "Sonarqube User")
	flag.StringVar(&sonarPassword, "password", "", "Sonarqube Password")
	flag.StringVar(&labelSeparator, "label-separator", "#", "Label Separator. For instance, "+
		"for Sonar with Label 'key#value', Prometheus attribute {project=\"my-project-name\"}")

	flag.BoolVar(&versionCmd, "version", false, "Show version")

	flag.Parse()

	if versionCmd {
		fmt.Printf("Git Revision: %s\n", gitRevision)
		fmt.Printf("UTC Build Date: %s\n", buildDate)
		os.Exit(0)
	}
}

func main() {
	// Setting up signal capturing
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	done := make(chan struct{}, 1)
	go func() {
		<-stop
		close(done)
	}()

	m := http.NewServeMux()
	m.Handle("/metrics", promhttp.Handler())
	server := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: m}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
	go initMetrics(done)

	// Waiting for SIGINT (pkill -2)
	<-done

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Println(err)
	}
}

func getMetricNames(metrics []*Metric) []string {
	names := make([]string, len(metrics))
	for i, m := range metrics {
		names[i] = m.Key
	}
	return names
}

func initMetrics(done <-chan struct{}) {
	sonar := NewSonarClient(sonarURL, sonarUser, sonarPassword)
	components, err := sonar.GetComponents()
	if err != nil {
		log.Fatal(err)
	}

	for _, cInfo := range components {
		componentKey := cInfo.Key
		component, err := sonar.GetComponent(componentKey)
		if err != nil {
			log.Fatal(err)
		}
		metrics, err := sonar.GetMetrics()
		if err != nil {
			log.Fatal(err)
		}

		labels := map[string]string{}
		if labelSeparator != "" {
			for _, tag := range component.Tags {
				parts := strings.Split(tag, labelSeparator)
				if len(parts) == 2 {
					labels[parts[0]] = parts[1]
				}
			}
		}

		exp := NewPrometheusExporter()
		if err := exp.Init(componentKey, metrics, labels); err != nil {
			log.Fatal(err)
		}

		mNames := getMetricNames(metrics)
		schedule(done, 0, scrapeTimeout, func() error {
			measures, err := sonar.GetMeasures(componentKey, mNames)
			if err != nil {
				log.Fatal(err)
			}

			return exp.Run(measures)
		})
	}
}

// schedule executes action with defined timeout until receives timeout signal
func schedule(done <-chan struct{}, initialDelay, timeout time.Duration, callback func() error) {
	var err error

	attemptTimer := time.After(initialDelay)
	for {
		select {
		case <-done:
			return
		case <-attemptTimer:
			err = callback()
			if err != nil {
				logrus.Errorf("Scheduler error: %v", err)
			}
			attemptTimer = time.After(timeout)
			logrus.Trace("Scheduler job run successfully")
		}
	}
}
