# ForgeHub Production Deployment Guide

This guide covers deploying ForgeHub to production environments using Docker Compose, Kubernetes, and reverse proxies with TLS termination.

---

## 1. Production Topologies

ForgeHub can be deployed on a single VM via Docker Compose or across a distributed cluster via Kubernetes.

### Architecture Topology
- **Edge Reverse Proxy**: Nginx, Caddy, or Traefik providing TLS termination (Let's Encrypt), rate limiting, and HTTP/2.
- **ForgeHub Web**: Caddy/Nginx serving static SPA assets with cache-control headers.
- **ForgeHub API**: Go binary running behind the reverse proxy with healthcheck probes.
- **PostgreSQL 16+**: High-availability database with streaming replication or managed RDS/Cloud SQL.
- **Redis 7+**: Cluster or Sentinel setup for queuing and realtime updates.
- **Persistent Volume**: Network-attached storage (NFS, Ceph, AWS EBS) for bare Git repositories.

---

## 2. Docker Compose Production Deployment

### 1. Configure Production Secrets
Create `.env` based on `.env.example`:
```bash
FORGEHUB_ENV=production
FORGEHUB_PORT=8080
DATABASE_URL=postgres://forgehub_user:STRONG_PASSWORD@postgres:5432/forgehub?sslmode=verify-full
REDIS_URL=redis://:REDIS_STRONG_PASSWORD@redis:6379/0
SESSION_SECRET=GENERATE_64_CHAR_HEX_KEY
GIT_ROOT_DIR=/data/git
```

### 2. Start Services
```bash
docker compose -f docker-compose.yml up -d
```

### 3. Verify Health
```bash
curl -f http://localhost:8080/healthz
```

---

## 3. Reverse Proxy Configuration (Caddy Example)

```caddy
forgehub.example.com {
    encode gzip zstd
    
    # API and Git Smart HTTP endpoints
    handle /api/* {
        reverse_proxy api:8080
    }
    handle /*.git/* {
        reverse_proxy api:8080
    }
    
    # Static Web SPA
    handle {
        reverse_proxy web:80
    }
}
```

---

## 4. Backups and Disaster Recovery

### PostgreSQL Backup
```bash
pg_dump -Fc -h localhost -U forgehub -d forgehub > /backups/forgehub_db_$(date +%Y%m%d_%H%M%S).dump
```

### Git Repositories Backup
```bash
rsync -aAX --delete /data/git/ /backups/git_mirrors/
```
Schedule daily automated snapshots with 30-day retention policies.
