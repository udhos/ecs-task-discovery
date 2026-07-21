package main

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/udhos/ecs-task-discovery/discovery"
)

func TestServeHTTPWithoutGroupcacheSuccess(t *testing.T) {
	var gotService string
	app := &application{
		clusterName:      "demo",
		groupcacheEnable: false,
		findTasksFunc: func(_ context.Context, serviceName string) ([]byte, error) {
			gotService = serviceName
			return []byte(`[{"arn":"a","address":"10.0.0.1"}]`), nil
		},
	}

	req := httptest.NewRequest("GET", "/tasks/svc-a", nil)
	req.SetPathValue("service", "svc-a")
	res := httptest.NewRecorder()

	app.ServeHTTP(res, req)

	if gotService != "svc-a" {
		t.Fatalf("expected serviceName %q, got %q", "svc-a", gotService)
	}

	if res.Code != 200 {
		t.Fatalf("expected status 200, got %d", res.Code)
	}

	if got := res.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("expected Content-Type=%q, got %q", "application/json; charset=utf-8", got)
	}

	if got := res.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options=%q, got %q", "nosniff", got)
	}

	if got := strings.TrimSpace(res.Body.String()); got != `[{"arn":"a","address":"10.0.0.1"}]` {
		t.Fatalf("unexpected response body: %q", got)
	}
}

func TestServeHTTPWithoutGroupcacheError(t *testing.T) {
	app := &application{
		clusterName:      "demo",
		groupcacheEnable: false,
		findTasksFunc: func(_ context.Context, _ string) ([]byte, error) {
			return nil, errors.New("boom")
		},
	}

	req := httptest.NewRequest("GET", "/tasks/svc-a", nil)
	req.SetPathValue("service", "svc-a")
	res := httptest.NewRecorder()

	app.ServeHTTP(res, req)

	if res.Code != 500 {
		t.Fatalf("expected status 500, got %d", res.Code)
	}

	if !strings.Contains(res.Body.String(), "boom") {
		t.Fatalf("expected error response to contain %q, got %q", "boom", res.Body.String())
	}
}

func TestServeHTTPWithGroupcacheUsesCacheGetter(t *testing.T) {
	var cacheCalled bool
	var findCalled bool

	app := &application{
		clusterName:      "demo",
		groupcacheEnable: true,
		cacheGetFunc: func(_ context.Context, serviceName string) ([]byte, error) {
			cacheCalled = true
			if serviceName != "svc-cache" {
				t.Fatalf("expected cache serviceName %q, got %q", "svc-cache", serviceName)
			}
			return []byte(`[{"arn":"cached"}]`), nil
		},
		findTasksFunc: func(_ context.Context, _ string) ([]byte, error) {
			findCalled = true
			return []byte(`[]`), nil
		},
	}

	req := httptest.NewRequest("GET", "/tasks/svc-cache", nil)
	req.SetPathValue("service", "svc-cache")
	res := httptest.NewRecorder()

	app.ServeHTTP(res, req)

	if !cacheCalled {
		t.Fatal("expected cacheGetFunc to be called")
	}

	if findCalled {
		t.Fatal("did not expect findTasksFunc to be called when groupcache is enabled")
	}

	if res.Code != 200 {
		t.Fatalf("expected status 200, got %d", res.Code)
	}

	if got := strings.TrimSpace(res.Body.String()); got != `[{"arn":"cached"}]` {
		t.Fatalf("unexpected response body: %q", got)
	}
}

func TestHandlerHealth(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	res := httptest.NewRecorder()

	handlerHealth(res, req)

	if res.Code != 200 {
		t.Fatalf("expected status 200, got %d", res.Code)
	}

	if got := strings.TrimSpace(res.Body.String()); got != "ok" {
		t.Fatalf("unexpected health response: %q", got)
	}
}

func TestMetricsHandler(t *testing.T) {
	app := &application{registry: prometheus.NewRegistry()}
	h := app.metricsHandler()

	req := httptest.NewRequest("GET", "/metrics", nil)
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != 200 {
		t.Fatalf("expected status 200, got %d", res.Code)
	}
}

func TestFindTasksSuccess(t *testing.T) {
	oldDiscoveryTasksFunc := discoveryTasksFunc
	discoveryTasksFunc = func(_ context.Context, _ *ecs.Client, cluster, serviceName string) ([]discovery.Task, error) {
		if cluster != "demo" {
			t.Fatalf("expected cluster %q, got %q", "demo", cluster)
		}
		if serviceName != "svc" {
			t.Fatalf("expected service %q, got %q", "svc", serviceName)
		}
		return []discovery.Task{{ARN: "a", Address: "10.0.0.1", HealthStatus: "HEALTHY", LastStatus: "RUNNING"}}, nil
	}
	t.Cleanup(func() { discoveryTasksFunc = oldDiscoveryTasksFunc })

	data, err := findTasks(context.Background(), nil, "demo", "svc")
	if err != nil {
		t.Fatalf("findTasks() unexpected error: %v", err)
	}

	body := string(data)
	if !strings.Contains(body, `"arn":"a"`) {
		t.Fatalf("expected ARN in json body, got %q", body)
	}
}

func TestFindTasksError(t *testing.T) {
	oldDiscoveryTasksFunc := discoveryTasksFunc
	discoveryTasksFunc = func(_ context.Context, _ *ecs.Client, _, _ string) ([]discovery.Task, error) {
		return nil, errors.New("discovery failure")
	}
	t.Cleanup(func() { discoveryTasksFunc = oldDiscoveryTasksFunc })

	_, err := findTasks(context.Background(), nil, "demo", "svc")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "cluster=demo") || !strings.Contains(errStr, "service=svc") || !strings.Contains(errStr, "discovery failure") {
		t.Fatalf("unexpected findTasks error: %q", errStr)
	}
}
