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

	t.Setenv(envVarMetadataURI, ts.URL)

	clusterArn, err := FindCluster()
	if err != nil {
		t.Error(err)
	}

	const expected = "arn:aws:ecs:us-east-1:111122223333:cluster/demo"

	if clusterArn != expected {
		t.Errorf("bad cluster ARN: expected=%s got=%s", expected, clusterArn)
	}
}

const metadata = `{"Cluster":"arn:aws:ecs:us-east-1:111122223333:cluster/demo"}`
