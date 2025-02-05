[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/ecs-task-discovery/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/udhos/ecs-task-discovery)](https://goreportcard.com/report/github.com/udhos/ecs-task-discovery)
[![Go Reference](https://pkg.go.dev/badge/github.com/udhos/ecs-task-discovery.svg)](https://pkg.go.dev/github.com/udhos/ecs-task-discovery)

# ecs-task-discovery

[ecs-task-discovery](ecs-task-discovery) is a Go module that performs service discovery for ECS tasks.

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
