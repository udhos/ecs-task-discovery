package groupcachediscovery

import (
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/udhos/aws-emf/emf"
	"github.com/udhos/cloudwatchlog/cwlog"
	"github.com/udhos/dogstatsdclient/dogstatsdclient"
)

type metrics struct {
	//
	// prometheus
	//
	peers  prometheus.Gauge
	events prometheus.Counter

	//
	// datadog
	//
	dogstatsdClient dogstatsdclient.DogstatsdClient
	sampleRate      float64
	extraTags       []string

	//
	// aws cloudwatch emf
	//
	metricContext *emf.Metric
	cwlogClient   *cwlog.Log
	dimensions    map[string]string
}

var (
	metricEvents = emf.MetricDefinition{Name: "events", Unit: "Count"}
	metricPeers  = emf.MetricDefinition{Name: "peers", Unit: "Count"}
)

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

	//
	// EMF
	//
	if m.metricContext != nil {

		namespace := "groupcachediscovery"

		m.metricContext.Record(namespace, metricEvents, m.dimensions, 1)
		m.metricContext.Record(namespace, metricPeers, m.dimensions, int(peers))

		if m.cwlogClient == nil {
			// send metrics to stdout
			m.metricContext.Println()
		} else {
			// send metrics to cloudwatch logs
			events := m.metricContext.CloudWatchLogEvents()
			if err := m.cwlogClient.PutLogEvents(events); err != nil {
				slog.Error(fmt.Sprintf("metrics.update export error: %v", err))
			}
		}
	}
}

func newMetrics(namespace string, registerer prometheus.Registerer,
	client dogstatsdclient.DogstatsdClient, dogstatsdExtraTags []string,
	emfEnable, emfSend bool, emfApplication string,
	emfDimensions map[string]string,
	awsConfig *aws.Config) (*metrics, error) {

	m := &metrics{
		dogstatsdClient: client,
		extraTags:       dogstatsdExtraTags,
		sampleRate:      1,
		dimensions:      emfDimensions,
	}

	if registerer != nil {
		m.peers = newGauge(
			registerer,
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "peers",
				Help:      "Number of peers discovered.",
			},
		)
		m.events = newCounter(
			registerer,
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "events",
				Help:      "Number of events received.",
			},
		)
	}

	if emfEnable {
		m.metricContext = emf.New(emf.Options{})
		if emfSend {
			if awsConfig == nil {
				return nil, fmt.Errorf("EmfSend: need non-nil AwsConfig to send EMF directly to CloudWatch logs")
			}
			cw, errCwlog := cwlog.New(cwlog.Options{
				AwsConfig: *awsConfig,
				LogGroup:  "/groupcachediscovery/" + emfApplication,
			})
			if errCwlog != nil {
				return nil, errCwlog
			}
			m.cwlogClient = cw
		}
	}

	return m, nil
}

func newGauge(registerer prometheus.Registerer,
	opts prometheus.GaugeOpts) prometheus.Gauge {
	return promauto.With(registerer).NewGauge(opts)
}

func newCounter(registerer prometheus.Registerer,
	opts prometheus.CounterOpts) prometheus.Counter {
	return promauto.With(registerer).NewCounter(opts)
}
