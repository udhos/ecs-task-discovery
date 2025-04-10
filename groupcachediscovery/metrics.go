package groupcachediscovery

import (
	"fmt"
	"log/slog"

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type metrics struct {
	peers  prometheus.Gauge
	events prometheus.Counter

	dogstatsdClient *statsd.Client
	sampleRate      float64
	extraTags       []string
}

func (m *metrics) update(peers int) {

	peersFloat64 := float64(peers)

	//
	// Prometheus
	//
	if m.events != nil {
		m.events.Inc()
	}
	if m.peers != nil {
		m.peers.Set(peersFloat64)
	}

	//
	// Dogstatsd
	//
	if m.dogstatsdClient != nil {
		if err := m.dogstatsdClient.Count("events", 1, m.extraTags, m.sampleRate); err != nil {
			slog.Error(fmt.Sprintf("metrics.update: Count error: %v", err))
		}
		if err := m.dogstatsdClient.Gauge("peers", peersFloat64, m.extraTags, m.sampleRate); err != nil {
			slog.Error(fmt.Sprintf("metrics.update: Gauge error: %v", err))
		}
	}
}

func newMetrics(namespace string, registerer prometheus.Registerer,
	client *statsd.Client, dogstatsdExtraTags []string) *metrics {

	m := &metrics{
		dogstatsdClient: client,
		extraTags:       dogstatsdExtraTags,
		sampleRate:      1,
	}

	if registerer == nil {
		return m
	}

	const subsystem = "groupcachediscovery"

	m.peers = newGauge(
		registerer,
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "peers",
			Help:      "Number of peers discovered.",
		},
	)

	m.events = newCounter(
		registerer,
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "events",
			Help:      "Number of events received.",
		},
	)

	return m
}

func newGauge(registerer prometheus.Registerer,
	opts prometheus.GaugeOpts) prometheus.Gauge {
	return promauto.With(registerer).NewGauge(opts)
}

func newCounter(registerer prometheus.Registerer,
	opts prometheus.CounterOpts) prometheus.Counter {
	return promauto.With(registerer).NewCounter(opts)
}
