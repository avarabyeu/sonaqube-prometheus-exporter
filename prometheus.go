package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	unsupportedMetricTypes = map[string]struct{}{"DATA": {}}
	promNamePattern        = regexp.MustCompile("[^a-zA-Z_:]")
	componentNameLabel     = "component"
)

type PrometheusExporter struct {
	metrics map[string]*promMetric
	mut     sync.RWMutex
	ns      string
	labels  []string
}

type promMetric struct {
	metric     *prometheus.GaugeVec
	metricType string
}

func NewPrometheusExporter(ns string, labels []string) *PrometheusExporter {
	p := &PrometheusExporter{
		ns:      ns,
		metrics: map[string]*promMetric{},
		mut:     sync.RWMutex{},
	}

	// make sure label names are OK
	for idx, l := range labels {
		labels[idx] = p.escapeName(l)
	}
	// adds default component name label
	labels = append(labels, componentNameLabel)
	sort.Strings(labels)
	p.labels = labels

	return p
}

func (pe *PrometheusExporter) InitMetrics(
	staticLabels map[string]string,
	metrics []*Metric,
) ([]string, error) {
	return pe.registerMetrics(pe.labels, staticLabels, metrics)
}

func (pe *PrometheusExporter) Report(component *Component, measures *Measures) {
	pe.mut.Lock()
	defer pe.mut.Unlock()

	labels := pe.tagsToLabels(component.Tags)
	// adds default component name label
	labels[componentNameLabel] = component.Key
	pe.filterSupported(labels)

	if len(labels) != len(pe.labels) {
		log.Debugf("Ignoreing component %s due to incorrect list of labels: [%s] != [%s]", component.Key, labels, pe.labels)
		return
	}

	for _, measure := range measures.Component.Measures {
		pMetric, found := pe.metrics[measure.Metric]
		if !found || pMetric == nil {
			log.Debugf("Metric isn't found: %s", measure.Metric)

			continue
		}

		val, err := pe.getFloatValue(pMetric.metricType, measure)
		if err != nil {
			log.Debugf("Unable to convert metric: %s[%s]", measure.Metric, measure.Value)

			continue
		}

		(*pMetric.metric).With(labels).Set(val)
	}
}

func (pe *PrometheusExporter) registerMetrics(labelNames []string, labels map[string]string, metrics []*Metric) ([]string, error) {
	pe.mut.RLock()
	defer pe.mut.RUnlock()

	var mNames []string
	for _, m := range metrics {
		if _, ok := pe.metrics[m.Key]; ok {
			// metric has already been registered
			continue
		}
		if !pe.supportsMetric(m) {
			// the metric is not supported
			continue
		}
		pMetric := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   pe.ns,
				Name:        m.Key,
				Help:        m.Description,
				ConstLabels: labels,
			}, labelNames)
		if err := prometheus.Register(pMetric); err != nil {
			return nil, fmt.Errorf("unable to register metric: %w", err)
		}
		pe.metrics[m.Key] = &promMetric{
			metric:     pMetric,
			metricType: m.Type,
		}
		mNames = append(mNames, m.Key)
	}
	return mNames, nil
}

// tagsToLabels converts Sonar's project tags to Prometheus's labels
// tags are supposed to be separated with separator, e.g. key#value
func (pe *PrometheusExporter) tagsToLabels(tags []string) map[string]string {
	labels := map[string]string{}
	if tagSeparator != "" {
		for _, tag := range tags {
			parts := strings.SplitN(tag, tagSeparator, 2)
			if len(parts) == 2 {
				labels[pe.escapeName(parts[0])] = parts[1]
			}
		}
	}
	return labels
}

// tagsToLabels converts Sonar's project tags to Prometheus's labels
func (pe *PrometheusExporter) tagsToLabelNames(tags []string) []string {
	var labels []string
	if tagSeparator != "" {
		for _, tag := range tags {
			parts := strings.Split(tag, tagSeparator)
			if len(parts) == 2 {
				labels = append(labels, pe.escapeName(parts[0]))
			}
		}
	}
	return labels
}

func (pe *PrometheusExporter) supportsMetric(m *Metric) bool {
	_, unsupported := unsupportedMetricTypes[m.Type]
	return !unsupported
}

// filterSupported removes unsupported labels
func (pe *PrometheusExporter) filterSupported(labels map[string]string) {
	for k := range labels {
		if !pe.supportsLabel(k) {
			delete(labels, k)
		}
	}
}

// supportsLabel checks whether label is supported
// not list of labels MUST be ordered
func (pe *PrometheusExporter) supportsLabel(l string) bool {
	idx := sort.SearchStrings(pe.labels, l)
	return idx < len(pe.labels) && pe.labels[idx] == l
}

// escapeName escapes unsupported symbols
func (pe *PrometheusExporter) escapeName(n string) string {
	return promNamePattern.ReplaceAllString(n, "_")
}

// getFloatValue gets value from measure converting it to float64 as prometheus requires
func (pe *PrometheusExporter) getFloatValue(mType string, measure *Measure) (fVar float64, err error) {
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
