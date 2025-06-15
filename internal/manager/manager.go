package manager

import (
	"context"
	"fmt"
	"mini-cloud/internal/docker"
	"mini-cloud/internal/resourcemanager"
	"sync"
	"time"
)

// ContainerInfo holds metadata about a running container
type ContainerInfo struct {
	ID        string
	Name      string
	Image     string
	CPU       float64
	MemoryMB  int64
	CreatedAt time.Time
	Status    string
	TTL       time.Duration
}

// Manager controls the lifecycle of containers
type Manager struct {
	docker    *docker.DockerClient
	mutex     sync.Mutex
	state     map[string]*ContainerInfo
	resources *resourcemanager.ResourceManager
}

// NewManager initializes a Manager instance
func NewManager(dc *docker.DockerClient, rm *resourcemanager.ResourceManager) *Manager {
	return &Manager{
		docker:    dc,
		state:     make(map[string]*ContainerInfo),
		resources: rm,
	}
}

func (m *Manager) AddContainer(id string, info *ContainerInfo) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.state[id] = info
}

// ProvisionContainer creates and starts a container
func (m *Manager) ProvisionContainer(ctx context.Context, spec docker.ContainerSpec) (*ContainerInfo, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	rSpec := resourcemanager.ResourceSpec{
		CPU:    spec.CPU,
		Memory: int(spec.Memory),
	}

	if !m.resources.CanAllocate(rSpec) {
		return nil, fmt.Errorf("insufficient resources to allocate container")
	}

	if !m.resources.Allocate(spec.Name, rSpec) {
		return nil, fmt.Errorf("failed to reserve resources")
	}

	if err := m.docker.PullImage(ctx, spec.Image); err != nil {
		m.resources.Release(spec.Name)
		return nil, fmt.Errorf("failed to pull image: %w", err)
	}

	id, err := m.docker.CreateContainer(ctx, spec)
	if err != nil {
		m.resources.Release(spec.Name)
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	if err := m.docker.StartContainer(ctx, id); err != nil {
		m.resources.Release(spec.Name)
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	info := &ContainerInfo{
		ID:        id,
		Name:      spec.Name,
		Image:     spec.Image,
		CPU:       spec.CPU,
		MemoryMB:  spec.Memory,
		CreatedAt: time.Now(),
		Status:    "running",
		TTL:       spec.TTL,
	}
	m.state[id] = info

	return info, nil
}

// TerminateContainer stops and removes a container
func (m *Manager) TerminateContainer(ctx context.Context, id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	info, exists := m.state[id]
	if !exists {
		return fmt.Errorf("container not found")
	}

	if err := m.docker.StopContainer(ctx, id); err != nil {
		return fmt.Errorf("stop error: %w", err)
	}

	if err := m.docker.RemoveContainer(ctx, id); err != nil {
		return fmt.Errorf("remove error: %w", err)
	}

	m.resources.Release(info.Name)
	delete(m.state, id)
	return nil
}

// GetContainerStatus returns metadata about a container
func (m *Manager) GetContainerStatus(ctx context.Context, id string) (*ContainerInfo, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	info, ok := m.state[id]
	if !ok {
		return nil, fmt.Errorf("container not found")
	}
	return info, nil
}

// ListActiveContainers returns all tracked containers
func (m *Manager) ListActiveContainers(ctx context.Context) ([]*ContainerInfo, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var containers []*ContainerInfo
	for _, info := range m.state {
		containers = append(containers, info)
	}
	return containers, nil
}

func (m *Manager) StartExpirationLoop(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.cleanupExpiredContainers(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (m *Manager) cleanupExpiredContainers(ctx context.Context) {
	now := time.Now()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	for id, info := range m.state {
		if info.TTL > 0 && info.CreatedAt.Add(info.TTL).Before(now) {
			// Unlock temporarily while terminating (avoid deadlock)
			m.mutex.Unlock()
			err := m.TerminateContainer(ctx, id)
			m.mutex.Lock()

			if err != nil {
				fmt.Printf("Failed to auto-terminate expired container %s: %v\n", id, err)
			} else {
				fmt.Printf("Auto-terminated expired container %s\n", id)
			}
		}
	}
}
