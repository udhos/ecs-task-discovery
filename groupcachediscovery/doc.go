/*
Package groupcachediscovery implements groupcache task discovery for ECS.

# Usage

## Step 1: Start the package “ in your application

See `start groupcache peer discovery` below.

	//
	// create groupcache pool
	//
	groupcachePort := ":5000"
	myURL, errURL := groupcachediscovery.FindMyURL(groupcachePort)
	if errURL != nil {
	    log.Fatalf("groupcache my URL: %v", errURL)
	}
	workspace := groupcache.NewWorkspace()
	pool := groupcache.NewHTTPPoolOptsWithWorkspace(workspace, myURL, &groupcache.HTTPPoolOptions{})

	//
	// start groupcache server
	//
	groupcacheServer := &http.Server{Addr: groupcachePort, Handler: pool}
	go func() {
	    err := groupcacheServer.ListenAndServe()
	}()

	//
	// start groupcache peer discovery
	//
	discOptions := groupcachediscovery.Options{
	    Pool:           pool,
	    Client:         clientEcs,
	    GroupCachePort: groupcachePort,
	    ServiceName:    "app-service-name-on-ecs", // self
	}
	errDisc := groupcachediscovery.Run(discOptions)
	if errDisc != nil {
	    log.Fatalf("groupcache discovery error: %v", errDisc)
	}

	//
	// create cache
	//
	getter := groupcache.GetterFunc(
	    func(c context.Context, key string, dest groupcache.Sink) error {
	        data, err := retrieveKeySomehow(key) // from DB or whatever
	        if err != nil {
	            return err
	        }
	        expire := time.Now().Add(10 * time.Minute)
	        return dest.SetBytes(data, expire)
	    },
	)
	groupcacheOptions := groupcache.Options{
	    Workspace:    workspace,
	    Name:         "my-objects",
	    PurgeExpired: true,
	    CacheBytes:   1_000_000,
	    Getter:       getter,
	}
	cache = groupcache.NewGroupWithWorkspace(groupcacheOptions)

	//
	// use the cache
	//
	var data []byte
	err := cache.Get(context.TODO(), "key1", groupcache.AllocatingByteSliceSink(&data))

## Step 2: (Optional) Run the ecs-task-discovery-agent

Running the agent `ecs-task-discovery-agent` on the same ECS cluster as your
application is optional but recommended.
If you don't run it, your application will perform the task discovery autonomously.
If you run the agent, the application will offload the task discovery to the agent.

### Agent URL

By default the discovery package in your application will attempt to contact
the agent at the URL `http://ecs-task-discovery-agent.{ClusterName}:8080/tasks`.
The cluster name is used as ECS namespace.

It is recommended to create an ECS namespace with the same name as the cluster.
Upon creation of the ECS Service for the discovery agent, use the namespace to
enable "Service Discovery", and specify the name `ecs-task-discovery-agent` as
DNS.

If the discovery agent is running on a namespace distinct from the cluster
name, or if other name than `ecs-task-discovery-agent` is used as DNS in the
"Service Discovery", then you must tell your application how to find the
discovery agent by setting the env var `ECS_TASK_DISCOVERY_AGENT_URL` as
follows:

	ECS_TASK_DISCOVERY_AGENT_URL=http://{AgentServiceDiscoveryDNS}.{AgentNamespace}:8080/tasks

### Agent Checklist

  - Enable the discovery agent if possible.

  - Use the port 8080 for the agent in both ECS task definition and service.

  - Create an ECS Service for the agent with name `ecs-task-discovery-agent`;
    otherwise, on ECS task definition for the agent, set the env var
    `ECS_TASK_DISCOVERY_AGENT_SERVICE` to the name of the agent ECS Service.

  - Enable Service Discovery on the ECS Service created for the agent. If
    possible, create a nameservice for the Service Discovery using the same
    name as the cluster, and in the Service Discovery use the name
    `ecs-task-discovery-agent` as DNS. Otherwise set the env var
    `ECS_TASK_DISCOVERY_AGENT_URL` in the ECS task definition of your
    application as follows:

    ECS_TASK_DISCOVERY_AGENT_URL=http://{AgentServiceDiscoveryDNS}.{AgentNamespace}:8080/tasks

    {AgentServiceDiscoveryDNS}: replace with the the DNS specified in the service
    discovery created in the ECS service for the agent.

    {AgentNamespace}: replace with the namespace used in the service discovery
    created in the ECS service for the agent.

  - Agent docker image: docker.io/udhos/ecs-task-discovery-agent:latest
*/
package groupcachediscovery
