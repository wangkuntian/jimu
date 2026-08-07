# Data Import + WebSocket Real-time Communication Design

**Date:** 2026-08-08
**Status:** Approved

## Overview

Two parallel features:
1. **Data Import** - CSV/Excel bulk import with validation, preview, partial success, error reporting
2. **WebSocket Real-time Communication** - Notifications, online status, instant messaging

---

## Part 1: Data Import

### Architecture

```
internal/platform/importer/
├── importer.go          # Import interface + registry
├── csv_importer.go      # CSV parser
├── excel_importer.go    # Excel parser (excelize)
├── validator.go         # Data validation engine
└── result.go            # Import result (success/errors/preview)
```

### Import Flow

```
Upload File → Parse (CSV/Excel) → Validate → Preview/Import → Error Report
                ↓
         Column Mapping (header → field)
                ↓
         Row Validation (type/required/uniqueness)
                ↓
         Partial Success Import + Error Row Report
```

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/admin/users/import/preview` | Preview import result |
| POST | `/api/v1/admin/users/import` | Execute import |
| GET | `/api/v1/admin/users/import/:id` | Import status/result |
| GET | `/api/v1/admin/users/import/template` | Download import template |

### Error Handling Modes

1. **Preview Mode** - Validate only, return result without importing
2. **Partial Success** - Valid rows imported, errors reported per row
3. **Detailed Error Report** - Row number, field name, error message, original value

### Validation Rules

- Type checking (string/int/date/email)
- Required field validation
- Uniqueness check (username, email)
- Format validation (email, phone, ID card)
- Custom validators per entity type

### Import Result

```go
type ImportResult struct {
    TotalRows   int            `json:"total_rows"`
    SuccessRows int            `json:"success_rows"`
    ErrorRows   int            `json:"error_rows"`
    Errors      []ImportError  `json:"errors,omitempty"`
    Duration    string         `json:"duration"`
}

type ImportError struct {
    Row     int    `json:"row"`
    Field   string `json:"field"`
    Message string `json:"message"`
    Value   string `json:"value"`
}
```

### Dependencies

- `excelize/v2` - Excel file parsing
- `encoding/csv` - CSV parsing (stdlib)
- `github.com/gocarina/gcsv` - CSV struct mapping (optional)

---

## Part 2: WebSocket Real-time Communication

### Architecture

```
internal/platform/ws/
├── hub.go               # Connection management (extend existing)
├── handler.go           # WebSocket HTTP upgrader
├── message.go           # Message protocol
├── presence.go          # Online status management
└── channels.go          # Channel/room management
```

### Message Protocol

```go
type WSMessage struct {
    Type    string      `json:"type"`    // notification/chat/presence/ping
    Channel string      `json:"channel"` // user:123 / room:abc / broadcast
    Payload interface{} `json:"payload"`
    Time    time.Time   `json:"time"`
}
```

### Real-time Scenarios

| Scenario | Message Type | Description |
|----------|--------------|-------------|
| **Notification** | `notification` | System alerts, task completion, security events |
| **Online Status** | `presence` | User online/offline/typing |
| **Instant Message** | `chat` | User-to-user messaging |

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| WS | `/api/v1/ws` | WebSocket connection endpoint |
| POST | `/api/v1/ws/push` | HTTP push notification (fallback) |
| GET | `/api/v1/ws/presence/:userId` | Query user online status |

### Connection Flow

```
Client → JWT Auth → Upgrade WS → Register to Hub → Subscribe Channel
                                    ↓
                              Heartbeat + Auto-reconnect
```

### Presence Management

- User connects → status = online
- User disconnects → status = offline (with delay)
- Heartbeat every 30s
- Typing indicator (3s timeout)

### Channel Types

| Channel | Pattern | Description |
|---------|---------|-------------|
| **User** | `user:{id}` | Direct messages to specific user |
| **Room** | `room:{id}` | Group chat room |
| **Broadcast** | `broadcast` | System-wide notifications |

### WebSocket Config

```go
type WSConfig struct {
    HeartbeatInterval int   `mapstructure:"heartbeat_interval"` // default 30s
    WriteTimeout      int   `mapstructure:"write_timeout"`      // default 10s
    MaxMessageSize    int64 `json:"max_message_size"`          // default 64KB
    ReconnectInterval int   `json:"reconnect_interval"`        // default 5s
}
```

---

## Implementation Order

### Phase 1: Data Import (independent)
1. Import interface + CSV parser
2. Excel parser
3. Validation engine
4. Admin HTTP endpoints

### Phase 2: WebSocket (independent)
1. Message protocol + handler
2. Presence management
3. Channel/room management
4. Admin HTTP endpoints

### Phase 3: Integration
1. Import completion → WebSocket notification
2. Online status in admin dashboard

---

## Database Migration

### 011_create_import_jobs.sql
```sql
CREATE TABLE IF NOT EXISTS import_jobs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    type VARCHAR(64) NOT NULL COMMENT 'import type (users/products)',
    filename VARCHAR(255) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT 'pending/processing/completed/failed',
    total_rows INT NOT NULL DEFAULT 0,
    success_rows INT NOT NULL DEFAULT 0,
    error_rows INT NOT NULL DEFAULT 0,
    errors TEXT COMMENT 'JSON error details',
    created_by BIGINT UNSIGNED,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    PRIMARY KEY (id),
    INDEX idx_status (status),
    INDEX idx_created_by (created_by)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```
