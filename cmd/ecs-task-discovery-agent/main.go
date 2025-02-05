// Package main implements the tool.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/udhos/boilerplate/awsconfig"
	"github.com/udhos/boilerplate/boilerplate"

	"github.com/udhos/ecs-task-discovery/discovery"
)

type application struct {
	clusterName string
	listenAddr  string

	awsConfig aws.Config
	clientEcs *ecs.Client
}

func main() {
	//
	// command-line
	//
	var showVersion bool
	flag.BoolVar(&showVersion, "version", showVersion, "show version")
	flag.Parse()

	me := filepath.Base(os.Args[0])

	//
	// version
	//
	{
		v := boilerplate.LongVersion(me + " version=" + version)
		if showVersion {
			fmt.Print(v)
			fmt.Println()
			return
		}
		slog.Info(v)
	}

	app := &application{
		clusterName: mustClusterName(),
		listenAddr:  envString("LISTEN_ADDR", ":8080"),
		awsConfig:   mustAwsConfig(),
	}

	app.clientEcs = ecs.NewFromConfig(app.awsConfig)

	slog.Info(fmt.Sprintf("clusterName: %s", app.clusterName))

	const route = "/tasks/{service}"
	slog.Info(fmt.Sprintf("registering route: %s", route))
	http.Handle(route, app)

	slog.Info(fmt.Sprintf("listening on HTTP %s", app.listenAddr))
	err := http.ListenAndServe(app.listenAddr, nil)
	fatalf("listen error: %v", err)
}

func fatalf(format string, a ...any) {
	slog.Error("FATAL: " + fmt.Sprintf(format, a...))
	os.Exit(1)
}

func mustClusterName() string {
	clusterArn, err := discovery.FindCluster()
	if err != nil {
		fatalf("find cluster error: %v", err)
	}
	lastSlash := strings.LastIndexByte(clusterArn, '/')
	return clusterArn[lastSlash+1:]
}

func mustAwsConfig() aws.Config {
	awsCfg, errCfg := awsconfig.AwsConfig(awsconfig.Options{})
	if errCfg != nil {
		fatalf("aws config error: %v", errCfg)
	}
	return awsCfg.AwsConfig
}

func (app *application) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	serviceName := r.PathValue("service")

	slog.Info(fmt.Sprintf("service=%s", serviceName))

	begin := time.Now()

	tasks, err := discovery.Tasks(app.clientEcs, app.clusterName, serviceName)

	elapsed := time.Since(begin)

	if err != nil {
		msg := fmt.Sprintf("service=%s elapsed=%v error:%v",
			serviceName, elapsed, err)
		slog.Error(msg)
		http.Error(w, msg, 500)
		return
	}

	data, errJSON := json.Marshal(tasks)
	if errJSON != nil {
		msg := fmt.Sprintf("service=%s elapsed=%v error:%v",
			serviceName, elapsed, errJSON)
		slog.Error(msg)
		http.Error(w, msg, 500)
		return
	}

	slog.Info(fmt.Sprintf("service=%s elapsed=%v found %d tasks",
		serviceName, elapsed, len(tasks)))

	h := w.Header()
	h.Set("Content-Length", strconv.Itoa(len(data)))
	h.Set("Content-Type", "text/json; charset=utf-8")
	h.Set("X-Content-Type-Options", "nosniff")
	w.Write(data)
}
