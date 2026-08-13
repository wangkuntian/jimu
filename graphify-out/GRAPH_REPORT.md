# Graph Report - jimu  (2026-08-13)

## Corpus Check
- 408 files · ~131,719 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3547 nodes · 7822 edges · 252 communities (204 shown, 48 thin omitted)
- Extraction: 81% EXTRACTED · 19% INFERRED · 0% AMBIGUOUS · INFERRED: 1448 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `1a74737d`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- ImportService
- ImportResult
- New
- Context
- fakeRedis
- auth/apikey.go
- common.sh
- Jimu Helm Chart Values
- NewAdminAPIKeyService
- T
- New
- SessionStore
- generator/module.go
- JobData
- Permission
- WorkerPool
- AuthHandler
- ImportJob
- Context
- Hub
- NewSnowflake
- JobRegistry
- AdminUserService
- Wrap
- config/config.go
- Worker
- New
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
- ConnectWithRetry
- message.go
- Message
- NewWithStore
- New
- Context
- JWT
- gorm_logger_test.go
- DeadLetter
- JobHistory
- Manager
- mysqlPermissionRepository
- snowflake_ext_test.go
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
- securityRouter
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
- ClientHub
- routerLimiterRedis
- newResetService
- i18n.go
- Outbox
- .Consume
- S3Storage
- .RegisterHTTP
- Container
- AdminTaskService
- SetupRouter
- LocalStorage
- LoginFailureTracker
- New
- OAuthBinding
- TestManagementRouterExposure
- Implementation Plan: Jimu 后端框架能力规格
- NewAdminUserService
- AuditMiddleware
- adminAuthRouter
- responseBodyWriter
- DefaultCSRFConfig
- mysqlAPIKeyRepository
- New
- CI Workflow
- Jimu Constitution
- newMockGormDB
- AdminMonitoringService
- .Validate
- DBConfig
- Jimu API Swagger Definition
- newEncryptionTestDB
- bindImportFile
- Role
- fakeContainer
- Tasks: Jimu 后端框架能力规格
- NewUserService
- signature_test.go
- NewWebhook
- NewAdminConfigService
- NewChannelManager
- Now
- Jimu Collaboration Rules (CLAUDE.md)
- UserService
- NewCSVImporter
- 统一响应契约
- Speckit SDD Workflow
- migrate_test.go
- gzipRouter
- NewWeChatProvider
- Fail
- newWSHandler
- validator/validator.go
- TestDB
- fakePermissionRepository
- oauth/interfaces/handler_test.go
- Context
- NewEmail
- New
- Context
- InitSnowflake
- Redis Service
- AuthorizationMiddleware
- newMockGormDB
- SMS
- queue/worker_test.go
- New
- KafkaQueue
- EventBus
- Timeout
- UserInfo
- Page
- NewDBAPIKeyStore
- As
- Registry
- Duration
- Server
- IdempotencyMiddleware
- Metrics
- AuditService
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
- NewHandler
- circuit
- Context
- 010_create_jobs.sql
- Security Policy
- 001_create_users.sql
- CaptchaHandler
- .validateCommon
- 002_create_roles.sql
- AdminJobHandler
- NewAuthHandler
- 003_create_permissions.sql
- openapi.go
- 004_create_user_roles.sql
- RegisterEncryptionHooks
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
- fakeDispatcher
- New
- TestWebSocketNotification
- ExcelImporter
- tracing.go
- Format
- Module
- newRedisTestQueue
- mockNotification
- MySQLStore
- admin/module_test.go
- setupTestCache
- Bootstrap
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
- newHistoryTestDB
- newRepoTestDB
- AdminAPIKeyHandler
- PermissionService
- Pagination
- AdminAuthMiddleware
- PermissionMiddleware
- wsFixture
- NewPathEnforcer
- BenchmarkSnowflakeNextID
- fakeComponent
- Policy
- request.go
- 015_add_user_contact_encrypted.sql

## God Nodes (most connected - your core abstractions)
1. `T()` - 699 edges
2. `New()` - 85 edges
3. `Now()` - 83 edges
4. `Fail()` - 82 edges
5. `User` - 71 edges
6. `OK()` - 55 edges
7. `New()` - 52 edges
8. `Wrap()` - 49 edges
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

## Communities (252 total, 48 thin omitted)

### Community 0 - "ImportService"
Cohesion: 0.17
Nodes (17): ImportService, DB, newSqliteDB(), Context, DB, Format, Reader, NewImportService() (+9 more)

### Community 1 - "ImportResult"
Cohesion: 0.27
Nodes (5): ImportError, ImportResult, Context, Time, NewImportResult()

### Community 2 - "New"
Cohesion: 0.18
Nodes (21): Module, AuthConfig, RouterGroup, Service, RegisterAuthRoutes(), RegisterCaptchaRoute(), Engine, newRouterLimiter() (+13 more)

### Community 3 - "Context"
Cohesion: 0.20
Nodes (5): Context, Duration, fakeConsumer, fakeJobRepo, fakeProducer

### Community 4 - "fakeRedis"
Cohesion: 0.05
Nodes (42): fakePipeline, fakeRedis, Limiter, BoolCmd, Cmder, IntCmd, Context, Duration (+34 more)

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
Cohesion: 0.11
Nodes (20): AdminAPIKeyService, CreateKeyInput, APIKey, APIKeyRepository, APIKey, Context, NewAdminAPIKeyService(), TestAdminAPIKeyServiceCreateKey() (+12 more)

### Community 9 - "T"
Cohesion: 0.06
Nodes (38): assertQueryParams(), readOpenAPI(), TestOpenAPIIncludesCRUDContract(), TestImportJobTableNameAndStatus(), TestOAuthBindingTableName(), TestEntityValues(), TestTableNames(), TestUserFields() (+30 more)

### Community 10 - "New"
Cohesion: 0.14
Nodes (9): PermissionHandler, Context, NewPermissionHandler(), RouterGroup, RegisterPermissionRoutes(), DB, EventBus, New() (+1 more)

### Community 11 - "SessionStore"
Cohesion: 0.22
Nodes (12): redisClientAdapter, RedisSessionStore, sessionPipeline, sessionRedis, SessionStore, Client, Context, Duration (+4 more)

### Community 12 - "generator/module.go"
Cohesion: 0.16
Nodes (25): targetFile, templateData, copyGoSum(), copyRootFile(), TestGeneratedModuleCompiles(), writeFileForTest(), writeStubPackages(), camel() (+17 more)

### Community 13 - "JobData"
Cohesion: 0.17
Nodes (10): Context, Duration, Client, Context, Duration, NewRedisQueue(), errorQueue, fakeQueue (+2 more)

### Community 14 - "Permission"
Cohesion: 0.25
Nodes (10): fakePermissionRepository, Permission, NewPermissionService(), Context, permissionAppCode(), TestPermissionServiceCreateMapsDuplicateNameToConflict(), TestPermissionServiceDeleteWrapsRepositoryError(), TestPermissionServiceListPassesPagination() (+2 more)

### Community 15 - "WorkerPool"
Cohesion: 0.15
Nodes (12): CancelFunc, TraceFromContext(), GetWorker(), Context, Duration, MySQLStore, Consumer, Queue (+4 more)

### Community 16 - "AuthHandler"
Cohesion: 0.33
Nodes (6): AuthHandler, authContext(), Context, Duration, normalizeUsername(), writeAuthRateLimitHeaders()

### Community 17 - "ImportJob"
Cohesion: 0.15
Nodes (9): fakeImportJobRepo, ImportJob, ImportJobRepository, mysqlImportJobRepository, fakeImportJobRepo, Time, Context, DB (+1 more)

### Community 18 - "Context"
Cohesion: 0.17
Nodes (5): handlerNotifier, handlerSessionStore, handlerUserRepo, Context, Duration

### Community 19 - "Hub"
Cohesion: 0.17
Nodes (9): Channel, Context, RWMutex, NewWebSocket(), Connection, Hub, Registration, WebSocket (+1 more)

### Community 20 - "NewSnowflake"
Cohesion: 0.13
Nodes (15): Generator, snowflake, uuidGenerator, FuzzSnowflakeWorkerID(), F, binaryID(), Mutex, NewSnowflake() (+7 more)

### Community 21 - "JobRegistry"
Cohesion: 0.08
Nodes (13): fakeAuthzModule, fakeBusinessModule, ComponentProvider, ErrorSource, EventBus, HTTPMiddlewareProvider, JobRegistry, Module (+5 more)

### Community 22 - "AdminUserService"
Cohesion: 0.21
Nodes (9): AdminCreateUserRequest, AdminUpdateUserRequest, AdminUser, AdminUserService, ListUserFilter, userRole, Context, DB (+1 more)

### Community 23 - "Wrap"
Cohesion: 0.20
Nodes (10): AuthService, AuthServiceInterface, TokenPair, ErrAccountLocked(), Context, Duration, invalidCredentials(), normalizeUsername() (+2 more)

### Community 24 - "config/config.go"
Cohesion: 0.13
Nodes (28): AuditConfig, CacheConfig, CaptchaConfig, CaptchaResult, EmailConfig, GRPCConfig, HTTPClientConfig, HTTPConfig (+20 more)

### Community 25 - "Worker"
Cohesion: 0.20
Nodes (10): Worker, Context, RWMutex, NewWorker(), Duration, testWorker(), TestWorkerFlushesConfiguredBatch(), TestWorkerRejectsWhenQueueFull() (+2 more)

### Community 26 - "New"
Cohesion: 0.20
Nodes (13): Cipher, New(), TestBlindIndexDeterministicAndDistinct(), TestBlindIndexEmpty(), TestBlindIndexWithoutKey(), TestDecryptTamperedFails(), TestDecryptWrongKeyFails(), TestEmptyValuePassthrough() (+5 more)

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
Cohesion: 0.16
Nodes (10): fakeAuditRepository, fakeBatchRepository, AuditLog, fakeAuditRepository, fakeQueue, Context, Context, Mutex (+2 more)

### Community 31 - "OK"
Cohesion: 0.11
Nodes (10): AdminConfigHandler, AdminTaskHandler, AdminWSHandler, Context, Context, Context, NewAdminTaskHandler(), Context (+2 more)

### Community 32 - "Job"
Cohesion: 0.18
Nodes (8): Job, JobRepository, mysqlJobRepository, fakeJobRepo, Context, DB, NewMysqlJobRepository(), Time

### Community 33 - "WSMessage"
Cohesion: 0.31
Nodes (3): RawMessage, Time, WSMessage

### Community 34 - "User"
Cohesion: 0.11
Nodes (15): fakeOutboxUserRepo, recordingOutboxStore, User, appCode(), createOutboxUserService(), fakeUserRepository, Context, TestCreateWritesOutbox() (+7 more)

### Community 35 - "OAuthService"
Cohesion: 0.27
Nodes (7): OAuthService, BindingRepository, Client, Context, DB, NewOAuthService(), Provider

### Community 36 - "PresenceManager"
Cohesion: 0.16
Nodes (4): RWMutex, Time, Presence, PresenceManager

### Community 37 - "ConnectWithRetry"
Cohesion: 0.29
Nodes (9): RedisConfig, ConnectWithRetry(), Client, New(), Miniredis, newTestClient(), TestConnectWithRetry_Exhausted(), TestConnectWithRetry_Success() (+1 more)

### Community 38 - "message.go"
Cohesion: 0.14
Nodes (14): BuildRoomChannel(), BuildUserChannel(), TestBuildRoomChannel(), TestBuildUserChannel(), TestNewMessage(), TestNewMessageMarshalError(), TestWSMessageDecodePayload(), TestWSMessageDecodePayloadError() (+6 more)

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
Cohesion: 0.20
Nodes (5): fakeUserRepo, fakeSessionStore, sessionRecord, Context, Duration

### Community 43 - "JWT"
Cohesion: 0.16
Nodes (12): Claims, JWT, FuzzJWTParse(), F, Duration, New(), NewWithRotation(), TestJWTPopulatesTypedClaims() (+4 more)

### Community 44 - "gorm_logger_test.go"
Cohesion: 0.11
Nodes (26): gormLogger, Interface, Context, Duration, Time, isSensitiveField(), NewGormLogger(), sanitizeArgs() (+18 more)

### Community 45 - "DeadLetter"
Cohesion: 0.17
Nodes (8): DeadLetter, DeadLetterRepository, mysqlDeadLetterRepository, Context, DB, NewMysqlDeadLetterRepository(), Time, fakeDeadRepo

### Community 46 - "JobHistory"
Cohesion: 0.17
Nodes (9): JobHistory, JobHistoryRepository, mysqlJobHistoryRepository, Context, DB, NewMysqlJobHistoryRepository(), Time, Mutex (+1 more)

### Community 47 - "Manager"
Cohesion: 0.14
Nodes (16): contextKey, Flag, Manager, AdminFeatureHandler, NewAdminFeatureHandler(), TestAdminFeatureHandlerList(), TestAdminFeatureHandlerUpdate(), FromContext() (+8 more)

### Community 48 - "mysqlPermissionRepository"
Cohesion: 0.24
Nodes (5): PermissionRepository, mysqlPermissionRepository, Context, DB, NewMysqlPermissionRepository()

### Community 49 - "snowflake_ext_test.go"
Cohesion: 0.20
Nodes (12): stringKeyModel, TestIndirect(), TestInitSnowflake_InvalidWorkerID(), TestIsIntegerID(), TestRegisterSnowflakeHook_NilDB(), TestSnowflakeHook_NonIntegerPK(), DB, Field (+4 more)

### Community 50 - "ChannelManager"
Cohesion: 0.19
Nodes (3): RWMutex, Channel, ChannelManager

### Community 51 - "Application"
Cohesion: 0.17
Nodes (11): Application, main(), run(), Component, forwardError(), Context, Duration, NewApplication() (+3 more)

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
Cohesion: 0.10
Nodes (32): SecurityConfig, DefaultSecurityConfig(), CORSMiddleware(), DefaultLogConfig(), HandlerFunc, isAllowedOrigin(), Logger(), Recovery() (+24 more)

### Community 56 - "NewEventBusPublisher"
Cohesion: 0.32
Nodes (6): Context, EventBus, RawMessage, NewEventBusPublisher(), EventBusPublisher, EventPayload

### Community 57 - "ws_integration_test.go"
Cohesion: 0.18
Nodes (32): NewMessage(), connID(), Conn, Duration, Mutex, mustDecodeTitle(), newTestConn(), newWSFixture() (+24 more)

### Community 58 - "oauth/application/service_test.go"
Cohesion: 0.23
Nodes (33): oauthStateKey(), dupSubjectProviders(), githubProviders(), Client, DB, Miniredis, newRedisClient(), newRepoResult() (+25 more)

### Community 59 - "AuditLogResponse"
Cohesion: 0.46
Nodes (5): AuditLogResponse, Time, ToAuditLogResponse(), ToAuditLogResponses(), Context

### Community 60 - "securityRouter"
Cohesion: 0.23
Nodes (10): TestSecurityAddVary(), addVary(), Context, HandlerFunc, Security(), Engine, securityRouter(), TestSecurityHandlesAllowedPreflight() (+2 more)

### Community 61 - "Logger"
Cohesion: 0.05
Nodes (38): AtomicLevel, ClientConn, Config, PingServer, pingService, Server, Context, Server (+30 more)

### Community 62 - "mysqlAuditRepository"
Cohesion: 0.23
Nodes (8): mysqlAuditRepository, AdminAuditHandler, DB, NewAdminAuditHandler(), deserializeChanges(), Context, DB, NewMysqlAuditRepository()

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
Cohesion: 0.17
Nodes (8): Context, RWMutex, Context, Time, failStore, JobDef, MemoryStore, Store

### Community 69 - "NewService"
Cohesion: 0.16
Nodes (10): fakeRouter, Service, Client, Time, NewService(), TestService(), Engine, RouterGroup (+2 more)

### Community 70 - "NewCleanupService"
Cohesion: 0.14
Nodes (19): CleanupConfig, cleanupModel, CleanupResult, CleanupService, CleanupTable, noTableNameModel, DefaultCleanupConfig(), Context (+11 more)

### Community 71 - "MySQLStore"
Cohesion: 0.17
Nodes (10): Context, DB, DeletedAt, Time, NewMySQLStore(), DB, testDB(), TestMySQLStore() (+2 more)

### Community 72 - "ValidateJSON"
Cohesion: 0.15
Nodes (14): RouterGroup, RegisterRoleRoutes(), RouterGroup, RegisterUserRoutes(), Context, HandlerFunc, localeOf(), TestValidateJSONTranslatesFieldErrors() (+6 more)

### Community 73 - "NewMysqlRepository"
Cohesion: 0.23
Nodes (15): TestMysqlRepositoryMySQLIntegration(), NewMysqlRepository(), DB, newTestUser(), newUserTestDB(), TestMysqlRepositoryCreateAndFindByID(), TestMysqlRepositoryFindByEmailHash(), TestMysqlRepositoryFindByPhoneHash() (+7 more)

### Community 74 - "dispatcher"
Cohesion: 0.18
Nodes (9): Channel, Channel, Context, dispatcher, RWMutex, NewDispatcher(), TestDispatcherDispatch(), TestDispatcherDispatchBatch() (+1 more)

### Community 75 - "New"
Cohesion: 0.24
Nodes (8): Client, Queue, New(), TestNew_InvalidType(), TestNew_Redis(), Config, KafkaConfig, Type

### Community 76 - "RabbitMQQueue"
Cohesion: 0.23
Nodes (7): Context, Delivery, Duration, NewRabbitMQQueue(), RabbitMQChannel, RabbitMQConfig, RabbitMQQueue

### Community 77 - "Lock"
Cohesion: 0.33
Nodes (8): generateToken(), Client, Context, Duration, Time, NewLock(), AcquireResult, Lock

### Community 78 - "ClientHub"
Cohesion: 0.16
Nodes (10): Conn, Context, HandlerFunc, RWMutex, Time, mustEncode(), WSHandler(), broadcastMsg (+2 more)

### Community 79 - "routerLimiterRedis"
Cohesion: 0.29
Nodes (6): routerLimiterRedis, BoolSliceCmd, Cmd, Context, StringCmd, routerLimiterInt()

### Community 80 - "newResetService"
Cohesion: 0.16
Nodes (27): fakeSessionStore, BenchmarkLogin(), benchUser(), B, Client, Miniredis, newResetRedis(), newResetService() (+19 more)

### Community 81 - "i18n.go"
Cohesion: 0.18
Nodes (7): HandlerFunc, Locale(), ParseAcceptLanguage(), TestParseAcceptLanguage(), TestTDefaultsToChinese(), TestTfFormatsArgs(), Tf()

### Community 82 - "Outbox"
Cohesion: 0.07
Nodes (28): Context, Queue, RawMessage, NewMQPublisher(), TestMQPublisher_Publish(), TestMQPublisher_PublishError(), traceFromMetadata(), Context (+20 more)

### Community 83 - ".Consume"
Cohesion: 0.24
Nodes (9): Context, Delivery, TestRabbitMQQueue_ConsumeTimeout(), TestRabbitMQQueue_ConsumeUnmarshalError(), TestRabbitMQQueue_SubmitConsume(), TestRabbitMQQueueImplementsInterfaces(), Publishing, fakeRabbitMQChannel (+1 more)

### Community 84 - "S3Storage"
Cohesion: 0.20
Nodes (6): Client, Context, Duration, ReadCloser, Reader, S3Storage

### Community 85 - ".RegisterHTTP"
Cohesion: 0.23
Nodes (6): Module, Client, DB, EventBus, HandlerFunc, Service

### Community 86 - "Container"
Cohesion: 0.15
Nodes (14): Container, Client, Config, Context, DB, EventBus, Server, Service (+6 more)

### Community 87 - "AdminTaskService"
Cohesion: 0.29
Nodes (5): AdminTaskService, TaskExecution, TaskInfo, Context, Time

### Community 88 - "SetupRouter"
Cohesion: 0.19
Nodes (16): ConfigureTrustedProxies(), Engine, SetupRouter(), freeAddr(), TestConfigureTrustedProxies(), testLogger(), TestNewServerPlain(), TestNewServerTLSInvalidFiles() (+8 more)

### Community 89 - "LocalStorage"
Cohesion: 0.09
Nodes (25): New(), newOSSStorage(), Context, Duration, ReadCloser, Reader, NewLocalStorage(), TestLocalStorageFilePersisted() (+17 more)

### Community 90 - "LoginFailureTracker"
Cohesion: 0.33
Nodes (8): LockoutConfig, LoginFailureTracker, DefaultLockoutConfig(), Client, Context, Duration, lockKey(), NewLoginFailureTracker()

### Community 91 - "New"
Cohesion: 0.19
Nodes (11): buildProviders(), Client, DB, EventBus, New(), newTestModule(), TestBuildProvidersFiltersEnabled(), TestModuleNameAndContract() (+3 more)

### Community 92 - "OAuthBinding"
Cohesion: 0.15
Nodes (8): fakeBindingRepo, OAuthBinding, fakeBindingRepo, fakeSessionStore, Time, Context, Context, Duration

### Community 93 - "TestManagementRouterExposure"
Cohesion: 0.40
Nodes (3): passingChecker, Context, TestManagementRouterExposure()

### Community 94 - "Implementation Plan: Jimu 后端框架能力规格"
Cohesion: 0.07
Nodes (26): Content Quality, Feature Readiness, Notes, Requirement Completeness, Specification Quality Checklist: Jimu 后端框架能力规格, Complexity Tracking, Constitution Check, Documentation (this feature) (+18 more)

### Community 95 - "NewAdminUserService"
Cohesion: 0.29
Nodes (9): testRole, NewAdminUserService(), TestAdminUserServiceAssignRoles(), TestAdminUserServiceAssignRolesRoleQueryError(), TestAdminUserServiceCreateUser(), TestAdminUserServiceDisableUser(), TestAdminUserServiceGetUser(), TestAdminUserServiceListUsers() (+1 more)

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

### Community 100 - "mysqlAPIKeyRepository"
Cohesion: 0.29
Nodes (5): mysqlAPIKeyRepository, APIKey, Context, DB, NewMysqlAPIKeyRepository()

### Community 101 - "New"
Cohesion: 0.14
Nodes (12): Module, AuditHandler, NewAuditHandler(), TestAuditGetInvalidIDReturnsStableBadRequest(), TestAuditGetReturnsLogDTO(), TestAuditListInvalidQueryReturnsStableBadRequest(), RouterGroup, RegisterAuditRoutes() (+4 more)

### Community 102 - "CI Workflow"
Cohesion: 0.27
Nodes (11): CI Build Job, CI Docker Build Job, CI Lint Job, CI Security Scan Job, CI Image Smoke Test Job, CI Test Job, CI Workflow, GolangCI Lint Configuration (+3 more)

### Community 103 - "Jimu Constitution"
Cohesion: 0.20
Nodes (10): Principle II: API Stability & SemVer, Principle I: Business-Agnostic, Jimu Constitution, Principle VI: Documentation & Examples, Governance & Revision Process, Principle IV: Simplicity & Minimal Dependencies, Principle V: Verification & Quality, Constitution Template (+2 more)

### Community 104 - "newMockGormDB"
Cohesion: 0.25
Nodes (13): DB, Sqlmock, newMockGormDB(), TestConfigurePool(), DB, TestTransaction_Commit(), TestTransaction_Rollback(), TestWithTx_BeginError() (+5 more)

### Community 105 - "AdminMonitoringService"
Cohesion: 0.13
Nodes (17): AdminMonitoringService, HealthStatus, MemoryStats, SystemStatus, AdminMonitoringHandler, Client, Context, Time (+9 more)

### Community 106 - ".Validate"
Cohesion: 0.33
Nodes (8): FieldRule, FieldType, ValidationRules, Validator, TestValidateUnique(), checkType(), Context, NewValidator()

### Community 107 - "DBConfig"
Cohesion: 0.25
Nodes (13): DBConfig, configurePool(), ConnectWithRetry(), dsn(), Context, DB, open(), pingDB() (+5 more)

### Community 108 - "Jimu API Swagger Definition"
Cohesion: 0.33
Nodes (10): Jimu API Swagger Definition, Audit Log Endpoints, Auth Flow Endpoints (login/refresh/logout/register), BearerAuth Security (API Key JWT), Captcha Endpoint, contract.ErrorResponse, OAuth Endpoints, contract.PageResponse (+2 more)

### Community 109 - "newEncryptionTestDB"
Cohesion: 0.24
Nodes (9): contact, contactPtr, DB, newEncryptionTestDB(), TestEncryptionHookBatchCreate(), TestEncryptionHookEmptySourceStoresNullAndCoexists(), TestEncryptionHookEncryptsOnWriteDecryptsOnRead(), TestEncryptionHookNoKeyStoresPlaintextButHashes() (+1 more)

### Community 110 - "bindImportFile"
Cohesion: 0.15
Nodes (18): AdminImportHandler, TestAdminAuditHandlerList(), DB, newSqliteDB(), bindImportFile(), Buffer, Context, Format (+10 more)

### Community 111 - "Role"
Cohesion: 0.05
Nodes (36): AssignPermissionsRequest, CreateRoleRequest, fakeRoleRepository, RoleResponse, RoleService, UpdateRoleRequest, Role, RoleRepository (+28 more)

### Community 112 - "fakeContainer"
Cohesion: 0.44
Nodes (8): bridgeFn(), fakeContainer(), newTestLogger(), TestBridgeWorkerConversionFailureErrors(), TestBridgeWorkerPublishesStrongTypeToBareTopic(), TestBridgeWorkerUnknownTypeErrors(), TestEventBusBridgePublishesToBareTopic(), TestRegisterOutboxWorkersRegistersAll()

### Community 113 - "Tasks: Jimu 后端框架能力规格"
Cohesion: 0.08
Nodes (24): Dependencies & Execution Order, Format: `[ID] [P?] [Story] Description`, Implementation for User Story 1, Implementation for User Story 2, Implementation for User Story 3, Implementation for User Story 4, Implementation Strategy, Incremental Delivery (+16 more)

### Community 114 - "NewUserService"
Cohesion: 0.16
Nodes (9): UserRepository, NewUserService(), NewUserHandler(), Context, fakeUserRepository, TestUserCreateReturnsCreatedDTO(), TestUserDeleteReturnsNoContent(), TestUserListRejectsInvalidPaginationContract() (+1 more)

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
Cohesion: 0.27
Nodes (11): NewChannel(), NewChannelManager(), TestChannelManagerGetChannelMissing(), TestChannelManagerGetSubscribers(), TestChannelManagerPreCreatedBroadcast(), TestChannelManagerSubscribeCreatesWithType(), TestChannelManagerSubscribeExisting(), TestChannelManagerUnsubscribe() (+3 more)

### Community 119 - "Now"
Cohesion: 0.15
Nodes (20): NewClientHub(), NewPresenceManager(), TestPresenceIsOnline(), TestPresenceIsTyping(), TestPresenceManagerAllPresences(), TestPresenceManagerHeartbeat(), TestPresenceManagerHeartbeatRestoresOnlineStatus(), TestPresenceManagerOfflineMissing() (+12 more)

### Community 120 - "Jimu Collaboration Rules (CLAUDE.md)"
Cohesion: 0.83
Nodes (4): Jimu Collaboration Rules (CLAUDE.md), Development Config (app.yaml), Production Config (app.prod.yaml), Configuration Reference Table

### Community 121 - "UserService"
Cohesion: 0.24
Nodes (10): BatchDeleteRequest, BatchResult, CreateUserRequest, UpdateUserRequest, UserResponse, UserService, Time, ToUserResponse() (+2 more)

### Community 122 - "NewCSVImporter"
Cohesion: 0.21
Nodes (9): CSVImporter, Reader, NewCSVImporter(), FuzzCSVImporterParse(), F, csvReader(), Reader, TestCSVParseAndValidate() (+1 more)

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

### Community 128 - "Fail"
Cohesion: 0.22
Nodes (8): UserHandler, Context, Context, Context, Config, DB, New(), Fail()

### Community 129 - "newWSHandler"
Cohesion: 0.73
Nodes (5): NewAdminWSHandler(), newWSHandler(), TestAdminWSHandlerOnlineUsers(), TestAdminWSHandlerPresence(), TestAdminWSHandlerPush()

### Community 130 - "validator/validator.go"
Cohesion: 0.25
Nodes (8): FieldLevel, FuzzValidateRules(), F, Validate(), validateIDCard(), validateMobile(), validatePassword(), validateUsername()

### Community 131 - "TestDB"
Cohesion: 0.32
Nodes (7): DB, mysqlEnvDBConfig(), mysqlReachable(), NewTestDB(), NewTestDBWithPool(), SkipUnlessMysql(), TestDB

### Community 132 - "fakePermissionRepository"
Cohesion: 0.31
Nodes (4): fakePermissionRepository, Context, TestPermissionCreateReturnsCreated(), TestPermissionDeleteReturnsNoContent()

### Community 133 - "oauth/interfaces/handler_test.go"
Cohesion: 0.23
Nodes (17): OAuthHandler, NewOAuthHandler(), assertCode(), doRequest(), githubProvider(), Engine, newTestHandler(), newTestRouter() (+9 more)

### Community 134 - "Context"
Cohesion: 0.14
Nodes (6): fakeAPIKeyRepo, fakeEventBus, APIKey, fakeUserRepository, Context, Mutex

### Community 135 - "NewEmail"
Cohesion: 0.16
Nodes (14): buildEmailHeaders(), Channel, Context, NewEmail(), Conn, Listener, newFakeSMTPServer(), TestBuildEmailHeaders() (+6 more)

### Community 136 - "New"
Cohesion: 0.18
Nodes (22): fakeUserRepository, NewAdminJobHandler(), TestAdminJobHandlerGet(), TestAdminJobHandlerList(), TestAdminJobHandlerListDeadLetters(), TestAdminJobHandlerResolveDeadLetter(), TestAdminJobHandlerRetry(), TestAdminJobHandlerSubmit() (+14 more)

### Community 137 - "Context"
Cohesion: 0.22
Nodes (9): Duration, refreshTTL(), fakeSessionStore, sessionRecord, Context, Duration, TestRefreshTTLNilExpiry(), TestRefreshTTLPositive() (+1 more)

### Community 138 - "InitSnowflake"
Cohesion: 0.33
Nodes (10): snowflakeModel, TestSnowflakeHook_PointerSlice(), TestSnowflakeHook_SinglePointerElement(), InitSnowflake(), DB, openTestDB(), TestSnowflakeHook_AssignsID(), TestSnowflakeHook_BatchCreate() (+2 more)

### Community 139 - "Redis Service"
Cohesion: 0.20
Nodes (12): Development Environment, Adminer Service, MariaDB Service, Redis Service, Server Service, Graphical Captcha, Docker & K8s Deployment, Redis Distributed Lock (+4 more)

### Community 140 - "AuthorizationMiddleware"
Cohesion: 0.14
Nodes (17): AuthorizationStore, DBAuthorizationStore, Enforcer, HandlerFunc, ProtectedMiddleware(), HandlerFunc, AuthorizationMiddleware(), Context (+9 more)

### Community 141 - "newMockGormDB"
Cohesion: 0.24
Nodes (14): MySQLBindingRepository, DB, NewMySQLBindingRepository(), bindingRows(), DB, Sqlmock, newMockGormDB(), TestCreateError() (+6 more)

### Community 142 - "SMS"
Cohesion: 0.27
Nodes (7): Channel, NewSMS(), TestSMSAliyunReject(), TestSMSAliyunSend(), TestSMSUnknownProvider(), SMS, SMSConfig

### Community 143 - "queue/worker_test.go"
Cohesion: 0.32
Nodes (13): NewMySQLStore(), NewWorkerPool(), RegisterWorker(), fakeStore(), MySQLStore, newFakeJobRepo(), TestWorkerPoolConsumeJob(), TestWorkerPoolConsumeRestoresTrace() (+5 more)

### Community 144 - "New"
Cohesion: 0.19
Nodes (22): New(), basePermissions(), DB, RunSeed(), RunSeedWithCasbin(), SeedCasbinPolicies(), expectRunSeedQueries(), DB (+14 more)

### Community 145 - "KafkaQueue"
Cohesion: 0.28
Nodes (6): Context, Duration, NewKafkaQueue(), KafkaMessageReader, KafkaMessageWriter, KafkaQueue

### Community 146 - "EventBus"
Cohesion: 0.19
Nodes (9): EventBus, Handler, RWMutex, New(), TestEventBus_Clear(), TestEventBus_HandlerPanicRecovered(), TestEventBus_MultipleHandlers(), TestEventBus_PublishAsync() (+1 more)

### Community 147 - "Timeout"
Cohesion: 0.22
Nodes (6): Duration, HandlerFunc, TestTimeoutOnlyPropagatesContextDeadline(), Timeout(), RouterGroup, RegisterSwagger()

### Community 148 - "UserInfo"
Cohesion: 0.14
Nodes (9): fakeProvider, fakeProvider, Client, Config, Context, NewGitHubProvider(), GitHubConfig, GitHubProvider (+1 more)

### Community 149 - "Page"
Cohesion: 0.36
Nodes (10): Created(), FailWithDetails(), Context, localeFrom(), NoContent(), Page(), requestID(), StatusForCode() (+2 more)

### Community 150 - "NewDBAPIKeyStore"
Cohesion: 0.31
Nodes (13): createKey(), APIKey, DB, newTestDB(), TestDBAPIKeyStore_GetByKeyHash(), TestDBAPIKeyStore_GetByKeyHashNotFound(), TestDBAPIKeyStore_UpdateLastUsed(), TestVerifyWithDBStore() (+5 more)

### Community 151 - "As"
Cohesion: 0.27
Nodes (6): AppError, ErrorInfo, isDuplicateKey(), AllErrorCodes(), As(), HTTPStatus()

### Community 152 - "Registry"
Cohesion: 0.62
Nodes (4): Exporter, Format, Registry, NewRegistry()

### Community 153 - "Duration"
Cohesion: 0.43
Nodes (4): Duration, ExponentialBackoff, FixedRetry, RetryStrategy

### Community 154 - "Server"
Cohesion: 0.29
Nodes (4): Server, formatAddr(), Context, TestFormatAddr()

### Community 155 - "IdempotencyMiddleware"
Cohesion: 0.20
Nodes (15): Int32, Client, Duration, HandlerFunc, IdempotencyMiddleware(), Client, Engine, idempotencyRouter() (+7 more)

### Community 156 - "Metrics"
Cohesion: 0.33
Nodes (7): HandlerFunc, Metrics(), gatherHTTPMetric(), Engine, setupMetricsEngine(), TestMetricsLabelsUseRouteTemplate(), TestMetricsRecordsUnmatchedPath()

### Community 157 - "AuditService"
Cohesion: 0.23
Nodes (8): AuditService, AuditRepository, Change, NewAuditService(), serializeChanges(), auditAppCode(), TestAuditServiceGetMapsNotFound(), TestAuditServiceListReturnsDTOAndPassesPagination()

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
Cohesion: 0.26
Nodes (5): AdminUserHandler, Context, Context, NewAdminUserHandler(), paginationFromQuery()

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

### Community 168 - "NewHandler"
Cohesion: 0.32
Nodes (5): Handler, Context, Service, NewHandler(), TestHandlerGetErrorCodes()

### Community 169 - "circuit"
Cohesion: 0.16
Nodes (15): circuit, circuitState, Client, Config, hostRateLimiter, Context, Duration, Limit (+7 more)

### Community 170 - "Context"
Cohesion: 0.11
Nodes (9): fakeAPIKeyRepo, fakeDeadLetterRepo, fakeEventBus, testRole, testUserRole, APIKey, Context, fakeUserRepository (+1 more)

### Community 171 - "010_create_jobs.sql"
Cohesion: 0.50
Nodes (3): dead_letters, job_history, jobs

### Community 172 - "Security Policy"
Cohesion: 0.50
Nodes (4): No .env & Secrets Hook, Password Strength Rule, Security Policy, Vulnerability Reporting Process

### Community 174 - "CaptchaHandler"
Cohesion: 0.83
Nodes (3): CaptchaHandler, Service, NewCaptchaHandler()

### Community 178 - "NewAuthHandler"
Cohesion: 0.22
Nodes (10): ResetStore, Client, Context, Duration, NewResetStore(), Service, NewAuthHandler(), newHandlerService() (+2 more)

### Community 183 - "RegisterEncryptionHooks"
Cohesion: 0.57
Nodes (7): applyBlindIndexFields(), applyEncryptedFields(), DB, Field, Value, RegisterEncryptionHooks(), walkElements()

### Community 210 - "mockTokenServer"
Cohesion: 0.18
Nodes (16): Client, Server, mockClient(), mockTokenServer(), TestGitHubProviderExchange(), TestGoogleProviderExchange(), TestGoogleProviderExchangeBadUserInfo(), TestGoogleProviderExchangeTokenError() (+8 more)

### Community 211 - "mysqlRepository"
Cohesion: 0.27
Nodes (3): Context, DB, mysqlRepository

### Community 212 - "fakeDispatcher"
Cohesion: 0.33
Nodes (4): fakeDispatcher, Channel, Context, Mutex

### Community 213 - "New"
Cohesion: 0.21
Nodes (6): RoleHandler, Context, DB, EventBus, New(), Module

### Community 214 - "TestWebSocketNotification"
Cohesion: 0.33
Nodes (5): NewHub(), TestHubLifecycle(), TestWebSocketNotification(), waitHub(), mockConn

### Community 215 - "ExcelImporter"
Cohesion: 0.38
Nodes (4): ExcelImporter, Context, Reader, NewExcelImporter()

### Community 216 - "tracing.go"
Cohesion: 0.43
Nodes (7): ContextWithTrace(), DefaultTracingConfig(), Context, TracerProvider, InitTracing(), ShutdownTracing(), TracingConfig

### Community 217 - "Format"
Cohesion: 0.54
Nodes (5): Format, Importer, Registry, NewRegistry(), TestRegistryGetUnsupported()

### Community 218 - "Module"
Cohesion: 0.29
Nodes (3): Client, EventBus, Module

### Community 219 - "newRedisTestQueue"
Cohesion: 0.39
Nodes (7): Client, newRedisTestQueue(), TestQueueContract_SubmitConsume(), TestRedisQueueAckRemovesFromProcessing(), TestRedisQueueImplementsInterfaces(), TestRedisQueueNackRequeues(), TestRedisQueueRequeueExpired()

### Community 220 - "mockNotification"
Cohesion: 0.47
Nodes (3): Channel, Context, mockNotification

### Community 222 - "admin/module_test.go"
Cohesion: 0.29
Nodes (4): fakeEventBus, TestModuleInitWSIdempotent(), TestModuleNameAndNew(), TestModuleWSHandler()

### Community 223 - "setupTestCache"
Cohesion: 0.52
Nodes (6): setupTestCache(), TestRedisCache_Delete(), TestRedisCache_DeletePattern(), TestRedisCache_GetOrSet(), TestRedisCache_GetOrSetStampede(), TestRedisCache_SetAndGet()

### Community 224 - "Bootstrap"
Cohesion: 0.24
Nodes (11): moduleLogger, registerRouter, Handler, Bootstrap(), registerEventBusBridge(), registerHTTP(), registerOutboxWorkers(), Server (+3 more)

### Community 236 - "newTestScheduler"
Cohesion: 0.62
Nodes (6): NewAdminTaskService(), newTestScheduler(), TestAdminTaskServiceGetHistory(), TestAdminTaskServiceListTasks(), TestAdminTaskServiceToggleTask(), TestAdminTaskServiceTriggerTask()

### Community 238 - "newHistoryTestDB"
Cohesion: 0.40
Nodes (4): TestMysqlDeadLetterRepositoryCRUD(), DB, newHistoryTestDB(), TestMysqlJobHistoryRepositoryCreateAndList()

### Community 239 - "newRepoTestDB"
Cohesion: 0.53
Nodes (5): DB, newRepoTestDB(), TestMysqlAPIKeyRepository(), TestMysqlImportJobRepository(), TestMysqlJobRepository()

### Community 241 - "PermissionService"
Cohesion: 0.20
Nodes (10): CreatePermissionRequest, PermissionService, UpdatePermissionRequest, PermissionResponse, Time, ToPermissionResponse(), ToPermissionResponses(), isDuplicateKey() (+2 more)

### Community 243 - "AdminAuthMiddleware"
Cohesion: 0.40
Nodes (3): AdminAuthMiddleware(), HandlerFunc, TestAdminAuthMiddleware()

### Community 244 - "PermissionMiddleware"
Cohesion: 0.50
Nodes (3): Enforcer, HandlerFunc, PermissionMiddleware()

### Community 245 - "wsFixture"
Cohesion: 0.50
Nodes (3): Server, Time, wsFixture

### Community 246 - "NewPathEnforcer"
Cohesion: 0.24
Nodes (7): fakeAuthzStore, Context, TestProtectedMiddlewareRequiresAccessToken(), DB, Enforcer, NewEnforcer(), NewPathEnforcer()

### Community 247 - "BenchmarkSnowflakeNextID"
Cohesion: 0.67
Nodes (3): BenchmarkSnowflakeNextID(), BenchmarkUUIDNextID(), B

### Community 251 - "Policy"
Cohesion: 0.60
Nodes (3): fakeAuthorizationStore, Policy, Context

### Community 252 - "request.go"
Cohesion: 0.40
Nodes (4): forgotPasswordRequest, loginRequest, refreshRequest, resetPasswordRequest

## Knowledge Gaps
- **205 isolated node(s):** `common.sh script`, `jimu`, `CaptchaResult`, `LogConfig`, `UserCreatedEvent` (+200 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **48 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `T()` connect `T` to `ImportService`, `New`, `fakeRedis`, `NewAdminAPIKeyService`, `generator/module.go`, `Permission`, `NewSnowflake`, `JobRegistry`, `Worker`, `New`, `health.go`, `User`, `ConnectWithRetry`, `message.go`, `NewWithStore`, `New`, `JWT`, `gorm_logger_test.go`, `Manager`, `snowflake_ext_test.go`, `Application`, `RedisStore`, `upload_handler_test.go`, `middleware/middleware_test.go`, `ws_integration_test.go`, `oauth/application/service_test.go`, `securityRouter`, `Logger`, `config/config_test.go`, `NewUserRateLimiter`, `NewCSVExporter`, `NewService`, `NewCleanupService`, `MySQLStore`, `ValidateJSON`, `NewMysqlRepository`, `dispatcher`, `New`, `newResetService`, `i18n.go`, `Outbox`, `.Consume`, `SetupRouter`, `LocalStorage`, `New`, `TestManagementRouterExposure`, `NewAdminUserService`, `AuditMiddleware`, `adminAuthRouter`, `responseBodyWriter`, `DefaultCSRFConfig`, `New`, `newMockGormDB`, `AdminMonitoringService`, `.Validate`, `DBConfig`, `newEncryptionTestDB`, `bindImportFile`, `Role`, `fakeContainer`, `NewUserService`, `signature_test.go`, `NewWebhook`, `NewAdminConfigService`, `NewChannelManager`, `Now`, `NewCSVImporter`, `migrate_test.go`, `gzipRouter`, `newWSHandler`, `TestDB`, `fakePermissionRepository`, `oauth/interfaces/handler_test.go`, `NewEmail`, `New`, `Context`, `InitSnowflake`, `AuthorizationMiddleware`, `newMockGormDB`, `SMS`, `queue/worker_test.go`, `New`, `EventBus`, `Timeout`, `Page`, `NewDBAPIKeyStore`, `Server`, `IdempotencyMiddleware`, `Metrics`, `AuditService`, `NewServer`, `NewHandler`, `NewAuthHandler`, `mockTokenServer`, `TestWebSocketNotification`, `Format`, `newRedisTestQueue`, `admin/module_test.go`, `setupTestCache`, `Bootstrap`, `newTestScheduler`, `newHistoryTestDB`, `newRepoTestDB`, `AdminAuthMiddleware`, `NewPathEnforcer`?**
  _High betweenness centrality (0.568) - this node is a cross-community bridge._
- **Why does `Now()` connect `Now` to `ImportService`, `ImportResult`, `Fail`, `auth/apikey.go`, `NewAdminAPIKeyService`, `Context`, `T`, `AuthorizationMiddleware`, `newMockGormDB`, `JobData`, `WorkerPool`, `New`, `NewSnowflake`, `NewDBAPIKeyStore`, `Wrap`, `RedisCache`, `Metrics`, `health.go`, `CronScheduler`, `PresenceManager`, `NewWithStore`, `circuit`, `JWT`, `gorm_logger_test.go`, `DeadLetter`, `upload_handler_test.go`, `middleware/middleware_test.go`, `ws_integration_test.go`, `NewUserRateLimiter`, `JobDef`, `NewService`, `NewCleanupService`, `NewMysqlRepository`, `Lock`, `ClientHub`, `Outbox`, `TestWebSocketNotification`, `SetupRouter`, `newRedisTestQueue`, `MySQLStore`, `NewAdminUserService`, `AuditMiddleware`, `DefaultCSRFConfig`, `mysqlAPIKeyRepository`, `AdminMonitoringService`, `.Validate`, `signature_test.go`, `NewWebhook`?**
  _High betweenness centrality (0.090) - this node is a cross-community bridge._
- **Why does `New()` connect `New` to `fakeRedis`, `Context`, `JWT`, `SessionStore`, `AuthHandler`, `newResetService`, `NewAuthHandler`, `NewPathEnforcer`, `Wrap`, `config/config.go`, `LoginFailureTracker`?**
  _High betweenness centrality (0.044) - this node is a cross-community bridge._
- **Are the 4 inferred relationships involving `T()` (e.g. with `translateValidationDetails()` and `ValidateJSON()`) actually correct?**
  _`T()` has 4 INFERRED edges - model-reasoned connections that need verification._
- **Are the 81 inferred relationships involving `New()` (e.g. with `.CreateKey()` and `.ToggleTask()`) actually correct?**
  _`New()` has 81 INFERRED edges - model-reasoned connections that need verification._
- **Are the 81 inferred relationships involving `Now()` (e.g. with `.CreateKey()` and `.generateResetCode()`) actually correct?**
  _`Now()` has 81 INFERRED edges - model-reasoned connections that need verification._
- **Are the 80 inferred relationships involving `Fail()` (e.g. with `.Generate()` and `.HandleDelete()`) actually correct?**
  _`Fail()` has 80 INFERRED edges - model-reasoned connections that need verification._