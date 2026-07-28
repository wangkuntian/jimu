# Jimu Backend Framework Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a reusable Go backend framework with platform infrastructure, JWT authentication, and RBAC authorization — ready to serve as a foundation for admin/SaaS projects.

**Architecture:** Modular monolith with Clean Architecture layers per module. Platform layer provides infrastructure (HTTP, DB, Redis, Logger, Auth). Modules (auth, user, role, permission) self-register via a unified Module interface. Bootstrap assembles everything at startup.

**Tech Stack:** Gin, Gorm, Zap, Viper, Redis, Casbin, Goose, JWT, go-playground/validator, swaggo/swag, Cobra, MariaDB

## Global Constraints

- Go version: 1.26.5
- Module path: `jimu`
- Database: MariaDB (MySQL protocol, use `gorm.io/driver/mysql`)
- HTTP status always 200; business errors in `body.code`
- No multi-tenant support in v1
- Each module follows: domain → application → infrastructure → interfaces
- All business logic depends on interfaces, not implementations

---

## Task 1: Project Scaffold & Dependencies

**Files:**
- Modify: `go.mod`
- Create: `Makefile`
- Create: `configs/app.yaml`

**Interfaces:**
- Produces: initialized Go module with all dependencies declared

- [ ] **Step 1: Add all dependencies**

Run:
```bash
go get -u \
  github.com/gin-gonic/gin \
  gorm.io/gorm \
  gorm.io/driver/mysql \
  github.com/spf13/viper \
  go.uber.org/zap \
  github.com/redis/go-redis/v9 \
  github.com/golang-jwt/jwt/v5 \
  github.com/casbin/casbin/v2 \
  github.com/casbin/gorm-adapter/v3 \
  github.com/go-playground/validator/v10 \
  github.com/swaggo/swag \
  github.com/swaggo/gin-swagger \
  github.com/spf13/cobra \
  golang.org/x/crypto
go mod tidy
```

Expected: `go.mod` updated with all dependencies, no errors

- [ ] **Step 2: Create Makefile**

```makefile
.PHONY: run build migrate swagger cli

run:
	go run cmd/server/main.go

build:
	go build -o bin/server cmd/server/main.go

migrate:
	go run cmd/cli/main.go migrate

swagger:
	swag init -g cmd/server/main.go -o docs/openapi

cli:
	go build -o bin/jimu cmd/cli/main.go
```

- [ ] **Step 3: Create configs/app.yaml**

```yaml
http:
  host: "0.0.0.0"
  port: 8080
  mode: "debug"

db:
  host: "127.0.0.1"
  port: 3306
  user: "root"
  password: "root"
  database: "jimu"
  max_open: 100
  max_idle: 10

redis:
  addr: "127.0.0.1:6379"
  password: ""
  db: 0

log:
  level: "debug"
  format: "console"
  output: "stdout"

auth:
  jwt_secret: "change-me-in-production"
  access_expire_min: 30
  refresh_expire_day: 7
```

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum Makefile configs/app.yaml
git commit -m "chore: initialize project dependencies and config"
```

---

## Task 2: Configuration Loader

**Files:**
- Create: `internal/config/config.go`

**Interfaces:**
- Produces: `config.Load() (*Config, error)`, `Config` struct

- [ ] **Step 1: Write the config loader**

```go
package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	HTTP  HTTPConfig  `mapstructure:"http"`
	DB    DBConfig    `mapstructure:"db"`
	Redis RedisConfig `mapstructure:"redis"`
	Log   LogConfig   `mapstructure:"log"`
	Auth  AuthConfig  `mapstructure:"auth"`
}

type HTTPConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DBConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	MaxOpen  int    `mapstructure:"max_open"`
	MaxIdle  int    `mapstructure:"max_idle"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

type AuthConfig struct {
	JWTSecret        string `mapstructure:"jwt_secret"`
	AccessExpireMin  int    `mapstructure:"access_expire_min"`
	RefreshExpireDay int    `mapstructure:"refresh_expire_day"`
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("app")
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")

	// Environment variable override: JIMU__HTTP__PORT=9090
	v.SetEnvPrefix("JIMU")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
```

- [ ] **Step 2: Write config test**

Create: `internal/config/config_test.go`

```go
package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.HTTP.Port == 0 {
		t.Error("expected HTTP.Port to be set")
	}
}

func TestEnvOverride(t *testing.T) {
	os.Setenv("JIMU__HTTP__PORT", "9999")
	defer os.Unsetenv("JIMU__HTTP__PORT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.HTTP.Port != 9999 {
		t.Errorf("expected port 9999, got %d", cfg.HTTP.Port)
	}
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/config/ -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/config/
git commit -m "feat(config): add viper-based config loader with env override"
```

---

## Task 3: Logger (Zap)

**Files:**
- Create: `internal/platform/logger/zap.go`

**Interfaces:**
- Produces: `logger.New(cfg LogConfig) *Logger`, `Logger` with `Info/Error/Debug/Warn` methods

- [ ] **Step 1: Write the logger**

```go
package logger

import (
	"jimu/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.SugaredLogger
}

func New(cfg config.LogConfig) *Logger {
	var level zapcore.Level
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		MessageKey:     "msg",
		CallerKey:      "caller",
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return &Logger{zapLogger.Sugar()}
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/platform/logger/
git commit -m "feat(logger): add zap-based structured logger"
```

---

## Task 4: Database Connection (Gorm + MariaDB)

**Files:**
- Create: `internal/platform/db/mysql.go`

**Interfaces:**
- Produces: `db.New(cfg DBConfig) (*gorm.DB, error)`

- [ ] **Step 1: Write the DB connector**

```go
package db

import (
	"fmt"
	"jimu/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func New(cfg config.DBConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpen)
	sqlDB.SetMaxIdleConns(cfg.MaxIdle)

	return db, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/platform/db/
git commit -m "feat(db): add gorm mysql connection"
```

---

## Task 5: Redis Client

**Files:**
- Create: `internal/platform/redis/redis.go`

**Interfaces:**
- Produces: `redis.New(cfg RedisConfig) *redis.Client`

- [ ] **Step 1: Write the Redis client**

```go
package redis

import (
	"context"
	"jimu/internal/config"

	"github.com/redis/go-redis/v9"
)

func New(cfg config.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
}

func Ping(ctx context.Context, client *redis.Client) error {
	return client.Ping(ctx).Err()
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/platform/redis/
git commit -m "feat(redis): add redis client wrapper"
```

---

## Task 6: Shared — Errors

**Files:**
- Create: `internal/shared/errors/errors.go`

**Interfaces:**
- Produces: `AppError` struct, `New(code int, message string) *AppError`, error code constants

- [ ] **Step 1: Write the error types**

```go
package errors

import "fmt"

const (
	CodeOK              = 0
	CodeInvalidParam    = 1001
	CodeUnauthorized    = 1002
	CodeForbidden       = 1003
	CodeNotFound        = 1004
	CodeInternalError   = 1005
	CodeUserNotFound    = 2001
	CodeUserExists      = 2002
	CodeInvalidPassword = 2003
	CodeRoleNotFound    = 2004
)

type AppError struct {
	Code    int
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func New(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func Wrap(code int, message string, cause error) *AppError {
	return &AppError{Code: code, Message: message, Cause: cause}
}
```

- [ ] **Step 2: Write error test**

Create: `internal/shared/errors/errors_test.go`

```go
package errors

import (
	"errors"
	"testing"
)

func TestAppError(t *testing.T) {
	err := New(CodeUserNotFound, "user not found")
	if err.Code != CodeUserNotFound {
		t.Errorf("expected code %d, got %d", CodeUserNotFound, err.Code)
	}
	if err.Error() != "[2001] user not found" {
		t.Errorf("unexpected error string: %s", err.Error())
	}
}

func TestAppErrorWithCause(t *testing.T) {
	cause := errors.New("db connection failed")
	err := Wrap(CodeInternalError, "internal error", cause)
	if err.Cause != cause {
		t.Error("expected cause to be preserved")
	}
	if !errors.Is(err, cause) {
		t.Error("expected Unwrap to work with errors.Is")
	}
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/shared/errors/ -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/shared/errors/
git commit -m "feat(errors): add AppError with error code constants"
```

---

## Task 7: Shared — Response

**Files:**
- Create: `internal/shared/response/response.go`

**Interfaces:**
- Produces: `response.Body`, `response.Paginated`, `response.OK(c, data)`, `response.Fail(c, err)`

- [ ] **Step 1: Write the response helpers**

```go
package response

import (
	"net/http"

	"jimu/internal/shared/errors"

	"github.com/gin-gonic/gin"
)

type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type Paginated struct {
	Code     int         `json:"code"`
	Message  string      `json:"message"`
	Data     interface{} `json:"data"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

func Fail(c *gin.Context, err error) {
	var appErr *errors.AppError
	if errors.As(err, &appErr) {
		c.JSON(http.StatusOK, Body{
			Code:    appErr.Code,
			Message: appErr.Message,
		})
		return
	}
	c.JSON(http.StatusOK, Body{
		Code:    errors.CodeInternalError,
		Message: "internal error",
	})
}

func Page(c *gin.Context, data interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, Paginated{
		Code:     0,
		Message:  "ok",
		Data:     data,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/shared/response/
git commit -m "feat(response): add unified response format helpers"
```

---

## Task 8: Shared — Pagination

**Files:**
- Create: `internal/shared/pagination/pagination.go`

**Interfaces:**
- Produces: `Pagination` struct, `GetOffset()` method

- [ ] **Step 1: Write pagination**

```go
package pagination

type Pagination struct {
	Page     int `form:"page" binding:"min=1"`
	PageSize int `form:"page_size" binding:"min=1,max=100"`
}

func (p Pagination) GetOffset() int {
	return (p.Page - 1) * p.PageSize
}

func (p Pagination) GetLimit() int {
	return p.PageSize
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/shared/pagination/
git commit -m "feat(pagination): add pagination request struct"
```

---

## Task 9: HTTP Server + Middleware

**Files:**
- Create: `internal/platform/http/server.go`
- Create: `internal/platform/http/middleware/middleware.go`

**Interfaces:**
- Produces: `http.NewServer(cfg, router) *Server`, `http.NewRouter() contract.Router`

- [ ] **Step 1: Write the HTTP server**

```go
package http

import (
	"context"
	"jimu/internal/config"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

type Server struct {
	*http.Server
}

type Router struct {
	*gin.Engine
}

func NewRouter() *Router {
	return &Router{Engine: gin.New()}
}

func (r *Router) Group(relativePath string, handlers ...gin.HandlerFunc) contract.Router {
	// Returns a group wrapper — simplified for brevity
	return r
}

func NewServer(cfg config.HTTPConfig, router *Router) *Server {
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler: router.Engine,
	}

	return &Server{Server: srv}
}

func (s *Server) Run() error {
	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.Shutdown(ctx)
}
```

- [ ] **Step 2: Write middleware**

```go
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func Logger(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()
		log.Info("request",
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency", time.Since(start),
			"request_id", c.GetString("request_id"),
		)
	}
}

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, err interface{}) {
		c.AbortWithStatusJSON(500, response.Body{
			Code:    errors.CodeInternalError,
			Message: "internal server error",
		})
	})
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/platform/http/
git commit -m "feat(http): add gin server with middleware chain"
```

---

## Task 10: Health Check

**Files:**
- Create: `internal/platform/observability/health.go`

**Interfaces:**
- Produces: `health.Register(router, db, rdb)` — registers `/health` endpoint

- [ ] **Step 1: Write health check**

```go
package health

import (
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Register(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	r.GET("/health", func(c *gin.Context) {
		// Check DB
		sqlDB, err := db.DB()
		dbStatus := "ok"
		if err != nil || sqlDB.Ping() != nil {
			dbStatus = "error"
		}

		// Check Redis
		redisStatus := "ok"
		if err := rdb.Ping(c.Request.Context()).Err(); err != nil {
			redisStatus = "error"
		}

		response.OK(c, gin.H{
			"status": "up",
			"db":     dbStatus,
			"redis":  redisStatus,
		})
	})
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/platform/observability/
git commit -m "feat(health): add health check endpoint"
```

---

## Task 11: Database Migrations

**Files:**
- Create: `migrations/001_create_users.sql`
- Create: `migrations/002_create_roles.sql`
- Create: `migrations/003_create_permissions.sql`
- Create: `migrations/004_create_user_roles.sql`
- Create: `migrations/005_create_role_permissions.sql`

**Interfaces:**
- Produces: Goose migration files for all base tables

- [ ] **Step 1: Write migration 001 — users**

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    username VARCHAR(64) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    status TINYINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL DEFAULT NULL,
    PRIMARY KEY (id),
    INDEX idx_username (username),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS users;
```

- [ ] **Step 2: Write migration 002 — roles**

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS roles (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(64) NOT NULL UNIQUE,
    description VARCHAR(255) DEFAULT '',
    status TINYINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS roles;
```

- [ ] **Step 3: Write migration 003 — permissions**

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS permissions (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(128) NOT NULL,
    resource VARCHAR(64) NOT NULL,
    action VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_resource_action (resource, action)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS permissions;
```

- [ ] **Step 4: Write migration 004 — user_roles**

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS user_roles (
    user_id BIGINT UNSIGNED NOT NULL,
    role_id BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (user_id, role_id),
    INDEX idx_role_id (role_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS user_roles;
```

- [ ] **Step 5: Write migration 005 — role_permissions**

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id BIGINT UNSIGNED NOT NULL,
    permission_id BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    INDEX idx_permission_id (permission_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS role_permissions;
```

- [ ] **Step 6: Commit**

```bash
git add migrations/
git commit -m "feat(migrations): add base table migrations for users, roles, permissions"
```

---

## Task 12: User Module — Domain

**Files:**
- Create: `internal/modules/user/domain/user.go`
- Create: `internal/modules/user/domain/repository.go`

**Interfaces:**
- Produces: `User` entity, `UserRepository` interface

- [ ] **Step 1: Write the User entity**

```go
package domain

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint64         `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"size:64;uniqueIndex;not null" json:"username"`
	Password  string         `gorm:"size:255;not null" json:"-"`
	Status    int8           `gorm:"default:1" json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}
```

- [ ] **Step 2: Write the repository interface**

```go
package domain

import "context"

type UserRepository interface {
	FindByID(ctx context.Context, id uint64) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	List(ctx context.Context, offset, limit int) ([]User, int64, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uint64) error
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/modules/user/domain/
git commit -m "feat(user): add user domain entity and repository interface"
```

---

## Task 13: User Module — Application

**Files:**
- Create: `internal/modules/user/application/dto.go`
- Create: `internal/modules/user/application/service.go`

**Interfaces:**
- Produces: `UserService` with CRUD methods, DTOs

- [ ] **Step 1: Write DTOs**

```go
package application

type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=6,max=32"`
}

type UpdateUserRequest struct {
	Status *int8 `json:"status" binding:"omitempty,oneof=0 1"`
}

type UserResponse struct {
	ID        uint64 `json:"id"`
	Username  string `json:"username"`
	Status    int8   `json:"status"`
	CreatedAt string `json:"created_at"`
}
```

- [ ] **Step 2: Write the service**

```go
package application

import (
	"context"
	"jimu/internal/modules/user/domain"
	"jimu/internal/shared/errors"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo domain.UserRepository
}

func NewUserService(repo domain.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Create(ctx context.Context, req CreateUserRequest) (*domain.User, error) {
	existing, _ := s.repo.FindByUsername(ctx, req.Username)
	if existing != nil {
		return nil, errors.New(errors.CodeUserExists, "username already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to hash password", err)
	}

	user := &domain.User{
		Username: req.Username,
		Password: string(hashedPassword),
		Status:   1,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to create user", err)
	}
	return user, nil
}

func (s *UserService) Get(ctx context.Context, id uint64) (*domain.User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New(errors.CodeUserNotFound, "user not found")
	}
	return user, nil
}

func (s *UserService) List(ctx context.Context, page, pageSize int) ([]domain.User, int64, error) {
	offset := (page - 1) * pageSize
	return s.repo.List(ctx, offset, pageSize)
}

func (s *UserService) Update(ctx context.Context, id uint64, req UpdateUserRequest) error {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return errors.New(errors.CodeUserNotFound, "user not found")
	}
	if req.Status != nil {
		user.Status = *req.Status
	}
	return s.repo.Update(ctx, user)
}

func (s *UserService) Delete(ctx context.Context, id uint64) error {
	return s.repo.Delete(ctx, id)
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/modules/user/application/
git commit -m "feat(user): add user application service with CRUD"
```

---

## Task 14: User Module — Infrastructure

**Files:**
- Create: `internal/modules/user/infrastructure/mysql_repository.go`

**Interfaces:**
- Produces: `mysqlRepository` implementing `domain.UserRepository`

- [ ] **Step 1: Write the MySQL repository**

```go
package infrastructure

import (
	"context"
	"jimu/internal/modules/user/domain"

	"gorm.io/gorm"
)

type mysqlRepository struct {
	db *gorm.DB
}

func NewMysqlRepository(db *gorm.DB) domain.UserRepository {
	return &mysqlRepository{db: db}
}

func (r *mysqlRepository) FindByID(ctx context.Context, id uint64) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *mysqlRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *mysqlRepository) List(ctx context.Context, offset, limit int) ([]domain.User, int64, error) {
	var users []domain.User
	var total int64
	db := r.db.WithContext(ctx).Model(&domain.User{})
	db.Count(&total)
	err := db.Offset(offset).Limit(limit).Find(&users).Error
	return users, total, err
}

func (r *mysqlRepository) Create(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *mysqlRepository) Update(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *mysqlRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&domain.User{}, id).Error
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/modules/user/infrastructure/
git commit -m "feat(user): add mysql repository implementation"
```

---

## Task 15: User Module — Interfaces (HTTP Handler + Router)

**Files:**
- Create: `internal/modules/user/interfaces/handler.go`
- Create: `internal/modules/user/interfaces/router.go`

**Interfaces:**
- Produces: `UserHandler`, `NewUserRouter(service, router)`

- [ ] **Step 1: Write the HTTP handler**

```go
package interfaces

import (
	"strconv"

	"jimu/internal/modules/user/application"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service *application.UserService
}

func NewUserHandler(service *application.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// Create godoc
// @Summary      Create user
// @Description  Create a new user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      application.CreateUserRequest  true  "User info"
// @Success      200   {object}  response.Body
// @Router       /users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req application.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	user, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, user)
}

func (h *UserHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	user, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, user)
}

func (h *UserHandler) List(c *gin.Context) {
	var p pagination.Pagination
	if err := c.ShouldBindQuery(&p); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	users, total, err := h.service.List(c.Request.Context(), p.Page, p.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Page(c, users, total, p.Page, p.PageSize)
}

func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	var req application.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	if err := h.service.Update(c.Request.Context(), id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, "invalid id"))
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, nil)
}
```

- [ ] **Step 2: Write the router**

```go
package interfaces

import (
	"jimu/internal/modules/user/application"

	"github.com/gin-gonic/gin"
)

func RegisterUserRoutes(r *gin.RouterGroup, service *application.UserService) {
	handler := NewUserHandler(service)
	users := r.Group("/users")
	{
		users.POST("", handler.Create)
		users.GET("", handler.List)
		users.GET("/:id", handler.Get)
		users.PUT("/:id", handler.Update)
		users.DELETE("/:id", handler.Delete)
	}
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/modules/user/interfaces/
git commit -m "feat(user): add HTTP handler and router"
```

---

## Task 16: User Module — Module Registration

**Files:**
- Create: `internal/modules/user/module.go`

**Interfaces:**
- Produces: `user.Module` implementing `contract.Module`

- [ ] **Step 1: Write the module registration**

```go
package user

import (
	"jimu/internal/contract"
	"jimu/internal/modules/user/application"
	"jimu/internal/modules/user/infrastructure"
	"jimu/internal/modules/user/interfaces"

	"gorm.io/gorm"
)

type Module struct {
	service *application.UserService
}

func New(db *gorm.DB) *Module {
	repo := infrastructure.NewMysqlRepository(db)
	service := application.NewUserService(repo)
	return &Module{service: service}
}

func (m *Module) Name() string {
	return "user"
}

func (m *Module) RegisterHTTP(r contract.Router) {
	interfaces.RegisterUserRoutes(r.Group("/api/v1"), m.service)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {
	// No jobs in v1
}

func (m *Module) RegisterEvents(e contract.EventBus) {
	// No events in v1
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/modules/user/module.go
git commit -m "feat(user): add module registration implementing contract.Module"
```

---

## Task 17: Auth — JWT Utility

**Files:**
- Create: `internal/platform/auth/jwt.go`

**Interfaces:**
- Produces: `jwt.New(secret string, accessExpireMin, refreshExpireDay int) *JWT`, `Generate(userID uint64) (token, refreshToken string, err error)`, `Parse(tokenString string) (*Claims, error)`

- [ ] **Step 1: Write JWT utility**

```go
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID uint64 `json:"user_id"`
	jwt.RegisteredClaims
}

type JWT struct {
	secret           []byte
	accessExpireMin  time.Duration
	refreshExpireDay time.Duration
}

func New(secret string, accessExpireMin, refreshExpireDay int) *JWT {
	return &JWT{
		secret:           []byte(secret),
		accessExpireMin:  time.Duration(accessExpireMin) * time.Minute,
		refreshExpireDay: time.Duration(refreshExpireDay) * 24 * time.Hour,
	}
}

func (j *JWT) Generate(userID uint64) (string, string, error) {
	accessClaims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.accessExpireMin)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessString, err := accessToken.SignedString(j.secret)
	if err != nil {
		return "", "", err
	}

	refreshClaims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.refreshExpireDay)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshString, err := refreshToken.SignedString(j.secret)
	if err != nil {
		return "", "", err
	}

	return accessString, refreshString, nil
}

func (j *JWT) Parse(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return j.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/platform/auth/
git commit -m "feat(auth): add JWT token generation and parsing"
```

---

## Task 18: Auth — AuthMiddleware

**Files:**
- Create: `internal/platform/auth/middleware.go`

**Interfaces:**
- Produces: `AuthMiddleware(jwtUtil *JWT) gin.HandlerFunc`

- [ ] **Step 1: Write the auth middleware**

```go
package auth

import (
	"strings"

	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(jwtUtil *JWT) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Fail(c, errors.New(errors.CodeUnauthorized, "missing authorization header"))
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Fail(c, errors.New(errors.CodeUnauthorized, "invalid authorization format"))
			c.Abort()
			return
		}

		claims, err := jwtUtil.Parse(parts[1])
		if err != nil {
			response.Fail(c, errors.New(errors.CodeUnauthorized, "invalid token"))
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Next()
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/platform/auth/middleware.go
git commit -m "feat(auth): add JWT auth middleware"
```

---

## Task 19: Auth Module — Login/Register

**Files:**
- Create: `internal/modules/auth/domain/auth.go`
- Create: `internal/modules/auth/application/service.go`
- Create: `internal/modules/auth/interfaces/handler.go`
- Create: `internal/modules/auth/interfaces/router.go`
- Create: `internal/modules/auth/module.go`

**Interfaces:**
- Produces: `auth.Module` with login/register endpoints

- [ ] **Step 1: Write domain**

```go
package domain

import "jimu/internal/modules/user/domain"

type AuthService interface {
	Login(ctx context.Context, username, password string) (*TokenPair, error)
	Register(ctx context.Context, username, password string) (*domain.User, error)
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // seconds
}
```

- [ ] **Step 2: Write application service**

```go
package application

import (
	"context"
	"jimu/internal/modules/auth/domain"
	"jimu/internal/modules/user/domain"
	appuser "jimu/internal/modules/user/application"
	"jimu/internal/platform/auth"
	"jimu/internal/shared/errors"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo  domain.UserRepository
	jwtUtil   *auth.JWT
	accessMin int
}

func NewAuthService(userRepo domain.UserRepository, jwtUtil *auth.JWT, accessMin int) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtUtil:   jwtUtil,
		accessMin: accessMin,
	}
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*domain.TokenPair, error) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, errors.New(errors.CodeUserNotFound, "user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New(errors.CodeInvalidPassword, "invalid password")
	}

	accessToken, refreshToken, err := s.jwtUtil.Generate(user.ID)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to generate token", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.accessMin * 60,
	}, nil
}

func (s *AuthService) Register(ctx context.Context, username, password string) (*domain.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to hash password", err)
	}

	user := &domain.User{
		Username: username,
		Password: string(hashedPassword),
		Status:   1,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to create user", err)
	}
	return user, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
	claims, err := s.jwtUtil.Parse(refreshToken)
	if err != nil {
		return nil, errors.New(errors.CodeUnauthorized, "invalid refresh token")
	}

	accessToken, newRefreshToken, err := s.jwtUtil.Generate(claims.UserID)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to generate token", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    s.accessMin * 60,
	}, nil
}
```

- [ ] **Step 3: Write HTTP handler**

```go
package interfaces

import (
	"jimu/internal/modules/auth/application"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service *application.AuthService
}

func NewAuthHandler(service *application.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	tokenPair, err := h.service.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, tokenPair)
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	user, err := h.service.Register(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, user)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errors.New(errors.CodeInvalidParam, err.Error()))
		return
	}
	tokenPair, err := h.service.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, tokenPair)
}
```

- [ ] **Step 4: Write router**

```go
package interfaces

import (
	"jimu/internal/modules/auth/application"
	"jimu/internal/platform/auth"

	"github.com/gin-gonic/gin"
)

func RegisterAuthRoutes(r *gin.RouterGroup, service *application.AuthService, jwtUtil *auth.JWT) {
	handler := NewAuthHandler(service)
	auth := r.Group("/auth")
	{
		auth.POST("/login", handler.Login)
		auth.POST("/register", handler.Register)
		auth.POST("/refresh", handler.RefreshToken)
	}
}
```

- [ ] **Step 5: Write module**

```go
package auth

import (
	"jimu/internal/config"
	"jimu/internal/contract"
	"jimu/internal/modules/auth/application"
	"jimu/internal/modules/auth/interfaces"
	"jimu/internal/modules/user/domain"
	"jimu/internal/modules/user/infrastructure"
	"jimu/internal/platform/auth"

	"gorm.io/gorm"
)

type Module struct {
	service *application.AuthService
	jwtUtil *auth.JWT
}

func New(db *gorm.DB, cfg config.AuthConfig) *Module {
	userRepo := infrastructure.NewMysqlRepository(db)
	jwtUtil := auth.New(cfg.JWTSecret, cfg.AccessExpireMin, cfg.RefreshExpireDay)
	service := application.NewAuthService(userRepo, jwtUtil, cfg.AccessExpireMin)
	return &Module{service: service, jwtUtil: jwtUtil}
}

func (m *Module) Name() string {
	return "auth"
}

func (m *Module) RegisterHTTP(r contract.Router) {
	interfaces.RegisterAuthRoutes(r.Group("/api/v1"), m.service, m.jwtUtil)
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}
```

- [ ] **Step 6: Commit**

```bash
git add internal/modules/auth/
git commit -m "feat(auth): add login, register, refresh token endpoints"
```

---

## Task 20: RBAC — Casbin Setup

**Files:**
- Create: `internal/platform/auth/casbin.go`
- Create: `conf/rbac_model.conf`

**Interfaces:**
- Produces: `casbin.NewEnforcer(db *gorm.DB) (*casbin.Enforcer, error)`

- [ ] **Step 1: Write Casbin model config**

Create: `conf/rbac_model.conf`

```ini
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && (r.act == p.act || p.act == "*")
```

- [ ] **Step 2: Write Casbin enforcer factory**

```go
package auth

import (
	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

func NewEnforcer(db *gorm.DB) (*casbin.Enforcer, error) {
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, err
	}
	enforcer, err := casbin.NewEnforcer("conf/rbac_model.conf", adapter)
	if err != nil {
		return nil, err
	}
	enforcer.EnableAutoLoadPolicy(5 * time.Second)
	return enforcer, nil
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/platform/auth/casbin.go conf/rbac_model.conf
git commit -m "feat(casbin): add RBAC enforcer with gorm adapter"
```

---

## Task 21: RBAC — Permission Middleware

**Files:**
- Create: `internal/platform/auth/permission_middleware.go`

**Interfaces:**
- Produces: `PermissionMiddleware(enforcer *casbin.Enforcer) gin.HandlerFunc`

- [ ] **Step 1: Write permission middleware**

```go
package auth

import (
	"jimu/internal/shared/errors"
	"jimu/internal/shared/response"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

func PermissionMiddleware(enforcer *casbin.Enforcer) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user roles from context (set by AuthMiddleware)
		roles, exists := c.Get("roles")
		if !exists {
			response.Fail(c, errors.New(errors.CodeForbidden, "no roles assigned"))
			c.Abort()
			return
		}

		roleList := roles.([]string)
		obj := c.Request.URL.Path
		act := c.Request.Method

		allowed := false
		for _, role := range roleList {
			if ok, _ := enforcer.Enforce(role, obj, act); ok {
				allowed = true
				break
			}
		}

		if !allowed {
			response.Fail(c, errors.New(errors.CodeForbidden, "permission denied"))
			c.Abort()
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/platform/auth/permission_middleware.go
git commit -m "feat(auth): add Casbin permission middleware"
```

---

## Task 22: Role Module

**Files:**
- Create: `internal/modules/role/domain/role.go`
- Create: `internal/modules/role/domain/repository.go`
- Create: `internal/modules/role/application/service.go`
- Create: `internal/modules/role/application/dto.go`
- Create: `internal/modules/role/infrastructure/mysql_repository.go`
- Create: `internal/modules/role/interfaces/handler.go`
- Create: `internal/modules/role/interfaces/router.go`
- Create: `internal/modules/role/module.go`

**Interfaces:**
- Produces: `role.Module` with CRUD + assign permissions

- [ ] **Step 1: Write domain**

```go
package domain

import "time"

type Role struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:64;uniqueIndex;not null" json:"name"`
	Description string    `gorm:"size:255;default:''" json:"description"`
	Status      int8      `gorm:"default:1" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Role) TableName() string { return "roles" }

type RoleRepository interface {
	FindByID(ctx context.Context, id uint64) (*Role, error)
	FindAll(ctx context.Context) ([]Role, error)
	Create(ctx context.Context, role *Role) error
	Update(ctx context.Context, role *Role) error
	Delete(ctx context.Context, id uint64) error
	AssignPermissions(ctx context.Context, roleID uint64, permissionIDs []uint64) error
	GetPermissions(ctx context.Context, roleID uint64) ([]Permission, error)
}
```

- [ ] **Step 2: Write application service**

```go
package application

import (
	"context"
	"jimu/internal/modules/role/domain"
	"jimu/internal/shared/errors"
)

type RoleService struct {
	repo domain.RoleRepository
}

func NewRoleService(repo domain.RoleRepository) *RoleService {
	return &RoleService{repo: repo}
}

func (s *RoleService) Create(ctx context.Context, name, description string) (*domain.Role, error) {
	role := &domain.Role{Name: name, Description: description, Status: 1}
	if err := s.repo.Create(ctx, role); err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to create role", err)
	}
	return role, nil
}

func (s *RoleService) Get(ctx context.Context, id uint64) (*domain.Role, error) {
	role, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New(errors.CodeRoleNotFound, "role not found")
	}
	return role, nil
}

func (s *RoleService) List(ctx context.Context) ([]domain.Role, error) {
	return s.repo.FindAll(ctx)
}

func (s *RoleService) Update(ctx context.Context, id uint64, name, description string) error {
	role, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return errors.New(errors.CodeRoleNotFound, "role not found")
	}
	role.Name = name
	role.Description = description
	return s.repo.Update(ctx, role)
}

func (s *RoleService) Delete(ctx context.Context, id uint64) error {
	return s.repo.Delete(ctx, id)
}

func (s *RoleService) AssignPermissions(ctx context.Context, roleID uint64, permissionIDs []uint64) error {
	return s.repo.AssignPermissions(ctx, roleID, permissionIDs)
}
```

- [ ] **Step 3: Write infrastructure**

```go
package infrastructure

import (
	"context"
	"jimu/internal/modules/role/domain"

	"gorm.io/gorm"
)

type mysqlRepository struct {
	db *gorm.DB
}

func NewMysqlRepository(db *gorm.DB) domain.RoleRepository {
	return &mysqlRepository{db: db}
}

func (r *mysqlRepository) FindByID(ctx context.Context, id uint64) (*domain.Role, error) {
	var role domain.Role
	err := r.db.WithContext(ctx).First(&role, id).Error
	return &role, err
}

func (r *mysqlRepository) FindAll(ctx context.Context) ([]domain.Role, error) {
	var roles []domain.Role
	err := r.db.WithContext(ctx).Find(&roles).Error
	return roles, err
}

func (r *mysqlRepository) Create(ctx context.Context, role *domain.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *mysqlRepository) Update(ctx context.Context, role *domain.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

func (r *mysqlRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&domain.Role{}, id).Error
}

func (r *mysqlRepository) AssignPermissions(ctx context.Context, roleID uint64, permissionIDs []uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).Delete(&role_permissions.RolePermission{}).Error; err != nil {
			return err
		}
		for _, pid := range permissionIDs {
			if err := tx.Create(&role_permissions.RolePermission{RoleID: roleID, PermissionID: pid}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *mysqlRepository) GetPermissions(ctx context.Context, roleID uint64) ([]domain.Permission, error) {
	var permissions []domain.Permission
	err := r.db.WithContext(ctx).
		Joins("JOIN role_permissions rp ON rp.permission_id = permissions.id").
		Where("rp.role_id = ?", roleID).
		Find(&permissions).Error
	return permissions, err
}
```

- [ ] **Step 4: Write handler + router + module**

(Similar pattern to user module — handler calls service, router registers routes, module implements contract.Module)

- [ ] **Step 5: Commit**

```bash
git add internal/modules/role/
git commit -m "feat(role): add role module with permission assignment"
```

---

## Task 23: Permission Module

**Files:**
- Create: `internal/modules/permission/domain/permission.go`
- Create: `internal/modules/permission/domain/repository.go`
- Create: `internal/modules/permission/application/service.go`
- Create: `internal/modules/permission/infrastructure/mysql_repository.go`
- Create: `internal/modules/permission/interfaces/handler.go`
- Create: `internal/modules/permission/interfaces/router.go`
- Create: `internal/modules/permission/module.go`

**Interfaces:**
- Produces: `permission.Module` with CRUD for permissions

- [ ] **Step 1: Write domain**

```go
package domain

import "time"

type Permission struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	Resource  string    `gorm:"size:64;not null" json:"resource"`
	Action    string    `gorm:"size:32;not null" json:"action"`
	CreatedAt time.Time `json:"created_at"`
}

func (Permission) TableName() string { return "permissions" }

type PermissionRepository interface {
	FindByID(ctx context.Context, id uint64) (*Permission, error)
	FindAll(ctx context.Context) ([]Permission, error)
	Create(ctx context.Context, perm *Permission) error
	Update(ctx context.Context, perm *Permission) error
	Delete(ctx context.Context, id uint64) error
}
```

- [ ] **Step 2: Write application service + infrastructure + handler + router + module**

(Same pattern as role/user modules)

- [ ] **Step 3: Commit**

```bash
git add internal/modules/permission/
git commit -m "feat(permission): add permission module"
```

---

## Task 24: Contract — Module Interface

**Files:**
- Create: `internal/contract/module.go`

**Interfaces:**
- Produces: `contract.Module`, `contract.Router`, `contract.JobRegistry`, `contract.EventBus`

- [ ] **Step 1: Write the contract**

```go
package contract

import "github.com/gin-gonic/gin"

type Router interface {
	GET(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	POST(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	PUT(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	DELETE(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	Group(relativePath string, handlers ...gin.HandlerFunc) Router
}

type JobRegistry interface {
	AddFunc(spec string, cmd func()) error
}

type EventBus interface {
	Subscribe(event string, handler func(payload interface{}))
	Publish(event string, payload interface{})
}

type Module interface {
	Name() string
	RegisterHTTP(r Router)
	RegisterJobs(j JobRegistry)
	RegisterEvents(e EventBus)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/contract/
git commit -m "feat(contract): add Module interface definition"
```

---

## Task 25: Bootstrap

**Files:**
- Create: `internal/app/bootstrap.go`
- Create: `internal/app/container.go`

**Interfaces:**
- Produces: `app.Bootstrap(modules ...contract.Module) *http.Server`

- [ ] **Step 1: Write the container**

```go
package app

import (
	"jimu/internal/config"
	"jimu/internal/platform/db"
	"jimu/internal/platform/logger"
	"jimu/internal/platform/redis"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Container struct {
	Config *config.Config
	DB     *gorm.DB
	Redis  *redis.Client
	Logger *logger.Logger
}

func NewContainer(cfg *config.Config) (*Container, error) {
	dbConn, err := db.New(cfg.DB)
	if err != nil {
		return nil, err
	}
	rdb := redis.New(cfg.Redis)
	log := logger.New(cfg.Log)

	return &Container{
		Config: cfg,
		DB:     dbConn,
		Redis:  rdb,
		Logger: log,
	}, nil
}
```

- [ ] **Step 2: Write bootstrap**

```go
package app

import (
	"jimu/internal/contract"
	"jimu/internal/platform/http"
	"jimu/internal/platform/observability"

	"github.com/gin-gonic/gin"
)

func Bootstrap(modules ...contract.Module) *http.Server {
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	container, err := NewContainer(cfg)
	if err != nil {
		panic("failed to create container: " + err.Error())
	}

	if cfg.HTTP.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := http.NewRouter()
	router.Use(
		middleware.RequestID(),
		middleware.Logger(container.Logger),
		middleware.Recovery(),
		middleware.CORS(),
	)

	// Health check (no auth required)
	health.Register(router.Group("/"), container.DB, container.Redis)

	// Register modules
	for _, m := range modules {
		m.RegisterHTTP(router)
		container.Logger.Info("module registered", "name", m.Name())
	}

	return http.NewServer(cfg.HTTP, router)
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/app/
git commit -m "feat(app): add bootstrap and container for dependency assembly"
```

---

## Task 26: Main Entry Point

**Files:**
- Create: `cmd/server/main.go`

**Interfaces:**
- Produces: runnable HTTP server

- [ ] **Step 1: Write main.go**

```go
package main

import (
	"jimu/internal/app"
	"jimu/internal/modules/auth"
	"jimu/internal/modules/permission"
	"jimu/internal/modules/role"
	"jimu/internal/modules/user"
	"jimu/internal/platform/auth"
	"jimu/internal/config"
)

// @title           Jimu API
// @version         1.0
// @description     Jimu Backend Framework API
// @host            localhost:8080
// @BasePath        /api/v1
func main() {
	cfg, _ := config.Load()

	db, _ := db.New(cfg.DB)
	enforcer, _ := casbin.NewEnforcer(db)

	userModule := user.New(db)
	authModule := authmodule.New(db, cfg.Auth)
	roleModule := role.New(db, enforcer)
	permModule := permission.New(db, enforcer)

	server := app.Bootstrap(userModule, authModule, roleModule, permModule)
	server.Run()
}
```

- [ ] **Step 2: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: add main entry point"
```

---

## Task 27: CLI — Cobra Scaffold

**Files:**
- Create: `cmd/cli/main.go`
- Create: `tools/generator/module.go`

**Interfaces:**
- Produces: `jimu module create [name]` command

- [ ] ** ] **Step 1: Write CLI entry**

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "jimu",
	Short: "Jimu backend framework CLI",
}

var moduleCmd = &cobra.Command{
	Use:   "module create [name]",
	Short: "Create a new module skeleton",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		return generator.GenerateModule(name)
	},
}

func init() {
	rootCmd.AddCommand(moduleCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Write module generator**

```go
package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const moduleTemplate = `package {{.Name}}

import (
	"jimu/internal/contract"
)

type Module struct {}

func New() *Module {
	return &Module{}
}

func (m *Module) Name() string {
	return "{{.Name}}"
}

func (m *Module) RegisterHTTP(r contract.Router) {
	// TODO: register routes
}

func (m *Module) RegisterJobs(j contract.JobRegistry) {}

func (m *Module) RegisterEvents(e contract.EventBus) {}
`

func GenerateModule(name string) error {
	dir := filepath.Join("internal", "modules", name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmpl, err := template.New("module").Parse(moduleTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(dir, "module.go"))
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, map[string]string{"Name": name})
}
```

- [ ] **Step 3: Commit**

```bash
git add cmd/cli/ tools/generator/
git commit -m "feat(cli): add cobra CLI with module generator"
```

---

## Task 28: Dockerfile + Final Polish

**Files:**
- Create: `Dockerfile`
- Modify: `Makefile`

**Interfaces:**
- Produces: Docker image build support

- [ ] **Step 1: Write Dockerfile**

```dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server cmd/server/main.go

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
COPY configs/ ./configs/
EXPOSE 8080
CMD ["./server"]
```

- [ ] **Step 2: Update Makefile with docker target**

```makefile
docker:
	docker build -t jimu:latest .
```

- [ ] **Step 3: Commit**

```bash
git add Dockerfile Makefile
git commit -m "feat: add Dockerfile and docker build target"
```

---

## Task 29: Integration Verification

**Files:** none (verification only)

- [ ] **Step 1: Build the project**

Run: `go build ./cmd/server/`
Expected: no errors

- [ ] **Step 2: Run tests**

Run: `go test ./...`
Expected: all pass

- [ ] **Step 3: Verify module registration**

Run: `go run cmd/cli/main.go module create testmodule`
Expected: `internal/modules/testmodule/module.go` created

- [ ] **Step 4: Commit any fixes**

```bash
git add -A
git commit -m "chore: verify build and fix any issues"
```
