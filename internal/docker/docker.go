package docker

import (
	"context"
	containerTypes "github.com/docker/docker/api/types/container"
	imageTypes "github.com/docker/docker/api/types/image"
	networkTypes "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"io"
	"os"
	"time"
)

// DockerClient wraps the Docker SDK client
type DockerClient struct {
	cli *client.Client
}

// NewDockerClient creates a new Docker client instance
func NewDockerClient() (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerClient{cli: cli}, nil
}

// PullImage ensures the image is present locally
func (dc *DockerClient) PullImage(ctx context.Context, image string) error {
	out, err := dc.cli.ImagePull(ctx, image, imageTypes.PullOptions{})
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(os.Stdout, out)
	return err
}

// ContainerSpec defines parameters to create a container
type ContainerSpec struct {
	Image   string
	Name    string
	CPU     float64 // in cores
	Memory  int64   // in MB
	Command []string
	TTL     time.Duration
}

// CreateContainer creates a container with the given spec
func (dc *DockerClient) CreateContainer(ctx context.Context, spec ContainerSpec) (string, error) {
	config := &containerTypes.Config{
		Image: spec.Image,
		Cmd:   spec.Command,
	}

	hostConfig := &containerTypes.HostConfig{
		Resources: containerTypes.Resources{
			NanoCPUs: int64(spec.CPU * 1e9), // convert to nanoseconds
			Memory:   spec.Memory * 1024 * 1024,
		},
	}

	networkingConfig := &networkTypes.NetworkingConfig{}

	resp, err := dc.cli.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, spec.Name)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

// StartContainer starts a container by ID
func (dc *DockerClient) StartContainer(ctx context.Context, id string) error {
	return dc.cli.ContainerStart(ctx, id, containerTypes.StartOptions{})
}

// StopContainer stops a running container
func (dc *DockerClient) StopContainer(ctx context.Context, id string) error {
	return dc.cli.ContainerStop(ctx, id, containerTypes.StopOptions{})
}

// RemoveContainer deletes a container
func (dc *DockerClient) RemoveContainer(ctx context.Context, id string) error {
	return dc.cli.ContainerRemove(ctx, id, containerTypes.RemoveOptions{Force: true})
}

// ListContainers returns containers created by this tool
func (dc *DockerClient) ListContainers(ctx context.Context) ([]containerTypes.Summary, error) {
	return dc.cli.ContainerList(ctx, containerTypes.ListOptions{All: true})
}

// InspectContainer returns detailed container info
func (dc *DockerClient) InspectContainer(ctx context.Context, id string) (containerTypes.InspectResponse, error) {
	return dc.cli.ContainerInspect(ctx, id)
}
