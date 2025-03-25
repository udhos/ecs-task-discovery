package main

import (
	"context"
	"net/http"
	"time"

	"github.com/modernprogram/groupcache/v2"
	"github.com/udhos/ecs-task-discovery/groupcachediscovery"
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

	metricsNamespace := ""

	discOptions := groupcachediscovery.Options{
		Pool:           pool,
		Client:         app.clientEcs,
		GroupCachePort: app.groupcachePort,
		ServiceName:    app.ecsTaskDiscoveryAgentService, // self
		// ForceSingleTask: see below
		DisableAgentQuery: true, // do not query ourselves
		MetricsNamespace:  metricsNamespace,
		MetricsRegisterer: app.registry,
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
		func(c context.Context, key string, dest groupcache.Sink) error {
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

	//
	// expose prometheus metrics for groupcache
	//
	g := modernprogram.New(app.cache)
	labels := map[string]string{}
	collector := groupcache_exporter.NewExporter(metricsNamespace, labels, g)
	app.registry.MustRegister(collector)

	return func() { disc.Stop() }
}
