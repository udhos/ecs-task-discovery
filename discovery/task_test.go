package discovery

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type mockECSClient struct {
	DescribeServicesFunc       func(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error)
	DescribeTaskDefinitionFunc func(ctx context.Context, params *ecs.DescribeTaskDefinitionInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTaskDefinitionOutput, error)
}

func (m *mockECSClient) DescribeServices(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
	if m.DescribeServicesFunc != nil {
		return m.DescribeServicesFunc(ctx, params, optFns...)
	}
	return nil, errors.New("DescribeServicesFunc not implemented")
}

func (m *mockECSClient) DescribeTaskDefinition(ctx context.Context, params *ecs.DescribeTaskDefinitionInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTaskDefinitionOutput, error) {
	if m.DescribeTaskDefinitionFunc != nil {
		return m.DescribeTaskDefinitionFunc(ctx, params, optFns...)
	}
	return nil, errors.New("DescribeTaskDefinitionFunc not implemented")
}

func TestIsHealthCheckEnabled(t *testing.T) {
	tests := []struct {
		name                 string
		services             []types.Service
		describeServicesErr  error
		taskDefinition       *types.TaskDefinition
		describeTaskDefErr   error
		expectedResult       bool
		expectedErrSubstring string
	}{
		{
			name:                 "Service not found",
			services:             []types.Service{},
			expectedResult:       false,
			expectedErrSubstring: "service my-service not found in cluster my-cluster",
		},
		{
			name:                 "DescribeServices error",
			describeServicesErr:  errors.New("API error"),
			expectedResult:       false,
			expectedErrSubstring: "describe services: API error",
		},
		{
			name: "No task definition ARN on service",
			services: []types.Service{
				{
					ServiceName: aws.String("my-service"),
				},
			},
			expectedResult:       false,
			expectedErrSubstring: "no task definition associated with service my-service",
		},
		{
			name: "DescribeTaskDefinition error",
			services: []types.Service{
				{
					ServiceName:    aws.String("my-service"),
					TaskDefinition: aws.String("arn:aws:ecs:us-east-1:123456789012:task-definition/my-task:1"),
				},
			},
			describeTaskDefErr:   errors.New("task def error"),
			expectedResult:       false,
			expectedErrSubstring: "describe task definition: task def error",
		},
		{
			name: "Task definition is nil in output",
			services: []types.Service{
				{
					ServiceName:    aws.String("my-service"),
					TaskDefinition: aws.String("arn:aws:ecs:us-east-1:123456789012:task-definition/my-task:1"),
				},
			},
			taskDefinition:       nil,
			expectedResult:       false,
			expectedErrSubstring: "task definition not found for ARN arn:aws:ecs:us-east-1:123456789012:task-definition/my-task:1",
		},
		{
			name: "Essential container without health check",
			services: []types.Service{
				{
					ServiceName:    aws.String("my-service"),
					TaskDefinition: aws.String("arn:aws:ecs:us-east-1:123456789012:task-definition/my-task:1"),
				},
			},
			taskDefinition: &types.TaskDefinition{
				ContainerDefinitions: []types.ContainerDefinition{
					{
						Name:      aws.String("app"),
						Essential: aws.Bool(true),
					},
				},
			},
			expectedResult: false,
		},
		{
			name: "Essential container with health check",
			services: []types.Service{
				{
					ServiceName:    aws.String("my-service"),
					TaskDefinition: aws.String("arn:aws:ecs:us-east-1:123456789012:task-definition/my-task:1"),
				},
			},
			taskDefinition: &types.TaskDefinition{
				ContainerDefinitions: []types.ContainerDefinition{
					{
						Name:      aws.String("app"),
						Essential: aws.Bool(true),
						HealthCheck: &types.HealthCheck{
							Command: []string{"CMD-SHELL", "exit 0"},
						},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "Essential container with default essential status (nil Essential pointer) and health check",
			services: []types.Service{
				{
					ServiceName:    aws.String("my-service"),
					TaskDefinition: aws.String("arn:aws:ecs:us-east-1:123456789012:task-definition/my-task:1"),
				},
			},
			taskDefinition: &types.TaskDefinition{
				ContainerDefinitions: []types.ContainerDefinition{
					{
						Name:      aws.String("app"),
						Essential: nil, // Defaults to true
						HealthCheck: &types.HealthCheck{
							Command: []string{"CMD-SHELL", "exit 0"},
						},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "Non-essential container with health check, essential container without",
			services: []types.Service{
				{
					ServiceName:    aws.String("my-service"),
					TaskDefinition: aws.String("arn:aws:ecs:us-east-1:123456789012:task-definition/my-task:1"),
				},
			},
			taskDefinition: &types.TaskDefinition{
				ContainerDefinitions: []types.ContainerDefinition{
					{
						Name:      aws.String("app"),
						Essential: aws.Bool(true),
					},
					{
						Name:      aws.String("sidecar"),
						Essential: aws.Bool(false),
						HealthCheck: &types.HealthCheck{
							Command: []string{"CMD-SHELL", "exit 0"},
						},
					},
				},
			},
			expectedResult: false,
		},
		{
			name: "Two essential containers, one with check, one without",
			services: []types.Service{
				{
					ServiceName:    aws.String("my-service"),
					TaskDefinition: aws.String("arn:aws:ecs:us-east-1:123456789012:task-definition/my-task:1"),
				},
			},
			taskDefinition: &types.TaskDefinition{
				ContainerDefinitions: []types.ContainerDefinition{
					{
						Name:      aws.String("app1"),
						Essential: aws.Bool(true),
					},
					{
						Name:      aws.String("app2"),
						Essential: aws.Bool(true),
						HealthCheck: &types.HealthCheck{
							Command: []string{"CMD-SHELL", "exit 0"},
						},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "Two essential containers, both with check",
			services: []types.Service{
				{
					ServiceName:    aws.String("my-service"),
					TaskDefinition: aws.String("arn:aws:ecs:us-east-1:123456789012:task-definition/my-task:1"),
				},
			},
			taskDefinition: &types.TaskDefinition{
				ContainerDefinitions: []types.ContainerDefinition{
					{
						Name:      aws.String("app1"),
						Essential: aws.Bool(true),
						HealthCheck: &types.HealthCheck{
							Command: []string{"CMD-SHELL", "exit 0"},
						},
					},
					{
						Name:      aws.String("app2"),
						Essential: aws.Bool(true),
						HealthCheck: &types.HealthCheck{
							Command: []string{"CMD-SHELL", "exit 0"},
						},
					},
				},
			},
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockECSClient{
				DescribeServicesFunc: func(_ context.Context, _ *ecs.DescribeServicesInput, _ ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
					if tt.describeServicesErr != nil {
						return nil, tt.describeServicesErr
					}
					return &ecs.DescribeServicesOutput{
						Services: tt.services,
					}, nil
				},
				DescribeTaskDefinitionFunc: func(_ context.Context, _ *ecs.DescribeTaskDefinitionInput, _ ...func(*ecs.Options)) (*ecs.DescribeTaskDefinitionOutput, error) {
					if tt.describeTaskDefErr != nil {
						return nil, tt.describeTaskDefErr
					}
					return &ecs.DescribeTaskDefinitionOutput{
						TaskDefinition: tt.taskDefinition,
					}, nil
				},
			}

			result, err := IsHealthCheckEnabled(context.Background(), client, "my-cluster", "my-service")

			if tt.expectedErrSubstring != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.expectedErrSubstring)
				}
				// check error string contains substring
				importString := err.Error()
				if !strings.Contains(importString, tt.expectedErrSubstring) {
					t.Errorf("expected error containing %q, got %q", tt.expectedErrSubstring, importString)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if result != tt.expectedResult {
					t.Errorf("expected result %t, got %t", tt.expectedResult, result)
				}
			}
		})
	}
}
