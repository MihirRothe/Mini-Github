# ForgeHub

<div align="center">

**A Self-Hosted, Production-Grade Developer Collaboration Platform**

[![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8?style=flat&logo=go)](https://go.dev)
[![React](https://img.shields.io/badge/React-19-61DAFB?style=flat&logo=react)](https://react.dev)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.x-3178C6?style=flat&logo=typescript)](https://www.typescriptlang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16+-4169E1?style=flat&logo=postgresql)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7+-DC382D?style=flat&logo=redis)](https://redis.io/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

</div>

---

## Overview

ForgeHub is an independent, self-hostable developer platform designed to deliver the capabilities of modern Git collaboration suites without vendor lock-in. Built from the ground up with a clean modular architecture, ForgeHub provides:

- **Git Repository Hosting**: Full Git Smart HTTP push/pull compatibility, branch/tag management, and bare repository storage.
- **Developer Collaboration**: Issue tracking, code review, inline discussions, pull requests, approvals, and atomic merge engine.
- **CI/CD Workflows**: GitHub Actions-style workflow definitions (`.forgehub/workflows/*.yml`) executed in isolated Docker containers with real-time log streaming.
- **Organizations & Access Control**: Granular team permissions, public/private repository scoping, and comprehensive audit trails.
- **AI-Assisted Development (ForgeAI)**: Architectural explanations, contextual code breakdown, pull request security reviews, and test generation.
- **Administrative Telemetry**: System health metrics, active jobs, storage monitoring, and audit log inspection.

---

## Repository Structure

```text
forgehub/
├── apps/
│   ├── web/               # React 19 + TypeScript + Vite + Tailwind CSS frontend
│   └── api/               # Go 1.27 modular monolith REST API & Git engine
├── migrations/            # Versioned SQL database migrations
├── docker/                # Production multi-stage Dockerfiles
├── docs/                  # Architecture, Security, API, and Deployment documentation
├── scripts/               # Developer automation scripts
├── docker-compose.yml     # Multi-service container orchestrator
├── Makefile               # Cross-platform development tasks
├── PROJECT_STATUS.md      # Milestones, ADRs, and implementation progress
└── README.md
```

---

## Quick Start

### Option 1: Docker Compose (Recommended for Production / Containers)

```bash
# Clone and enter the directory
cd forgehub

# Copy environment variables
cp .env.example .env

# Start all services (PostgreSQL, Redis, API, Web)
docker compose up -d

# Access the platform
# Frontend: http://localhost:5173 (or http://localhost:80 in production)
# API:      http://localhost:8080
```

### Option 2: Local Native Development

#### Prerequisites
- Go 1.23+ / 1.27+
- Node.js 20+ and npm
- Git 2.40+

```powershell
# Windows PowerShell Quick Start:
.\scripts\dev.ps1
```

Or run services independently:

```bash
# Terminal 1: Start API Server
cd apps/api
go run cmd/server/main.go

# Terminal 2: Start Frontend Web Shell
cd apps/web
npm install
npm run dev
```

---

## Architecture

ForgeHub is structured as a **modular monolith** with well-defined service and repository interfaces:

```
[ Browser / Git CLI ]
         │
         ├─── HTTP / Git Smart HTTP ───► [ Chi Router / Security Middleware ]
         │                                               │
         ▼                                               ├──► [ Auth & Sessions ]
[ React 19 Frontend ]                                    ├──► [ Git Storage Engine ]
  (TanStack Query, Monaco)                               ├──► [ Issues & Pull Requests ]
                                                         ├──► [ Workflow Engine & Queue ]
                                                         └──► [ ForgeAI Assistant ]
                                                                 │
                                                ┌────────────────┴───────────────┐
                                                ▼                                ▼
                                       [ PostgreSQL 16 ]                   [ Redis 7 ]
                                      (Data, Audits, PRs)             (Queues, PubSub, Cache)
```

For more details, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

---

## Documentation

- [Architecture & Design](docs/ARCHITECTURE.md)
- [Local Development Guide](docs/DEVELOPMENT.md)
- [Production Deployment](docs/DEPLOYMENT.md)
- [Security Policy & Threat Model](docs/SECURITY.md)

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
