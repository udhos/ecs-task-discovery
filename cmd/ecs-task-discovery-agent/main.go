// Package main implements the tool.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/modernprogram/groupcache/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/udhos/boilerplate/awsconfig"
	"github.com/udhos/boilerplate/boilerplate"

	"github.com/udhos/ecs-task-discovery/discovery"
	"github.com/udhos/ecs-task-discovery/internal/shared"
)

type application struct {
	clusterName                           string
	listenAddr                            string
	groupcachePort                        string
	groupcachePurgeExpired                bool
	groupcacheExpiredKeysEvictionInterval time.Duration
	groupcacheSizeBytes                   int64
	groupcacheEnable                      bool
	cacheTTL                              time.Duration
	ecsTaskDiscoveryAgentService          string
	forceSingleTask                       bool
	metricsPath                           string
	healthPath                            string

	awsConfig        aws.Config
	clientEcs        *ecs.Client
	groupcacheServer *http.Server
	cache            *groupcache.Group
	registry         *prometheus.Registry
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
		v := boilerplate.LongVersion(me + " version=" + shared.Version)
		if showVersion {
			fmt.Print(v)
			fmt.Println()
			return
		}
		slog.Info(v)
	}

	//
	// create application
	//

	app := &application{
		clusterName:                           discovery.MustClusterName(),
		listenAddr:                            envString("LISTEN_ADDR", ":8080"),
		groupcachePort:                        envString("GROUPCACHE_PORT", ":5000"),
		groupcachePurgeExpired:                envBool("GROUPCACHE_PURGE_EXPIRED", true),
		groupcacheExpiredKeysEvictionInterval: envDuration("GROUPCACHE_EXPIRED_KEYS_EVICTION_INTERVAL", 30*time.Minute),
		groupcacheSizeBytes:                   envInt64("GROUPCACHE_SIZE_BYTES", 1_000_000),
		groupcacheEnable:                      envBool("GROUPCACHE_ENABLE", true),
		cacheTTL:                              envDuration("CACHE_TTL", 20*time.Second),
		ecsTaskDiscoveryAgentService:          envString("ECS_TASK_DISCOVERY_AGENT_SERVICE", "ecs-task-discovery-agent"),
		forceSingleTask:                       envBool("FORCE_SINGLE_TASK", false),
		metricsPath:                           envString("METRICS_PATH", "/metrics"),
		healthPath:                            envString("HEALTH_PATH", "/health"),

		awsConfig: mustAwsConfig(),
		registry:  prometheus.NewRegistry(),
	}

	app.clientEcs = ecs.NewFromConfig(app.awsConfig)

	slog.Info(fmt.Sprintf("clusterName: %s", app.clusterName))

	//
	// start groupcache
	//

	startGroupcache(app)

	//
	// start health check
	//

	http.HandleFunc(app.healthPath, handlerHealth)

	//
	// start metrics server
	//

	{
		/*
			log.Info().Msgf("registering metrics route: %s %s",
				app.cfg.metricsAddr, app.cfg.metricsPath)

			mux := http.NewServeMux()
			app.serverMetrics = &http.Server{Addr: app.cfg.metricsAddr, Handler: mux}
		*/
		http.Handle(app.metricsPath, app.metricsHandler())
		/*
			mux.Handle(app.cfg.metricsPath, )

			go func() {
				log.Info().Msgf("metrics server: listening on %s %s",
					app.cfg.metricsAddr, app.cfg.metricsPath)
				err := app.serverMetrics.ListenAndServe()
				log.Error().Msgf("metrics server: exited: %v", err)
			}()
		*/
	}

	//
	// start server
	//

	const route = "/tasks/{service}"
	slog.Info(fmt.Sprintf("registering route: %s", route))
	http.Handle(route, app)

	slog.Info(fmt.Sprintf("listening on HTTP %s", app.listenAddr))
	err := http.ListenAndServe(app.listenAddr, nil)
	fatalf("listen error: %v", err)
}

func handlerHealth(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintln(w, "ok")
}

func mustAwsConfig() aws.Config {
	awsCfg, errCfg := awsconfig.AwsConfig(awsconfig.Options{})
	if errCfg != nil {
		fatalf("aws config error: %v", errCfg)
	}
	return awsCfg.AwsConfig
}

func (app *application) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	const me = "application.ServeHTTP"

	serviceName := r.PathValue("service")

	var data []byte
	var err error

	begin := time.Now()

	if app.groupcacheEnable {
		err = app.cache.Get(context.TODO(), serviceName,
			groupcache.AllocatingByteSliceSink(&data), nil)
	} else {
		data, err = findTasks(context.TODO(), app.clientEcs, app.clusterName,
			serviceName)
	}

	elapsed := time.Since(begin)

	infof("%s: cluster=%s service=%s elapsed=%v",
		me, app.clusterName, serviceName, elapsed)

	if err != nil {
		msg := fmt.Sprintf("%s: error: %v",
			me, err)
		slog.Error(msg)
		http.Error(w, msg, 500)
		return
	}

	h := w.Header()
	h.Set("Content-Length", strconv.Itoa(len(data)))
	h.Set("Content-Type", "text/json; charset=utf-8")
	h.Set("X-Content-Type-Options", "nosniff")
	w.Write(data)
}

func findTasks(ctx context.Context, clientEcs *ecs.Client, clusterName, serviceName string) ([]byte, error) {
	const me = "findTasks"

	begin := time.Now()

	tasks, err := discovery.Tasks(ctx, clientEcs, clusterName, serviceName)

	elapsed := time.Since(begin)

	if err != nil {
		return nil, fmt.Errorf("%s: service=%s elapsed=%v error:%v",
			me, serviceName, elapsed, err)
	}

	data, errJSON := json.Marshal(tasks)
	if errJSON != nil {
		return nil, fmt.Errorf("%s: service=%s elapsed=%v error:%v",
			me, serviceName, elapsed, errJSON)
	}

	slog.Info(fmt.Sprintf("%s: service=%s elapsed=%v found %d tasks",
		me, serviceName, elapsed, len(tasks)))

	return data, nil
}

func (app *application) metricsHandler() http.Handler {
	registerer := app.registry
	gatherer := app.registry
	return promhttp.InstrumentMetricHandler(
		registerer, promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{}),
	)
}
