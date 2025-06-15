package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"mini-cloud/internal/cluster"
	"mini-cloud/internal/docker"
)

// provisionRequest defines the JSON format for provisioning a container
type provisionRequest struct {
	Name   string  `json:"name"`
	Image  string  `json:"image"`
	CPU    float64 `json:"cpu"`
	Memory int64   `json:"memory"`
	TTL    string  `json:"ttl"`
}

// ClusterServer exposes HTTP endpoints for a multi-node mini-cloud
type ClusterServer struct {
	cluster *cluster.ClusterManager
	ctx     context.Context
}

// NewClusterServer creates and configures the API server using a ClusterManager
func NewClusterServer(cm *cluster.ClusterManager) *ClusterServer {
	return &ClusterServer{
		cluster: cm,
		ctx:     context.Background(),
	}
}

// Run starts the HTTP server
func (s *ClusterServer) Run(addr string) error {
	http.HandleFunc("/provision", s.handleProvision)
	http.HandleFunc("/terminate/", s.handleTerminate) // expects /terminate/{id}
	http.HandleFunc("/status/", s.handleStatus)       // expects /status/{id}
	http.HandleFunc("/list", s.handleList)

	log.Printf("Starting cluster server on %s...", addr)
	return http.ListenAndServe(addr, nil)
}

// handleProvision creates a container across any available node
func (s *ClusterServer) handleProvision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req provisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	ttl, err := time.ParseDuration(req.TTL)
	if err != nil {
		http.Error(w, "Invalid TTL format (example: \"10s\", \"5m\"): "+err.Error(), http.StatusBadRequest)
		return
	}

	spec := docker.ContainerSpec{
		Name:   req.Name,
		Image:  req.Image,
		CPU:    req.CPU,
		Memory: req.Memory,
		TTL:    ttl,
	}

	info, err := s.cluster.Schedule(s.ctx, spec)
	if err != nil {
		http.Error(w, "Provision failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(info)
}

// handleTerminate deletes a container regardless of which node it's on
func (s *ClusterServer) handleTerminate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/terminate/")
	if id == "" {
		http.Error(w, "Missing container ID", http.StatusBadRequest)
		return
	}

	if err := s.cluster.TerminateContainer(s.ctx, id); err != nil {
		http.Error(w, "Terminate failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, "Container terminated")
}

func (s *ClusterServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/status/")
	if id == "" {
		http.Error(w, "Missing container ID", http.StatusBadRequest)
		return
	}

	info, err := s.cluster.GetContainerStatus(s.ctx, id)
	if err != nil {
		http.Error(w, "Status lookup failed: "+err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(info)
	if err != nil {
		return
	}
}

// handleList lists all active containers across all nodes
func (s *ClusterServer) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	containers := s.cluster.ListAllContainers(s.ctx)
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(containers)
	if err != nil {
		return
	}
}
