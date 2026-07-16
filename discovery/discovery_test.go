package discovery

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

type captureTransport struct {
	requestedURL string
}

func (c *captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.requestedURL = req.URL.String()

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("[]")),
		Header:     make(http.Header),
	}, nil
}

func TestFindCluster(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, metadata)
	}))
	defer ts.Close()

	t.Setenv(envVarMetadataURI, ts.URL)

	clusterArn, err := FindCluster()
	if err != nil {
		t.Error(err)
	}

	const expected = "arn:aws:ecs:us-east-1:111122223333:cluster/demo"

	if clusterArn != expected {
		t.Errorf("bad cluster ARN: expected=%s got=%s", expected, clusterArn)
	}
}

func TestNewDiscoveryModes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, metadata)
	}))
	defer ts.Close()

	t.Setenv(envVarMetadataURI, ts.URL)

	// Create a dummy ecs client
	client := ecs.NewFromConfig(aws.Config{})

	t.Run("True mode", func(t *testing.T) {
		d, err := New(Options{
			ServiceName:                  "dummy-service",
			Client:                       client,
			TaskDefinitionHasHealthCheck: HealthCheckModeTrue,
			Callback:                     func(_ []Task) {},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer d.Stop()

		if !d.healthCheckEnabled {
			t.Error("expected healthCheckEnabled to be true")
		}
	})

	t.Run("False mode", func(t *testing.T) {
		d, err := New(Options{
			ServiceName:                  "dummy-service",
			Client:                       client,
			TaskDefinitionHasHealthCheck: HealthCheckModeFalse,
			Callback:                     func(_ []Task) {},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer d.Stop()

		if d.healthCheckEnabled {
			t.Error("expected healthCheckEnabled to be false")
		}
	})

	t.Run("Detect mode fails with dummy client", func(t *testing.T) {
		_, err := New(Options{
			ServiceName:                  "dummy-service",
			Client:                       client,
			TaskDefinitionHasHealthCheck: HealthCheckModeDetect,
			Callback:                     func(_ []Task) {},
		})
		if err == nil {
			t.Fatal("expected error in Detect mode with dummy client, got nil")
		}
	})

	t.Run("DetectAndHandleErrorAsFalse mode succeeds with dummy client", func(t *testing.T) {
		d, err := New(Options{
			ServiceName:                  "dummy-service",
			Client:                       client,
			TaskDefinitionHasHealthCheck: HealthCheckModeDetectAndHandleErrorAsFalse,
			Callback:                     func(_ []Task) {},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer d.Stop()

		if d.healthCheckEnabled {
			t.Error("expected healthCheckEnabled to be false")
		}
	})
}

func TestDiscoveryRunStopsQuickly(t *testing.T) {
	callbackDone := make(chan struct{}, 1)

	d := &Discovery{
		options: Options{
			ServiceName: "svc",
			Interval:    5 * time.Second,
			Callback: func(_ []Task) {
				select {
				case callbackDone <- struct{}{}:
				default:
				}
			},
			DisableAgentQuery: true,
			ForceSingleTask:   "127.0.0.1",
		},
		clusterName: "cluster",
		done:        make(chan struct{}),
	}

	exited := make(chan struct{})
	go func() {
		d.run()
		close(exited)
	}()

	select {
	case <-callbackDone:
		// first poll completed, loop should now be waiting for next interval
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first discovery cycle")
	}

	d.Stop()

	select {
	case <-exited:
		// exit should be prompt even with long interval
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Discovery.run did not stop quickly")
	}
}

func TestDiscoveryStopTwice(_ *testing.T) {
	d := &Discovery{done: make(chan struct{})}

	d.Stop()
	d.Stop()
}

func TestQueryAgentURLPrecedence(t *testing.T) {
	t.Run("Options.AgentURL takes precedence over env and default", func(t *testing.T) {
		t.Setenv(envAgentURL, "http://from-env.example/tasks")

		transport := &captureTransport{}
		d := &Discovery{
			options: Options{
				ServiceName: "svc",
				AgentURL:    "http://from-options.example/tasks",
			},
			clusterName: "demo-cluster",
			httpClient:  &http.Client{Transport: transport},
		}

		if _, err := d.queryAgent(); err != nil {
			t.Fatalf("queryAgent() unexpected error: %v", err)
		}

		const expected = "http://from-options.example/tasks/svc"
		if transport.requestedURL != expected {
			t.Fatalf("queryAgent() URL mismatch: expected=%q got=%q", expected, transport.requestedURL)
		}
	})

	t.Run("env takes precedence over default when option is empty", func(t *testing.T) {
		t.Setenv(envAgentURL, "http://from-env.example/tasks")

		transport := &captureTransport{}
		d := &Discovery{
			options: Options{
				ServiceName: "svc",
			},
			clusterName: "demo-cluster",
			httpClient:  &http.Client{Transport: transport},
		}

		if _, err := d.queryAgent(); err != nil {
			t.Fatalf("queryAgent() unexpected error: %v", err)
		}

		const expected = "http://from-env.example/tasks/svc"
		if transport.requestedURL != expected {
			t.Fatalf("queryAgent() URL mismatch: expected=%q got=%q", expected, transport.requestedURL)
		}
	})

	t.Run("default is used when option and env are empty", func(t *testing.T) {
		t.Setenv(envAgentURL, "")

		transport := &captureTransport{}
		d := &Discovery{
			options: Options{
				ServiceName: "svc",
			},
			clusterName: "demo-cluster",
			httpClient:  &http.Client{Transport: transport},
		}

		if _, err := d.queryAgent(); err != nil {
			t.Fatalf("queryAgent() unexpected error: %v", err)
		}

		const expected = "http://ecs-task-discovery-agent.demo-cluster:8080/tasks/svc"
		if transport.requestedURL != expected {
			t.Fatalf("queryAgent() URL mismatch: expected=%q got=%q", expected, transport.requestedURL)
		}
	})
}

const metadata = `{"Cluster":"arn:aws:ecs:us-east-1:111122223333:cluster/demo"}`
