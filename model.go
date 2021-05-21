package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const sonarDateFormat = "2006-01-02T15:04:05-0700"

type Paging struct {
	PageIndex int `json:"pageIndex"`
	PageSize  int `json:"pageSize"`
	Total     int `json:"total"`
}

type ComponentInfo struct {
	Organization string `json:"organization,omitempty"`
	Key          string `json:"key,omitempty"`
	Name         string `json:"name,omitempty"`
	Qualifier    string `json:"qualifier,omitempty"`
	Project      string `json:"project,omitempty"`
}

type Component struct {
	ComponentInfo
	Description    string    `json:"description,omitempty"`
	AnalysisDate   sonarDate `json:"analysisDate,omitempty"`
	Tags           []string  `json:"tags,omitempty"`
	Visibility     string    `json:"visibility,omitempty"`
	LeakPeriodDate sonarDate `json:"leakPeriodDate,omitempty"`
	Version        string    `json:"version,omitempty"`
	NeedIssueSync  bool      `json:"needIssueSync,omitempty"`
}

type Components struct {
	Paging     *Paging          `json:"paging,omitempty"`
	Components []*ComponentInfo `json:"components,omitempty"`
}

type Metrics struct {
	Metrics []*Metric `json:"metrics,omitempty"`
	Total   int       `json:"total"`
	P       int       `json:"p"`
	Ps      int       `json:"ps"`
}

type Metric struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Domain      string `json:"domain"`
	Direction   int    `json:"direction"`
	Qualitative bool   `json:"qualitative"`
	Hidden      bool   `json:"hidden"`
	Custom      bool   `json:"custom"`
}

type Measures struct {
	Component struct {
		Key       string     `json:"key"`
		Name      string     `json:"name"`
		Qualifier string     `json:"qualifier"`
		Language  string     `json:"language"`
		Path      string     `json:"path"`
		Measures  []*Measure `json:"measures"`
	} `json:"component"`
	Period  *Period   `json:"period"`
	Metrics []*Metric `json:"metrics"`
}

type Measure struct {
	Metric string `json:"metric"`
	Value  string `json:"value,omitempty"`
	Period struct {
		Value     string `json:"value"`
		BestValue bool   `json:"bestValue"`
	} `json:"period"`
}

type Period struct {
	Mode      string    `json:"mode"`
	Date      sonarDate `json:"date"`
	Parameter string    `json:"parameter"`
}

// Date type alias
type sonarDate time.Time

func (j *sonarDate) UnmarshalJSON(b []byte) error {
	t, err := time.Parse(sonarDateFormat, strings.Trim(string(b), "\""))
	if err != nil {
		return fmt.Errorf("unable to parse date: %w", err)
	}
	*j = sonarDate(t)
	return nil
}

func (j sonarDate) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.format(sonarDateFormat))
}

func (j sonarDate) format(s string) string {
	t := time.Time(j)

	return t.Format(s)
}
