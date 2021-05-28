package main

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// Collector schedules measures collection and executes exporter in order to get them converted to prometheus format
type Collector struct {
	sonar    *SonarClient
	exporter *PrometheusExporter
}

func NewCollector(sonar *SonarClient, exporter *PrometheusExporter) *Collector {
	return &Collector{sonar: sonar, exporter: exporter}
}

func (c *Collector) Schedule(done <-chan struct{}, initialDelay, timeout time.Duration) error {
	allMetrics, err := c.sonar.GetMetrics()
	if err != nil {
		return fmt.Errorf("unable to get sonar metrics: %w", err)
	}

	// registers metrics to be gathered
	metricNames, err := c.exporter.InitMetrics(allMetrics)
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
			go c.reportComponent(metricNames)(cInfo.Key)
		}
		return nil
	})
	return nil
}

func (c *Collector) reportComponent(metricNames []string) func(componentKey string) {
	return func(componentKey string) {
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

		c.exporter.Report(component, measures)
	}
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
