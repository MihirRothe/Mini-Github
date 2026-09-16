# ForgeHub — Project Status & Roadmap

ForgeHub is an independent, production-grade, self-hostable developer collaboration platform featuring Git repository hosting, code review, pull requests, issues, containerized CI/CD, real-time updates, AI assistance, organizations, teams, and administrative auditing.

---

## Current Milestone

* **Current Phase**: **PHASE 8 — Webhooks & Automation** (Up Next)
* **Completed Milestones**: Phase 1 (Foundation), Phase 2 (Authentication & User Management), Phase 3 (Organizations, Teams & RBAC), Phase 4 (Git Storage & Smart HTTP Engine), Phase 5 (Issues & Collaboration), Phase 6 (Pull Requests & Code Review), Phase 7 (CI/CD System & Container Runners)
* **Target**: Event dispatcher, HMAC-SHA256 signed payloads, webhook deliveries, retry queue, delivery history inspection.

---

## Phase Roadmap

| Phase | Description | Status | Verification |
|---|---|---|---|
| **Phase 1** | **Foundation** (Monorepo, API, Web, Postgres, Redis, Health, Docker, Tests) | 🟢 **Completed** | Backend & frontend running, tests pass, healthcheck 200 OK |
| **Phase 2** | **Authentication & User Management** (Argon2id, Sessions, Cookies, Profiles, Auth Middleware) | 🟢 **Completed** | Argon2id tests, Registration, Login, Session cookies, Tokens |
| **Phase 3** | **Organizations, Teams & RBAC** (Orgs, Teams, Granular Repository Permissions) | 🟢 **Completed** | RBAC permission matrix tests, Org/Team CRUD, live API workflow |
| **Phase 4** | **Git Storage & Smart HTTP** (Bare repos, Git HTTP clone/push, branch/tree browsing) | 🟢 **Completed** | Git CLI clone/push over HTTP, Smart HTTP auth/permission tests, file browser UI |
| **Phase 5** | **Issues & Collaboration** (Issue tracker, comments, labels, milestones, assignments) | 🟢 **Completed** | Issues CRUD, comments thread, state toggling, labels, milestones, filtering |
| **Phase 6** | **Pull Requests & Code Review** (Diff viewer, PR lifecycle, reviews, approvals, merge engine) | 🟢 **Completed** | PR creation, reviews, approvals, comments, 3-way merge & squash tests |
| **Phase 7** | **CI/CD System & Container Runners** (Workflow parser, job queue, Docker runner, log streaming) | 🟢 **Completed** | YAML parser, step execution, failing step skipping, live logs |
| **Phase 8** | **Webhooks & Automation** (Event dispatcher, signed payloads, delivery retries) | ⚪ Planned | Webhook delivery & retry tests |
| **Phase 9** | **Global Search** (Full-text repository, issue, PR, and code search) | ⚪ Planned | Search relevance & filter tests |
| **Phase 10** | **ForgeAI** (Provider abstraction, code explanation, PR review, test generation) | ⚪ Planned | AI prompt security, response validation tests |
| **Phase 11** | **Admin Dashboard & Observability** (Admin metrics, audit logs, user management, metrics) | ⚪ Planned | Audit logging & metrics endpoint tests |
| **Phase 12** | **Production Hardening** (Security audit, performance testing, backup/restore, hardening) | ⚪ Planned | Security & benchmark reports |

---

## Architectural Decisions Log (ADR)

1. **ADR-001: Modular Monorepo Architecture**
   - *Context*: ForgeHub requires tight integration between API, Web, Worker, and Git services without prematurely introducing microservice networking overhead.
   - *Decision*: Adopt a modular monolith in Go with explicit package boundaries (`apps/api/internal/modules/*`), standard data access abstractions, and a modern React/Vite/TypeScript frontend in `apps/web`.

2. **ADR-002: Dual-Mode Persistence & Zero-Dependency Local Dev**
   - *Context*: Production uses PostgreSQL 16+ and Redis 7+ via Docker Compose. Local developers or environments without Docker should still be able to run and test immediately.
   - *Decision*: The backend natively connects to PostgreSQL and Redis when configured. For environments lacking external daemons, it supports embedded miniredis and SQLite/test driver fallbacks, ensuring 100% testability anywhere.

3. **ADR-003: Structured Errors & API Contract**
   - *Context*: Client-server communication must be predictable, secure, and devoid of leaked stack traces.
   - *Decision*: Standardize on `{ "error": { "code": "STRING_ENUM", "message": "Human readable", "details": ... } }` for all non-2xx responses.

4. **ADR-004: Native Bare Git Repositories**
   - *Context*: Git hosting requires absolute reliability and compatibility with standard Git tooling.
   - *Decision*: Store bare Git repositories in isolated directory hierarchies (`/data/git/users/:user/:repo.git` and `/data/git/orgs/:org/:repo.git`) behind a `RepositoryStorage` interface, preventing any path traversal.

---

## Known Issues & Notes

- Host environment is Windows with Node v24, npm 11, Git 2.53, and Go 1.27. Docker Desktop is not present on the host; full Docker Compose files are supplied and validated for Linux/container deployments.
