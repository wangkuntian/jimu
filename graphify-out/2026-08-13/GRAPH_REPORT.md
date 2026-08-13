# Graph Report - jimu  (2026-08-13)

## Corpus Check
- 388 files · ~124,766 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3330 nodes · 7310 edges · 249 communities (198 shown, 51 thin omitted)
- Extraction: 82% EXTRACTED · 18% INFERRED · 0% AMBIGUOUS · INFERRED: 1351 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `30d6ff98`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- Permission
- ImportResult
- New
- LocalStorage
- fakeRedis
- auth/apikey.go
- common.sh
- Jimu Helm Chart Values
- NewAdminAPIKeyService
- T
- New
- message.go
- generator/module.go
- JobData
- PermissionService
- WorkerPool
- AuthHandler
- mysqlAPIKeyRepository
- Module
- Hub
- NewMQPublisher
- JobRegistry
- AdminUserService
- Wrap
- config/config.go
- Worker
- fakeStorage
- RedisCache
- CronScheduler
- health.go
- AuditLog
- OK
- Job
- fakePermissionRepository
- Context
- OAuthService
- PresenceManager
- ConnectWithRetry
- InitSnowflake
- Message
- NewWithStore
- New
- User
- JWT
- gorm_logger_test.go
- DeadLetter
- JobHistory
- Manager
- Fail
- MySQLStore
- NewChannelManager
- Application
- RedisStore
- Jimu Backend Framework README
- upload_handler_test.go
- middleware/middleware_test.go
- NewEventBusPublisher
- ws_integration_test.go
- oauth/application/service_test.go
- AuditLogResponse
- securityRouter
- Logger
- mysqlAuditRepository
- Data Model: Jimu 框架持久化实体
- config/config_test.go
- NewUserRateLimiter
- ImportJob
- Context
- JobDef
- NewService
- NewCleanupService
- MySQLStore
- ValidateJSON
- NewMysqlRepository
- dispatcher
- KafkaQueue
- RabbitMQQueue
- Lock
- ClientHub
- routerLimiterRedis
- auth/application/service_test.go
- i18n.go
- Outbox
- .Consume
- TestDB
- .RegisterHTTP
- NewContainer
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
- ChannelManager
- interfaces/fakes_test.go
- CI Workflow
- Jimu Constitution
- New
- AdminMonitoringService
- Client
- DBConfig
- Jimu API Swagger Definition
- Server
- bindImportFile
- Role
- Bootstrap
- Tasks: Jimu 后端框架能力规格
- newRedisTestQueue
- signature_test.go
- NewWebhook
- NewAdminConfigService
- NewAdminUserService
- Now
- Jimu Collaboration Rules (CLAUDE.md)
- UserService
- CSVImporter
- 统一响应契约
- Speckit SDD Workflow
- migrate_test.go
- gzipRouter
- NewWeChatProvider
- New
- fakeComponent
- validator/validator.go
- fakePermissionRepository
- newWSHandler
- oauth/interfaces/handler_test.go
- Context
- NewEmail
- New
- Context
- SecurityHeadersFromConfig
- Redis Service
- AuthorizationMiddleware
- newMockGormDB
- NewSMS
- queue/worker_test.go
- New
- setupTestCache
- EventBus
- Timeout
- UserInfo
- NewGoogleProvider
- NewDBAPIKeyStore
- newMonitoringHandler
- fakeAuditRepository
- Duration
- NewImportService
- idempotencyRouter
- Metrics
- New
- NewServer
- Release Workflow
- Module Layering & contract.Module
- response.go
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
- testEnforcer
- newMockGormDB
- 003_create_permissions.sql
- openapi.go
- 004_create_user_roles.sql
- ImportService
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
- email_test.go
- Page
- TestWebSocketNotification
- As
- NewAdminJobHandler
- Format
- newTestScheduler
- UserHandler
- mockNotification
- newLock
- NewPathEnforcer
- admin/module_test.go
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
- NewAdminAuditHandler
- fakeRouter
- newRepoTestDB
- AdminAPIKeyHandler
- Pagination
- New
- fakeEventBus
- IdempotencyMiddleware
- TestGeneratedModuleCompiles
- Policy
- BenchmarkLogin
- BenchmarkWebhookSend

## God Nodes (most connected - your core abstractions)
1. `T()` - 649 edges
2. `New()` - 85 edges
3. `Fail()` - 80 edges
4. `Now()` - 80 edges
5. `OK()` - 53 edges
6. `User` - 50 edges
7. `Wrap()` - 47 edges
8. `New()` - 38 edges
9. `JobData` - 33 edges
10. `newService()` - 32 edges

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

## Communities (249 total, 51 thin omitted)

### Community 0 - "Permission"
Cohesion: 0.26
Nodes (6): Permission, mysqlPermissionRepository, Context, DB, NewMysqlPermissionRepository(), Time

### Community 1 - "ImportResult"
Cohesion: 0.14
Nodes (15): ExcelImporter, FieldRule, FieldType, ImportError, ImportResult, ValidationRules, Validator, Context (+7 more)

### Community 2 - "New"
Cohesion: 0.16
Nodes (23): Module, AuthConfig, Service, NewAuthHandler(), RouterGroup, Service, RegisterAuthRoutes(), RegisterCaptchaRoute() (+15 more)

### Community 3 - "LocalStorage"
Cohesion: 0.06
Nodes (37): redisClientAdapter, RedisSessionStore, sessionPipeline, sessionRedis, SessionStore, Client, Context, Duration (+29 more)

### Community 4 - "fakeRedis"
Cohesion: 0.06
Nodes (42): fakePipeline, fakeRedis, Limiter, BoolCmd, Cmder, IntCmd, Context, Duration (+34 more)

### Community 5 - "auth/apikey.go"
Cohesion: 0.18
Nodes (11): APIKey, APIKeyContextKey, dbAPIKeyStore, APIKeyFromContext(), ContextWithAPIKey(), Context, DB, Time (+3 more)

### Community 6 - "common.sh"
Cohesion: 0.08
Nodes (17): check-prerequisites.sh script, check_dir(), check_file(), get_feature_paths(), get_repo_root(), has_jq(), _persist_feature_json(), resolve_specify_init_dir() (+9 more)

### Community 7 - "Jimu Helm Chart Values"
Cohesion: 0.12
Nodes (35): Grafana Prometheus Datasource, Jimu Helm Chart, Helm ConfigMap Template, Helm Deployment Template, Helm HorizontalPodAutoscaler Template, Helm Ingress Template, Helm NetworkPolicy Template, Helm PodDisruptionBudget Template (+27 more)

### Community 8 - "NewAdminAPIKeyService"
Cohesion: 0.12
Nodes (18): AdminAPIKeyService, CreateKeyInput, APIKey, APIKeyRepository, APIKey, Context, NewAdminAPIKeyService(), TestAdminAPIKeyServiceCreateKey() (+10 more)

### Community 9 - "T"
Cohesion: 0.05
Nodes (45): assertQueryParams(), readOpenAPI(), TestOpenAPIIncludesCRUDContract(), TestAPIKeyTableName(), TestHashKey(), TestImportJobTableNameAndStatus(), TestOAuthBindingTableName(), TestEntityValues() (+37 more)

### Community 10 - "New"
Cohesion: 0.16
Nodes (8): RoleHandler, Context, NewRoleHandler(), DB, EventBus, New(), NoContent(), Module

### Community 11 - "message.go"
Cohesion: 0.29
Nodes (6): ChatPayload, NotificationPayload, PingPayload, PongPayload, PresencePayload, SubscribePayload

### Community 12 - "generator/module.go"
Cohesion: 0.20
Nodes (20): targetFile, templateData, camel(), GenerateModule(), GenerateModuleAt(), lowerCamel(), mkdirAllTracked(), nextMigrationNumber() (+12 more)

### Community 13 - "JobData"
Cohesion: 0.17
Nodes (10): Context, Duration, Client, Context, Duration, NewRedisQueue(), errorQueue, fakeQueue (+2 more)

### Community 14 - "PermissionService"
Cohesion: 0.17
Nodes (12): CreatePermissionRequest, PermissionService, UpdatePermissionRequest, PermissionRepository, PermissionResponse, Time, ToPermissionResponse(), ToPermissionResponses() (+4 more)

### Community 15 - "WorkerPool"
Cohesion: 0.11
Nodes (19): CancelFunc, ContextWithTrace(), DefaultTracingConfig(), Context, TracerProvider, InitTracing(), ShutdownTracing(), TraceFromContext() (+11 more)

### Community 16 - "AuthHandler"
Cohesion: 0.26
Nodes (8): AuthHandler, loginRequest, refreshRequest, authContext(), Context, Duration, normalizeUsername(), writeAuthRateLimitHeaders()

### Community 17 - "mysqlAPIKeyRepository"
Cohesion: 0.29
Nodes (5): mysqlAPIKeyRepository, APIKey, Context, DB, NewMysqlAPIKeyRepository()

### Community 18 - "Module"
Cohesion: 0.17
Nodes (5): Module, RouterGroup, RegisterAuditRoutes(), EventBus, HandlerFunc

### Community 19 - "Hub"
Cohesion: 0.17
Nodes (9): Channel, Context, RWMutex, NewWebSocket(), Connection, Hub, Registration, WebSocket (+1 more)

### Community 20 - "NewMQPublisher"
Cohesion: 0.25
Nodes (8): Context, Queue, RawMessage, NewMQPublisher(), TestMQPublisher_Publish(), TestMQPublisher_PublishError(), traceFromMetadata(), MQPublisher

### Community 21 - "JobRegistry"
Cohesion: 0.08
Nodes (13): fakeAuthzModule, fakeBusinessModule, ComponentProvider, ErrorSource, EventBus, HTTPMiddlewareProvider, JobRegistry, Module (+5 more)

### Community 22 - "AdminUserService"
Cohesion: 0.19
Nodes (9): AdminCreateUserRequest, AdminUpdateUserRequest, AdminUser, AdminUserService, ListUserFilter, userRole, UserRepository, Context (+1 more)

### Community 23 - "Wrap"
Cohesion: 0.23
Nodes (10): AuthService, AuthServiceInterface, TokenPair, ErrAccountLocked(), Context, Duration, invalidCredentials(), normalizeUsername() (+2 more)

### Community 24 - "config/config.go"
Cohesion: 0.15
Nodes (23): AuditConfig, CacheConfig, CaptchaConfig, CaptchaResult, EmailConfig, HTTPClientConfig, HTTPConfig, IDConfig (+15 more)

### Community 25 - "Worker"
Cohesion: 0.20
Nodes (11): Worker, AuditRepository, Context, RWMutex, NewWorker(), Duration, testWorker(), TestWorkerFlushesConfiguredBatch() (+3 more)

### Community 26 - "fakeStorage"
Cohesion: 0.22
Nodes (5): fakeStorage, Context, Duration, ReadCloser, Reader

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
Cohesion: 0.15
Nodes (10): fakeAuditRepository, fakeBatchRepository, AuditLog, Change, fakeQueue, serializeChanges(), Context, Context (+2 more)

### Community 31 - "OK"
Cohesion: 0.13
Nodes (9): AdminConfigHandler, AdminTaskHandler, Context, Context, Context, NewAdminTaskHandler(), Context, Context (+1 more)

### Community 32 - "Job"
Cohesion: 0.18
Nodes (8): Job, JobRepository, mysqlJobRepository, fakeJobRepo, Context, DB, NewMysqlJobRepository(), Time

### Community 33 - "fakePermissionRepository"
Cohesion: 0.29
Nodes (7): fakePermissionRepository, Context, permissionAppCode(), TestPermissionServiceCreateMapsDuplicateNameToConflict(), TestPermissionServiceDeleteWrapsRepositoryError(), TestPermissionServiceListPassesPagination(), TestPermissionServiceUpdateMapsNotFound()

### Community 34 - "Context"
Cohesion: 0.14
Nodes (10): fakeOutboxUserRepo, recordingOutboxStore, appCode(), createOutboxUserService(), fakeUserRepository, Context, TestCreateWritesOutbox(), TestUpdateAndDeleteWriteOutbox() (+2 more)

### Community 35 - "OAuthService"
Cohesion: 0.27
Nodes (7): OAuthService, BindingRepository, Client, Context, DB, NewOAuthService(), Provider

### Community 36 - "PresenceManager"
Cohesion: 0.16
Nodes (4): RWMutex, Time, Presence, PresenceManager

### Community 37 - "ConnectWithRetry"
Cohesion: 0.29
Nodes (9): RedisConfig, ConnectWithRetry(), Client, New(), Miniredis, newTestClient(), TestConnectWithRetry_Exhausted(), TestConnectWithRetry_Success() (+1 more)

### Community 38 - "InitSnowflake"
Cohesion: 0.07
Nodes (38): snowflakeModel, stringKeyModel, Field, Generator, snowflake, uuidGenerator, TestIndirect(), TestInitSnowflake_InvalidWorkerID() (+30 more)

### Community 39 - "Message"
Cohesion: 0.19
Nodes (9): Dispatcher, Channel, Context, Context, Channel, Message, SMS, fakeKafkaReader (+1 more)

### Community 40 - "NewWithStore"
Cohesion: 0.27
Nodes (12): NewMemoryStore(), TestSetEnabledNotFound(), TestSetEnabledToggle(), TestTriggerJobNotFound(), TestTriggerJobRunsCommand(), NewWithStore(), newTestLogger(), TestAddNamedFuncPersistsAndRuns() (+4 more)

### Community 41 - "New"
Cohesion: 0.19
Nodes (12): New(), TestAppError(), TestAppErrorWithCause(), TestFailDoesNotLeakInfrastructureDetails(), TestOKIncludesStableEnvelopeAndRequestID(), TestPageIncludesStableEnvelopeAndPagination(), TestCreatedUsesStandardEnvelope(), TestFailHidesInternalCauseAndIncludesRequestID() (+4 more)

### Community 42 - "User"
Cohesion: 0.19
Nodes (7): fakeUserRepo, User, Context, DeletedAt, Time, Context, fakeUserRepository

### Community 43 - "JWT"
Cohesion: 0.20
Nodes (10): Claims, JWT, Duration, New(), NewWithRotation(), TestJWTPopulatesTypedClaims(), TestParseRejectsWrongTokenType(), AuthMiddleware() (+2 more)

### Community 44 - "gorm_logger_test.go"
Cohesion: 0.11
Nodes (27): gormLogger, Interface, Context, Duration, Time, isSensitiveField(), NewGormLogger(), sanitizeArgs() (+19 more)

### Community 45 - "DeadLetter"
Cohesion: 0.10
Nodes (14): DeadLetter, DeadLetterRepository, mysqlDeadLetterRepository, fakeDeadLetterRepo, Context, DB, NewMysqlDeadLetterRepository(), TestMysqlDeadLetterRepositoryCRUD() (+6 more)

### Community 46 - "JobHistory"
Cohesion: 0.19
Nodes (8): JobHistory, JobHistoryRepository, mysqlJobHistoryRepository, Context, DB, NewMysqlJobHistoryRepository(), Time, fakeHistoryRepo

### Community 47 - "Manager"
Cohesion: 0.14
Nodes (16): contextKey, Flag, Manager, AdminFeatureHandler, NewAdminFeatureHandler(), TestAdminFeatureHandlerList(), TestAdminFeatureHandlerUpdate(), FromContext() (+8 more)

### Community 48 - "Fail"
Cohesion: 0.24
Nodes (7): AdminUserHandler, Context, Context, NewAdminUserHandler(), paginationFromQuery(), Context, Fail()

### Community 50 - "NewChannelManager"
Cohesion: 0.26
Nodes (11): NewChannel(), NewChannelManager(), TestChannelManagerGetChannelMissing(), TestChannelManagerGetSubscribers(), TestChannelManagerPreCreatedBroadcast(), TestChannelManagerSubscribeCreatesWithType(), TestChannelManagerSubscribeExisting(), TestChannelManagerUnsubscribe() (+3 more)

### Community 51 - "Application"
Cohesion: 0.17
Nodes (11): Application, main(), run(), Component, forwardError(), Context, Duration, NewApplication() (+3 more)

### Community 52 - "RedisStore"
Cohesion: 0.22
Nodes (9): RedisStore, Service, Client, Context, Duration, NewRedisStore(), NewService(), TestGenerate() (+1 more)

### Community 53 - "Jimu Backend Framework README"
Cohesion: 0.14
Nodes (15): Unreleased Feature Set, Error Code Ranges, Snowflake ID Primary Keys, Uniform Response Format, Contributing Guide, CI/CD & Security Scanning, Snowflake Distributed ID, Event Bus (+7 more)

### Community 54 - "upload_handler_test.go"
Cohesion: 0.09
Nodes (38): FileHeader, fakeStorage, UploadConfig, UploadHandler, UploadResponse, Context, HandlerFunc, Reader (+30 more)

### Community 55 - "middleware/middleware_test.go"
Cohesion: 0.14
Nodes (24): CORSMiddleware(), DefaultLogConfig(), HandlerFunc, isAllowedOrigin(), Logger(), Recovery(), RequestID(), shouldLogBody() (+16 more)

### Community 56 - "NewEventBusPublisher"
Cohesion: 0.32
Nodes (6): Context, EventBus, RawMessage, NewEventBusPublisher(), EventBusPublisher, EventPayload

### Community 57 - "ws_integration_test.go"
Cohesion: 0.17
Nodes (33): NewMessage(), connID(), Conn, Duration, Mutex, Time, mustDecodeTitle(), newTestConn() (+25 more)

### Community 58 - "oauth/application/service_test.go"
Cohesion: 0.24
Nodes (32): oauthStateKey(), dupSubjectProviders(), githubProviders(), Client, DB, Miniredis, newRedisClient(), newRepoResult() (+24 more)

### Community 59 - "AuditLogResponse"
Cohesion: 0.46
Nodes (5): AuditLogResponse, Time, ToAuditLogResponse(), ToAuditLogResponses(), Context

### Community 60 - "securityRouter"
Cohesion: 0.23
Nodes (10): TestSecurityAddVary(), addVary(), Context, HandlerFunc, Security(), Engine, securityRouter(), TestSecurityHandlesAllowedPreflight() (+2 more)

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

### Community 66 - "ImportJob"
Cohesion: 0.15
Nodes (9): fakeImportJobRepo, ImportJob, ImportJobRepository, mysqlImportJobRepository, fakeImportJobRepo, Time, Context, DB (+1 more)

### Community 67 - "Context"
Cohesion: 0.20
Nodes (5): Context, Duration, fakeConsumer, fakeJobRepo, fakeProducer

### Community 68 - "JobDef"
Cohesion: 0.18
Nodes (8): Context, RWMutex, Context, Time, failStore, JobDef, MemoryStore, Store

### Community 69 - "NewService"
Cohesion: 0.17
Nodes (9): Service, Handler, Client, Time, NewService(), TestService(), Service, NewHandler() (+1 more)

### Community 70 - "NewCleanupService"
Cohesion: 0.14
Nodes (19): CleanupConfig, cleanupModel, CleanupResult, CleanupService, CleanupTable, noTableNameModel, DefaultCleanupConfig(), Context (+11 more)

### Community 71 - "MySQLStore"
Cohesion: 0.17
Nodes (10): Context, DB, DeletedAt, Time, NewMySQLStore(), DB, testDB(), TestMySQLStore() (+2 more)

### Community 72 - "ValidateJSON"
Cohesion: 0.12
Nodes (17): RouterGroup, RegisterPermissionRoutes(), RouterGroup, RegisterRoleRoutes(), RouterGroup, RegisterUserRoutes(), Context, HandlerFunc (+9 more)

### Community 73 - "NewMysqlRepository"
Cohesion: 0.22
Nodes (12): TestMysqlRepositoryMySQLIntegration(), NewMysqlRepository(), DB, newTestUser(), newUserTestDB(), TestMysqlRepositoryCreateAndFindByID(), TestMysqlRepositoryFindByUsername(), TestMysqlRepositoryListCountAndPagination() (+4 more)

### Community 74 - "dispatcher"
Cohesion: 0.22
Nodes (8): Channel, Context, dispatcher, RWMutex, NewDispatcher(), TestDispatcherDispatch(), TestDispatcherDispatchBatch(), Notification

### Community 75 - "KafkaQueue"
Cohesion: 0.30
Nodes (5): Context, Duration, KafkaMessageReader, KafkaMessageWriter, KafkaQueue

### Community 76 - "RabbitMQQueue"
Cohesion: 0.23
Nodes (7): Context, Delivery, Duration, NewRabbitMQQueue(), RabbitMQChannel, RabbitMQConfig, RabbitMQQueue

### Community 77 - "Lock"
Cohesion: 0.33
Nodes (8): generateToken(), Client, Context, Duration, Time, NewLock(), AcquireResult, Lock

### Community 78 - "ClientHub"
Cohesion: 0.15
Nodes (8): mustEncode(), BuildUserChannel(), RawMessage, Time, TestBuildUserChannel(), broadcastMsg, ClientHub, WSMessage

### Community 79 - "routerLimiterRedis"
Cohesion: 0.29
Nodes (6): routerLimiterRedis, BoolSliceCmd, Cmd, Context, StringCmd, routerLimiterInt()

### Community 80 - "auth/application/service_test.go"
Cohesion: 0.25
Nodes (14): NewAuthService(), appCode(), fakeSessionStore, sessionRecord, Duration, newFakeSessionStore(), newTestService(), TestLoginCreatesRefreshSession() (+6 more)

### Community 81 - "i18n.go"
Cohesion: 0.29
Nodes (3): HandlerFunc, Locale(), ParseAcceptLanguage()

### Community 82 - "Outbox"
Cohesion: 0.09
Nodes (20): Context, DB, NewMySQLStore(), TestMySQLStore_AddWithinTx(), TestMySQLStore_AddWithNilTx(), Context, RawMessage, Time (+12 more)

### Community 83 - ".Consume"
Cohesion: 0.24
Nodes (9): Context, Delivery, TestRabbitMQQueue_ConsumeTimeout(), TestRabbitMQQueue_ConsumeUnmarshalError(), TestRabbitMQQueue_SubmitConsume(), TestRabbitMQQueueImplementsInterfaces(), Publishing, fakeRabbitMQChannel (+1 more)

### Community 84 - "TestDB"
Cohesion: 0.32
Nodes (7): DB, mysqlEnvDBConfig(), mysqlReachable(), NewTestDB(), NewTestDBWithPool(), SkipUnlessMysql(), TestDB

### Community 85 - ".RegisterHTTP"
Cohesion: 0.23
Nodes (6): Module, Client, DB, EventBus, HandlerFunc, Service

### Community 86 - "NewContainer"
Cohesion: 0.16
Nodes (13): Container, Client, Config, Context, DB, EventBus, Service, TracerProvider (+5 more)

### Community 87 - "AdminTaskService"
Cohesion: 0.29
Nodes (5): AdminTaskService, TaskExecution, TaskInfo, Context, Time

### Community 88 - "SetupRouter"
Cohesion: 0.18
Nodes (16): ConfigureTrustedProxies(), formatAddr(), Engine, SetupRouter(), freeAddr(), TestConfigureTrustedProxies(), TestFormatAddr(), testLogger() (+8 more)

### Community 89 - "S3Storage"
Cohesion: 0.20
Nodes (6): Client, Context, Duration, ReadCloser, Reader, S3Storage

### Community 90 - "LoginFailureTracker"
Cohesion: 0.33
Nodes (8): LockoutConfig, LoginFailureTracker, DefaultLockoutConfig(), Client, Context, Duration, lockKey(), NewLoginFailureTracker()

### Community 91 - "New"
Cohesion: 0.17
Nodes (13): OAuthConfig, OAuthProviderConfig, buildProviders(), Client, DB, EventBus, New(), newTestModule() (+5 more)

### Community 92 - "OAuthBinding"
Cohesion: 0.15
Nodes (8): fakeBindingRepo, OAuthBinding, fakeBindingRepo, fakeSessionStore, Time, Context, Context, Duration

### Community 93 - "NewManagementServer"
Cohesion: 0.22
Nodes (7): Handler, passingChecker, HealthRouter(), NewManagementServer(), Context, TestManagementRouterExposure(), TestNewManagementServer()

### Community 94 - "Implementation Plan: Jimu 后端框架能力规格"
Cohesion: 0.07
Nodes (26): Content Quality, Feature Readiness, Notes, Requirement Completeness, Specification Quality Checklist: Jimu 后端框架能力规格, Complexity Tracking, Constitution Check, Documentation (this feature) (+18 more)

### Community 95 - "New"
Cohesion: 0.22
Nodes (6): PermissionHandler, Context, DB, EventBus, New(), Module

### Community 96 - "AuditMiddleware"
Cohesion: 0.31
Nodes (8): Queue, AuditMiddleware(), Context, HandlerFunc, isManagementPath(), optionalString(), optionalUint64(), TestAuditMiddlewareAllowsAnonymousRequest()

### Community 97 - "adminAuthRouter"
Cohesion: 0.27
Nodes (9): AdminAuth(), HandlerFunc, adminAuthRouter(), Engine, TestAdminAuthAllowsAdminRole(), TestAdminAuthAllowsEnglishAdminRole(), TestAdminAuthRejectsNonAdminRole(), TestAdminAuthRequiresAuthentication() (+1 more)

### Community 98 - "responseBodyWriter"
Cohesion: 0.17
Nodes (10): Buffer, ResponseWriter, newResponseBodyWriter(), sanitizeBody(), sanitizeJSONField(), TestResponseBodyWriterCapturesAndTruncates(), TestResponseBodyWriterWriteHeaderPassthrough(), TestSanitizeBody() (+2 more)

### Community 99 - "DefaultCSRFConfig"
Cohesion: 0.18
Nodes (23): CSRF(), DefaultCSRFConfig(), generateToken(), Context, HandlerFunc, isSafeMethod(), setCSRFCookie(), csrfRouter() (+15 more)

### Community 100 - "ChannelManager"
Cohesion: 0.21
Nodes (3): RWMutex, Channel, ChannelManager

### Community 101 - "interfaces/fakes_test.go"
Cohesion: 0.22
Nodes (4): fakeEventBus, testRole, testUserRole, Mutex

### Community 102 - "CI Workflow"
Cohesion: 0.31
Nodes (10): CI Build Job, CI Docker Build Job, CI Lint Job, CI Security Scan Job, CI Image Smoke Test Job, CI Test Job, CI Workflow, GolangCI Lint Configuration (+2 more)

### Community 103 - "Jimu Constitution"
Cohesion: 0.20
Nodes (10): Principle II: API Stability & SemVer, Principle I: Business-Agnostic, Jimu Constitution, Principle VI: Documentation & Examples, Governance & Revision Process, Principle IV: Simplicity & Minimal Dependencies, Principle V: Verification & Quality, Constitution Template (+2 more)

### Community 104 - "New"
Cohesion: 0.23
Nodes (9): Client, Queue, New(), TestNew_InvalidType(), TestNew_Redis(), NewKafkaQueue(), Config, KafkaConfig (+1 more)

### Community 105 - "AdminMonitoringService"
Cohesion: 0.22
Nodes (10): AdminMonitoringService, HealthStatus, MemoryStats, SystemStatus, Client, Context, Time, NewAdminMonitoringService() (+2 more)

### Community 106 - "Client"
Cohesion: 0.27
Nodes (7): Conn, Context, HandlerFunc, RWMutex, Time, WSHandler(), Client

### Community 107 - "DBConfig"
Cohesion: 0.23
Nodes (14): DBConfig, configurePool(), ConnectWithRetry(), dsn(), Context, DB, open(), pingDB() (+6 more)

### Community 108 - "Jimu API Swagger Definition"
Cohesion: 0.33
Nodes (10): Jimu API Swagger Definition, Audit Log Endpoints, Auth Flow Endpoints (login/refresh/logout/register), BearerAuth Security (API Key JWT), Captcha Endpoint, contract.ErrorResponse, OAuth Endpoints, contract.PageResponse (+2 more)

### Community 110 - "bindImportFile"
Cohesion: 0.18
Nodes (16): AdminImportHandler, DB, newSqliteDB(), bindImportFile(), Buffer, Context, NewAdminImportHandler(), DB (+8 more)

### Community 111 - "Role"
Cohesion: 0.05
Nodes (36): AssignPermissionsRequest, CreateRoleRequest, fakeRoleRepository, RoleResponse, RoleService, UpdateRoleRequest, Role, RoleRepository (+28 more)

### Community 112 - "Bootstrap"
Cohesion: 0.26
Nodes (14): moduleLogger, registerRouter, Bootstrap(), bridgeFn(), registerEventBusBridge(), registerHTTP(), registerOutboxWorkers(), fakeContainer() (+6 more)

### Community 113 - "Tasks: Jimu 后端框架能力规格"
Cohesion: 0.08
Nodes (24): Dependencies & Execution Order, Format: `[ID] [P?] [Story] Description`, Implementation for User Story 1, Implementation for User Story 2, Implementation for User Story 3, Implementation for User Story 4, Implementation Strategy, Incremental Delivery (+16 more)

### Community 114 - "newRedisTestQueue"
Cohesion: 0.39
Nodes (7): Client, newRedisTestQueue(), TestQueueContract_SubmitConsume(), TestRedisQueueAckRemovesFromProcessing(), TestRedisQueueImplementsInterfaces(), TestRedisQueueNackRequeues(), TestRedisQueueRequeueExpired()

### Community 115 - "signature_test.go"
Cohesion: 0.19
Nodes (20): buildSignString(), DefaultSignatureConfig(), Context, Duration, HandlerFunc, hmacSign(), Signature(), SignRequest() (+12 more)

### Community 116 - "NewWebhook"
Cohesion: 0.22
Nodes (10): Channel, Client, Context, NewWebhook(), TestWebhookSendBatch(), TestWebhookSendNilClient(), TestWebhookSendNon2xx(), TestWebhookSendSuccess() (+2 more)

### Community 117 - "NewAdminConfigService"
Cohesion: 0.14
Nodes (20): AdminConfigService, Client, Context, EventBus, NewAdminConfigService(), Miniredis, newConfigTestService(), TestAdminConfigServiceConfigKey() (+12 more)

### Community 118 - "NewAdminUserService"
Cohesion: 0.26
Nodes (10): testRole, int8Ptr(), NewAdminUserService(), TestAdminUserServiceAssignRoles(), TestAdminUserServiceAssignRolesRoleQueryError(), TestAdminUserServiceCreateUser(), TestAdminUserServiceDisableUser(), TestAdminUserServiceGetUser() (+2 more)

### Community 119 - "Now"
Cohesion: 0.15
Nodes (20): NewClientHub(), NewPresenceManager(), TestPresenceIsOnline(), TestPresenceIsTyping(), TestPresenceManagerAllPresences(), TestPresenceManagerHeartbeat(), TestPresenceManagerHeartbeatRestoresOnlineStatus(), TestPresenceManagerOfflineMissing() (+12 more)

### Community 120 - "Jimu Collaboration Rules (CLAUDE.md)"
Cohesion: 0.83
Nodes (4): Jimu Collaboration Rules (CLAUDE.md), Development Config (app.yaml), Production Config (app.prod.yaml), Configuration Reference Table

### Community 121 - "UserService"
Cohesion: 0.22
Nodes (11): BatchDeleteRequest, BatchResult, CreateUserRequest, UpdateUserRequest, UserResponse, UserService, Time, ToUserResponse() (+3 more)

### Community 122 - "CSVImporter"
Cohesion: 0.23
Nodes (9): CSVImporter, Context, Reader, NewCSVImporter(), csvReader(), Reader, TestCSVParseAndValidate(), TestCSVParseEmptyFile() (+1 more)

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
Nodes (12): HandlerFunc, ResponseWriter, GzipCompression(), isAlreadyCompressed(), Engine, gzipRouter(), TestGzipCompressionEncodesBody(), TestGzipCompressionNoAcceptEncoding() (+4 more)

### Community 127 - "NewWeChatProvider"
Cohesion: 0.27
Nodes (6): Client, Config, Context, NewWeChatProvider(), WeChatConfig, WeChatProvider

### Community 128 - "New"
Cohesion: 0.16
Nodes (12): NewUserService(), NewUserHandler(), TestUserCreateReturnsCreatedDTO(), TestUserDeleteReturnsNoContent(), TestUserListRejectsInvalidPaginationContract(), TestUserListUsesDefaultPaginationBeforeService(), Client, Config (+4 more)

### Community 130 - "validator/validator.go"
Cohesion: 0.39
Nodes (6): FieldLevel, Validate(), validateIDCard(), validateMobile(), validatePassword(), validateUsername()

### Community 131 - "fakePermissionRepository"
Cohesion: 0.26
Nodes (5): fakePermissionRepository, NewPermissionHandler(), Context, TestPermissionCreateReturnsCreated(), TestPermissionDeleteReturnsNoContent()

### Community 132 - "newWSHandler"
Cohesion: 0.54
Nodes (6): AdminWSHandler, NewAdminWSHandler(), newWSHandler(), TestAdminWSHandlerOnlineUsers(), TestAdminWSHandlerPresence(), TestAdminWSHandlerPush()

### Community 133 - "oauth/interfaces/handler_test.go"
Cohesion: 0.23
Nodes (17): OAuthHandler, NewOAuthHandler(), assertCode(), doRequest(), githubProvider(), Engine, newTestHandler(), newTestRouter() (+9 more)

### Community 134 - "Context"
Cohesion: 0.25
Nodes (4): fakeAPIKeyRepo, APIKey, fakeUserRepository, Context

### Community 135 - "NewEmail"
Cohesion: 0.29
Nodes (7): buildEmailHeaders(), Channel, Context, NewEmail(), TestBuildEmailHeaders(), Email, EmailConfig

### Community 136 - "New"
Cohesion: 0.18
Nodes (18): fakeUserRepository, AdminAuthMiddleware(), HandlerFunc, TestAdminAuthMiddleware(), newTaskHandler(), TestAdminTaskHandlerHistory(), TestAdminTaskHandlerList(), TestAdminTaskHandlerToggle() (+10 more)

### Community 137 - "Context"
Cohesion: 0.20
Nodes (10): Duration, refreshTTL(), fakeSessionStore, sessionRecord, Context, Duration, TestCreateUserWithBindingTransactionFailure(), TestRefreshTTLNilExpiry() (+2 more)

### Community 138 - "SecurityHeadersFromConfig"
Cohesion: 0.31
Nodes (8): SecurityConfig, DefaultSecurityConfig(), TestSecurityHeadersNilValuesSkipped(), TestSecurityHeadersRespectsCustomValues(), Context, HandlerFunc, SecurityHeadersFromConfig(), writeSecurityHeaders()

### Community 139 - "Redis Service"
Cohesion: 0.20
Nodes (12): Development Environment, Adminer Service, MariaDB Service, Redis Service, Server Service, Graphical Captcha, Docker & K8s Deployment, Redis Distributed Lock (+4 more)

### Community 140 - "AuthorizationMiddleware"
Cohesion: 0.18
Nodes (12): AuthorizationStore, DBAuthorizationStore, Enforcer, HandlerFunc, ProtectedMiddleware(), HandlerFunc, AuthorizationMiddleware(), Context (+4 more)

### Community 141 - "newMockGormDB"
Cohesion: 0.24
Nodes (14): MySQLBindingRepository, DB, NewMySQLBindingRepository(), bindingRows(), DB, Sqlmock, newMockGormDB(), TestCreateError() (+6 more)

### Community 142 - "NewSMS"
Cohesion: 0.43
Nodes (5): NewSMS(), TestSMSAliyunReject(), TestSMSAliyunSend(), TestSMSUnknownProvider(), SMSConfig

### Community 143 - "queue/worker_test.go"
Cohesion: 0.32
Nodes (13): NewMySQLStore(), NewWorkerPool(), RegisterWorker(), fakeStore(), MySQLStore, newFakeJobRepo(), TestWorkerPoolConsumeJob(), TestWorkerPoolConsumeRestoresTrace() (+5 more)

### Community 144 - "New"
Cohesion: 0.19
Nodes (22): New(), basePermissions(), DB, RunSeed(), RunSeedWithCasbin(), SeedCasbinPolicies(), expectRunSeedQueries(), DB (+14 more)

### Community 145 - "setupTestCache"
Cohesion: 0.52
Nodes (6): setupTestCache(), TestRedisCache_Delete(), TestRedisCache_DeletePattern(), TestRedisCache_GetOrSet(), TestRedisCache_GetOrSetStampede(), TestRedisCache_SetAndGet()

### Community 146 - "EventBus"
Cohesion: 0.29
Nodes (3): EventBus, Handler, RWMutex

### Community 147 - "Timeout"
Cohesion: 0.22
Nodes (6): Duration, HandlerFunc, TestTimeoutOnlyPropagatesContextDeadline(), Timeout(), RouterGroup, RegisterSwagger()

### Community 148 - "UserInfo"
Cohesion: 0.14
Nodes (9): fakeProvider, fakeProvider, Client, Config, Context, NewGitHubProvider(), GitHubConfig, GitHubProvider (+1 more)

### Community 149 - "NewGoogleProvider"
Cohesion: 0.27
Nodes (6): Client, Config, Context, NewGoogleProvider(), GoogleConfig, GoogleProvider

### Community 150 - "NewDBAPIKeyStore"
Cohesion: 0.27
Nodes (15): APIKeyStore, APIKeyVerifier, createKey(), APIKey, DB, newTestDB(), TestDBAPIKeyStore_GetByKeyHash(), TestDBAPIKeyStore_GetByKeyHashNotFound() (+7 more)

### Community 151 - "newMonitoringHandler"
Cohesion: 0.27
Nodes (7): AdminMonitoringHandler, Context, NewAdminMonitoringHandler(), newMonitoringHandler(), TestAdminMonitoringHandlerHealth(), TestAdminMonitoringHandlerMetrics(), TestAdminMonitoringHandlerStatus()

### Community 153 - "Duration"
Cohesion: 0.43
Nodes (4): Duration, ExponentialBackoff, FixedRetry, RetryStrategy

### Community 154 - "NewImportService"
Cohesion: 0.32
Nodes (11): DB, newSqliteDB(), DB, NewImportService(), DB, newImportService(), TestImportServiceGetImportJob(), TestImportServiceImport() (+3 more)

### Community 155 - "idempotencyRouter"
Cohesion: 0.36
Nodes (10): Int32, Client, Engine, idempotencyRouter(), newRedisForTest(), TestIdempotencyCachedResponseSerializes(), TestIdempotencyCachesAndReplays(), TestIdempotencyDoesNotCacheFailure() (+2 more)

### Community 156 - "Metrics"
Cohesion: 0.33
Nodes (7): HandlerFunc, Metrics(), gatherHTTPMetric(), Engine, setupMetricsEngine(), TestMetricsLabelsUseRouteTemplate(), TestMetricsRecordsUnmatchedPath()

### Community 157 - "New"
Cohesion: 0.19
Nodes (13): AuditService, AuditHandler, NewAuditService(), auditAppCode(), TestAuditServiceGetMapsNotFound(), TestAuditServiceListReturnsDTOAndPassesPagination(), Context, NewAuditHandler() (+5 more)

### Community 158 - "NewServer"
Cohesion: 0.44
Nodes (10): NewServer(), New(), TestCircuitHalfOpenOnlyAllowsProbe(), TestCircuitOpensAfterConsecutiveFailures(), TestCircuitRecoversAfterCooldown(), TestDoCancelsDuringRetryWait(), TestDoInjectsTraceParent(), TestDoNoRetryOn4xx() (+2 more)

### Community 159 - "Release Workflow"
Cohesion: 0.40
Nodes (5): Release Workflow, Jimu Changelog, Conventional Commits, Release Note Structure, Commit Style

### Community 160 - "Module Layering & contract.Module"
Cohesion: 0.40
Nodes (5): Principle III: Composable Modules, Module Layering & contract.Module, Module Generator CLI, Clean Architecture, Scaffold CLI

### Community 161 - "response.go"
Cohesion: 0.31
Nodes (8): Created(), FailWithDetails(), Context, localeFrom(), requestID(), StatusForCode(), Body, Paginated

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
Cohesion: 0.17
Nodes (12): circuit, circuitState, Client, Config, Context, Duration, Mutex, Request (+4 more)

### Community 170 - "Context"
Cohesion: 0.25
Nodes (4): fakeAPIKeyRepo, APIKey, Context, fakeUserRepository

### Community 171 - "010_create_jobs.sql"
Cohesion: 0.50
Nodes (3): dead_letters, job_history, jobs

### Community 172 - "Security Policy"
Cohesion: 0.50
Nodes (4): No .env & Secrets Hook, Password Strength Rule, Security Policy, Vulnerability Reporting Process

### Community 174 - "CaptchaHandler"
Cohesion: 0.83
Nodes (3): CaptchaHandler, Service, NewCaptchaHandler()

### Community 177 - "testEnforcer"
Cohesion: 0.53
Nodes (5): Enforcer, TestAuthorizationMiddlewareAllowsRolePolicy(), TestAuthorizationMiddlewareHidesStoreErrors(), TestAuthorizationMiddlewareRejectsMissingRole(), testEnforcer()

### Community 178 - "newMockGormDB"
Cohesion: 0.29
Nodes (11): Sqlmock, newMockGormDB(), DB, TestTransaction_Commit(), TestTransaction_Rollback(), TestWithTx_BeginError(), TestWithTx_Commit(), TestWithTx_PanicRollsBack() (+3 more)

### Community 183 - "ImportService"
Cohesion: 0.36
Nodes (5): ImportService, Context, Reader, rulesFor(), TestRulesFor()

### Community 210 - "mockTokenServer"
Cohesion: 0.47
Nodes (9): Client, mockClient(), mockTokenServer(), TestGitHubProviderExchange(), TestGoogleProviderExchange(), TestGoogleProviderExchangeBadUserInfo(), TestGoogleProviderExchangeTokenError(), TestWeChatProviderExchange() (+1 more)

### Community 211 - "mysqlRepository"
Cohesion: 0.31
Nodes (3): Context, DB, mysqlRepository

### Community 212 - "email_test.go"
Cohesion: 0.29
Nodes (7): Conn, newFakeSMTPServer(), TestEmailSendEmptyRecipient(), TestEmailSendMissingConfig(), TestEmailSendSMTPIntegration(), Listener, fakeSMTPServer

### Community 213 - "Page"
Cohesion: 0.42
Nodes (3): AdminJobHandler, Context, Page()

### Community 214 - "TestWebSocketNotification"
Cohesion: 0.33
Nodes (5): NewHub(), TestHubLifecycle(), TestWebSocketNotification(), waitHub(), mockConn

### Community 215 - "As"
Cohesion: 0.27
Nodes (6): AppError, ErrorInfo, Context, AllErrorCodes(), As(), HTTPStatus()

### Community 216 - "NewAdminJobHandler"
Cohesion: 0.36
Nodes (7): NewAdminJobHandler(), TestAdminJobHandlerGet(), TestAdminJobHandlerList(), TestAdminJobHandlerListDeadLetters(), TestAdminJobHandlerResolveDeadLetter(), TestAdminJobHandlerRetry(), TestAdminJobHandlerSubmit()

### Community 217 - "Format"
Cohesion: 0.54
Nodes (5): Format, Importer, Registry, NewRegistry(), TestRegistryGetUnsupported()

### Community 218 - "newTestScheduler"
Cohesion: 0.62
Nodes (6): NewAdminTaskService(), newTestScheduler(), TestAdminTaskServiceGetHistory(), TestAdminTaskServiceListTasks(), TestAdminTaskServiceToggleTask(), TestAdminTaskServiceTriggerTask()

### Community 220 - "mockNotification"
Cohesion: 0.47
Nodes (3): Channel, Context, mockNotification

### Community 221 - "newLock"
Cohesion: 0.43
Nodes (7): Client, newLock(), TestLock_AcquireRelease(), TestLock_ConcurrentAcquire(), TestLock_Extend(), TestLock_ReleaseOnlyOwnToken(), TestLock_WithLock()

### Community 222 - "NewPathEnforcer"
Cohesion: 0.24
Nodes (7): fakeAuthzStore, Context, TestProtectedMiddlewareRequiresAccessToken(), DB, Enforcer, NewEnforcer(), NewPathEnforcer()

### Community 223 - "admin/module_test.go"
Cohesion: 0.29
Nodes (4): fakeEventBus, TestModuleInitWSIdempotent(), TestModuleNameAndNew(), TestModuleWSHandler()

### Community 224 - "PermissionMiddleware"
Cohesion: 0.50
Nodes (3): Enforcer, HandlerFunc, PermissionMiddleware()

### Community 236 - "NewAdminAuditHandler"
Cohesion: 0.40
Nodes (4): AdminAuditHandler, DB, NewAdminAuditHandler(), TestAdminAuditHandlerList()

### Community 238 - "fakeRouter"
Cohesion: 0.38
Nodes (5): fakeRouter, Engine, RouterGroup, newFakeRouter(), TestAdminRoutesUseAPIV1Prefix()

### Community 239 - "newRepoTestDB"
Cohesion: 0.53
Nodes (5): DB, newRepoTestDB(), TestMysqlAPIKeyRepository(), TestMysqlImportJobRepository(), TestMysqlJobRepository()

### Community 242 - "New"
Cohesion: 0.48
Nodes (6): New(), TestEventBus_Clear(), TestEventBus_HandlerPanicRecovered(), TestEventBus_MultipleHandlers(), TestEventBus_PublishAsync(), TestEventBus_SubscribeAndPublish()

### Community 244 - "IdempotencyMiddleware"
Cohesion: 0.33
Nodes (5): Client, Duration, HandlerFunc, IdempotencyMiddleware(), cachedResponse

### Community 245 - "TestGeneratedModuleCompiles"
Cohesion: 0.73
Nodes (5): copyGoSum(), copyRootFile(), TestGeneratedModuleCompiles(), writeFileForTest(), writeStubPackages()

### Community 246 - "Policy"
Cohesion: 0.60
Nodes (3): fakeAuthorizationStore, Policy, Context

### Community 247 - "BenchmarkLogin"
Cohesion: 0.67
Nodes (3): BenchmarkLogin(), benchUser(), B

## Knowledge Gaps
- **202 isolated node(s):** `common.sh script`, `jimu`, `CaptchaResult`, `LogConfig`, `UserCreatedEvent` (+197 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **51 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `T()` connect `T` to `New`, `LocalStorage`, `fakeRedis`, `NewAdminAPIKeyService`, `generator/module.go`, `NewMQPublisher`, `JobRegistry`, `Worker`, `health.go`, `fakePermissionRepository`, `Context`, `ConnectWithRetry`, `InitSnowflake`, `NewWithStore`, `New`, `JWT`, `gorm_logger_test.go`, `DeadLetter`, `Manager`, `NewChannelManager`, `Application`, `RedisStore`, `upload_handler_test.go`, `middleware/middleware_test.go`, `ws_integration_test.go`, `oauth/application/service_test.go`, `securityRouter`, `Logger`, `config/config_test.go`, `NewUserRateLimiter`, `NewService`, `NewCleanupService`, `MySQLStore`, `ValidateJSON`, `NewMysqlRepository`, `dispatcher`, `ClientHub`, `auth/application/service_test.go`, `i18n.go`, `Outbox`, `.Consume`, `TestDB`, `SetupRouter`, `New`, `NewManagementServer`, `AuditMiddleware`, `adminAuthRouter`, `responseBodyWriter`, `DefaultCSRFConfig`, `New`, `AdminMonitoringService`, `DBConfig`, `bindImportFile`, `Role`, `Bootstrap`, `newRedisTestQueue`, `signature_test.go`, `NewWebhook`, `NewAdminConfigService`, `NewAdminUserService`, `Now`, `UserService`, `CSVImporter`, `migrate_test.go`, `gzipRouter`, `New`, `fakePermissionRepository`, `newWSHandler`, `oauth/interfaces/handler_test.go`, `NewEmail`, `New`, `Context`, `SecurityHeadersFromConfig`, `newMockGormDB`, `NewSMS`, `queue/worker_test.go`, `New`, `setupTestCache`, `Timeout`, `NewDBAPIKeyStore`, `newMonitoringHandler`, `NewImportService`, `idempotencyRouter`, `Metrics`, `New`, `NewServer`, `response.go`, `NewLogChannel`, `testEnforcer`, `newMockGormDB`, `ImportService`, `mockTokenServer`, `email_test.go`, `TestWebSocketNotification`, `NewAdminJobHandler`, `Format`, `newTestScheduler`, `newLock`, `NewPathEnforcer`, `admin/module_test.go`, `NewAdminAuditHandler`, `fakeRouter`, `newRepoTestDB`, `New`, `TestGeneratedModuleCompiles`?**
  _High betweenness centrality (0.561) - this node is a cross-community bridge._
- **Why does `Now()` connect `Now` to `ImportResult`, `auth/apikey.go`, `NewAdminAPIKeyService`, `Context`, `T`, `AuthorizationMiddleware`, `newMockGormDB`, `JobData`, `WorkerPool`, `New`, `mysqlAPIKeyRepository`, `NewDBAPIKeyStore`, `RedisCache`, `Metrics`, `health.go`, `CronScheduler`, `PresenceManager`, `InitSnowflake`, `NewWithStore`, `circuit`, `JWT`, `gorm_logger_test.go`, `DeadLetter`, `MySQLStore`, `upload_handler_test.go`, `ImportService`, `middleware/middleware_test.go`, `ws_integration_test.go`, `NewUserRateLimiter`, `JobDef`, `NewService`, `NewCleanupService`, `NewMysqlRepository`, `Lock`, `ClientHub`, `Outbox`, `TestWebSocketNotification`, `UserHandler`, `AuditMiddleware`, `DefaultCSRFConfig`, `AdminMonitoringService`, `Client`, `newRedisTestQueue`, `signature_test.go`, `NewAdminUserService`?**
  _High betweenness centrality (0.073) - this node is a cross-community bridge._
- **Why does `Wrap()` connect `Wrap` to `OAuthService`, `NewAdminAPIKeyService`, `New`, `AuthorizationMiddleware`, `PermissionService`, `Role`, `Page`, `AdminUserService`, `ImportService`, `upload_handler_test.go`, `UserService`, `As`, `AuditLogResponse`?**
  _High betweenness centrality (0.046) - this node is a cross-community bridge._
- **Are the 4 inferred relationships involving `T()` (e.g. with `translateValidationDetails()` and `ValidateJSON()`) actually correct?**
  _`T()` has 4 INFERRED edges - model-reasoned connections that need verification._
- **Are the 81 inferred relationships involving `New()` (e.g. with `.CreateKey()` and `.ToggleTask()`) actually correct?**
  _`New()` has 81 INFERRED edges - model-reasoned connections that need verification._
- **Are the 78 inferred relationships involving `Fail()` (e.g. with `.Generate()` and `.HandleDelete()`) actually correct?**
  _`Fail()` has 78 INFERRED edges - model-reasoned connections that need verification._
- **Are the 78 inferred relationships involving `Now()` (e.g. with `.CreateKey()` and `.Import()`) actually correct?**
  _`Now()` has 78 INFERRED edges - model-reasoned connections that need verification._