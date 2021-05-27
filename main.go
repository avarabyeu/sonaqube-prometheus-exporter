package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	port             string
	scrapeTimeout    time.Duration
	sonarURL         string
	sonarUser        string
	sonarPassword    string
	metricsNamespace string
	labelSeparator   string
	loggingLevel     string
)

var (
	gitRevision = "HEAD"
	buildDate   = "unknown"
	version     = "unknown"

	versionCmd bool
	helpCmd    bool
)

// nolint:gochecknoinits
func init() {
	var scrapeTimeoutStr string

	flag.StringVar(&port, "port", getEnv("PORT", "8080"), "Exporter port. Default 8080")
	flag.StringVar(&scrapeTimeoutStr, "scrape-timeout", getEnv("SONAR_SCRAPE_TIMEOUT", "1m"), "Metrics scraper timeout. Default: 1m")
	flag.StringVar(&sonarURL, "url", getEnv("SONAR_URL", ""), "Required. Sonarqube URL")
	flag.StringVar(&sonarUser, "user", getEnv("SONAR_USER", ""), "Required. Sonarqube User")
	flag.StringVar(&sonarPassword, "password", getEnv("SONAR_PASSWORD", ""), "Required. Sonarqube Password")
	flag.StringVar(&labelSeparator, "label-separator", getEnv("LABEL_SEPARATOR", "#"), "Label Separator. For instance, "+
		"for Sonar with Label 'key#value', Prometheus attribute {project=\"my-project-name\"}")
	flag.StringVar(&metricsNamespace, "metrics-ns", getEnv("METRICS_NAMESPACE", "sonarxxx"), "Prometheus metrics namespace. Default: sonar")
	flag.StringVar(&loggingLevel, "log", getEnv("LOGGING_LEVEL", "info"), "Logging level, e.g. debug,info. Default: debug")

	flag.BoolVar(&versionCmd, "version", false, "Show version")
	flag.BoolVar(&helpCmd, "help", false, "Show help")

	flag.Parse()

	if versionCmd {
		log.Printf("Version: %s\n", version)
		log.Printf("Git Revision: %s\n", gitRevision)
		log.Printf("UTC Build Date: %s\n", buildDate)
		os.Exit(0)
	}
	if helpCmd {
		flag.Usage()
		os.Exit(0)
	}

	initLogger(loggingLevel)

	var err error
	scrapeTimeout, err = time.ParseDuration(scrapeTimeoutStr)
	if err != nil {
		log.Fatalf("Unable to parse scrape duration")
	}

	if sonarURL == "" || sonarUser == "" || sonarPassword == "" {
		flag.Usage()
		log.Fatal("make sure all required flags are provided")
	}
	if _, err := strconv.ParseUint(port, 10, 32); err != nil {
		flag.Usage()
		log.Fatalf("incorrect port provided: %s", port)
	}
}

func initLogger(level string) {
	log.SetOutput(os.Stdout)
	l, err := log.ParseLevel(level)
	if err != nil {
		log.Fatal()
	}
	log.SetLevel(l)
	log.SetFormatter(&log.TextFormatter{ForceColors: true, FullTimestamp: true})
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
	server := &http.Server{Addr: fmt.Sprintf(":%s", port), Handler: m}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
	go func() {
		if err := initMetrics(done); err != nil {
			log.Fatalf("Unable to init metrics: %v", err)
		}
	}()

	// Waiting for SIGINT (pkill -2)
	<-done

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Error(err)
	}
}

func initMetrics(done <-chan struct{}) error {
	sonar := NewSonarClient(sonarURL, sonarUser, sonarPassword)
	components, err := sonar.GetComponents()
	if err != nil {
		return fmt.Errorf("unable to get sonar components: %w", err)
	}

	for _, cInfo := range components {
		componentKey := cInfo.Key
		component, err := sonar.GetComponent(componentKey)
		if err != nil {
			return err
		}
		allMetrics, err := sonar.GetMetrics()
		if err != nil {
			return fmt.Errorf("unable to get sonar metrics: %w", err)
		}

		exp := NewPrometheusExporter(metricsNamespace)
		metrics, err := exp.Init(component, allMetrics)
		if err != nil {
			return fmt.Errorf("unable to init metrics exporter: %w", err)
		}

		schedule(done, 0, scrapeTimeout, func() error {
			measures, err := sonar.GetMeasures(componentKey, metrics)
			if err != nil {
				return fmt.Errorf("unable to get sonar measures: %w", err)
			}

			return exp.Run(measures)
		})
	}
	return nil
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

func getEnv(name, def string) string {
	envVar := os.Getenv(name)
	if envVar == "" {
		envVar = def
	}
	return envVar
}
