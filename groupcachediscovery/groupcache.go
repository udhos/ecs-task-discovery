package groupcachediscovery

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/groupcache/groupcache-go/v3/transport/peer"
	"github.com/udhos/ecs-task-discovery/discovery"
)

// PeerGroup is an interface to plug in a target for delivering peering
// updates. *groupcache.HTTPPool, created with
// groupcache.NewHTTPPoolOpts(), implements this interface.
type PeerGroup interface {
	Set(peers ...string)
}

// PeerSet is an interface to plug in a target for delivering peering
// updates. *groupcache.daemon, created with
// groupcache.ListenAndServe(), implements this interface.
type PeerSet interface {
	SetPeers(ctx context.Context, peers []peer.Info) error
}

// Options specify options.
type Options struct {
	// Pool is an interface to plug in a target for delivering peering
	// updates. *groupcache.HTTPPool, created with
	// groupcache.NewHTTPPoolOpts(), implements this interface.
	// Pool supports groupcache2.
	Pool PeerGroup

	// Peers is an interface to plug in a target for delivering peering
	// updates. *groupcache.Daemon, created with
	// groupcache.ListenAndServe(), implements this interface.
	// Peers supports groupcache3.
	Peers PeerSet

	// Client provides ECS client.
	Client *ecs.Client

	// GroupCachePort is the listening port used by groupcache peering http
	// server. For instance, ":5000".
	GroupCachePort string

	// ServiceName filters tasks by service name.
	ServiceName string

	// ForceSingleTask forces our local IP address.
	// If defined, it should be set to our actual IP address.
	// The function FindMyAddr() provides a suitable address.
	// It is useful only for locally running the application.
	ForceSingleTask string

	// DisableAgentQuery skips querying the task discovery agent.
	// The task discovery agent sets DisableAgentQuery to true in order to not query itself.
	// Most applications should leave it undefined (set to false).
	DisableAgentQuery bool
}

func buildURL(addr, groupcachePort string) string {
	return "http://" + addr + groupcachePort
}

// Run starts the discovery.
func Run(options Options) error {

	const me = "groupcachediscovery.Run: callback"

	myAddr, errAddr := FindMyAddr()
	if errAddr != nil {
		return errAddr
	}

	callback := func(tasks []discovery.Task) {

		size := len(tasks)

		infof("%s: %d tasks", me, size)

		if size == 0 {
			return
		}

		if options.Peers != nil {
			//
			// groupcache3
			//

			peers := make([]peer.Info, 0, size)

			for i, t := range tasks {
				hostPort := t.Address + options.GroupCachePort
				isSelf := myAddr == t.Address

				infof("%s: %d/%d: service=%s task=%s addr=%s health_status=%s last_status=%s host_port=%s is_self=%t",
					me, i+1, size, options.ServiceName, t.ARN, t.Address, t.HealthStatus, t.LastStatus, hostPort, isSelf)

				peers = append(peers, peer.Info{
					Address: hostPort,
					IsSelf:  isSelf,
				})
			}

			err := options.Peers.SetPeers(context.TODO(), peers)
			if err != nil {
				errorf("%s: groupcache3 set peers error: %v", me, err)
			}
		} else {
			//
			// groupcache2
			//
			peers := make([]string, 0, size)

			for i, t := range tasks {
				infof("%s: %d/%d: service=%s task=%s addr=%s health_status=%s last_status=%s",
					me, i+1, size, options.ServiceName, t.ARN, t.Address, t.HealthStatus, t.LastStatus)

				peers = append(peers, buildURL(t.Address, options.GroupCachePort))
			}

			options.Pool.Set(peers...)
		}

	}

	disc, err := discovery.New(discovery.Options{
		ServiceName:       options.ServiceName,
		Client:            options.Client,
		Callback:          callback,
		ForceSingleTask:   options.ForceSingleTask,
		DisableAgentQuery: options.DisableAgentQuery,
	})
	if err != nil {
		return err
	}

	go disc.Run()

	return nil
}

// FindMyURL returns my URL for groupcache pool.
// groupcachePort example: ":5000".
// Sample resulting URL: "http://10.0.0.1:5000"
func FindMyURL(groupcachePort string) (string, error) {
	addr, errAddr := FindMyAddr()
	if errAddr != nil {
		return "", errAddr
	}
	url := buildURL(addr, groupcachePort)
	return url, nil
}

// FindMyAddr returns our local IP address.
func FindMyAddr() (string, error) {
	host, errHost := os.Hostname()
	if errHost != nil {
		return "", errHost
	}
	addrs, errAddr := net.LookupHost(host)
	if errAddr != nil {
		return "", errAddr
	}
	if len(addrs) < 1 {
		return "", fmt.Errorf("findMyAddr: hostname '%s': no addr found", host)
	}
	addr := addrs[0]
	if len(addrs) > 1 {
		return addr, fmt.Errorf("findMyAddr: hostname '%s': found multiple addresses: %v", host, addrs)
	}
	return addr, nil
}
