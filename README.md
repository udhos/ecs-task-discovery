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

ecs-task-discovery-example -task-definition-health-check-mode=false
```

# Testing agent

```bash
# mock metadata server
cd samples
python3 -m http.server 8000

# run agent pointing to mocked metadata
export FORCE_SINGLE_TASK=true
export ECS_CONTAINER_METADATA_URI_V4=http://localhost:8000
export TASK_DEFINITION_HEALTH_CHECK_MODE=false ;# do not query ECS API
ecs-task-discovery-agent

curl localhost:8080/tasks/ecs-task-discovery-example

export ECS_TASK_DISCOVERY_AGENT_URL=http://localhost:8080/tasks
export ECS_CONTAINER_METADATA_URI_V4=http://localhost:8000
ecs-task-discovery-example -task-definition-health-check-mode=false
```

# Agent health check mode

The agent accepts env var `TASK_DEFINITION_HEALTH_CHECK_MODE` to control how task definition health check detection is handled.

Allowed values:

- `detect` (default): detect using ECS API; errors fail startup.
- `detectandhandleerrorasfalse`: detect using ECS API; errors fallback to false.
- `true`: force health checks enabled.
- `false`: force health checks disabled.

# PRODUCTION READINESS REVIEW

Tests that improve confidence on ECS task autodiscovery behavior:

1. [x] Discovery agent URL precedence in `Discovery.queryAgent()`: verify `Options.AgentURL` > `ECS_TASK_DISCOVERY_AGENT_URL` > default URL.
2. [x] Agent fallback to ECS API in `Discovery.listTasks()`: simulate agent HTTP error and verify `Tasks()` fallback result is used.
3. [x] Health filtering semantics in `filterByHealth()`: verify HEALTHY-only behavior when task definition health check mode resolves to enabled, and pass-through when disabled.
4. [x] Groupcache v2 peer update callback in `groupcachediscovery.New()`: verify `Pool.Set(peers...)` receives expected URLs.
5. [x] Groupcache v3 peer update callback in `groupcachediscovery.New()`: verify `SetPeers()` receives expected peer list and `IsSelf` flag mapping.
6. [x] Address extraction edge cases in `describeTasks()/findAddress()`: verify task attachment parsing and skip behavior when `privateIPv4Address` is missing.

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
