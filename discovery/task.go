package discovery

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

// ecsClient defines the subset of *ecs.Client methods required for health check detection.
// This allows mocking the client in unit tests.
type ecsClient interface {
	DescribeServices(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error)
	DescribeTaskDefinition(ctx context.Context, params *ecs.DescribeTaskDefinitionInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTaskDefinitionOutput, error)
}

// IsHealthCheckEnabled checks if the service's active task definition has container health check enabled
// on any of its essential containers.
func IsHealthCheckEnabled(ctx context.Context, client ecsClient, cluster, serviceName string) (bool, error) {
	out, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  aws.String(cluster),
		Services: []string{serviceName},
	})
	if err != nil {
		return false, fmt.Errorf("describe services: %w", err)
	}
	if len(out.Services) == 0 {
		return false, fmt.Errorf("service %s not found in cluster %s", serviceName, cluster)
	}

	taskDefArn := out.Services[0].TaskDefinition
	if taskDefArn == nil {
		return false, fmt.Errorf("no task definition associated with service %s", serviceName)
	}

	outDef, err := client.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: taskDefArn,
	})
	if err != nil {
		return false, fmt.Errorf("describe task definition: %w", err)
	}
	if outDef.TaskDefinition == nil {
		return false, fmt.Errorf("task definition not found for ARN %s", aws.ToString(taskDefArn))
	}

	for _, containerDef := range outDef.TaskDefinition.ContainerDefinitions {
		// Only check essential containers, since they are the ones determining task health status.
		// ContainerDefinition.Essential is a pointer to bool. If nil, it defaults to true.
		isEssential := containerDef.Essential == nil || *containerDef.Essential
		if isEssential && containerDef.HealthCheck != nil {
			return true, nil
		}
	}

	return false, nil
}
