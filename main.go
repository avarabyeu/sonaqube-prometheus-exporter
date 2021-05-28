package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
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

	labels map[string]string

	tagKeys      []string
	tagSeparator string

	loggingLevel string
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
	var labelsStr string
	var tagKeysStr string

	flag.StringVar(&port, "port", getEnv("PORT", "8080"), "Exporter port. Default 8080")
	flag.StringVar(&scrapeTimeoutStr, "scrape-timeout", getEnv("SONAR_SCRAPE_TIMEOUT", "1m"), "Metrics scraper timeout. Default: 1m")
	flag.StringVar(&sonarURL, "url", getEnv("SONAR_URL", ""), "Required. Sonarqube URL")
	flag.StringVar(&sonarUser, "user", getEnv("SONAR_USER", ""), "Required. Sonarqube User")
	flag.StringVar(&sonarPassword, "password", getEnv("SONAR_PASSWORD", ""), "Required. Sonarqube Password")
	flag.StringVar(&metricsNamespace, "metrics-ns", getEnv("METRICS_NAMESPACE", "sonar"), "Prometheus metrics namespace. Default: sonar")

	flag.StringVar(&labelsStr, "labels", getEnv("LABELS", ""), "Static labels to be added to all metrics. In form 'label1=labelvalue,label2=labelValue'")
	flag.StringVar(&tagKeysStr, "tag-keys", getEnv("TAG_KEYS", ""), "List of tag keys to be used as metric labels")
	flag.StringVar(&tagSeparator, "tag-separator", getEnv("TAG_SEPARATOR", "#"), "Tag Separator. For instance, "+
		"for Sonar project with tag 'key#value', Prometheus will have label {project=\"my-project-name\"} if defined in TAG_KEYS list")

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
	if _, err = strconv.ParseUint(port, 10, 32); err != nil {
		flag.Usage()
		log.Fatalf("incorrect port provided: %s", port)
	}

	// parses tag keys
	if tagKeysStr != "" {
		tagKeys = strings.Split(tagKeysStr, ",")
	}

	// parses static prometheus metrics
	labels, err = parseMap(labelsStr)
	if err != nil {
		flag.Usage()
		log.Fatalf(err.Error())
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
		if err := initScheduler(done); err != nil {
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

func initScheduler(done <-chan struct{}) error {
	sonar := NewSonarClient(sonarURL, sonarUser, sonarPassword)
	exporter := NewPrometheusExporter(metricsNamespace, labels, tagKeys)
	return NewCollector(sonar, exporter).Schedule(done, 0, scrapeTimeout)
}

func getEnv(name, def string) string {
	envVar := os.Getenv(name)
	if envVar == "" {
		envVar = def
	}
	return envVar
}

func parseMap(str string) (map[string]string, error) {
	m := map[string]string{}
	if str == "" {
		return m, nil
	}
	entries := strings.Split(str, ",")

	for _, entry := range entries {
		kv := strings.SplitN(entry, "=", 2)
		if len(kv) < 2 {
			return nil, fmt.Errorf("incorrect format: %s", str)
		}
		m[kv[0]] = kv[1]
	}
	return m, nil
}
