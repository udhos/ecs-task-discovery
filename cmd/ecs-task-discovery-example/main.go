// Package main implements the tool.
package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/udhos/ecs-task-discovery/discovery"
)

func main() {

	cluster := os.Getenv("CLUSTER")
	if cluster == "" {
		cluster = "demo"
	}
	service := os.Getenv("SERVICE")
	if service == "" {
		service = "demo"
	}

	slog.Info(fmt.Sprintf("CLUSTER=%s", cluster))
	slog.Info(fmt.Sprintf("SERVICE=%s", service))

	disc, err := discovery.New(discovery.Options{
		Cluster:     cluster,
		ServiceName: service,
		Callback:    callback,
		Interval:    10 * time.Second,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("FATAL: %v", err))
		os.Exit(1)
	}
	go disc.Run()
	select {} // wait forever
}

func callback(tasks []discovery.Task) {
	for i, t := range tasks {
		slog.Info(fmt.Sprintf("task %d/%d", i+1, len(tasks)),
			"addr", t.Address,
			"healthStatus", t.HealthStatus,
			"lastStatus", t.LastStatus,
		)
	}
}
