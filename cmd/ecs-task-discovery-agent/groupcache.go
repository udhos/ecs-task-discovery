package main

import (
	"context"
	"net/http"
	"time"

	"github.com/modernprogram/groupcache/v2"
	"github.com/udhos/ecs-task-discovery/groupcachediscovery"
	"github.com/udhos/kubegroup/kubegroup"
)

func startGroupcache(app *application) {

	workspace := groupcache.NewWorkspace()

	//
	// create groupcache pool
	//

	myURL, errURL := kubegroup.FindMyURL(app.groupcachePort)
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

	discOptions := groupcachediscovery.Options{
		Pool:           pool,
		Client:         app.clientEcs,
		GroupCachePort: app.groupcachePort,
		Cluster:        app.clusterName,
		ServiceName:    app.ecsTaskDiscoveryAgentService, // self
	}

	errDisc := groupcachediscovery.Run(discOptions)
	if errDisc != nil {
		fatalf("groupcache discovery error: %v", errDisc)
	}

	/*
		clientsetOpt := kubeclient.Options{DebugLog: app.cfg.kubegroupDebug}
		clientset, errClientset := kubeclient.New(clientsetOpt)
		if errClientset != nil {
			fatalf("startGroupcache: kubeclient: %v", errClientset)
		}

		options := kubegroup.Options{
			Client:                clientset,
			LabelSelector:         app.cfg.kubegroupLabelSelector,
			Pool:                  pool,
			GroupCachePort:        app.cfg.groupcachePort,
			MetricsRegisterer:     app.registry,
			MetricsGatherer:       app.registry,
			MetricsNamespace:      app.cfg.kubegroupMetricsNamespace,
			Debug:                 app.cfg.kubegroupDebug,
			ForceNamespaceDefault: forceNamespaceDefault,
		}

		kg, errKg := kubegroup.UpdatePeers(options)
		if errKg != nil {
			log.Fatal().Msgf("kubegroup error: %v", errKg)
		}
	*/

	//
	// create cache
	//

	getter := groupcache.GetterFunc(
		func(c context.Context, key string, dest groupcache.Sink) error {

			const me = "groupcache.getter"

			data, err := findTasks(c, app.clientEcs, app.clusterName, key)
			if err != nil {
				return err
			}

			expire := time.Now().Add(app.cacheTTL)

			return dest.SetBytes(data, expire)
		},
	)

	groupcacheOptions := groupcache.Options{
		Workspace:    workspace,
		Name:         "tasks",
		PurgeExpired: app.groupcachePurgeExpired,
		CacheBytes:   app.groupcacheSizeBytes,
		Getter:       getter,
	}

	// https://talks.golang.org/2013/oscon-dl.slide#46
	//
	// 64 MB max per-node memory usage
	app.cache = groupcache.NewGroupWithWorkspace(groupcacheOptions)

	//
	// expose prometheus metrics for groupcache
	//
	/*
		g := modernprogram.New(app.cache)
		labels := map[string]string{}
		namespace := ""
		collector := groupcache_exporter.NewExporter(namespace, labels, g)
		app.registry.MustRegister(collector)
	*/
}
