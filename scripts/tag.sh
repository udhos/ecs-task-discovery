
version=$(go run ./cmd/ecs-task-discovery-agent -version | awk '{ print $2 }' | awk -F= '{ print $2 }')

git tag v${version}
