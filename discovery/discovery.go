// Package discovery discovers ecs tasks.
package discovery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/udhos/boilerplate/awsconfig"
)

// Discovery is used for performing task discovery.
type Discovery struct {
	options   Options
	awsConfig awsconfig.Output
	clientEcs *ecs.Client
}

// Options define settings for creating a Discovery.
type Options struct {
	// Cluster is required cluster short name or full cluster ARN.
	Cluster string

	// ServiceName filters tasks that belong to service.
	ServiceName string

	// Interval for polling, defaults to 30s if undefined.
	Interval time.Duration

	// Callback is required callback for list of discovered tasks.
	Callback func(tasks []Task)
}

// Task represents a task.
type Task struct {
	ARN          string
	Address      string
	HealthStatus string
	LastStatus   string
}

// New creates a Discovery.
func New(options Options) (*Discovery, error) {

	if options.Cluster == "" {
		return nil, errors.New("Cluster is required")
	}

	if options.ServiceName == "" {
		return nil, errors.New("ServiceName is required")
	}

	if options.Interval == 0 {
		options.Interval = 30 * time.Second
	}

	if options.Callback == nil {
		return nil, errors.New("Callback is required")
	}

	awsCfg, errCfg := awsconfig.AwsConfig(awsconfig.Options{})
	if errCfg != nil {
		return nil, errCfg
	}

	return &Discovery{
		options:   options,
		awsConfig: awsCfg,
		clientEcs: ecs.NewFromConfig(awsCfg.AwsConfig),
	}, nil
}

// Run runs a Discovery.
func (d *Discovery) Run() {
	for {
		begin := time.Now()

		tasks, err := Tasks(d.clientEcs, d.options.Cluster, d.options.ServiceName)
		if err != nil {
			slog.Error(fmt.Sprintf("Tasks: error: %v", err))
		} else {
			slog.Info(fmt.Sprintf("Run: tasksFound=%d elapsed=%v", len(tasks), time.Since(begin)))
			d.options.Callback(tasks) // deliver result
		}

		slog.Info(fmt.Sprintf("Run: tasksFound=%d elapsed=%v", len(tasks), time.Since(begin)))

		slog.Info(fmt.Sprintf("Run: sleeping %v", d.options.Interval))
		time.Sleep(d.options.Interval)
	}
}

// Tasks discovers running ECS tasks.
func Tasks(clientEcs *ecs.Client, cluster, serviceName string) ([]Task, error) {

	desiredStatus := "RUNNING"
	maxResults := int32(100) // 1..100

	input := ecs.ListTasksInput{
		Cluster:       aws.String(cluster),
		ServiceName:   aws.String(serviceName),
		MaxResults:    aws.Int32(maxResults),
		DesiredStatus: types.DesiredStatus(desiredStatus),
	}

	var tasks []Task // collect all tasks

	//
	// scan over pages of ListTasks responses
	//
	for {
		out, errList := clientEcs.ListTasks(context.TODO(), &input)
		if errList != nil {
			return nil, errList
		}

		slog.Info(fmt.Sprintf("ListTasks: found %d of maxResults=%d tasks",
			len(out.TaskArns), maxResults))

		list, errDesc := describeTasks(clientEcs, cluster, out.TaskArns)
		if errDesc != nil {
			return nil, errDesc
		}
		tasks = append(tasks, list...)
		if out.NextToken == nil {
			break // finished last page
		}
		input.NextToken = out.NextToken // next page
	}

	return tasks, nil
}

// describeTasks describes a batch of tasks.
func describeTasks(clientEcs *ecs.Client, cluster string, taskArns []string) ([]Task, error) {
	if len(taskArns) == 0 {
		return nil, nil
	}
	input := ecs.DescribeTasksInput{
		Tasks:   taskArns,
		Cluster: aws.String(cluster),
	}
	out, err := clientEcs.DescribeTasks(context.TODO(), &input)
	if err != nil {
		return nil, err
	}

	var tasks []Task
	for _, t := range out.Tasks {
		//
		// find task address
		//

		switch {
		case len(t.Attachments) == 0:
			// log only
			slog.Error("describeTasks: task missing network attachment",
				"ARN", aws.ToString(t.TaskArn),
				"healthStatus", t.HealthStatus,
				"lastStatus", aws.ToString(t.LastStatus),
			)
		case len(t.Attachments) == 1:
			// quiet
		default:
			// log only
			slog.Error(fmt.Sprintf("describeTasks: task has multiple network attachments: %d",
				len(t.Attachments)),
				"ARN", aws.ToString(t.TaskArn),
				"healthStatus", t.HealthStatus,
				"lastStatus", aws.ToString(t.LastStatus),
			)
		}

		addr := findAddress(t.Attachments)

		if addr == "" {
			slog.Error("describeTasks: task missing privateIPv4Address",
				"ARN", aws.ToString(t.TaskArn),
				"healthStatus", t.HealthStatus,
				"lastStatus", aws.ToString(t.LastStatus),
			)
			continue // actual error: missing address
		}

		// task address found

		tasks = append(tasks, Task{
			ARN:          aws.ToString(t.TaskArn),
			Address:      addr,
			HealthStatus: string(t.HealthStatus),
			LastStatus:   aws.ToString(t.LastStatus),
		})
	}

	return tasks, nil
}

func findAddress(attachments []types.Attachment) string {
	for _, at := range attachments {
		for _, kv := range at.Details {
			if aws.ToString(kv.Name) == "privateIPv4Address" {
				return aws.ToString(kv.Value)
			}
		}
	}
	return ""
}
