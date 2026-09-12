# ForgeHub Security Policy & Threat Model

ForgeHub takes software security and data protection seriously. This document details the security posture, threat mitigation strategies, and vulnerability reporting procedures.

---

## 1. Threat Mitigation Strategies

### 1.1 Authentication & Credential Storage
- Passwords are hashed using **Argon2id** (memory: 64MB, iterations: 3, parallelism: 2). Plaintext passwords are never logged, cached, or transmitted insecurely.
- Session IDs are 256-bit cryptographic random tokens. The database stores only the SHA-256 hash of the session token.
- Cookies use `HttpOnly`, `SameSite=Lax`, and `Secure` attributes.

### 1.2 Path Traversal & Injection Defenses
- **Git Repositories**: Repository directories are strictly determined by validated slugs (`^[a-zA-Z0-9._-]+$`). Direct concatenation of user input into filesystem paths is strictly prohibited.
- **SQL Injection**: All database queries use parameterized prepared statements through `pgx/v5`.
- **Command Injection**: Arbitrary workflow commands are executed exclusively within isolated, disposable Docker containers with non-root users, restricted memory/CPU, and no access to host sockets.

### 1.3 Authorization Boundaries
- Every API endpoint validates server-side permissions (Organization role, Team membership, Repository access tier). Client-provided roles or permissions are discarded.
- Insecure Direct Object References (IDOR) are mitigated by verifying ownership/read access on every entity query.

### 1.4 Rate Limiting & Denial of Service
- Critical endpoints (Login, Registration, Password Reset, Git HTTP push) are protected by IP and account-based rate limiters.
- Maximum payload limits (10MB for JSON, 100MB for Git packfiles) are enforced at the HTTP gateway level.

---

## 2. Reporting a Vulnerability

If you discover a security vulnerability in ForgeHub, please report it responsibly:
- **Email**: security@forgehub.local (or submit an encrypted issue).
- Do not disclose security vulnerabilities publicly before a fix is released.
- You will receive an initial response within 24 hours acknowledging receipt.
