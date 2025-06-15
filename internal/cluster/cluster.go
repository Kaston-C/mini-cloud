package cluster

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"

	"mini-cloud/internal/docker"
	"mini-cloud/internal/manager"
	"mini-cloud/internal/resourcemanager"
)

// Node represents a physical/virtual host running containers
type Node struct {
	ID        string
	Docker    *docker.DockerClient
	Resources *resourcemanager.ResourceManager
	Manager   *manager.Manager // per-node manager to track TTL etc.
}

// ClusterManager handles multi-node container scheduling
type ClusterManager struct {
	mu          sync.Mutex
	nodes       map[string]*Node
	assignments map[string]string // containerID -> nodeName
}

// NewClusterManager creates a new cluster from a slice of nodes
func NewClusterManager(nodes map[string]*Node) *ClusterManager {
	return &ClusterManager{
		nodes:       nodes,
		assignments: make(map[string]string),
	}
}

// Schedule schedules a container on a node with enough resources
func (cm *ClusterManager) Schedule(ctx context.Context, spec docker.ContainerSpec) (*manager.ContainerInfo, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var selectedNode *Node
	var minLeftover float64 = math.MaxFloat64

	for _, node := range cm.nodes {
		if node.Resources.CanAllocate(resourcemanager.ResourceSpec{
			CPU:    spec.CPU,
			Memory: int(spec.Memory),
		}) {
			// Calculate leftover resources after allocation
			leftoverCPU := node.Resources.TotalCPU - (node.Resources.AllocatedCPUSum() + spec.CPU)
			leftoverMem := float64(node.Resources.TotalMemory - (node.Resources.AllocatedMemorySum() + int(spec.Memory)))

			// Combine leftover CPU and Memory into a single metric (weighted sum)
			leftover := leftoverCPU + leftoverMem/1024.0 // normalize memory to cores roughly

			if leftover < minLeftover {
				minLeftover = leftover
				selectedNode = node
			}
		}
	}

	if selectedNode == nil {
		return nil, errors.New("no node has enough resources")
	}

	containerID := uuid.New().String()
	ok := selectedNode.Resources.Allocate(containerID, resourcemanager.ResourceSpec{
		CPU:    spec.CPU,
		Memory: int(spec.Memory),
	})
	if !ok {
		return nil, errors.New("failed to allocate resources")
	}

	spec.Name = containerID

	id, err := selectedNode.Docker.CreateContainer(ctx, spec)
	if err != nil {
		selectedNode.Resources.Release(containerID)
		return nil, err
	}

	err = selectedNode.Docker.StartContainer(ctx, id)
	if err != nil {
		err := selectedNode.Docker.RemoveContainer(ctx, id)
		if err != nil {
			return nil, err
		}
		selectedNode.Resources.Release(containerID)
		return nil, err
	}

	info := &manager.ContainerInfo{
		ID:        id,
		Name:      spec.Name,
		Image:     spec.Image,
		CPU:       spec.CPU,
		MemoryMB:  spec.Memory,
		CreatedAt: time.Now(),
		Status:    "Running",
		TTL:       spec.TTL,
	}

	selectedNode.Manager.AddContainer(id, info)
	return info, nil
}

// ListAllContainers lists all containers across all nodes
func (cm *ClusterManager) ListAllContainers(ctx context.Context) []*manager.ContainerInfo {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var all []*manager.ContainerInfo
	for _, node := range cm.nodes {
		containers, _ := node.Manager.ListActiveContainers(ctx)
		all = append(all, containers...)
	}
	return all
}

func (cm *ClusterManager) GetContainerStatus(ctx context.Context, id string) (*manager.ContainerInfo, error) {
	cm.mu.Lock()
	nodeName, ok := cm.assignments[id]
	cm.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("container %s not found", id)
	}

	node, exists := cm.nodes[nodeName]
	if !exists {
		return nil, fmt.Errorf("node %s not found for container %s", nodeName, id)
	}

	return node.Manager.GetContainerStatus(ctx, id)
}

// TerminateContainer finds and terminates container on any node
func (cm *ClusterManager) TerminateContainer(ctx context.Context, id string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for _, node := range cm.nodes {
		err := node.Manager.TerminateContainer(ctx, id)
		if err == nil {
			node.Resources.Release(id)
			return nil
		}
	}
	return errors.New("container not found")
}
