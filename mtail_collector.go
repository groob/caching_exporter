package main

import (
	"encoding/json"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/google/mtail/metrics"
	"github.com/google/mtail/mtail"
	"github.com/prometheus/client_golang/prometheus"
)

type mtailCollector struct {
	mtail   *mtail.Mtail
	metrics []*metrics.Metric
	mu      *sync.Mutex
}

func newMtailCollector(m *mtail.Mtail) *mtailCollector {
	c := &mtailCollector{mtail: m, mu: &sync.Mutex{}}
	err := c.mtail.StartTailing()
	if err != nil {
		log.Fatal(err)
	}
	return c
}

func (c *mtailCollector) process() {
	r, w := io.Pipe()
	go func(w *io.PipeWriter) {
		err := c.mtail.WriteMetrics(w)
		if err != nil {
			log.Println(err)
		}
	}(w)
	err := json.NewDecoder(r).Decode(&c.metrics)
	if err != nil {
		log.Println(err)
	}
}

func newDesc(m *metrics.Metric, l *metrics.LabelSet) *prometheus.Desc {
	labels := prometheus.Labels{}
	for k, v := range l.Labels {
		labels[k] = v
	}
	name := m.Name
	help := strings.ToLower(m.Kind.String())
	return prometheus.NewDesc(name, help, []string{}, labels)
}

func newMetric(m *metrics.Metric, l *metrics.LabelSet) (prometheus.Metric, error) {
	var value float64
	var valueType prometheus.ValueType
	switch m.Kind {
	case metrics.Counter:
		valueType = prometheus.CounterValue
		value = float64(l.Datum.Get())
	case metrics.Gauge:
		valueType = prometheus.GaugeValue
		value = float64(l.Datum.Get())
	}
	return prometheus.NewConstMetric(newDesc(m, l), valueType, value)
}

// Collect implements prometheus.Collector.
func (c *mtailCollector) Collect(ch chan<- prometheus.Metric) {
	c.process()
	for _, metric := range c.metrics {
		lc := make(chan *metrics.LabelSet)
		go metric.EmitLabelSets(lc)
		for l := range lc {
			m, err := newMetric(metric, l)
			if err != nil {
				log.Println(err)
				continue
			}
			ch <- m
		}
	}
}

// Describe implements prometheus.Collector.
func (c *mtailCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collected.Desc()
}
