// Package main implements the tool.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/udhos/boilerplate/awsconfig"
	"github.com/udhos/boilerplate/boilerplate"
	"github.com/udhos/ecs-task-discovery/discovery"
	"github.com/udhos/ecs-task-discovery/internal/shared"
)

func main() {
	//
	// command-line
	//
	var showVersion bool
	flag.BoolVar(&showVersion, "version", showVersion, "show version")
	flag.Parse()

	me := filepath.Base(os.Args[0])

	//
	// version
	//
	{
		v := boilerplate.LongVersion(me + " version=" + shared.Version)
		if showVersion {
			fmt.Print(v)
			fmt.Println()
			return
		}
		slog.Info(v)
	}

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

	awsConfig := mustAwsConfig()

	disc, err := discovery.New(discovery.Options{
		Cluster:     cluster,
		ServiceName: service,
		Callback:    callback,
		Interval:    10 * time.Second,
		Client:      ecs.NewFromConfig(awsConfig),
	})
	if err != nil {
		fatalf("discovery.New: error: %v", err)
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

func mustAwsConfig() aws.Config {
	awsCfg, errCfg := awsconfig.AwsConfig(awsconfig.Options{})
	if errCfg != nil {
		fatalf("aws config error: %v", errCfg)
	}
	return awsCfg.AwsConfig
}

func fatalf(format string, a ...any) {
	slog.Error("FATAL: " + fmt.Sprintf(format, a...))
	os.Exit(1)
}
