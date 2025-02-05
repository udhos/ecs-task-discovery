package discovery

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFindCluster(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, metadata)
	}))
	defer ts.Close()

	t.Setenv("ECS_CONTAINER_METADATA_URI", ts.URL)

	clusterArn, err := FindCluster()
	if err != nil {
		t.Error(err)
	}

	const expected = "arn:aws:ecs:us-east-1:111122223333:cluster/demo"

	if clusterArn != expected {
		t.Errorf("bad cluster ARN: expected=%s got=%s", expected, clusterArn)
	}
}

const metadata = `{"DockerId":"1641160e91d34bbf880549bde1981fb3-1209491550","Name":"miniapi","DockerName":"miniapi","Image":"docker.io/udhos/miniapi","ImageID":"sha256:8a7042db82ceec77774a31567db4e621d95a9d95a2074d34917ec6703d66946c","Labels":{"com.amazonaws.ecs.cluster":"arn:aws:ecs:us-east-1:111122223333:cluster/demo","com.amazonaws.ecs.container-name":"miniapi","com.amazonaws.ecs.task-arn":"arn:aws:ecs:us-east-1:111122223333:task/demo/1641160e91d34bbf880549bde1981fb3","com.amazonaws.ecs.task-definition-family":"miniapi","com.amazonaws.ecs.task-definition-version":"2"},"DesiredStatus":"RUNNING","KnownStatus":"RUNNING","Limits":{"CPU":2},"CreatedAt":"2025-02-05T16:33:31.18575081Z","StartedAt":"2025-02-05T16:33:31.18575081Z","Type":"NORMAL","Networks":[{"NetworkMode":"awsvpc","IPv4Addresses":["172.31.50.241"]}]}`
