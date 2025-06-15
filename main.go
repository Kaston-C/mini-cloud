package main

import (
	"context"
	"log"
	"mini-cloud/internal/api"
	"mini-cloud/internal/cluster"
	"mini-cloud/internal/docker"
	"mini-cloud/internal/manager"
	"mini-cloud/internal/resourcemanager"
	"time"
)

func main() {
	ctx := context.Background()

	// Create node 1
	dc1, err := docker.NewDockerClient()
	if err != nil {
		log.Fatalf("failed to create docker client 1: %v", err)
	}
	rm1 := resourcemanager.NewResourceManager(4.0, 8192)
	mgr1 := manager.NewManager(dc1, rm1)
	mgr1.StartExpirationLoop(ctx, 15*time.Second)

	// Create node 2
	dc2, err := docker.NewDockerClient()
	if err != nil {
		log.Fatalf("failed to create docker client 2: %v", err)
	}
	rm2 := resourcemanager.NewResourceManager(8.0, 16384)
	mgr2 := manager.NewManager(dc2, rm2)
	mgr2.StartExpirationLoop(ctx, 15*time.Second)

	node1 := &cluster.Node{ID: "node1", Docker: dc1, Resources: rm1, Manager: mgr1}
	node2 := &cluster.Node{ID: "node2", Docker: dc2, Resources: rm2, Manager: mgr2}

	nodes := map[string]*cluster.Node{
		node1.ID: node1,
		node2.ID: node2,
	}

	clusterMgr := cluster.NewClusterManager(nodes)
	srv := api.NewClusterServer(clusterMgr)

	log.Fatal(srv.Run(":8080"))
}
