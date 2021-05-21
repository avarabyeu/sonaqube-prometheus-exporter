package main

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var unsupportedTypes = map[string]struct{}{"DATA": {}}

type PrometheusExporter struct {
	metrics map[string]*promMetric
	mut     sync.Mutex
}

type promMetric struct {
	metric     *prometheus.Gauge
	metricType string
}

func NewPrometheusExporter() *PrometheusExporter {
	return &PrometheusExporter{
		metrics: map[string]*promMetric{},
		mut:     sync.Mutex{},
	}
}

func (pe *PrometheusExporter) Init(component string, metrics []*Metric, labels map[string]string) error {
	r := regexp.MustCompile("[^a-zA-Z_:]")
	compName := r.ReplaceAllString(component, "_")
	for _, m := range metrics {
		if _, unsupported := unsupportedTypes[m.Type]; unsupported {
			continue
		}
		pMetric := prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   "sonar",
				Subsystem:   compName,
				Name:        m.Key,
				Help:        m.Description,
				ConstLabels: labels,
			})
		if err := prometheus.Register(pMetric); err != nil {
			return fmt.Errorf("unable to register metric: %w", err)
		}
		pe.metrics[m.Key] = &promMetric{
			metric:     &pMetric,
			metricType: m.Type,
		}
	}
	return nil
}

func (pe *PrometheusExporter) Run(measures *Measures) error {
	pe.mut.Lock()
	defer pe.mut.Unlock()

	for _, measure := range measures.Component.Measures {
		pMetric, found := pe.metrics[measure.Metric]
		if !found || pMetric == nil {
			log.Printf("NO METRIC FOUND: %s", measure.Metric)

			continue
		}

		val, err := getFloatValue(pMetric.metricType, measure)
		if err != nil {
			log.Printf("Unable to convert metric: %s[%s]", measure.Metric, measure.Value)

			continue
		}
		(*pMetric.metric).Add(val)
	}
	return nil
}

func getFloatValue(mType string, measure *Measure) (fVar float64, err error) {
	var strVal string
	if measure.Value != "" {
		strVal = measure.Value
	} else {
		strVal = measure.Period.Value
	}

	if mType == "BOOL" {
		bVar, pErr := strconv.ParseBool(strVal)
		if pErr == nil {
			if bVar {
				fVar = 1
			} else {
				fVar = 0
			}
		}
	} else {
		fVar, err = strconv.ParseFloat(strVal, 64)
	}
	return
}

// nolint:deadcode
func getMetric(name string, metrics []*Metric) *Metric {
	for _, m := range metrics {
		if m.Name == name {
			return m
		}
	}
	log.Printf("NO METRIC FOUND: %s", name)
	return nil
}
