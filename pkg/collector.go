package pkg

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// Collector schedules measures collection and executes exporter in order to get them converted to prometheus format
type Collector struct {
	sonar             *SonarClient
	namespace         string
	tagSeparator      string
	tagLabels         []string
	staticLabels      map[string]string
	exportEmptyLabels bool
}

func NewCollector(
	sonar *SonarClient,
	tagSeparator, namespace string,
	staticLabels map[string]string, labels []string,
	exportEmptyLabels bool,
) *Collector {
	return &Collector{
		sonar:             sonar,
		tagSeparator:      tagSeparator,
		namespace:         namespace,
		tagLabels:         labels,
		exportEmptyLabels: exportEmptyLabels,
		staticLabels:      staticLabels,
	}
}

func (c *Collector) Schedule(done <-chan struct{}, initialDelay, timeout time.Duration) error {
	allMetrics, err := c.sonar.GetMetrics()
	if err != nil {
		return fmt.Errorf("unable to get sonar metrics: %w", err)
	}

	exporter := NewPrometheusExporter(c.namespace, c.staticLabels, c.tagLabels, c.exportEmptyLabels)

	// registers metrics to be gathered
	metricNames, err := exporter.InitMetrics(allMetrics)
	if err != nil {
		return fmt.Errorf("unable to init metrics exporter: %w", err)
	}
	log.Debugf("Metrics to be collected\n: %s", strings.Join(metricNames, ","))
	if len(metricNames) == 0 {
		return fmt.Errorf("no metrics to gather detected")
	}

	go c.schedule(done, initialDelay, timeout, func() error {
		// all components which are projects
		components, err := c.sonar.SearchComponents()
		if err != nil {
			return fmt.Errorf("unable to get all sonar components: %w", err)
		}

		// iterate over all components
		for _, cInfo := range components {
			go func(componentKey string) {
				defer func() {
					if r := recover(); r != nil {
						log.Errorf("unable to report component: %v", r)
					}
				}()
				c.reportComponent(exporter, componentKey, metricNames)
			}(cInfo.Key)
		}
		return nil
	})
	return nil
}

func (c *Collector) reportComponent(exporter *PrometheusExporter, componentKey string, metricNames []string) {
	log.Debugf("Updating metrics for project: %s", componentKey)

	// get component. Selected on each iteration since
	// list of tags can be changed
	component, cErr := c.sonar.GetComponent(componentKey)
	if cErr != nil {
		log.Errorf("unable to find component [%s]: %v", componentKey, cErr)
	}

	// get component measures to be transformed to prometheus metrics
	measures, mErr := c.sonar.GetMeasures(component.Key, metricNames)
	if mErr != nil {
		log.Errorf("unable to get sonar measures: %v", mErr)
	}

	labels := c.tagsToLabels(component.Tags)
	exporter.Report(component.Key, labels, measures)
}

// schedule executes action with defined timeout until receives timeout signal
func (c *Collector) schedule(done <-chan struct{}, initialDelay, timeout time.Duration, callback func() error) {
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

// tagsToLabels converts Sonar's project tags to Prometheus's labels
// tags are supposed to be separated with separator, e.g. key#value
func (c *Collector) tagsToLabels(tags []string) map[string]string {
	labels := map[string]string{}
	if c.tagSeparator != "" {
		for _, tag := range tags {
			parts := strings.SplitN(tag, c.tagSeparator, 2)
			if len(parts) == 2 {
				labels[escapeName(parts[0])] = parts[1]
			}
		}
	}
	return labels
}
