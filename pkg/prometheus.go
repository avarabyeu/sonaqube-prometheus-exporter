package pkg

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	unsupportedMetricTypes = map[string]struct{}{"DATA": {}}
	promNamePattern        = regexp.MustCompile("[^a-zA-Z_:]")
	componentNameLabel     = "component"
)

// PrometheusExporter is response for converting Sonarqube metrics to Prometheus format and reporting them
type PrometheusExporter struct {
	metrics           map[string]*promMetric
	mut               sync.RWMutex
	ns                string
	labels            []string
	staticLabels      map[string]string
	exportEmptyLabels bool

	componentLabels map[string]prometheus.Labels
}

type promMetric struct {
	metric     *prometheus.GaugeVec
	metricType string
}

// NewPrometheusExporter creates new exporter instance
func NewPrometheusExporter(ns string, staticLabels map[string]string, labels []string, exportEmptyLabels bool) *PrometheusExporter {
	// make sure names are escaped
	for i, label := range labels {
		labels[i] = escapeName(label)
	}

	// adds default component name label
	labels = append(labels, componentNameLabel)
	sort.Strings(labels)

	p := &PrometheusExporter{
		ns:                ns,
		metrics:           map[string]*promMetric{},
		componentLabels:   map[string]prometheus.Labels{},
		mut:               sync.RWMutex{},
		staticLabels:      staticLabels,
		exportEmptyLabels: exportEmptyLabels,
		labels:            labels,
	}

	return p
}

func (pe *PrometheusExporter) InitMetrics(
	metrics []*Metric,
) ([]string, error) {
	return pe.registerMetrics(metrics)
}

func (pe *PrometheusExporter) Report(component string, labels map[string]string, measures *Measures) {
	pe.mut.Lock()
	defer pe.mut.Unlock()

	// adds default component name label
	pe.fillDefaults(labels, component)

	// make sure unknown labels are removed
	pe.filterSupported(labels)

	if len(labels) != len(pe.labels) {
		log.Debugf("Ignoreing component %s due to incorrect list of labels: [%s] != [%s]", component, labels, pe.labels)
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
		pe.checkStaleLabels(component, pMetric, labels)
	}
}

func (pe *PrometheusExporter) registerMetrics(metrics []*Metric) ([]string, error) {
	pe.mut.RLock()
	defer pe.mut.RUnlock()

	//nolint:prealloc
	var mNames []string
	for _, m := range metrics {
		if _, ok := pe.metrics[m.Key]; ok {
			// metric has already been registered
			continue
		}
		if !pe.supportsMetric(m) {
			// the metric is not supported
			log.Debugf("Metric %s[%s] isn't supported", m.Key, m.Name)
			continue
		}
		pMetric := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   pe.ns,
				Name:        m.Key,
				Help:        m.Description,
				ConstLabels: pe.staticLabels,
			}, pe.labels)
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

// checkStaleLabels
func (pe *PrometheusExporter) checkStaleLabels(component string, metric *promMetric, labels prometheus.Labels) {
	oldLabels, found := pe.componentLabels[component]
	if !found {
		pe.componentLabels[component] = labels
		return
	}
	if !reflect.DeepEqual(labels, oldLabels) {
		metric.metric.Delete(oldLabels)
		pe.componentLabels[component] = labels
	}
}

// fillDefaults adds default metrics
func (pe *PrometheusExporter) fillDefaults(labels map[string]string, component string) {
	// component label is default one
	labels[componentNameLabel] = component

	if pe.exportEmptyLabels {
		// if some initial labels are not filled out, assign empty string
		// to avoid inconsistent metric cardinality error
		for _, l := range pe.labels {
			if _, ok := labels[l]; !ok {
				labels[l] = ""
			}
		}
	}
}

// supportsLabel checks whether label is supported
// not list of labels MUST be ordered
func (pe *PrometheusExporter) supportsLabel(l string) bool {
	idx := sort.SearchStrings(pe.labels, l)
	return idx < len(pe.labels) && pe.labels[idx] == l
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

// escapeName escapes unsupported symbols
func escapeName(n string) string {
	return promNamePattern.ReplaceAllString(n, "_")
}
