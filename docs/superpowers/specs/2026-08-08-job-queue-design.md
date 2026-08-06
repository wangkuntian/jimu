# Job Queue Design

**Date:** 2026-08-08
**Status:** Approved

## Overview

Full-featured job queue system with delayed tasks, async/sync execution, retry strategies, dead letter queue, and alerting. Redis for real-time queue + MySQL for persistence.

## Architecture

```
internal/platform/queue/
├── queue.go              # Queue interface + registry
├── redis_queue.go        # Redis queue (real-time + delayed)
├── mysql_store.go        # MySQL persistence (jobs + history)
├── worker.go             # Worker pool (consume + execute)
├── retry.go              # Retry strategies (fixed/exponential backoff)
└── dead_letter.go        # Dead letter queue handling
```

### Data Flow

**Real-time Job:**
```
Submit → Redis List (LPUSH) → Worker (BRPOP) → Execute → MySQL Update
```

**Delayed Job:**
```
SubmitDelayed → Redis ZSET (score=run_at) → Scanner → Move to List → Worker
```

**Failure Handling:**
```
Failed → attempts++ < max? → YES: Re-enqueue with delay
                           → NO: Dead Letter + Alert
```

## Data Models

### Job
```go
type Job struct {
    ID          uint64    `json:"id"`
    Type        string    `json:"type"`         // "send_email", "generate_report"
    Payload     string    `json:"payload"`      // JSON data
    Status      string    `json:"status"`       // pending/running/success/failed/dead
    Priority    int       `json:"priority"`     // 0-9, lower = higher priority
    Attempts    int       `json:"attempts"`     // Attempt count
    MaxAttempts int       `json:"max_attempts"` // Max retries
    NextRunAt   time.Time `json:"next_run_at"`  // Next execution time
    Error       string    `json:"error"`        // Last error message
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### JobHistory
```go
type JobHistory struct {
    ID        uint64    `json:"id"`
    JobID     uint64    `json:"job_id"`
    Status    string    `json:"status"`    // success/failed
    Error     string    `json:"error"`
    Duration  int64     `json:"duration"`  // Execution time in ms
    StartedAt time.Time `json:"started_at"`
    EndedAt   time.Time `json:"ended_at"`
}
```

### DeadLetter
```go
type DeadLetter struct {
    ID         uint64    `json:"id"`
    JobID      uint64    `json:"job_id"`
    Type       string    `json:"type"`
    Payload    string    `json:"payload"`
    FailReason string    `json:"fail_reason"`
    FailedAt   time.Time `json:"failed_at"`
    Resolved   bool      `json:"resolved"`
    ResolvedAt time.Time `json:"resolved_at,omitempty"`
}
```

## API

### Queue Interface
```go
type Queue interface {
    Submit(ctx context.Context, job *Job) error
    SubmitAndWait(ctx context.Context, job *Job, timeout time.Duration) (*JobResult, error)
    SubmitDelayed(ctx context.Context, job *Job, delay time.Duration) error
    GetJob(ctx context.Context, id uint64) (*Job, error)
    ListJobs(ctx context.Context, filter JobFilter, p pagination.Pagination) ([]Job, int64, error)
    RetryJob(ctx context.Context, id uint64) error
    ListDeadLetters(ctx context.Context, p pagination.Pagination) ([]DeadLetter, int64, error)
    ResolveDeadLetter(ctx context.Context, id uint64) error
}
```

### Worker Registration
```go
type WorkerFunc func(ctx context.Context, payload string) error

// Register a handler for a job type
func RegisterWorker(jobType string, fn WorkerFunc)
```

### HTTP Endpoints (Admin)
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/admin/jobs` | Submit a job |
| GET | `/api/v1/admin/jobs` | List jobs |
| GET | `/api/v1/admin/jobs/:id` | Get job detail |
| POST | `/api/v1/admin/jobs/:id/retry` | Manual retry |
| GET | `/api/v1/admin/jobs/dead-letters` | List dead letters |
| POST | `/api/v1/admin/jobs/dead-letters/:id/resolve` | Resolve dead letter |

## Retry Strategies

| Strategy | Description | Use Case |
|----------|-------------|----------|
| **Fixed** | Same interval between retries (e.g., 30s) | Simple scenarios |
| **Exponential Backoff** | Interval doubles each retry (1s → 2s → 4s → 8s...) | API calls, network requests |
| **Custom** | User-defined retry intervals | Special business logic |

**Default Config:**
- Max retries: 3
- Default strategy: Exponential backoff (initial 1s, max 60s)
- Formula: `min(2^attempt * baseDelay, maxDelay)`

## Failure Handling

```
Job Execution Failed
    ↓
attempts++ < maxAttempts?
    ├── YES → Calculate next run time → Re-enqueue (delayed queue)
    └── NO  → Move to Dead Letter Queue → Send Alert
```

### Dead Letter Queue
- Manual retry: Admin reviews and triggers retry
- Resolve: Mark as resolved after fixing issue
- Alert: Email/Webhook notification to admins

## Worker Pool

### Config
```go
type WorkerConfig struct {
    Workers     int           // Concurrency (default: 10)
    QueueName   string        // Redis queue name (default: "jimu:queue:default")
    PollTimeout time.Duration // Poll timeout (default: 5s)
    MaxRetries  int           // Default max retries (default: 3)
}
```

### Execution Mode
| Mode | Behavior | Return |
|------|----------|--------|
| **Async** | Submit → Return job ID immediately | `{job_id: "xxx"}` |
| **Sync** | SubmitAndWait → Block until complete | `{result: ..., error: ...}` |

### Worker Lifecycle
```
Start → Poll Redis Queue → Get Job → Execute → Update Status
            ↑                                    ↓
            └──────── Retry on Failure ←──────────┘
```

## Database Migration

### 010_create_jobs.sql
```sql
CREATE TABLE IF NOT EXISTS jobs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    type VARCHAR(64) NOT NULL,
    payload TEXT,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    priority INT NOT NULL DEFAULT 5,
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    next_run_at TIMESTAMP NULL,
    error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_status_next_run (status, next_run_at),
    INDEX idx_type (type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS job_history (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    job_id BIGINT UNSIGNED NOT NULL,
    status VARCHAR(16) NOT NULL,
    error TEXT,
    duration_ms BIGINT,
    started_at TIMESTAMP NULL,
    ended_at TIMESTAMP NULL,
    PRIMARY KEY (id),
    INDEX idx_job_id (job_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS dead_letters (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    job_id BIGINT UNSIGNED NOT NULL,
    type VARCHAR(64) NOT NULL,
    payload TEXT,
    fail_reason TEXT,
    failed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved TINYINT(1) NOT NULL DEFAULT 0,
    resolved_at TIMESTAMP NULL,
    PRIMARY KEY (id),
    INDEX idx_resolved (resolved)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

## Implementation Order

1. MySQL migrations (jobs, job_history, dead_letters tables)
2. Domain layer (Job, JobHistory, DeadLetter entities + repositories)
3. Redis queue (real-time + delayed)
4. MySQL store (persistence + history)
5. Worker pool (consume + execute)
6. Retry strategies (fixed + exponential backoff)
7. Dead letter queue
8. Admin HTTP endpoints
9. Worker registration + bootstrap integration
