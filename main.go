package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	helpCmd    bool
)

// nolint:gochecknoinits
func init() {
	flag.IntVar(&port, "port", 8080, "Exporter port")
	flag.DurationVar(&scrapeTimeout, "scrape-timeout", 1*time.Minute, "Metrics scraper timeout")
	flag.StringVar(&sonarURL, "url", "", "Required. Sonarqube URL")
	flag.StringVar(&sonarUser, "user", "", "Required. Sonarqube User")
	flag.StringVar(&sonarPassword, "password", "", "Required. Sonarqube Password")
	flag.StringVar(&labelSeparator, "label-separator", "#", "Label Separator. For instance, "+
		"for Sonar with Label 'key#value', Prometheus attribute {project=\"my-project-name\"}")

	flag.BoolVar(&versionCmd, "version", false, "Show version")
	flag.BoolVar(&helpCmd, "help", false, "Show help")

	flag.Parse()

	if versionCmd {
		fmt.Printf("Git Revision: %s\n", gitRevision)
		fmt.Printf("UTC Build Date: %s\n", buildDate)
		os.Exit(0)
	}
	if helpCmd {
		flag.Usage()
		os.Exit(0)
	}

	if sonarURL == "" || sonarUser == "" || sonarPassword == "" {
		flag.Usage()
		log.Fatal("make sure all required flags are provided")
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
		allMetrics, err := sonar.GetMetrics()
		if err != nil {
			log.Fatal(err)
		}

		exp := NewPrometheusExporter()
		metrics, err := exp.Init(component, allMetrics)
		if err != nil {
			log.Fatal(err)
		}

		schedule(done, 0, scrapeTimeout, func() error {
			measures, err := sonar.GetMeasures(componentKey, metrics)
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
				log.Printf("Scheduler error: %v\n", err)
			}
			attemptTimer = time.After(timeout)
			log.Println("Scheduler job run successfully")
		}
	}
}
