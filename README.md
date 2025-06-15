# ☁️ Mini-Cloud

A lightweight mini-cloud infrastructure, **Mini-Cloud**, built in **Go** that uses Docker containers (acting as lightweight VMs) with multi-node scheduling and resource management.

> Simulates a multi-node cluster with resource-aware container provisioning and lifecycle management.

---

## 🚀 Features

* 🖥️ Multi-node cluster management with resource-aware scheduling
* 📊 Per-node resource tracking (CPU cores, memory)
* 🐳 Container provisioning with TTL and lifecycle management
* 🔌 REST API for container operations (provision, terminate, status, list)
* 🕒 Automatic cleanup of expired containers via expiration loop
* 🛠️ Support for static nodes representing physical machines

---

## 🧱 Architecture Overview

* **ClusterManager** — orchestrates scheduling and container placement across nodes
* **Node** — represents a physical node with Docker client, resource manager, and container lifecycle manager
* **ResourceManager** — tracks CPU and memory allocation per node
* **ContainerManager** — manages container lifecycles on a single node
* **API Server** — HTTP server exposing endpoints for container operations

---

## ⚙️ Getting Started

### Prerequisites

* [Go 1.20+](https://golang.org/dl/)
* [Docker](https://www.docker.com/get-started) installed and running locally
* Go Docker SDK dependency

---

### Installation & Running

```bash
git clone https://github.com/Kaston-C/mini-cloud.git
cd mini-cloud
go run main.go
```

By default, `main.go` creates two static nodes on the same machine with different resource capacities.
The API server listens on port `8080`.

---

## 🛠️ API Endpoints

| Method | Endpoint          | Description                    |
| ------ | ----------------- | ------------------------------ |
| POST   | `/provision`      | Provision a new container (VM) |
| POST   | `/terminate/{id}` | Terminate a container by ID    |
| GET    | `/status/{id}`    | Get container metadata         |
| GET    | `/list`           | List all active containers     |

---

### Example Provision Request

```bash
curl -X POST http://localhost:8080/provision \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test1",
    "image": "nginx",
    "cpu": 1.0,
    "memory": 2048,
    "ttl": "10m"
  }'
```

---

## 💡 Design Decisions

* **Static Nodes:** Nodes represent fixed physical machines; no dynamic node registration
* **Best-Fit Scheduling:** Containers are scheduled on the node leaving the fewest remaining resources after placement
* **Container TTL:** Containers auto-expire and are cleaned up after their TTL

---

## 🔮 Future Improvements

* ❤️‍🔥 Add node health monitoring and failure simulation
* 🔄 Support container migration between nodes
* 🔐 Add authentication and multi-tenant support
* 📈 Enable dynamic node registration for scaling

---

## 📋 License

MIT License — see [LICENSE](LICENSE)
