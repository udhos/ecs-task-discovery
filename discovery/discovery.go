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

	desiredStatus := "RUNNING"
	maxResults := int32(100) // 1..100

	input := ecs.ListTasksInput{
		Cluster:       aws.String(d.options.Cluster),
		ServiceName:   aws.String(d.options.ServiceName),
		MaxResults:    aws.Int32(maxResults),
		DesiredStatus: types.DesiredStatus(desiredStatus),
	}

	for {

		begin := time.Now()

		var tasks []Task
		var errored bool

		for {
			out, errList := d.clientEcs.ListTasks(context.TODO(), &input)
			if errList == nil {
				slog.Info(fmt.Sprintf("ListTasks: found %d of maxResults=%d tasks",
					len(out.TaskArns), maxResults))
				list, errDesc := d.describeTasks(out.TaskArns)
				if errDesc != nil {
					slog.Error(fmt.Sprintf("DescribeTasks: error: %v", errDesc))
					errored = true // do not report partial result
					break          // do not hammer ListTasks on error
				}
				tasks = append(tasks, list...)
				input.NextToken = out.NextToken
				if out.NextToken == nil {
					break // finished
				}
				input.NextToken = out.NextToken
			} else {
				slog.Error(fmt.Sprintf("ListTasks: error: %v", errList))
				errored = true // do not report partial result
				break          // do not hammer ListTasks on error
			}
		}

		slog.Info(fmt.Sprintf("Run: tasksFound=%d elapsed=%v", len(tasks), time.Since(begin)))

		if !errored {
			d.options.Callback(tasks) // deliver result
		}

		slog.Info(fmt.Sprintf("Run: sleeping %v", d.options.Interval))
		time.Sleep(d.options.Interval)
	}
}

func (d *Discovery) describeTasks(taskArns []string) ([]Task, error) {
	if len(taskArns) == 0 {
		return nil, nil
	}
	input := ecs.DescribeTasksInput{
		Tasks:   taskArns,
		Cluster: aws.String(d.options.Cluster),
	}
	out, err := d.clientEcs.DescribeTasks(context.TODO(), &input)
	if err != nil {
		return nil, err
	}

	var tasks []Task
	for _, t := range out.Tasks {
		//
		// find address
		//
		var addr string
		if len(t.Attachments) > 0 {
			at := t.Attachments[0]
			for _, kv := range at.Details {
				if key := aws.ToString(kv.Name); key == "privateIPv4Address" {
					addr = aws.ToString(kv.Value)
					break
				}
			}
		}

		if addr == "" {
			continue // missing address
		}

		tasks = append(tasks, Task{
			ARN:          aws.ToString(t.TaskArn),
			Address:      addr,
			HealthStatus: string(t.HealthStatus),
			LastStatus:   aws.ToString(t.LastStatus),
		})
	}

	return tasks, nil
}
