#!/bin/bash

app=ecs-task-discovery-example

version=$(go run ./cmd/$app -version | awk '{ print $2 }' | awk -F= '{ print $2 }')

echo version=$version

docker build --no-cache \
    -t udhos/$app:latest \
    -t udhos/$app:$version \
    -f docker/Dockerfile-example .

echo push:
echo "docker push udhos/$app:$version; docker push udhos/$app:latest" > docker-push-example.sh
chmod a+rx docker-push-example.sh
echo docker-push-example.sh:
cat docker-push-example.sh
