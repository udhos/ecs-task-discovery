[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/ecs-task-discovery/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/udhos/ecs-task-discovery)](https://goreportcard.com/report/github.com/udhos/ecs-task-discovery)
[![Go Reference](https://pkg.go.dev/badge/github.com/udhos/ecs-task-discovery.svg)](https://pkg.go.dev/github.com/udhos/ecs-task-discovery)
[![Docker Pulls](https://img.shields.io/docker/pulls/udhos/ecs-task-discovery-agent)](https://hub.docker.com/r/udhos/ecs-task-discovery-agent)

# ecs-task-discovery

[ecs-task-discovery](https://github.com/udhos/ecs-task-discovery?tab=readme-ov-file) is a Go module that performs service discovery for ECS tasks.

# Usage

See example: [./cmd/ecs-task-discovery-example/main.go](./cmd/ecs-task-discovery-example/main.go)

# Build Example

```bash
./build.sh
```

# Run example

```bash
export CLUSTER=demo
export SERVICE=demo

ecs-task-discovery-example
```

# References

## ECS Exec Checker

ECS Checker: https://github.com/aws-containers/amazon-ecs-exec-checker

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
