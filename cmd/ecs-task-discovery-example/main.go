// Package main implements the tool.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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
	var repeat int
	var taskDefinitionHealthCheckMode string
	flag.BoolVar(&showVersion, "version", showVersion, "show version")
	flag.IntVar(&repeat, "repeat", repeat, "repeat count")
	flag.StringVar(&taskDefinitionHealthCheckMode, "task-definition-health-check-mode", "", "task definition health check mode: detect|detectandhandleerrorasfalse|true|false")
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

	service := os.Getenv("SERVICE")
	if service == "" {
		service = "ecs-task-discovery-example"
	}

	mode := taskDefinitionHealthCheckMode
	if mode == "" {
		mode = os.Getenv("TASK_DEFINITION_HEALTH_CHECK_MODE")
	}
	if mode == "" {
		mode = string(discovery.HealthCheckModeDetect)
	}
	mode = strings.ToLower(mode)

	healthCheckMode := discovery.HealthCheckMode(mode)
	switch healthCheckMode {
	case discovery.HealthCheckModeDetect,
		discovery.HealthCheckModeDetectAndHandleErrorAsFalse,
		discovery.HealthCheckModeTrue,
		discovery.HealthCheckModeFalse:
		// valid
	default:
		fatalf("invalid task definition health check mode: %s", mode)
	}

	slog.Info(fmt.Sprintf("SERVICE=%s", service))
	slog.Info(fmt.Sprintf("TASK_DEFINITION_HEALTH_CHECK_MODE=%s", healthCheckMode))

	awsConfig := mustAwsConfig()

	var count int

	for ; ; repeat-- {
		disc, err := discovery.New(discovery.Options{
			ServiceName:                  service,
			Callback:                     callback,
			Interval:                     10 * time.Second,
			Client:                       ecs.NewFromConfig(awsConfig),
			TaskDefinitionHasHealthCheck: healthCheckMode,
		})
		if err != nil {
			fatalf("discovery.New: error: %v", err)
		}
		if repeat > 0 {
			count++
			if count%10000 == 0 {
				slog.Info("loop", "count", count, "repeat", repeat)
			}
			disc.Stop()
			continue
		}
		if repeat == 0 {
			break
		}
	}

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
