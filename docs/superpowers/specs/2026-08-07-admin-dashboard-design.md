# Admin Dashboard Design

**Date:** 2026-08-07
**Status:** Approved
**Priority:** 1 (User/Permission Management) → 2 (Monitoring) → 3 (Config) → 4 (Tasks)

## Overview

Complete admin backend for operational management: user/permission administration, real-time monitoring, configuration hot-reload, task scheduling, and audit logging.

## Architecture

```
internal/modules/admin/
├── module.go                  # Aggregate module, registers all sub-routes
├── interfaces/
│   ├── users.go               # User management handlers
│   ├── apikeys.go             # API Key management handlers
│   ├── monitoring.go          # Monitoring handlers
│   ├── config.go              # Config hot-update handlers
│   ├── tasks.go               # Task scheduling handlers
│   └── audit.go               # Audit log handlers
├── application/
│   ├── users_service.go       # User CRUD, role assignment
│   ├── apikeys_service.go     # API Key CRUD, usage stats
│   ├── monitoring_service.go  # System + metrics aggregation
│   ├── config_service.go      # Config read/write with event publish
│   ├── tasks_service.go       # Task list, trigger, history
│   └── audit_service.go       # Audit logging
└── domain/
    ├── apikey.go              # API Key entity + repository interface
    └── audit.go               # Audit log entity + repository interface
```

## API Endpoints

### User Management (`/api/v1/admin/users`)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | List users (search, filter, sort, paginate) |
| POST | `/` | Create user (username, password, status, roles, metadata) |
| GET | `/:id` | Get user detail |
| PUT | `/:id` | Update user |
| DELETE | `/:id` | Disable user |
| POST | `/:id/roles` | Assign roles |
| DELETE | `/:id/roles/:roleID` | Revoke role |
| GET | `/export.csv` | Export user list |

### API Key Management (`/api/v1/admin/apikeys`)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | List API keys |
| POST | `/` | Create key (name, scopes, expires) |
| GET | `/:id` | Get key detail with usage stats |
| DELETE | `/:id` | Revoke key |

### Monitoring (`/api/v1/admin/monitoring`)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/status` | System status (version, uptime, goroutines, memory) |
| GET | `/metrics` | Real-time metrics (requests, latency, error rate) |
| GET | `/health` | Dependency health (DB, Redis) |

### Config Hot-Update (`/api/v1/admin/config`)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Get current runtime config |
| PUT | `/:key` | Update config value |
| POST | `/reload` | Trigger config reload |

### Task Scheduling (`/api/v1/admin/tasks`)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | List scheduled tasks |
| POST | `/:id/run` | Manually trigger task |
| POST | `/:id/toggle` | Pause/resume task |
| GET | `/:id/history` | Task execution history |

### Audit Log (`/api/v1/admin/audit`)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | List audit logs (filter by admin, action, time) |

## Data Models

### API Key
```go
type APIKey struct {
    ID        uint64    `json:"id"`
    Name      string    `json:"name"`
    KeyPrefix string    `json:"key_prefix"`
    KeyHash   string    `json:"-"`       // SHA-256, never expose
    Scopes    []string  `json:"scopes"`
    Enabled   bool      `json:"enabled"`
    ExpiresAt time.Time `json:"expires_at"`
    LastUsed  time.Time `json:"last_used"`
    UseCount  int64     `json:"use_count"`
    CreatedBy uint64    `json:"created_by"`
    CreatedAt time.Time `json:"created_at"`
}
```

### Audit Log
```go
type AuditLog struct {
    ID        uint64    `json:"id"`
    AdminID   uint64    `json:"admin_id"`
    AdminName string    `json:"admin_name"`
    Action    string    `json:"action"`   // "user.create", "config.update"
    Resource  string    `json:"resource"` // "user:123"
    Detail    string    `json:"detail"`   // JSON change detail
    IP        string    `json:"ip"`
    CreatedAt time.Time `json:"created_at"`
}
```

## Key Technical Decisions

### 1. API Key Security
- Plaintext returned only once (at creation), then only SHA-256 hash stored
- Verification uses `subtle.ConstantTimeCompare` to prevent timing attacks
- Key prefix (first 8 chars) stored for identification

### 2. Audit Log Strategy
- Async write via Event Bus (non-blocking)
- Independent audit table to avoid impacting business performance
- Records all admin actions with admin identity, timestamp, IP

### 3. Config Hot-Update Consistency
- Single node: update in-memory + Redis
- Multi-node: update Redis → publish event → all nodes refresh local cache
- No restart required

### 4. User List Search
- Username: LIKE query (with index)
- Status/Role: exact match filter
- Registration time: time range
- Sort field allowlist (prevent SQL injection)

### 5. Task Scheduling Extension
- Build on existing CronScheduler with execution history
- Task status: pending / running / success / failed
- Retain last 100 execution records per task

## Permissions

| Role | Capabilities |
|------|-------------|
| **Super Admin** | All operations including config, audit |
| **User Admin** | User CRUD, role assignment, API Key management |
| **Ops** | Monitoring view, task trigger, log view |
| **Read Only** | View-only access |

Implementation: Extend existing RBAC with admin scope middleware.

## Error Handling

| Scenario | HTTP | Behavior |
|----------|------|----------|
| User not found | 404 | Clear message |
| Permission denied | 403 | Required role indicated |
| Invalid params | 400 | Field-level details |
| Conflict (last admin) | 409 | Explain why |
| Server error | 500 | Generic message |

### Business Rules
- Cannot disable/delete self (prevent lockout)
- Cannot delete last super admin
- Validate role existence on assignment
- API Key scope must be within allowed set

## Testing Strategy

| Type | Coverage |
|------|----------|
| Unit | Service logic (user validation, API Key hash, config validation) |
| Integration | HTTP end-to-end (permission checks, CRUD flows, search) |
| Mock | Redis/DB via testutil or miniredis |

## Database Migrations

### 009_create_api_keys.sql
```sql
CREATE TABLE api_keys (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    key_prefix VARCHAR(16) NOT NULL,
    key_hash VARCHAR(64) NOT NULL,
    scopes TEXT COMMENT 'JSON array',
    enabled TINYINT(1) DEFAULT 1,
    expires_at TIMESTAMP NULL,
    last_used TIMESTAMP NULL,
    use_count BIGINT DEFAULT 0,
    created_by BIGINT UNSIGNED,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_key_hash (key_hash),
    INDEX idx_created_by (created_by)
);
```

### 009_create_audit_logs.sql
```sql
CREATE TABLE audit_logs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    admin_id BIGINT UNSIGNED NOT NULL,
    admin_name VARCHAR(64),
    action VARCHAR(64) NOT NULL,
    resource VARCHAR(128) NOT NULL,
    detail TEXT,
    ip VARCHAR(64),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_admin_id (admin_id),
    INDEX idx_action (action),
    INDEX idx_created_at (created_at)
);
```

## Implementation Order

1. **User/Permission Management** — highest frequency
2. **Monitoring Dashboard** — day-to-day ops
3. **Config Hot-Update** — reduce restarts
4. **Task Scheduling** — operational tooling
