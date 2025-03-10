[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/ecs-task-discovery/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/udhos/ecs-task-discovery)](https://goreportcard.com/report/github.com/udhos/ecs-task-discovery)
[![Go Reference](https://pkg.go.dev/badge/github.com/udhos/ecs-task-discovery.svg)](https://pkg.go.dev/github.com/udhos/ecs-task-discovery)
[![Docker Pulls Agent](https://img.shields.io/docker/pulls/udhos/ecs-task-discovery-agent)](https://hub.docker.com/r/udhos/ecs-task-discovery-agent)
[![Docker Pulls Example](https://img.shields.io/docker/pulls/udhos/ecs-task-discovery-example)](https://hub.docker.com/r/udhos/ecs-task-discovery-example)

# ecs-task-discovery

[ecs-task-discovery](https://github.com/udhos/ecs-task-discovery?tab=readme-ov-file) is a Go module that performs service discovery for ECS tasks.

# Groupcache Task Discovery

The package [groupcachediscovery](https://pkg.go.dev/github.com/udhos/ecs-task-discovery/groupcachediscovery) implements task discovery for groupcache-enabled applications running on AWS ECS.

See package documentation here: [groupcachediscovery](https://pkg.go.dev/github.com/udhos/ecs-task-discovery/groupcachediscovery)

Find an example toy application using [groupcache](https://github.com/modernprogram/groupcache) with [groupcachediscovery](https://pkg.go.dev/github.com/udhos/ecs-task-discovery/groupcachediscovery) here: https://github.com/udhos/kubecache/blob/main/cmd/kubecache/groupcache.go

# Usage

See example: [./cmd/ecs-task-discovery-example/main.go](./cmd/ecs-task-discovery-example/main.go)

# Build Example

```bash
./build.sh
```

# Run example

```bash
export SERVICE=ecs-task-discovery-example

ecs-task-discovery-example
```

# Testing agent

```bash
# mock metadata server
cd samples
python3 -m http.server 8000

# run agent pointing to mocked metadata
export FORCE_SINGLE_TASK=true
export ECS_CONTAINER_METADATA_URI_V4=http://localhost:8000
ecs-task-discovery-agent

curl localhost:8080/tasks/ecs-task-discovery-example

export ECS_TASK_DISCOVERY_AGENT_URL=http://localhost:8080/tasks
export ECS_CONTAINER_METADATA_URI_V4=http://localhost:8000
ecs-task-discovery-example
```

# References

## ECS Exec Checker

ECS Exec Checker: https://github.com/aws-containers/amazon-ecs-exec-checker

```bash
git clone https://github.com/aws-containers/amazon-ecs-exec-checker

cd amazon-ecs-exec-checker

./check-ecs-exec.sh demo 1641160e91d34bbf880549bde1981fb3
```

## Execute command

```bash
aws ecs update-service \
    --task-definition miniapi \
    --cluster demo \
    --service demo \
    --enable-execute-command \
    --force-new-deployment

aws ecs execute-command --cluster demo \
    --task 1641160e91d34bbf880549bde1981fb3 \
    --container miniapi \
    --interactive \
    --command "/bin/sh"
```
