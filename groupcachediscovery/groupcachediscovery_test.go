package groupcachediscovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/groupcache/groupcache-go/v3/transport/peer"
	"github.com/udhos/ecs-task-discovery/discovery"
)

type capturePool struct {
	ch chan []string
}

func (c *capturePool) Set(peers ...string) {
	copyPeers := append([]string(nil), peers...)
	c.ch <- copyPeers
}

type capturePeerSet struct {
	ch chan []peer.Info
}

func (c *capturePeerSet) SetPeers(_ context.Context, peers []peer.Info) error {
	copyPeers := append([]peer.Info(nil), peers...)
	c.ch <- copyPeers
	return nil
}

type agentTask struct {
	ARN          string `json:"arn"`
	Address      string `json:"address"`
	HealthStatus string `json:"health_status"`
	LastStatus   string `json:"last_status"`
}

func newMetadataServer(t *testing.T, clusterARN string) *httptest.Server {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/task" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprintf(w, `{"Cluster":"%s"}`+"\n", clusterARN)
	}))

	t.Cleanup(ts.Close)
	return ts
}

func newAgentServer(t *testing.T, serviceName string, tasks []agentTask) *httptest.Server {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/tasks/" + serviceName
		if r.URL.Path != expectedPath {
			http.NotFound(w, r)
			return
		}
		if err := json.NewEncoder(w).Encode(tasks); err != nil {
			t.Fatalf("encode agent response: %v", err)
		}
	}))

	t.Cleanup(ts.Close)
	return ts
}

func TestNewGroupcacheV2PoolSetReceivesExpectedURLs(t *testing.T) {
	const myAddr = "10.0.0.10"
	oldFindMyAddr := findMyAddrFunc
	findMyAddrFunc = func() (string, error) { return myAddr, nil }
	t.Cleanup(func() { findMyAddrFunc = oldFindMyAddr })

	metadataServer := newMetadataServer(t, "arn:aws:ecs:us-east-1:111122223333:cluster/demo")
	t.Setenv("ECS_CONTAINER_METADATA_URI_V4", metadataServer.URL)

	serviceName := "svc"
	tasks := []agentTask{
		{ARN: "task-1", Address: myAddr, HealthStatus: "HEALTHY", LastStatus: "RUNNING"},
		{ARN: "task-2", Address: "10.0.0.11", HealthStatus: "HEALTHY", LastStatus: "RUNNING"},
	}
	agentServer := newAgentServer(t, serviceName, tasks)
	t.Setenv("ECS_TASK_DISCOVERY_AGENT_URL", agentServer.URL+"/tasks")

	pool := &capturePool{ch: make(chan []string, 1)}
	d, err := New(Options{
		Pool:                         pool,
		Client:                       ecs.NewFromConfig(aws.Config{}),
		GroupCachePort:               ":5000",
		ServiceName:                  serviceName,
		TaskDefinitionHasHealthCheck: discovery.HealthCheckModeFalse,
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer d.Stop()

	select {
	case got := <-pool.ch:
		expected := []string{"http://10.0.0.10:5000", "http://10.0.0.11:5000"}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("Pool.Set peers mismatch: expected=%v got=%v", expected, got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Pool.Set callback")
	}
}

func TestNewGroupcacheV3SetPeersReceivesExpectedPeerInfo(t *testing.T) {
	const myAddr = "10.0.0.10"
	oldFindMyAddr := findMyAddrFunc
	findMyAddrFunc = func() (string, error) { return myAddr, nil }
	t.Cleanup(func() { findMyAddrFunc = oldFindMyAddr })

	metadataServer := newMetadataServer(t, "arn:aws:ecs:us-east-1:111122223333:cluster/demo")
	t.Setenv("ECS_CONTAINER_METADATA_URI_V4", metadataServer.URL)

	serviceName := "svc"
	tasks := []agentTask{
		{ARN: "task-1", Address: myAddr, HealthStatus: "HEALTHY", LastStatus: "RUNNING"},
		{ARN: "task-2", Address: "10.0.0.11", HealthStatus: "HEALTHY", LastStatus: "RUNNING"},
	}
	agentServer := newAgentServer(t, serviceName, tasks)
	t.Setenv("ECS_TASK_DISCOVERY_AGENT_URL", agentServer.URL+"/tasks")

	peerSet := &capturePeerSet{ch: make(chan []peer.Info, 1)}
	d, err := New(Options{
		Peers:                        peerSet,
		Client:                       ecs.NewFromConfig(aws.Config{}),
		GroupCachePort:               ":5000",
		ServiceName:                  serviceName,
		TaskDefinitionHasHealthCheck: discovery.HealthCheckModeFalse,
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer d.Stop()

	select {
	case got := <-peerSet.ch:
		expected := []peer.Info{
			{Address: "10.0.0.10:5000", IsSelf: true},
			{Address: "10.0.0.11:5000", IsSelf: false},
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("SetPeers peer list mismatch: expected=%v got=%v", expected, got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SetPeers callback")
	}
}
