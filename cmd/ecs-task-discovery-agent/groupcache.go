package main

import (
	"context"
	"net/http"
	"time"

	"github.com/modernprogram/groupcache/v2"
	"github.com/udhos/dogstatsdclient/dogstatsdclient"
	"github.com/udhos/ecs-task-discovery/groupcachediscovery"
	"github.com/udhos/groupcache_datadog/exporter"
	"github.com/udhos/groupcache_exporter"
	"github.com/udhos/groupcache_exporter/groupcache/modernprogram"
)

func startGroupcache(app *application) func() {

	workspace := groupcache.NewWorkspace()

	//
	// create groupcache pool
	//

	myURL, errURL := groupcachediscovery.FindMyURL(app.groupcachePort)
	if errURL != nil {
		fatalf("groupcache my URL: %v", errURL)
	}
	infof("groupcache my URL: %s", myURL)

	pool := groupcache.NewHTTPPoolOptsWithWorkspace(workspace, myURL, &groupcache.HTTPPoolOptions{})

	//
	// start groupcache server
	//

	app.groupcacheServer = &http.Server{Addr: app.groupcachePort, Handler: pool}

	go func() {
		infof("groupcache server: listening on %s", app.groupcachePort)
		err := app.groupcacheServer.ListenAndServe()
		errorf("groupcache server: exited: %v", err)
	}()

	//
	// start groupcache peer discovery
	//

	const (
		groupcacheMetricsNamespace          = "" // usually empty since "groupcache" is added as subsystem
		groupcacheDiscoveryMetricsNamespace = "groupcachediscovery"
	)

	discOptions := groupcachediscovery.Options{
		Pool:           pool,
		Client:         app.clientEcs,
		GroupCachePort: app.groupcachePort,
		ServiceName:    app.ecsTaskDiscoveryAgentService, // self
		// ForceSingleTask: see below
		DisableAgentQuery: true, // do not query ourselves
		MetricsNamespace:  groupcacheDiscoveryMetricsNamespace,
	}

	if app.prometheusEnable {
		discOptions.MetricsRegisterer = app.registry
	}

	if app.dogstatsdEnable {
		client, errClient := dogstatsdclient.New(dogstatsdclient.Options{
			Namespace: groupcacheDiscoveryMetricsNamespace,
			Debug:     app.dogstatsdDebug,
			TTL:       app.dogstatsdClientTTL,
		})
		if errClient != nil {
			fatalf("dogstatsd client: %v", errClient)
		}
		discOptions.DogstatsdClient = client
	}

	if app.forceSingleTask {
		myAddr, errAddr := groupcachediscovery.FindMyAddr()
		if errAddr != nil {
			fatalf("groupcache my address: %v", errAddr)
		}
		discOptions.ForceSingleTask = myAddr
	}

	disc, errDisc := groupcachediscovery.New(discOptions)
	if errDisc != nil {
		fatalf("groupcache discovery error: %v", errDisc)
	}

	//
	// create cache
	//

	getter := groupcache.GetterFunc(
		func(c context.Context, key string, dest groupcache.Sink, _ *groupcache.Info) error {
			data, err := findTasks(c, app.clientEcs, app.clusterName, key)
			if err != nil {
				return err
			}
			expire := time.Now().Add(app.cacheTTL)
			return dest.SetBytes(data, expire)
		},
	)

	groupcacheOptions := groupcache.Options{
		Workspace:                   workspace,
		Name:                        "tasks",
		PurgeExpired:                app.groupcachePurgeExpired,
		ExpiredKeysEvictionInterval: app.groupcacheExpiredKeysEvictionInterval,
		CacheBytesLimit:             app.groupcacheSizeBytes,
		Getter:                      getter,
	}

	// https://talks.golang.org/2013/oscon-dl.slide#46
	//
	// 64 MB max per-node memory usage
	app.cache = groupcache.NewGroupWithWorkspace(groupcacheOptions)

	extract := modernprogram.New(app.cache) // extract metrics from groupcache group

	unregister := func() {}

	if app.prometheusEnable {
		infof("starting groupcache metrics exporter for Prometheus")
		labels := map[string]string{}
		collector := groupcache_exporter.NewExporter(groupcacheMetricsNamespace, labels, extract)
		app.registry.MustRegister(collector)
		unregister = func() { app.registry.Unregister(collector) }
	}

	closeExporterDogstatsd := func() {}

	if discOptions.DogstatsdClient != nil {
		infof("starting groupcache metrics exporter for Dogstatsd")
		exporter := exporter.New(exporter.Options{
			Client:         discOptions.DogstatsdClient,
			Groups:         []groupcache_exporter.GroupStatistics{extract},
			ExportInterval: app.dogstatsdExportInterval,
		})
		closeExporterDogstatsd = func() { exporter.Close() }
	}

	return func() {
		disc.Stop()
		closeExporterDogstatsd()
		unregister()
	}
}
