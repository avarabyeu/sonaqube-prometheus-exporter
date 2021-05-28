package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

type SonarClient struct {
	c        *http.Client
	url      string
	user     string
	password string
}

func NewSonarClient(url, user, password string) *SonarClient {
	return &SonarClient{url: strings.TrimRight(url, "/"), user: user, password: password, c: http.DefaultClient}
}

func (s *SonarClient) SearchComponents() ([]*ComponentInfo, error) {
	var c Components
	err := s.executeGet(fmt.Sprintf("%s/api/components/search?qualifiers=TRK", s.url), &c)
	if err != nil {
		return nil, err
	}

	return c.Components, err
}

func (s *SonarClient) GetComponent(key string) (*Component, error) {
	var c struct {
		Component *Component `json:"component,omitempty"`
	}
	return c.Component, s.executeGet(fmt.Sprintf("%s/api/components/show?component=%s", s.url, key), &c)
}

func (s *SonarClient) GetMetrics() ([]*Metric, error) {
	var m Metrics
	err := s.executeGet(fmt.Sprintf("%s/api/metrics/search", s.url), &m)
	if err != nil {
		return nil, err
	}
	return m.Metrics, err
}

func (s *SonarClient) GetMeasures(key string, metrics []string) (*Measures, error) {
	var m Measures
	err := s.executeGet(fmt.Sprintf("%s/api/measures/component?component=%s&metricKeys=%s", s.url, key, strings.Join(metrics, ",")), &m)
	if err != nil {
		return nil, err
	}
	return &m, err
}

func (s *SonarClient) executeGet(u string, res interface{}) error {
	rq, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u, nil)
	if err != nil {
		return fmt.Errorf("unable to build request: %w", err)
	}
	rq.SetBasicAuth(s.user, s.password)

	log.Debugf("GET [%s]", rq.URL.String())

	rs, err := s.c.Do(rq)
	if err != nil {
		return fmt.Errorf("unable to execute request: %w", err)
	}
	defer func() {
		if rs.Body != nil {
			if err := rs.Body.Close(); err != nil {
				log.Error(err)
			}
		}
	}()
	if rs.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(rs.Body)
		return fmt.Errorf("request failed. status code %d. Error: %s", rs.StatusCode, string(body))
	}

	return json.NewDecoder(rs.Body).Decode(res)
}
