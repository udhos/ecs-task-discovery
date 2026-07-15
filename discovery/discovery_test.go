package discovery

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

func TestFindCluster(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, metadata)
	}))
	defer ts.Close()

	t.Setenv(envVarMetadataURI, ts.URL)

	clusterArn, err := FindCluster(NewHTTPClient())
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

const metadata = `{"Cluster":"arn:aws:ecs:us-east-1:111122223333:cluster/demo"}`
