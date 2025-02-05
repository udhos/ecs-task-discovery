// Package main implements the tool.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/udhos/boilerplate/boilerplate"
	"github.com/udhos/ecs-task-discovery/discovery"
)

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

	for {
		cluster, err := discovery.FindCluster()
		if err != nil {
			slog.Error(fmt.Sprintf("FATAL: find cluster error: %v", err))
		}
		slog.Info(fmt.Sprintf("cluster: '%s'", cluster))
		time.Sleep(10 * time.Second)
	}
}
