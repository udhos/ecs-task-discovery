// Package discovery discovers ecs tasks.
package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// Discovery is used for performing task discovery.
type Discovery struct {
	options     Options
	clusterName string
}

// Options define settings for creating a Discovery.
type Options struct {
	// ServiceName filters tasks that belong to service.
	ServiceName string

	// Interval for polling, defaults to 20s if undefined.
	Interval time.Duration

	// Callback is required callback for list of discovered tasks.
	Callback func(tasks []Task)

	// Client is required ECS client.
	Client *ecs.Client

	// ForceSingleTask forces our local IP address.
	// If defined, it should be set to our actual IP address.
	// The function FindMyAddr() provides a suitable address.
	// It is useful only for locally running the application.
	ForceSingleTask string

	// DisableAgentQuery skips querying the task discovery agent.
	// The task discovery agent sets DisableAgentQuery to true in order to not query itself.
	// Most applications should leave it undefined (set to false).
	DisableAgentQuery bool

	// AgentURL forces agent URL.
	// If undefined, retrieves value from env var ECS_TASK_DISCOVERY_AGENT_URL.
	// If ECS_TASK_DISCOVERY_AGENT_URL is undefined, defaults to http://ecs-task-discovery-agent.{Cluster}:8080/tasks.
	AgentURL string
}

const (
	defaultAgentURL = "http://ecs-task-discovery-agent.%s:8080/tasks"
	envAgentURL     = "ECS_TASK_DISCOVERY_AGENT_URL"
)

// Task represents a task.
type Task struct {
	ARN          string `json:"arn"`
	Address      string `json:"address"`
	HealthStatus string `json:"health_status"`
	LastStatus   string `json:"last_status"`
}

// New creates a Discovery.
func New(options Options) (*Discovery, error) {

	if options.ServiceName == "" {
		return nil, errors.New("option ServiceName is required")
	}

	if options.Interval == 0 {
		options.Interval = 20 * time.Second
	}

	if options.Callback == nil {
		return nil, errors.New("option Callback is required")
	}

	if options.Client == nil {
		return nil, errors.New("option Client is required")
	}

	return &Discovery{
		options:     options,
		clusterName: MustClusterName(),
	}, nil
}

// Run runs a Discovery.
func (d *Discovery) Run() {
	const me = "Discovery.Run"

	for {
		begin := time.Now()

		tasks := d.listTasks()

		elapsed := time.Since(begin)

		infof("%s: forceSingleTask=[%s] disableAgentQuery=%t tasksFound=%d elapsed=%v",
			me, d.options.ForceSingleTask, d.options.DisableAgentQuery, len(tasks), elapsed)

		if len(tasks) > 0 {
			d.options.Callback(tasks) // deliver result
		}

		infof("%s: sleeping %v", me, d.options.Interval)
		time.Sleep(d.options.Interval)
	}
}

func (d *Discovery) listTasks() []Task {
	const me = "Discovery.listTasks"

	var tasks []Task

	if !d.options.DisableAgentQuery {
		var err error
		tasks, err = d.queryAgent()
		if err == nil {
			infof("%s: query agent: %d tasks", me, len(tasks))
			return tasks
		}
		errorf("%s: query agent error: %v", me, err)
	}

	if d.options.ForceSingleTask != "" {
		tasks = []Task{
			{
				ARN:          "mockedSingleTaskARN",
				Address:      d.options.ForceSingleTask,
				HealthStatus: "UNKNOWN",
				LastStatus:   "RUNNING",
			},
		}
	} else {
		var err error
		tasks, err = Tasks(context.TODO(), d.options.Client, d.clusterName, d.options.ServiceName)
		if err != nil {
			errorf("%s: Tasks: error: %v", me, err)
		}
	}

	return tasks
}

func (d *Discovery) queryAgent() ([]Task, error) {
	const me = "Discovery.queryAgent"

	defaultURL := fmt.Sprintf(defaultAgentURL, d.clusterName)

	agentURL := d.options.AgentURL
	if agentURL == "" {
		agentURL = os.Getenv(envAgentURL)
		if agentURL == "" {
			agentURL = defaultURL
		}
	}

	infof("%s: agentURL: (1)AgentURL='%s' (2)%s='%s' (3)default=%s using value: '%s'",
		me, d.options.AgentURL, envAgentURL, os.Getenv(envAgentURL), defaultURL, agentURL)

	u, errJoin := url.JoinPath(agentURL, d.options.ServiceName)
	if errJoin != nil {
		return nil, errJoin
	}

	resp, errGet := http.Get(u)
	if errGet != nil {
		return nil, errGet
	}

	defer resp.Body.Close()

	body, errBody := io.ReadAll(resp.Body)
	if errBody != nil {
		return nil, fmt.Errorf("%s: status=%d url=%s body_error:%v",
			me, resp.StatusCode, u, errBody)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s: bad_status=%d url=%s error:%s",
			me, resp.StatusCode, u, string(body))
	}

	var tasks []Task

	if errJSON := json.Unmarshal(body, &tasks); errJSON != nil {
		return nil, fmt.Errorf("%s: status=%d url=%s json_error:%v",
			me, resp.StatusCode, u, errJSON)
	}

	return tasks, nil
}

// Tasks discovers running ECS tasks.
func Tasks(ctx context.Context, clientEcs *ecs.Client, cluster, serviceName string) ([]Task, error) {

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
		out, errList := clientEcs.ListTasks(ctx, &input)
		if errList != nil {
			return nil, errList
		}
		infof("Tasks: ListTasks: cluster=%s service=%s found %d of maxResults=%d tasks",
			cluster, serviceName, len(out.TaskArns), maxResults)

		list, errDesc := describeTasks(ctx, clientEcs, cluster, out.TaskArns)
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
func describeTasks(ctx context.Context, clientEcs *ecs.Client, cluster string, taskArns []string) ([]Task, error) {
	if len(taskArns) == 0 {
		return nil, nil
	}
	input := ecs.DescribeTasksInput{
		Tasks:   taskArns,
		Cluster: aws.String(cluster),
	}
	out, err := clientEcs.DescribeTasks(ctx, &input)
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
		}

		addr := findAddress(t.Attachments)

		if addr == "" {
			slog.Error("describeTasks: task missing privateIPv4Address",
				"ARN", aws.ToString(t.TaskArn),
				"healthStatus", t.HealthStatus,
				"lastStatus", aws.ToString(t.LastStatus),
				"networkAttachments", len(t.Attachments),
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

// MustClusterName returns ECS cluster name.
func MustClusterName() string {
	clusterArn, err := FindCluster()
	if err != nil {
		fatalf("find cluster error: %v", err)
	}
	// extract short cluster name from ARN
	lastSlash := strings.LastIndexByte(clusterArn, '/')
	return clusterArn[lastSlash+1:]
}

const envVarMetadataURI = "ECS_CONTAINER_METADATA_URI_V4"

// FindCluster finds ECS cluster ARN by querying container metadata.
//
// EC2: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4-response.html
// Fargate: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4-fargate-response.html
// Env var: ${ECS_CONTAINER_METADATA_URI_V4}/task
// Field: Cluster
func FindCluster() (string, error) {
	envValue := os.Getenv(envVarMetadataURI)
	if envValue == "" {
		return "", fmt.Errorf("env var '%s' is empty", envVarMetadataURI)
	}
	uri := envValue + "/task"
	resp, errGet := http.Get(uri)
	if errGet != nil {
		return "", errGet
	}
	defer resp.Body.Close()
	body, errBody := io.ReadAll(resp.Body)
	if errBody != nil {
		return "", fmt.Errorf("status:%d uri:%s body_error:%v", resp.StatusCode, uri, errBody)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("bad_status:%d uri:%s body:%s", resp.StatusCode, uri, string(body))
	}
	var metadata metadataFormat
	if err := json.Unmarshal(body, &metadata); err != nil {
		return "", fmt.Errorf("status:%d uri:%s json_error:%v", resp.StatusCode, uri, err)
	}
	return metadata.Cluster, nil
}

type metadataFormat struct {
	Cluster string `json:"Cluster"`
}
