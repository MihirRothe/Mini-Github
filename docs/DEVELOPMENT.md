# ForgeHub Local Development Guide

This guide covers setting up your local development environment, running services, executing tests, and developing new features.

---

## 1. Prerequisites

- **Go**: 1.23 or newer (tested on 1.27.0)
- **Node.js**: 20 LTS or newer (tested on v24.13.0)
- **npm**: 10 or newer
- **Git**: 2.40 or newer
- **Docker** *(optional for native dev, required for full containerized stack)*

---

## 2. Directory Layout

```
forgehub/
├── apps/
│   ├── api/             # Go REST API and Git services
│   └── web/             # React 19 / TypeScript / Vite frontend
├── migrations/          # SQL database migrations
├── scripts/             # Development scripts (e.g. dev.ps1)
└── docker-compose.yml   # Multi-service container orchestrator
```

---

## 3. Running Locally

### Starting with PowerShell (Windows)
```powershell
.\scripts\dev.ps1
```

### Starting Independently

**Terminal 1 — API Server**:
```powershell
cd apps/api
go run cmd/server/main.go
```
The API server starts on `http://localhost:8080`.
Check health: `curl http://localhost:8080/healthz`

**Terminal 2 — Web Frontend**:
```powershell
cd apps/web
npm install
npm run dev
```
The Vite development server starts on `http://localhost:5173` and proxies `/api` calls to `http://localhost:8080`.

---

## 4. Running Automated Tests

### Run all tests:
```bash
make test
```

### Backend tests only:
```bash
cd apps/api
go test -v ./...
```

### Frontend build & type check:
```bash
cd apps/web
npm run build
```

---

## 5. Coding Standards

- **Go**: Follow `gofmt` and standard idiomatic Go practices. Return structured errors using `internal/errors`.
- **TypeScript**: Strict mode enabled (`noImplicitAny`, `strictNullChecks`).
- **Commits**: Follow conventional commits (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`).
