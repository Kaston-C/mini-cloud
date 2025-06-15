package resourcemanager

import (
	"sync"
)

type ResourceSpec struct {
	CPU    float64 // in cores
	Memory int     // in MB
}

type ResourceManager struct {
	TotalCPU    float64
	TotalMemory int

	allocatedCPU    map[string]float64
	allocatedMemory map[string]int

	mu sync.Mutex
}

func NewResourceManager(cpu float64, memory int) *ResourceManager {
	return &ResourceManager{
		TotalCPU:        cpu,
		TotalMemory:     memory,
		allocatedCPU:    make(map[string]float64),
		allocatedMemory: make(map[string]int),
	}
}

func (rm *ResourceManager) CanAllocate(spec ResourceSpec) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	usedCPU := 0.0
	usedMem := 0
	for _, v := range rm.allocatedCPU {
		usedCPU += v
	}
	for _, v := range rm.allocatedMemory {
		usedMem += v
	}

	return (usedCPU+spec.CPU <= rm.TotalCPU) && (usedMem+spec.Memory <= rm.TotalMemory)
}

func (rm *ResourceManager) Allocate(id string, spec ResourceSpec) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	usedCPU := 0.0
	usedMem := 0
	for _, v := range rm.allocatedCPU {
		usedCPU += v
	}
	for _, v := range rm.allocatedMemory {
		usedMem += v
	}

	if usedCPU+spec.CPU > rm.TotalCPU || usedMem+spec.Memory > rm.TotalMemory {
		return false
	}

	rm.allocatedCPU[id] = spec.CPU
	rm.allocatedMemory[id] = spec.Memory
	return true
}

func (rm *ResourceManager) Release(id string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	delete(rm.allocatedCPU, id)
	delete(rm.allocatedMemory, id)
}

func (rm *ResourceManager) Usage() ResourceSpec {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	usedCPU := 0.0
	usedMem := 0
	for _, v := range rm.allocatedCPU {
		usedCPU += v
	}
	for _, v := range rm.allocatedMemory {
		usedMem += v
	}
	return ResourceSpec{
		CPU:    usedCPU,
		Memory: usedMem,
	}
}

func (rm *ResourceManager) AllocatedCPUSum() float64 {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	sum := 0.0
	for _, c := range rm.allocatedCPU {
		sum += c
	}
	return sum
}

func (rm *ResourceManager) AllocatedMemorySum() int {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	sum := 0
	for _, m := range rm.allocatedMemory {
		sum += m
	}
	return sum
}
