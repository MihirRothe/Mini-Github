# ForgeHub Architecture & Engineering Design

This document outlines the architectural blueprints, design decisions, and system constraints for the ForgeHub platform.

---

## 1. System Topology

ForgeHub is implemented as a high-performance **modular monolith** in Go paired with a modern React 19 single-page application. The architecture prioritizes clear module boundaries, strict layer separation, and zero leakage of database or filesystem structures to HTTP clients.

```
┌─────────────────────────────────────────────────────────────┐
│                       Client Layer                          │
│   Web Browser (React 19 SPA)   │    Native Git CLI (HTTP)   │
└──────────────┬────────────────────────────────┬─────────────┘
               │                                │
               ▼                                ▼
┌─────────────────────────────────────────────────────────────┐
│                 ForgeHub API Gateway / Router               │
│   - Security Headers (CSP, HSTS, X-Frame-Options)           │
│   - CORS & Request Correlation ID                           │
│   - Structured Request Logging                              │
│   - Panic Recovery & Standardized Error Serialization       │
└──────────────────────────────┬──────────────────────────────┘
                               │
               ┌───────────────┴───────────────┐
               ▼                               ▼
┌─────────────────────────────┐ ┌─────────────────────────────┐
│      REST API Modules       │ │       Git HTTP Engine       │
│  - Auth & Sessions          │ │  - /info/refs               │
│  - Organizations & Teams    │ │  - /git-upload-pack         │
│  - Repositories & Branches  │ │  - /git-receive-pack        │
│  - Issues & Pull Requests   │ │  - Bare repo storage        │
│  - CI Workflows & Artifacts │ │  - Permission validation    │
│  - ForgeAI Assistant        │ │  - Path traversal defense   │
│  - Audit Logs & Admin       │ │                             │
└──────────────┬──────────────┘ └──────────────┬──────────────┘
               │                               │
               ├───────────────────────────────┘
               ▼
┌─────────────────────────────────────────────────────────────┐
│                      Persistence Layer                      │
│   PostgreSQL 16 (Relational Entities & Audit Logs)          │
│   Redis 7 (Queues, Pub/Sub, Realtime Updates)               │
│   Local / Object Storage (Bare Git Repositories & Artifacts)│
└─────────────────────────────────────────────────────────────┘
```

---

## 2. Module Boundaries

Each module inside `apps/api/internal/modules/` adheres to a strict four-layer architecture:

1. **HTTP Handler (`handler.go`)**:
   - Parses and validates request bodies and query parameters.
   - Extracts authenticated identity from request context.
   - Delegates directly to the service layer.
   - Formats success responses or structured error envelopes.
   - **No business logic or SQL queries permitted in handlers.**

2. **Service Layer (`service.go`)**:
   - Executes business logic, permission evaluation, and domain event dispatching.
   - Coordinates multi-entity workflows inside database transactions.
   - Calls storage and external provider abstractions.

3. **Repository Layer (`repository.go`)**:
   - Encapsulates all SQL queries, parameter binding, and database scanning.
   - Returns typed domain structs or typed errors.

4. **Domain Models (`models.go`)**:
   - Defines immutable entity structs, request DTOs, response representations, and enum definitions.

---

## 3. Storage Abstractions

### RepositoryStorage Interface
ForgeHub decouples Git operations from direct filesystem manipulation using the `RepositoryStorage` interface:

```go
type RepositoryStorage interface {
    CreateRepository(ctx context.Context, namespace, name string) error
    DeleteRepository(ctx context.Context, namespace, name string) error
    Exists(ctx context.Context, namespace, name string) bool
    ListBranches(ctx context.Context, namespace, name string) ([]Branch, error)
    ListTags(ctx context.Context, namespace, name string) ([]Tag, error)
    GetCommit(ctx context.Context, namespace, name, sha string) (*Commit, error)
    GetTree(ctx context.Context, namespace, name, ref, path string) (*Tree, error)
    GetFile(ctx context.Context, namespace, name, ref, path string) (*Blob, error)
    GetDiff(ctx context.Context, namespace, name, baseRef, headRef string) (*Diff, error)
    Merge(ctx context.Context, opts MergeOptions) (*MergeResult, error)
}
```

This ensures future expansion into distributed storage or S3-backed Git mirrors without rewriting application logic.

---

## 4. CI/CD Runner Architecture

Workflows specified in `.forgehub/workflows/*.yml` are executed in strict isolation:
- Workflow definitions are parsed and validated against strict schemas.
- Jobs are enqueued into a Redis-backed queue (`asynq`).
- Dedicated worker processes launch disposable Docker containers with:
  - Strict CPU and memory limits.
  - Read-only root filesystem with isolated temporary working directories.
  - Network isolation options.
  - Non-root execution (`uid 1000`).
  - Execution timeouts (default 15 minutes).
- Job logs are streamed via WebSockets or Server-Sent Events to the web client and stored in the database.

---

## 5. Security Architecture

1. **Password Hashing**: Argon2id with memory = 64MB, iterations = 3, parallelism = 2.
2. **Session Security**: 256-bit cryptographically secure session tokens stored as SHA-256 hashes in PostgreSQL; cookies marked `HttpOnly`, `SameSite=Lax`, and `Secure` in production.
3. **Authorization**: Server-side Role-Based Access Control (RBAC) enforced on every operation. Client-provided roles or ownership are never trusted.
4. **Path Traversal Protection**: Repository paths are strictly constructed using sanitized, validated slugs. Any attempt to use `..`, `/`, `\`, or null bytes produces an immediate validation error.
