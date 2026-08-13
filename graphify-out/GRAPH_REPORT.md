# Graph Report - jimu  (2026-08-13)

## Corpus Check
- 400 files · ~127,423 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3419 nodes · 7476 edges · 247 communities (202 shown, 45 thin omitted)
- Extraction: 81% EXTRACTED · 19% INFERRED · 0% AMBIGUOUS · INFERRED: 1386 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `3dbb0b04`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- Role
- ImportResult
- RegisterAuthRoutes
- Storage
- fakeRedis
- auth/apikey.go
- common.sh
- Jimu Helm Chart Values
- NewAdminAPIKeyService
- T
- Page
- testConn
- generator/module.go
- JobData
- Permission
- WorkerPool
- AuthHandler
- mysqlRepository
- Router
- Hub
- NewSnowflake
- JobRegistry
- AdminUserService
- Wrap
- config/config.go
- Worker
- RateLimiter
- RedisCache
- CronScheduler
- health.go
- AuditLog
- OK
- Job
- WSMessage
- User
- OAuthService
- PresenceManager
- newLock
- InitSnowflake
- Message
- NewWithStore
- New
- Context
- JWT
- gormLogger
- DeadLetter
- MySQLStore
- Manager
- Limiter
- fakeRoleRepository
- ChannelManager
- Application
- RedisStore
- Jimu Backend Framework README
- upload_handler_test.go
- middleware/middleware_test.go
- NewEventBusPublisher
- ws_integration_test.go
- oauth/application/service_test.go
- AuditLogResponse
- Security
- Logger
- mysqlAuditRepository
- Data Model: Jimu 框架持久化实体
- config/config_test.go
- NewUserRateLimiter
- fakeStorage
- NewCSVExporter
- JobDef
- NewService
- NewCleanupService
- MySQLStore
- ValidateJSON
- NewMysqlRepository
- dispatcher
- New
- RabbitMQQueue
- Lock
- Now
- routerLimiterRedis
- auth/application/service_test.go
- i18n.go
- Outbox
- .Consume
- contract/module.go
- .RegisterHTTP
- Container
- AdminTaskService
- SetupRouter
- S3Storage
- LoginFailureTracker
- New
- OAuthBinding
- NewManagementServer
- Implementation Plan: Jimu 后端框架能力规格
- New
- AuditMiddleware
- adminAuthRouter
- responseBodyWriter
- DefaultCSRFConfig
- gorm_logger_test.go
- AuditService
- CI Workflow
- Jimu Constitution
- newMockGormDB
- AdminMonitoringService
- .Validate
- DBConfig
- Jimu API Swagger Definition
- RegisterRoleRoutes
- bindImportFile
- RoleService
- Bootstrap
- Tasks: Jimu 后端框架能力规格
- NewUserHandler
- signature_test.go
- NewWebhook
- NewAdminConfigService
- NewChannelManager
- NewPresenceManager
- Jimu Collaboration Rules (CLAUDE.md)
- UserService
- NewCSVImporter
- 统一响应契约
- Speckit SDD Workflow
- migrate_test.go
- gzipRouter
- NewWeChatProvider
- New
- newWSHandler
- validator/validator.go
- TestDB
- New
- oauth/interfaces/handler_test.go
- Context
- Email
- New
- Context
- SecurityHeadersFromConfig
- Redis Service
- AuthorizationMiddleware
- newMockGormDB
- SMS
- Context
- New
- setupTestCache
- EventBus
- Timeout
- GitHubProvider
- UserInfo
- NewDBAPIKeyStore
- Server
- Registry
- Duration
- Server
- idempotencyRouter
- setupMetricsEngine
- New
- NewServer
- Release Workflow
- Module Layering & contract.Module
- AdminUserHandler
- 配置契约
- Module 契约
- workerPoolComponent
- events.go
- Prometheus Service
- CLI 契约
- NewLogChannel
- circuit
- Context
- 010_create_jobs.sql
- Security Policy
- 001_create_users.sql
- CaptchaHandler
- .validateCommon
- 002_create_roles.sql
- NewEmail
- securityRouter
- 003_create_permissions.sql
- openapi.go
- 004_create_user_roles.sql
- NewMQPublisher
- smoke_api_contract.sh
- test_runtime_security.sh
- LoginRequest
- GitHub Flow Branch Strategy
- 005_create_role_permissions.sql
- backup.sh
- restore.sh
- Checklist Template
- Config Enum Startup Validation
- Goose Migration Convention
- Docker Compose Topology
- Bug Report Template
- Docker Images Update
- GitHub Actions Update
- Go Dependencies Update
- Feature Request Template
- Pull Request Template
- 006_create_audit_logs.sql
- 007_create_outbox_events.sql
- 008_add_audit_changes.sql
- 009_create_api_keys.sql
- 011_create_import_jobs.sql
- 013_create_user_oauth_bindings.sql
- 014_create_scheduled_jobs.sql
- mockTokenServer
- mysqlRepository
- RegisterUserRoutes
- Fail
- TestWebSocketNotification
- As
- RegisterPingServer
- Format
- importer_test.go
- newRedisTestQueue
- mockNotification
- NewLocalStorage
- admin/module_test.go
- NewGormLogger
- PermissionMiddleware
- loadtest.sh
- jimu
- Audit Logging
- Cache Abstraction
- Feature Flag
- Graceful Shutdown
- i18n Localization
- Management Server
- OAuth Login (Google/GitHub/WeChat)
- HTTP Security Boundary
- Technology Stack
- newTestScheduler
- fakeAuditRepository
- newRepoTestDB
- newHistoryTestDB
- Pagination
- testWorker
- testEnforcer
- storage.go
- TestGeneratedModuleCompiles
- Policy

## God Nodes (most connected - your core abstractions)
1. `T()` - 665 edges
2. `New()` - 85 edges
3. `Now()` - 82 edges
4. `Fail()` - 80 edges
5. `OK()` - 53 edges
6. `User` - 50 edges
7. `Wrap()` - 47 edges
8. `New()` - 38 edges
9. `NewServer()` - 35 edges
10. `JobData` - 33 edges

## Surprising Connections (you probably didn't know these)
- `Principle III: Composable Modules` --semantically_similar_to--> `Module Layering & contract.Module`  [INFERRED] [semantically similar]
  .specify/memory/constitution.md → CLAUDE.md
- `Module Layering & contract.Module` --semantically_similar_to--> `Clean Architecture`  [INFERRED] [semantically similar]
  CLAUDE.md → README.md
- `Uniform Response Format` --semantically_similar_to--> `Unified Response & Pagination`  [INFERRED] [semantically similar]
  CLAUDE.md → README.md
- `Snowflake ID Primary Keys` --semantically_similar_to--> `Snowflake Distributed ID`  [INFERRED] [semantically similar]
  CLAUDE.md → README.md
- `GitHub Flow Branch Strategy` --semantically_similar_to--> `Branch Strategy`  [INFERRED] [semantically similar]
  CLAUDE.md → CONTRIBUTING.md

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **CI Pipeline Stages** — _github_workflows_ci_lint_job, _github_workflows_ci_test_job, _github_workflows_ci_build_job, _github_workflows_ci_docker_job, _github_workflows_ci_security_job, _github_workflows_ci_smoke_job [EXTRACTED 1.00]
- **Observability Stack** — docker_compose_prometheus, docker_compose_grafana, deploy_prometheus_config, deploy_grafana_provisioning_dashboards_jimu_config, readme_observability [INFERRED 0.85]
- **Constitution Core Principles** — _specify_memory_constitution_business_agnostic, _specify_memory_constitution_api_stability, _specify_memory_constitution_composable_modules, _specify_memory_constitution_simplicity, _specify_memory_constitution_verification, _specify_memory_constitution_documentation [EXTRACTED 1.00]
- **Jimu Helm Chart Resource Set** — deploy_helm_chart, deploy_helm_values, deploy_helm_templates_configmap, deploy_helm_templates_deployment, deploy_helm_templates_hpa, deploy_helm_templates_ingress, deploy_helm_templates_networkpolicy, deploy_helm_templates_pdb, deploy_helm_templates_prometheusrule, deploy_helm_templates_service [EXTRACTED 1.00]
- **jimu-server Kubernetes Resource Set** — deploy_k8s_configmap, deploy_k8s_config_files, deploy_k8s_deployment, deploy_k8s_hpa, deploy_k8s_ingress, deploy_k8s_networkpolicy, deploy_k8s_pdb, deploy_k8s_prometheusrule, deploy_k8s_service [EXTRACTED 1.00]
- **Jimu Prometheus Alert Rules** — deploy_k8s_prometheusrule_jimuhigherrorrate, deploy_k8s_prometheusrule_jimuhighlatency, deploy_k8s_prometheusrule_jimupodcrashlooping, deploy_k8s_prometheusrule_jimupoddown, deploy_k8s_prometheusrule_jimudbpoolhigh, deploy_k8s_prometheusrule_jimureadinessfailing [EXTRACTED 1.00]

## Communities (247 total, 45 thin omitted)

### Community 0 - "Role"
Cohesion: 0.21
Nodes (11): fakeRoleRepository, Role, NewRoleService(), Context, roleAppCode(), TestRoleServiceCreateMapsDuplicateNameToConflict(), TestRoleServiceDeleteWrapsRepositoryError(), TestRoleServiceListPassesPagination() (+3 more)

### Community 1 - "ImportResult"
Cohesion: 0.19
Nodes (8): ExcelImporter, ImportError, ImportResult, Context, Reader, NewExcelImporter(), Time, NewImportResult()

### Community 2 - "RegisterAuthRoutes"
Cohesion: 0.29
Nodes (14): RouterGroup, Service, RegisterAuthRoutes(), RegisterCaptchaRoute(), Engine, newRouterLimiter(), testAuthConfig(), TestLoginRateLimitUsesIPScope() (+6 more)

### Community 3 - "Storage"
Cohesion: 0.44
Nodes (9): New(), newOSSStorage(), Config, newMinioStorage(), newS3CompatibleStorage(), newS3Storage(), Config, Storage (+1 more)

### Community 4 - "fakeRedis"
Cohesion: 0.06
Nodes (30): fakePipeline, fakeRedis, redisClientAdapter, RedisSessionStore, sessionPipeline, sessionRedis, SessionStore, BoolCmd (+22 more)

### Community 5 - "auth/apikey.go"
Cohesion: 0.17
Nodes (13): APIKey, APIKeyContextKey, APIKeyStore, APIKeyVerifier, dbAPIKeyStore, APIKeyFromContext(), ContextWithAPIKey(), Context (+5 more)

### Community 6 - "common.sh"
Cohesion: 0.08
Nodes (17): check-prerequisites.sh script, check_dir(), check_file(), get_feature_paths(), get_repo_root(), has_jq(), _persist_feature_json(), resolve_specify_init_dir() (+9 more)

### Community 7 - "Jimu Helm Chart Values"
Cohesion: 0.12
Nodes (35): Grafana Prometheus Datasource, Jimu Helm Chart, Helm ConfigMap Template, Helm Deployment Template, Helm HorizontalPodAutoscaler Template, Helm Ingress Template, Helm NetworkPolicy Template, Helm PodDisruptionBudget Template (+27 more)

### Community 8 - "NewAdminAPIKeyService"
Cohesion: 0.09
Nodes (23): AdminAPIKeyService, CreateKeyInput, APIKey, APIKeyRepository, mysqlAPIKeyRepository, APIKey, Context, NewAdminAPIKeyService() (+15 more)

### Community 9 - "T"
Cohesion: 0.05
Nodes (40): assertQueryParams(), readOpenAPI(), TestOpenAPIIncludesCRUDContract(), TestAPIKeyTableName(), TestHashKey(), TestImportJobTableNameAndStatus(), TestOAuthBindingTableName(), TestEntityValues() (+32 more)

### Community 10 - "Page"
Cohesion: 0.36
Nodes (10): Created(), FailWithDetails(), Context, localeFrom(), NoContent(), Page(), requestID(), StatusForCode() (+2 more)

### Community 11 - "testConn"
Cohesion: 0.29
Nodes (6): Conn, Duration, Mutex, newTestConn(), PresencePayload, testConn

### Community 12 - "generator/module.go"
Cohesion: 0.20
Nodes (20): targetFile, templateData, camel(), GenerateModule(), GenerateModuleAt(), lowerCamel(), mkdirAllTracked(), nextMigrationNumber() (+12 more)

### Community 13 - "JobData"
Cohesion: 0.17
Nodes (10): Context, Duration, Client, Context, Duration, NewRedisQueue(), errorQueue, fakeQueue (+2 more)

### Community 14 - "Permission"
Cohesion: 0.05
Nodes (36): CreatePermissionRequest, fakePermissionRepository, PermissionService, UpdatePermissionRequest, Permission, PermissionRepository, mysqlPermissionRepository, fakePermissionRepository (+28 more)

### Community 15 - "WorkerPool"
Cohesion: 0.11
Nodes (19): CancelFunc, ContextWithTrace(), DefaultTracingConfig(), Context, TracerProvider, InitTracing(), ShutdownTracing(), TraceFromContext() (+11 more)

### Community 16 - "AuthHandler"
Cohesion: 0.26
Nodes (8): AuthHandler, loginRequest, refreshRequest, authContext(), Context, Duration, normalizeUsername(), writeAuthRateLimitHeaders()

### Community 17 - "mysqlRepository"
Cohesion: 0.20
Nodes (6): RoleRepository, rolePermission, Context, DB, mysqlRepository, NewMysqlRepository()

### Community 18 - "Router"
Cohesion: 0.20
Nodes (5): Router, RouterGroup, RegisterAuditRoutes(), RouterGroup, RegisterPermissionRoutes()

### Community 19 - "Hub"
Cohesion: 0.17
Nodes (9): Channel, Context, RWMutex, NewWebSocket(), Connection, Hub, Registration, WebSocket (+1 more)

### Community 20 - "NewSnowflake"
Cohesion: 0.11
Nodes (18): Generator, snowflake, uuidGenerator, BenchmarkSnowflakeNextID(), BenchmarkUUIDNextID(), B, FuzzSnowflakeWorkerID(), F (+10 more)

### Community 21 - "JobRegistry"
Cohesion: 0.12
Nodes (6): fakeAuthzModule, fakeBusinessModule, JobRegistry, EventBus, HandlerFunc, TestBusinessRoutesRequireProtectedMiddleware()

### Community 22 - "AdminUserService"
Cohesion: 0.23
Nodes (8): AdminCreateUserRequest, AdminUpdateUserRequest, AdminUser, AdminUserService, ListUserFilter, userRole, Context, DB

### Community 23 - "Wrap"
Cohesion: 0.23
Nodes (10): AuthService, AuthServiceInterface, TokenPair, ErrAccountLocked(), Context, Duration, invalidCredentials(), normalizeUsername() (+2 more)

### Community 24 - "config/config.go"
Cohesion: 0.14
Nodes (26): CacheConfig, CaptchaResult, EmailConfig, GRPCConfig, HTTPClientConfig, HTTPConfig, IDConfig, LogConfig (+18 more)

### Community 25 - "Worker"
Cohesion: 0.29
Nodes (6): Worker, AuditConfig, Context, RWMutex, NewWorker(), Once

### Community 26 - "RateLimiter"
Cohesion: 0.20
Nodes (16): GlobalRateLimit(), HandlerFunc, Limit, RWMutex, NewRateLimiter(), Engine, Request, ratelimitRequest() (+8 more)

### Community 27 - "RedisCache"
Cohesion: 0.23
Nodes (7): Cache, RedisCache, Client, Context, Duration, NewRedisCache(), randomToken()

### Community 28 - "CronScheduler"
Cohesion: 0.17
Nodes (8): Cron, EntryID, Context, RWMutex, Time, CronScheduler, JobInfo, Store

### Community 29 - "health.go"
Cohesion: 0.14
Nodes (19): Client, Context, DB, Duration, ResponseWriter, NewReadiness(), NewRedisChecker(), NewSQLChecker() (+11 more)

### Community 30 - "AuditLog"
Cohesion: 0.21
Nodes (8): fakeAuditRepository, fakeBatchRepository, AuditLog, fakeQueue, Context, Context, Mutex, Time

### Community 31 - "OK"
Cohesion: 0.16
Nodes (8): AdminAPIKeyHandler, AdminConfigHandler, AdminTaskHandler, Context, Context, Context, NewAdminTaskHandler(), OK()

### Community 32 - "Job"
Cohesion: 0.18
Nodes (8): Job, JobRepository, mysqlJobRepository, fakeJobRepo, Context, DB, NewMysqlJobRepository(), Time

### Community 33 - "WSMessage"
Cohesion: 0.15
Nodes (9): RawMessage, Time, mustDecodeTitle(), ChatPayload, NotificationPayload, PingPayload, PongPayload, SubscribePayload (+1 more)

### Community 34 - "User"
Cohesion: 0.12
Nodes (15): fakeOutboxUserRepo, recordingOutboxStore, User, NewUserService(), appCode(), createOutboxUserService(), fakeUserRepository, Context (+7 more)

### Community 35 - "OAuthService"
Cohesion: 0.27
Nodes (7): OAuthService, BindingRepository, Client, Context, DB, NewOAuthService(), Provider

### Community 36 - "PresenceManager"
Cohesion: 0.18
Nodes (4): RWMutex, Time, Presence, PresenceManager

### Community 37 - "newLock"
Cohesion: 0.18
Nodes (16): RedisConfig, Client, newLock(), TestLock_AcquireRelease(), TestLock_ConcurrentAcquire(), TestLock_Extend(), TestLock_ReleaseOnlyOwnToken(), TestLock_WithLock() (+8 more)

### Community 38 - "InitSnowflake"
Cohesion: 0.14
Nodes (22): snowflakeModel, stringKeyModel, Field, TestIndirect(), TestInitSnowflake_InvalidWorkerID(), TestIsIntegerID(), TestRegisterSnowflakeHook_NilDB(), TestSnowflakeHook_NonIntegerPK() (+14 more)

### Community 39 - "Message"
Cohesion: 0.24
Nodes (7): Dispatcher, Context, Context, Channel, Message, fakeKafkaReader, fakeKafkaWriter

### Community 40 - "NewWithStore"
Cohesion: 0.23
Nodes (14): NewMemoryStore(), TestObserveCountsPanicAsFailure(), TestObserveEmitsSuccessMetrics(), TestSetEnabledNotFound(), TestSetEnabledToggle(), TestTriggerJobNotFound(), TestTriggerJobRunsCommand(), NewWithStore() (+6 more)

### Community 41 - "New"
Cohesion: 0.19
Nodes (12): New(), TestAppError(), TestAppErrorWithCause(), TestFailDoesNotLeakInfrastructureDetails(), TestOKIncludesStableEnvelopeAndRequestID(), TestPageIncludesStableEnvelopeAndPagination(), TestCreatedUsesStandardEnvelope(), TestFailHidesInternalCauseAndIncludesRequestID() (+4 more)

### Community 42 - "Context"
Cohesion: 0.22
Nodes (5): fakeUserRepo, fakeSessionStore, sessionRecord, Context, Duration

### Community 43 - "JWT"
Cohesion: 0.16
Nodes (12): Claims, JWT, FuzzJWTParse(), F, Duration, New(), NewWithRotation(), TestJWTPopulatesTypedClaims() (+4 more)

### Community 44 - "gormLogger"
Cohesion: 0.23
Nodes (9): gormLogger, Context, Time, isSensitiveField(), sanitizeArgs(), sanitizeSQL(), TestIsSensitiveField(), TestSanitizeArgs() (+1 more)

### Community 45 - "DeadLetter"
Cohesion: 0.14
Nodes (9): DeadLetter, DeadLetterRepository, mysqlDeadLetterRepository, fakeDeadLetterRepo, Context, DB, NewMysqlDeadLetterRepository(), Time (+1 more)

### Community 46 - "MySQLStore"
Cohesion: 0.13
Nodes (11): JobHistory, JobHistoryRepository, mysqlJobHistoryRepository, Context, DB, NewMysqlJobHistoryRepository(), Time, Context (+3 more)

### Community 47 - "Manager"
Cohesion: 0.12
Nodes (17): contextKey, Flag, Manager, AdminFeatureHandler, Context, NewAdminFeatureHandler(), TestAdminFeatureHandlerList(), TestAdminFeatureHandlerUpdate() (+9 more)

### Community 48 - "Limiter"
Cohesion: 0.18
Nodes (13): Limiter, Context, Duration, Scripter, limitKey(), NewLimiter(), newTestLimiter(), TestLimiterAllowsOnlyConfiguredWindow() (+5 more)

### Community 49 - "fakeRoleRepository"
Cohesion: 0.23
Nodes (5): fakeRoleRepository, NewRoleHandler(), Context, TestRoleCreateReturnsCreated(), TestRoleDeleteReturnsNoContent()

### Community 50 - "ChannelManager"
Cohesion: 0.13
Nodes (8): RWMutex, NewChannel(), TestNewChannel(), Server, Time, Channel, ChannelManager, wsFixture

### Community 51 - "Application"
Cohesion: 0.13
Nodes (13): Application, fakeComponent, main(), run(), Component, forwardError(), Context, Duration (+5 more)

### Community 52 - "RedisStore"
Cohesion: 0.22
Nodes (9): RedisStore, Service, Client, Context, Duration, NewRedisStore(), NewService(), TestGenerate() (+1 more)

### Community 53 - "Jimu Backend Framework README"
Cohesion: 0.15
Nodes (14): Unreleased Feature Set, Error Code Ranges, Snowflake ID Primary Keys, Uniform Response Format, Contributing Guide, Snowflake Distributed ID, Event Bus, Health Checks (+6 more)

### Community 54 - "upload_handler_test.go"
Cohesion: 0.09
Nodes (38): FileHeader, fakeStorage, UploadConfig, UploadHandler, UploadResponse, Context, HandlerFunc, Reader (+30 more)

### Community 55 - "middleware/middleware_test.go"
Cohesion: 0.15
Nodes (23): CORSMiddleware(), DefaultLogConfig(), HandlerFunc, isAllowedOrigin(), Logger(), Recovery(), RequestID(), shouldLogBody() (+15 more)

### Community 56 - "NewEventBusPublisher"
Cohesion: 0.32
Nodes (6): Context, EventBus, RawMessage, NewEventBusPublisher(), EventBusPublisher, EventPayload

### Community 57 - "ws_integration_test.go"
Cohesion: 0.27
Nodes (26): NewMessage(), connID(), newWSFixture(), TestClientChannels(), TestWSBroadcastAll(), TestWSChatBroadcast(), TestWSChatInvalidPayload(), TestWSConnectRegisters() (+18 more)

### Community 58 - "oauth/application/service_test.go"
Cohesion: 0.22
Nodes (33): oauthStateKey(), dupSubjectProviders(), githubProviders(), Client, DB, Miniredis, newRedisClient(), newRepoResult() (+25 more)

### Community 59 - "AuditLogResponse"
Cohesion: 0.46
Nodes (5): AuditLogResponse, Time, ToAuditLogResponse(), ToAuditLogResponses(), Context

### Community 60 - "Security"
Cohesion: 0.40
Nodes (5): TestSecurityAddVary(), addVary(), Context, HandlerFunc, Security()

### Community 61 - "Logger"
Cohesion: 0.17
Nodes (11): AtomicLevel, Context, LogConfig, New(), TestFileOutput(), TestNewConsoleStdout(), TestNewJSONLevels(), TestSetLevel() (+3 more)

### Community 62 - "mysqlAuditRepository"
Cohesion: 0.36
Nodes (5): mysqlAuditRepository, deserializeChanges(), Context, DB, NewMysqlAuditRepository()

### Community 63 - "Data Model: Jimu 框架持久化实体"
Cohesion: 0.06
Nodes (31): 1. User（用户）, 2. Role（角色）, 3. Permission（权限）, 4. APIKey（API 密钥）, 5. AuditLog（审计日志）, 6. OAuthBinding（第三方绑定）, 7. OutboxEvent（Outbox 事件）, 8. ScheduledJob（定时任务定义） (+23 more)

### Community 64 - "config/config_test.go"
Cohesion: 0.22
Nodes (14): Load(), Config, TestDBPasswordOverride(), TestJWTSecretOverride(), TestLoad(), TestStorageConfigFieldMapping(), TestValidateHTTPMode(), TestValidateLogFormat() (+6 more)

### Community 65 - "NewUserRateLimiter"
Cohesion: 0.19
Nodes (19): defaultKeyFunc(), Client, Context, Duration, HandlerFunc, Time, NewUserRateLimiter(), Engine (+11 more)

### Community 66 - "fakeStorage"
Cohesion: 0.22
Nodes (5): fakeStorage, Context, Duration, ReadCloser, Reader

### Community 67 - "NewCSVExporter"
Cohesion: 0.15
Nodes (12): CSVExporter, ExcelExporter, Context, Writer, NewCSVExporter(), Context, Writer, NewExcelExporter() (+4 more)

### Community 68 - "JobDef"
Cohesion: 0.18
Nodes (8): Context, RWMutex, Context, Time, failStore, JobDef, MemoryStore, Store

### Community 69 - "NewService"
Cohesion: 0.11
Nodes (15): fakeRouter, Service, Handler, Client, Time, NewService(), TestService(), Context (+7 more)

### Community 70 - "NewCleanupService"
Cohesion: 0.14
Nodes (19): CleanupConfig, cleanupModel, CleanupResult, CleanupService, CleanupTable, noTableNameModel, DefaultCleanupConfig(), Context (+11 more)

### Community 71 - "MySQLStore"
Cohesion: 0.17
Nodes (10): Context, DB, DeletedAt, Time, NewMySQLStore(), DB, testDB(), TestMySQLStore() (+2 more)

### Community 72 - "ValidateJSON"
Cohesion: 0.29
Nodes (10): Context, HandlerFunc, localeOf(), TestValidateJSONTranslatesFieldErrors(), translateValidationDetails(), translateValidationMessage(), ValidateJSON(), ValidateQuery() (+2 more)

### Community 73 - "NewMysqlRepository"
Cohesion: 0.22
Nodes (12): TestMysqlRepositoryMySQLIntegration(), NewMysqlRepository(), DB, newTestUser(), newUserTestDB(), TestMysqlRepositoryCreateAndFindByID(), TestMysqlRepositoryFindByUsername(), TestMysqlRepositoryListCountAndPagination() (+4 more)

### Community 74 - "dispatcher"
Cohesion: 0.22
Nodes (8): Channel, Context, dispatcher, RWMutex, NewDispatcher(), TestDispatcherDispatch(), TestDispatcherDispatchBatch(), Notification

### Community 75 - "New"
Cohesion: 0.13
Nodes (14): Client, Queue, New(), TestNew_InvalidType(), TestNew_Redis(), Context, Duration, NewKafkaQueue() (+6 more)

### Community 76 - "RabbitMQQueue"
Cohesion: 0.23
Nodes (7): Context, Delivery, Duration, NewRabbitMQQueue(), RabbitMQChannel, RabbitMQConfig, RabbitMQQueue

### Community 77 - "Lock"
Cohesion: 0.33
Nodes (8): generateToken(), Client, Context, Duration, Time, NewLock(), AcquireResult, Lock

### Community 78 - "Now"
Cohesion: 0.14
Nodes (14): Conn, Context, HandlerFunc, RWMutex, Time, mustEncode(), NewClientHub(), WSHandler() (+6 more)

### Community 79 - "routerLimiterRedis"
Cohesion: 0.29
Nodes (6): routerLimiterRedis, BoolSliceCmd, Cmd, Context, StringCmd, routerLimiterInt()

### Community 80 - "auth/application/service_test.go"
Cohesion: 0.33
Nodes (14): BenchmarkLogin(), benchUser(), B, NewAuthService(), appCode(), newFakeSessionStore(), newTestService(), TestLoginCreatesRefreshSession() (+6 more)

### Community 81 - "i18n.go"
Cohesion: 0.18
Nodes (7): HandlerFunc, Locale(), ParseAcceptLanguage(), TestParseAcceptLanguage(), TestTDefaultsToChinese(), TestTfFormatsArgs(), Tf()

### Community 82 - "Outbox"
Cohesion: 0.09
Nodes (20): Context, DB, NewMySQLStore(), TestMySQLStore_AddWithinTx(), TestMySQLStore_AddWithNilTx(), Context, RawMessage, Time (+12 more)

### Community 83 - ".Consume"
Cohesion: 0.24
Nodes (9): Context, Delivery, TestRabbitMQQueue_ConsumeTimeout(), TestRabbitMQQueue_ConsumeUnmarshalError(), TestRabbitMQQueue_SubmitConsume(), TestRabbitMQQueueImplementsInterfaces(), Publishing, fakeRabbitMQChannel (+1 more)

### Community 84 - "contract/module.go"
Cohesion: 0.29
Nodes (6): ComponentProvider, ErrorSource, EventBus, HTTPMiddlewareProvider, Module, ProtectedHTTPMiddlewareProvider

### Community 85 - ".RegisterHTTP"
Cohesion: 0.13
Nodes (11): Module, AdminAuditHandler, Context, DB, NewAdminAuditHandler(), TestAdminAuditHandlerList(), Client, DB (+3 more)

### Community 86 - "Container"
Cohesion: 0.15
Nodes (14): Container, Client, Config, Context, DB, EventBus, Server, Service (+6 more)

### Community 87 - "AdminTaskService"
Cohesion: 0.29
Nodes (5): AdminTaskService, TaskExecution, TaskInfo, Context, Time

### Community 88 - "SetupRouter"
Cohesion: 0.18
Nodes (15): HandlerFunc, Metrics(), SetupRouter(), freeAddr(), testLogger(), TestNewServerPlain(), TestNewServerTLSInvalidFiles(), TestNewServerTLSValid() (+7 more)

### Community 89 - "S3Storage"
Cohesion: 0.20
Nodes (6): Client, Context, Duration, ReadCloser, Reader, S3Storage

### Community 90 - "LoginFailureTracker"
Cohesion: 0.33
Nodes (8): LockoutConfig, LoginFailureTracker, DefaultLockoutConfig(), Client, Context, Duration, lockKey(), NewLoginFailureTracker()

### Community 91 - "New"
Cohesion: 0.14
Nodes (13): RouterGroup, RegisterOAuthRoutes(), buildProviders(), Client, DB, EventBus, New(), newTestModule() (+5 more)

### Community 92 - "OAuthBinding"
Cohesion: 0.15
Nodes (8): fakeBindingRepo, OAuthBinding, fakeBindingRepo, fakeSessionStore, Time, Context, Context, Duration

### Community 93 - "NewManagementServer"
Cohesion: 0.20
Nodes (8): Handler, passingChecker, Server, HealthRouter(), NewManagementServer(), Context, TestManagementRouterExposure(), TestNewManagementServer()

### Community 94 - "Implementation Plan: Jimu 后端框架能力规格"
Cohesion: 0.07
Nodes (26): Content Quality, Feature Readiness, Notes, Requirement Completeness, Specification Quality Checklist: Jimu 后端框架能力规格, Complexity Tracking, Constitution Check, Documentation (this feature) (+18 more)

### Community 95 - "New"
Cohesion: 0.23
Nodes (10): Module, AuthConfig, CaptchaConfig, Service, NewAuthHandler(), Client, DB, EventBus (+2 more)

### Community 96 - "AuditMiddleware"
Cohesion: 0.24
Nodes (9): Queue, AuditMiddleware(), Context, HandlerFunc, isManagementPath(), optionalString(), optionalUint64(), TestAuditMiddlewareAllowsAnonymousRequest() (+1 more)

### Community 97 - "adminAuthRouter"
Cohesion: 0.27
Nodes (9): AdminAuth(), HandlerFunc, adminAuthRouter(), Engine, TestAdminAuthAllowsAdminRole(), TestAdminAuthAllowsEnglishAdminRole(), TestAdminAuthRejectsNonAdminRole(), TestAdminAuthRequiresAuthentication() (+1 more)

### Community 98 - "responseBodyWriter"
Cohesion: 0.17
Nodes (10): Buffer, ResponseWriter, newResponseBodyWriter(), sanitizeBody(), sanitizeJSONField(), TestResponseBodyWriterCapturesAndTruncates(), TestResponseBodyWriterWriteHeaderPassthrough(), TestSanitizeBody() (+2 more)

### Community 99 - "DefaultCSRFConfig"
Cohesion: 0.18
Nodes (23): CSRF(), DefaultCSRFConfig(), generateToken(), Context, HandlerFunc, isSafeMethod(), setCSRFCookie(), csrfRouter() (+15 more)

### Community 100 - "gorm_logger_test.go"
Cohesion: 0.28
Nodes (12): Buffer, newBufferLogger(), newGormLogger(), TestGormLogger_ErrorRedactsSensitive(), TestGormLogger_InfoRedactsSensitive(), TestGormLogger_LogMode(), TestGormLogger_TraceError(), TestGormLogger_TraceFastQuerySilent() (+4 more)

### Community 101 - "AuditService"
Cohesion: 0.23
Nodes (8): AuditService, AuditRepository, Change, NewAuditService(), serializeChanges(), auditAppCode(), TestAuditServiceGetMapsNotFound(), TestAuditServiceListReturnsDTOAndPassesPagination()

### Community 102 - "CI Workflow"
Cohesion: 0.27
Nodes (11): CI Build Job, CI Docker Build Job, CI Lint Job, CI Security Scan Job, CI Image Smoke Test Job, CI Test Job, CI Workflow, GolangCI Lint Configuration (+3 more)

### Community 103 - "Jimu Constitution"
Cohesion: 0.20
Nodes (10): Principle II: API Stability & SemVer, Principle I: Business-Agnostic, Jimu Constitution, Principle VI: Documentation & Examples, Governance & Revision Process, Principle IV: Simplicity & Minimal Dependencies, Principle V: Verification & Quality, Constitution Template (+2 more)

### Community 104 - "newMockGormDB"
Cohesion: 0.29
Nodes (11): Sqlmock, newMockGormDB(), DB, TestTransaction_Commit(), TestTransaction_Rollback(), TestWithTx_BeginError(), TestWithTx_Commit(), TestWithTx_PanicRollsBack() (+3 more)

### Community 105 - "AdminMonitoringService"
Cohesion: 0.13
Nodes (17): AdminMonitoringService, HealthStatus, MemoryStats, SystemStatus, AdminMonitoringHandler, Client, Context, Time (+9 more)

### Community 106 - ".Validate"
Cohesion: 0.33
Nodes (8): FieldRule, FieldType, ValidationRules, Validator, TestValidateUnique(), checkType(), Context, NewValidator()

### Community 107 - "DBConfig"
Cohesion: 0.23
Nodes (14): DBConfig, configurePool(), ConnectWithRetry(), dsn(), Context, DB, open(), pingDB() (+6 more)

### Community 108 - "Jimu API Swagger Definition"
Cohesion: 0.33
Nodes (10): Jimu API Swagger Definition, Audit Log Endpoints, Auth Flow Endpoints (login/refresh/logout/register), BearerAuth Security (API Key JWT), Captcha Endpoint, contract.ErrorResponse, OAuth Endpoints, contract.PageResponse (+2 more)

### Community 110 - "bindImportFile"
Cohesion: 0.17
Nodes (17): AdminImportHandler, DB, newSqliteDB(), bindImportFile(), Buffer, Context, Format, NewAdminImportHandler() (+9 more)

### Community 111 - "RoleService"
Cohesion: 0.14
Nodes (15): AssignPermissionsRequest, CreateRoleRequest, RoleResponse, RoleService, UpdateRoleRequest, PermissionResponse, Time, TestRoleResponseUsesDTO() (+7 more)

### Community 112 - "Bootstrap"
Cohesion: 0.26
Nodes (14): moduleLogger, registerRouter, Bootstrap(), bridgeFn(), registerEventBusBridge(), registerHTTP(), registerOutboxWorkers(), fakeContainer() (+6 more)

### Community 113 - "Tasks: Jimu 后端框架能力规格"
Cohesion: 0.08
Nodes (24): Dependencies & Execution Order, Format: `[ID] [P?] [Story] Description`, Implementation for User Story 1, Implementation for User Story 2, Implementation for User Story 3, Implementation for User Story 4, Implementation Strategy, Incremental Delivery (+16 more)

### Community 114 - "NewUserHandler"
Cohesion: 0.24
Nodes (7): NewUserHandler(), Context, fakeUserRepository, TestUserCreateReturnsCreatedDTO(), TestUserDeleteReturnsNoContent(), TestUserListRejectsInvalidPaginationContract(), TestUserListUsesDefaultPaginationBeforeService()

### Community 115 - "signature_test.go"
Cohesion: 0.19
Nodes (20): buildSignString(), DefaultSignatureConfig(), Context, Duration, HandlerFunc, hmacSign(), Signature(), SignRequest() (+12 more)

### Community 116 - "NewWebhook"
Cohesion: 0.16
Nodes (15): BenchmarkWebhookSend(), B, Channel, Client, Context, NewWebhook(), signPayload(), TestWebhookNoSignatureWhenNoSecret() (+7 more)

### Community 117 - "NewAdminConfigService"
Cohesion: 0.14
Nodes (20): AdminConfigService, Client, Context, EventBus, NewAdminConfigService(), Miniredis, newConfigTestService(), TestAdminConfigServiceConfigKey() (+12 more)

### Community 118 - "NewChannelManager"
Cohesion: 0.24
Nodes (12): NewChannelManager(), TestChannelManagerGetChannelMissing(), TestChannelManagerGetSubscribers(), TestChannelManagerPreCreatedBroadcast(), TestChannelManagerSubscribeCreatesWithType(), TestChannelManagerSubscribeExisting(), TestChannelManagerUnsubscribe(), TestChannelManagerUnsubscribeAll() (+4 more)

### Community 119 - "NewPresenceManager"
Cohesion: 0.19
Nodes (14): NewPresenceManager(), TestPresenceIsOnline(), TestPresenceIsTyping(), TestPresenceManagerAllPresences(), TestPresenceManagerHeartbeat(), TestPresenceManagerHeartbeatRestoresOnlineStatus(), TestPresenceManagerOfflineMissing(), TestPresenceManagerOnlineCountFiltersOffline() (+6 more)

### Community 120 - "Jimu Collaboration Rules (CLAUDE.md)"
Cohesion: 0.83
Nodes (4): Jimu Collaboration Rules (CLAUDE.md), Development Config (app.yaml), Production Config (app.prod.yaml), Configuration Reference Table

### Community 121 - "UserService"
Cohesion: 0.24
Nodes (10): BatchDeleteRequest, BatchResult, CreateUserRequest, UpdateUserRequest, UserResponse, UserService, Time, ToUserResponse() (+2 more)

### Community 122 - "NewCSVImporter"
Cohesion: 0.24
Nodes (6): CSVImporter, Context, Reader, NewCSVImporter(), FuzzCSVImporterParse(), F

### Community 123 - "统一响应契约"
Cohesion: 0.13
Nodes (13): Admin 模块, Auth 模块, HTTP API 契约, 中间件, 健康与观测（非业务）, 完整接口文档, 分页, 国际化 (+5 more)

### Community 124 - "Speckit SDD Workflow"
Cohesion: 0.40
Nodes (5): Plan Template, Spec Template, Tasks Template, Spec/Plan Review Gates, Speckit SDD Workflow

### Community 125 - "migrate_test.go"
Cohesion: 0.19
Nodes (17): AutoMigrate(), findUp(), DB, isDir(), Migrate(), MigrateWithRetry(), MigrationDir(), runMigration() (+9 more)

### Community 126 - "gzipRouter"
Cohesion: 0.17
Nodes (12): HandlerFunc, ResponseWriter, Writer, GzipCompression(), isAlreadyCompressed(), Engine, gzipRouter(), TestGzipCompressionEncodesBody() (+4 more)

### Community 127 - "NewWeChatProvider"
Cohesion: 0.27
Nodes (6): Client, Config, Context, NewWeChatProvider(), WeChatConfig, WeChatProvider

### Community 128 - "New"
Cohesion: 0.18
Nodes (8): UserHandler, Context, Client, Config, DB, EventBus, New(), Module

### Community 129 - "newWSHandler"
Cohesion: 0.32
Nodes (7): AdminWSHandler, Context, NewAdminWSHandler(), newWSHandler(), TestAdminWSHandlerOnlineUsers(), TestAdminWSHandlerPresence(), TestAdminWSHandlerPush()

### Community 130 - "validator/validator.go"
Cohesion: 0.25
Nodes (8): FieldLevel, FuzzValidateRules(), F, Validate(), validateIDCard(), validateMobile(), validatePassword(), validateUsername()

### Community 131 - "TestDB"
Cohesion: 0.32
Nodes (7): DB, mysqlEnvDBConfig(), mysqlReachable(), NewTestDB(), NewTestDBWithPool(), SkipUnlessMysql(), TestDB

### Community 132 - "New"
Cohesion: 0.38
Nodes (10): ClientConn, New(), dial(), Server, newTestLogger(), startTestServer(), TestAddrBeforeStartIsEmpty(), TestHealthServing() (+2 more)

### Community 133 - "oauth/interfaces/handler_test.go"
Cohesion: 0.26
Nodes (16): OAuthHandler, Context, NewOAuthHandler(), assertCode(), doRequest(), githubProvider(), Engine, newTestHandler() (+8 more)

### Community 134 - "Context"
Cohesion: 0.06
Nodes (33): fakeAPIKeyRepo, fakeEventBus, fakeImportJobRepo, ImportService, ImportJob, ImportJobRepository, UserRepository, mysqlImportJobRepository (+25 more)

### Community 135 - "Email"
Cohesion: 0.29
Nodes (6): buildEmailHeaders(), Channel, Context, TestBuildEmailHeaders(), Email, EmailConfig

### Community 136 - "New"
Cohesion: 0.10
Nodes (35): testRole, fakeUserRepository, int8Ptr(), NewAdminUserService(), TestAdminUserServiceAssignRoles(), TestAdminUserServiceAssignRolesRoleQueryError(), TestAdminUserServiceCreateUser(), TestAdminUserServiceDisableUser() (+27 more)

### Community 137 - "Context"
Cohesion: 0.21
Nodes (9): Duration, refreshTTL(), fakeSessionStore, sessionRecord, Context, Duration, TestRefreshTTLNilExpiry(), TestRefreshTTLPositive() (+1 more)

### Community 138 - "SecurityHeadersFromConfig"
Cohesion: 0.27
Nodes (9): SecurityConfig, DefaultSecurityConfig(), TestSecurityHeadersMiddleware(), TestSecurityHeadersNilValuesSkipped(), TestSecurityHeadersRespectsCustomValues(), Context, HandlerFunc, SecurityHeadersFromConfig() (+1 more)

### Community 139 - "Redis Service"
Cohesion: 0.20
Nodes (12): Development Environment, Adminer Service, MariaDB Service, Redis Service, Server Service, Graphical Captcha, Docker & K8s Deployment, Redis Distributed Lock (+4 more)

### Community 140 - "AuthorizationMiddleware"
Cohesion: 0.14
Nodes (16): AuthorizationStore, DBAuthorizationStore, Enforcer, HandlerFunc, ProtectedMiddleware(), HandlerFunc, DB, Enforcer (+8 more)

### Community 141 - "newMockGormDB"
Cohesion: 0.24
Nodes (14): MySQLBindingRepository, DB, NewMySQLBindingRepository(), bindingRows(), DB, Sqlmock, newMockGormDB(), TestCreateError() (+6 more)

### Community 142 - "SMS"
Cohesion: 0.27
Nodes (7): Channel, NewSMS(), TestSMSAliyunReject(), TestSMSAliyunSend(), TestSMSUnknownProvider(), SMS, SMSConfig

### Community 143 - "Context"
Cohesion: 0.14
Nodes (18): NewMySQLStore(), NewWorkerPool(), RegisterWorker(), fakeStore(), Context, Duration, MySQLStore, newFakeJobRepo() (+10 more)

### Community 144 - "New"
Cohesion: 0.19
Nodes (22): New(), basePermissions(), DB, RunSeed(), RunSeedWithCasbin(), SeedCasbinPolicies(), expectRunSeedQueries(), DB (+14 more)

### Community 145 - "setupTestCache"
Cohesion: 0.52
Nodes (6): setupTestCache(), TestRedisCache_Delete(), TestRedisCache_DeletePattern(), TestRedisCache_GetOrSet(), TestRedisCache_GetOrSetStampede(), TestRedisCache_SetAndGet()

### Community 146 - "EventBus"
Cohesion: 0.19
Nodes (9): EventBus, Handler, RWMutex, New(), TestEventBus_Clear(), TestEventBus_HandlerPanicRecovered(), TestEventBus_MultipleHandlers(), TestEventBus_PublishAsync() (+1 more)

### Community 147 - "Timeout"
Cohesion: 0.22
Nodes (6): Duration, HandlerFunc, TestTimeoutOnlyPropagatesContextDeadline(), Timeout(), RouterGroup, RegisterSwagger()

### Community 148 - "GitHubProvider"
Cohesion: 0.27
Nodes (6): Client, Config, Context, NewGitHubProvider(), GitHubConfig, GitHubProvider

### Community 149 - "UserInfo"
Cohesion: 0.24
Nodes (4): fakeProvider, fakeProvider, Context, UserInfo

### Community 150 - "NewDBAPIKeyStore"
Cohesion: 0.31
Nodes (13): createKey(), APIKey, DB, newTestDB(), TestDBAPIKeyStore_GetByKeyHash(), TestDBAPIKeyStore_GetByKeyHashNotFound(), TestDBAPIKeyStore_UpdateLastUsed(), TestVerifyWithDBStore() (+5 more)

### Community 151 - "Server"
Cohesion: 0.22
Nodes (5): Config, Server, Context, Listener, ServiceDesc

### Community 152 - "Registry"
Cohesion: 0.62
Nodes (4): Exporter, Format, Registry, NewRegistry()

### Community 153 - "Duration"
Cohesion: 0.43
Nodes (4): Duration, ExponentialBackoff, FixedRetry, RetryStrategy

### Community 154 - "Server"
Cohesion: 0.20
Nodes (7): Server, ConfigureTrustedProxies(), formatAddr(), Context, Engine, TestConfigureTrustedProxies(), TestFormatAddr()

### Community 155 - "idempotencyRouter"
Cohesion: 0.36
Nodes (10): Int32, Client, Engine, idempotencyRouter(), newRedisForTest(), TestIdempotencyCachedResponseSerializes(), TestIdempotencyCachesAndReplays(), TestIdempotencyDoesNotCacheFailure() (+2 more)

### Community 156 - "setupMetricsEngine"
Cohesion: 0.60
Nodes (5): gatherHTTPMetric(), Engine, setupMetricsEngine(), TestMetricsLabelsUseRouteTemplate(), TestMetricsRecordsUnmatchedPath()

### Community 157 - "New"
Cohesion: 0.17
Nodes (10): Module, AuditHandler, Context, NewAuditHandler(), TestAuditGetInvalidIDReturnsStableBadRequest(), TestAuditGetReturnsLogDTO(), TestAuditListInvalidQueryReturnsStableBadRequest(), DB (+2 more)

### Community 158 - "NewServer"
Cohesion: 0.38
Nodes (12): NewServer(), New(), TestCircuitHalfOpenOnlyAllowsProbe(), TestCircuitOpensAfterConsecutiveFailures(), TestCircuitRecoversAfterCooldown(), TestDoCancelsDuringRetryWait(), TestDoInjectsTraceParent(), TestDoNoRetryOn4xx() (+4 more)

### Community 159 - "Release Workflow"
Cohesion: 0.40
Nodes (5): Release Workflow, Jimu Changelog, Conventional Commits, Release Note Structure, Commit Style

### Community 160 - "Module Layering & contract.Module"
Cohesion: 0.40
Nodes (5): Principle III: Composable Modules, Module Layering & contract.Module, Module Generator CLI, Clean Architecture, Scaffold CLI

### Community 161 - "AdminUserHandler"
Cohesion: 0.33
Nodes (4): AdminUserHandler, Context, NewAdminUserHandler(), paginationFromQuery()

### Community 162 - "配置契约"
Cohesion: 0.33
Nodes (5): 关键配置组, 加载优先级, 敏感值注入（Docker Secrets）, 枚举约束（非法值启动报错）, 配置契约

### Community 163 - "Module 契约"
Cohesion: 0.33
Nodes (5): Module 契约, 中间件挂载, 分层结构, 模块接口, 组件生命周期

### Community 165 - "events.go"
Cohesion: 0.40
Nodes (4): UserCreatedEvent, UserDeletedEvent, UserLoggedInEvent, UserUpdatedEvent

### Community 166 - "Prometheus Service"
Cohesion: 0.40
Nodes (5): Grafana Dashboard Provider Config, Prometheus Scrape Config, Grafana Service, Prometheus Service, Observability (OTel + Prometheus)

### Community 167 - "CLI 契约"
Cohesion: 0.40
Nodes (4): CLI 契约, 命令表, 种子数据, 迁移命名

### Community 168 - "NewLogChannel"
Cohesion: 0.29
Nodes (6): Channel, Context, NewLogChannel(), TestLogChannelSendBatchNoError(), TestLogChannelSendNoError(), LogChannel

### Community 169 - "circuit"
Cohesion: 0.16
Nodes (15): circuit, circuitState, Client, Config, hostRateLimiter, Context, Duration, Limit (+7 more)

### Community 170 - "Context"
Cohesion: 0.14
Nodes (8): fakeAPIKeyRepo, fakeEventBus, testRole, testUserRole, APIKey, Context, fakeUserRepository, Mutex

### Community 171 - "010_create_jobs.sql"
Cohesion: 0.50
Nodes (3): dead_letters, job_history, jobs

### Community 172 - "Security Policy"
Cohesion: 0.50
Nodes (4): No .env & Secrets Hook, Password Strength Rule, Security Policy, Vulnerability Reporting Process

### Community 174 - "CaptchaHandler"
Cohesion: 0.47
Nodes (4): CaptchaHandler, Context, Service, NewCaptchaHandler()

### Community 177 - "NewEmail"
Cohesion: 0.29
Nodes (8): NewEmail(), Conn, Listener, newFakeSMTPServer(), TestEmailSendEmptyRecipient(), TestEmailSendMissingConfig(), TestEmailSendSMTPIntegration(), fakeSMTPServer

### Community 178 - "securityRouter"
Cohesion: 0.53
Nodes (5): Engine, securityRouter(), TestSecurityHandlesAllowedPreflight(), TestSecurityRejectsOversizedBody(), TestSecurityUsesOriginAllowList()

### Community 183 - "NewMQPublisher"
Cohesion: 0.25
Nodes (8): Context, Queue, RawMessage, NewMQPublisher(), TestMQPublisher_Publish(), TestMQPublisher_PublishError(), traceFromMetadata(), MQPublisher

### Community 210 - "mockTokenServer"
Cohesion: 0.20
Nodes (15): Client, Server, mockClient(), mockTokenServer(), TestGitHubProviderExchange(), TestGoogleProviderExchange(), TestGoogleProviderExchangeBadUserInfo(), TestGoogleProviderExchangeTokenError() (+7 more)

### Community 211 - "mysqlRepository"
Cohesion: 0.31
Nodes (3): Context, DB, mysqlRepository

### Community 212 - "RegisterUserRoutes"
Cohesion: 0.22
Nodes (7): RouterGroup, RegisterUserRoutes(), Client, Duration, HandlerFunc, IdempotencyMiddleware(), cachedResponse

### Community 213 - "Fail"
Cohesion: 0.16
Nodes (9): AdminJobHandler, RoleHandler, Context, Context, DB, EventBus, New(), Fail() (+1 more)

### Community 214 - "TestWebSocketNotification"
Cohesion: 0.33
Nodes (5): NewHub(), TestHubLifecycle(), TestWebSocketNotification(), waitHub(), mockConn

### Community 215 - "As"
Cohesion: 0.36
Nodes (5): AppError, ErrorInfo, AllErrorCodes(), As(), HTTPStatus()

### Community 216 - "RegisterPingServer"
Cohesion: 0.29
Nodes (6): PingServer, pingService, Context, Server, RegisterPingServer(), StringValue

### Community 217 - "Format"
Cohesion: 0.54
Nodes (5): Format, Importer, Registry, NewRegistry(), TestRegistryGetUnsupported()

### Community 218 - "importer_test.go"
Cohesion: 0.60
Nodes (4): csvReader(), Reader, TestCSVParseAndValidate(), TestCSVParseEmptyFile()

### Community 219 - "newRedisTestQueue"
Cohesion: 0.39
Nodes (7): Client, newRedisTestQueue(), TestQueueContract_SubmitConsume(), TestRedisQueueAckRemovesFromProcessing(), TestRedisQueueImplementsInterfaces(), TestRedisQueueNackRequeues(), TestRedisQueueRequeueExpired()

### Community 220 - "mockNotification"
Cohesion: 0.47
Nodes (3): Channel, Context, mockNotification

### Community 221 - "NewLocalStorage"
Cohesion: 0.32
Nodes (6): NewLocalStorage(), TestLocalStorageFilePersisted(), TestLocalStorageUploadDownloadDelete(), TestLocalStorageURL(), TestNewS3RequiresBucket(), TestNewUnsupportedType()

### Community 222 - "admin/module_test.go"
Cohesion: 0.29
Nodes (4): fakeEventBus, TestModuleInitWSIdempotent(), TestModuleNameAndNew(), TestModuleWSHandler()

### Community 223 - "NewGormLogger"
Cohesion: 0.29
Nodes (6): Interface, Duration, NewGormLogger(), TestNewGormLogger_CustomThreshold(), TestNewGormLogger_DefaultThreshold(), LogLevel

### Community 224 - "PermissionMiddleware"
Cohesion: 0.50
Nodes (3): Enforcer, HandlerFunc, PermissionMiddleware()

### Community 236 - "newTestScheduler"
Cohesion: 0.62
Nodes (6): NewAdminTaskService(), newTestScheduler(), TestAdminTaskServiceGetHistory(), TestAdminTaskServiceListTasks(), TestAdminTaskServiceToggleTask(), TestAdminTaskServiceTriggerTask()

### Community 239 - "newRepoTestDB"
Cohesion: 0.53
Nodes (5): DB, newRepoTestDB(), TestMysqlAPIKeyRepository(), TestMysqlImportJobRepository(), TestMysqlJobRepository()

### Community 240 - "newHistoryTestDB"
Cohesion: 0.40
Nodes (4): TestMysqlDeadLetterRepositoryCRUD(), DB, newHistoryTestDB(), TestMysqlJobHistoryRepositoryCreateAndList()

### Community 242 - "testWorker"
Cohesion: 0.53
Nodes (5): Duration, testWorker(), TestWorkerFlushesConfiguredBatch(), TestWorkerRejectsWhenQueueFull(), TestWorkerStopDrainsAcceptedRecords()

### Community 243 - "testEnforcer"
Cohesion: 0.53
Nodes (5): Enforcer, TestAuthorizationMiddlewareAllowsRolePolicy(), TestAuthorizationMiddlewareHidesStoreErrors(), TestAuthorizationMiddlewareRejectsMissingRole(), testEnforcer()

### Community 244 - "storage.go"
Cohesion: 0.33
Nodes (5): Time, FileInfo, Lister, ListOptions, UploadOptions

### Community 245 - "TestGeneratedModuleCompiles"
Cohesion: 0.73
Nodes (5): copyGoSum(), copyRootFile(), TestGeneratedModuleCompiles(), writeFileForTest(), writeStubPackages()

### Community 246 - "Policy"
Cohesion: 0.24
Nodes (6): fakeAuthorizationStore, Policy, fakeAuthzStore, Context, TestProtectedMiddlewareRequiresAccessToken(), Context

## Knowledge Gaps
- **202 isolated node(s):** `common.sh script`, `jimu`, `CaptchaResult`, `LogConfig`, `UserCreatedEvent` (+197 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **45 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `T()` connect `T` to `Role`, `RegisterAuthRoutes`, `NewAdminAPIKeyService`, `Page`, `testConn`, `generator/module.go`, `Permission`, `NewSnowflake`, `JobRegistry`, `RateLimiter`, `health.go`, `WSMessage`, `User`, `newLock`, `InitSnowflake`, `NewWithStore`, `New`, `JWT`, `gormLogger`, `Manager`, `Limiter`, `fakeRoleRepository`, `ChannelManager`, `Application`, `RedisStore`, `upload_handler_test.go`, `middleware/middleware_test.go`, `ws_integration_test.go`, `oauth/application/service_test.go`, `Security`, `Logger`, `config/config_test.go`, `NewUserRateLimiter`, `NewCSVExporter`, `NewService`, `NewCleanupService`, `MySQLStore`, `ValidateJSON`, `NewMysqlRepository`, `dispatcher`, `New`, `auth/application/service_test.go`, `i18n.go`, `Outbox`, `.Consume`, `.RegisterHTTP`, `SetupRouter`, `New`, `NewManagementServer`, `AuditMiddleware`, `adminAuthRouter`, `responseBodyWriter`, `DefaultCSRFConfig`, `gorm_logger_test.go`, `AuditService`, `newMockGormDB`, `AdminMonitoringService`, `.Validate`, `DBConfig`, `bindImportFile`, `RoleService`, `Bootstrap`, `NewUserHandler`, `signature_test.go`, `NewWebhook`, `NewAdminConfigService`, `NewChannelManager`, `NewPresenceManager`, `migrate_test.go`, `gzipRouter`, `newWSHandler`, `TestDB`, `New`, `oauth/interfaces/handler_test.go`, `Context`, `Email`, `New`, `Context`, `SecurityHeadersFromConfig`, `newMockGormDB`, `SMS`, `Context`, `New`, `setupTestCache`, `EventBus`, `Timeout`, `NewDBAPIKeyStore`, `Server`, `idempotencyRouter`, `setupMetricsEngine`, `New`, `NewServer`, `NewLogChannel`, `NewEmail`, `securityRouter`, `NewMQPublisher`, `mockTokenServer`, `TestWebSocketNotification`, `Format`, `importer_test.go`, `newRedisTestQueue`, `NewLocalStorage`, `admin/module_test.go`, `NewGormLogger`, `newTestScheduler`, `newRepoTestDB`, `newHistoryTestDB`, `testWorker`, `testEnforcer`, `TestGeneratedModuleCompiles`, `Policy`?**
  _High betweenness centrality (0.585) - this node is a cross-community bridge._
- **Why does `Now()` connect `Now` to `New`, `ImportResult`, `auth/apikey.go`, `Context`, `NewAdminAPIKeyService`, `New`, `Context`, `T`, `AuthorizationMiddleware`, `newMockGormDB`, `JobData`, `WorkerPool`, `New`, `testConn`, `NewSnowflake`, `NewDBAPIKeyStore`, `RedisCache`, `CronScheduler`, `health.go`, `PresenceManager`, `NewWithStore`, `circuit`, `JWT`, `DeadLetter`, `MySQLStore`, `upload_handler_test.go`, `middleware/middleware_test.go`, `ws_integration_test.go`, `NewUserRateLimiter`, `JobDef`, `NewService`, `NewCleanupService`, `NewMysqlRepository`, `Lock`, `Outbox`, `TestWebSocketNotification`, `SetupRouter`, `newRedisTestQueue`, `AuditMiddleware`, `DefaultCSRFConfig`, `gorm_logger_test.go`, `AdminMonitoringService`, `.Validate`, `signature_test.go`, `NewWebhook`, `NewPresenceManager`?**
  _High betweenness centrality (0.076) - this node is a cross-community bridge._
- **Why does `Fail()` connect `Fail` to `New`, `newWSHandler`, `oauth/interfaces/handler_test.go`, `auth/apikey.go`, `New`, `Page`, `AuthorizationMiddleware`, `Permission`, `AuthHandler`, `RateLimiter`, `New`, `OK`, `AdminUserHandler`, `New`, `JWT`, `CaptchaHandler`, `Manager`, `upload_handler_test.go`, `middleware/middleware_test.go`, `NewUserRateLimiter`, `RegisterUserRoutes`, `.RegisterHTTP`, `PermissionMiddleware`, `adminAuthRouter`, `DefaultCSRFConfig`, `bindImportFile`, `signature_test.go`?**
  _High betweenness centrality (0.043) - this node is a cross-community bridge._
- **Are the 4 inferred relationships involving `T()` (e.g. with `translateValidationDetails()` and `ValidateJSON()`) actually correct?**
  _`T()` has 4 INFERRED edges - model-reasoned connections that need verification._
- **Are the 81 inferred relationships involving `New()` (e.g. with `.CreateKey()` and `.ToggleTask()`) actually correct?**
  _`New()` has 81 INFERRED edges - model-reasoned connections that need verification._
- **Are the 80 inferred relationships involving `Now()` (e.g. with `.CreateKey()` and `.Import()`) actually correct?**
  _`Now()` has 80 INFERRED edges - model-reasoned connections that need verification._
- **Are the 78 inferred relationships involving `Fail()` (e.g. with `.Generate()` and `.HandleDelete()`) actually correct?**
  _`Fail()` has 78 INFERRED edges - model-reasoned connections that need verification._