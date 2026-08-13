# Graph Report - jimu  (2026-08-13)

## Corpus Check
- 387 files · ~123,177 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3302 nodes · 7233 edges · 231 communities (182 shown, 49 thin omitted)
- Extraction: 82% EXTRACTED · 18% INFERRED · 0% AMBIGUOUS · INFERRED: 1334 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `45202e5b`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- mysqlPermissionRepository
- ImportService
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
- JobRegistry
- Hub
- NewMQPublisher
- Router
- AdminUserService
- AuthService
- config/config.go
- Worker
- fakeStorage
- RedisCache
- CronScheduler
- health.go
- AuditLog
- Fail
- Job
- Permission
- Context
- OAuthService
- newTaskHandler
- newLock
- InitSnowflake
- Message
- NewWithStore
- New
- User
- JWT
- Logger
- DeadLetter
- JobHistory
- Manager
- AdminUserHandler
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
- AuditService
- securityRouter
- RegisterRoleRoutes
- mysqlAuditRepository
- Data Model: Jimu 框架持久化实体
- config/config_test.go
- NewUserRateLimiter
- ImportJob
- Bootstrap
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
- MySQLStore
- .Consume
- DBConfig
- .RegisterHTTP
- NewContainer
- AdminTaskService
- SetupRouter
- NewAuditService
- LoginFailureTracker
- New
- OAuthBinding
- NewManagementServer
- Implementation Plan: Jimu 后端框架能力规格
- New
- AuditMiddleware
- adminAuthRouter
- IdempotencyMiddleware
- DefaultCSRFConfig
- RateLimiter
- interfaces/fakes_test.go
- CI Workflow
- Jimu Constitution
- New
- AdminMonitoringService
- ratelimit_user_test.go
- ConnectWithRetry
- Jimu API Swagger Definition
- Server
- bindImportFile
- Role
- fakeContainer
- Tasks: Jimu 后端框架能力规格
- newRedisTestQueue
- signature_test.go
- NewWebhook
- NewAdminConfigService
- Limiter
- Now
- Jimu Collaboration Rules (CLAUDE.md)
- Wrap
- TestReadinessBoundsCheckerDuration
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
- GitHubProvider
- UserInfo
- NewDBAPIKeyStore
- InitTracing
- fakeAuditRepository
- Duration
- AdminAuthMiddleware
- RegisterUserRoutes
- Metrics
- New
- Release Workflow
- Module Layering & contract.Module
- Page
- 配置契约
- Module 契约
- workerPoolComponent
- events.go
- Prometheus Service
- CLI 契约
- NewServer
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
- Outbox
- TestWebSocketNotification
- As
- newHistoryTestDB
- newTestScheduler
- mockNotification
- Policy
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
- AuditRepository
- newRepoTestDB
- Pagination

## God Nodes (most connected - your core abstractions)
1. `T()` - 641 edges
2. `New()` - 85 edges
3. `Fail()` - 80 edges
4. `Now()` - 79 edges
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

## Communities (231 total, 49 thin omitted)

### Community 0 - "mysqlPermissionRepository"
Cohesion: 0.24
Nodes (5): PermissionRepository, mysqlPermissionRepository, Context, DB, NewMysqlPermissionRepository()

### Community 1 - "ImportService"
Cohesion: 0.05
Nodes (47): ImportService, ImportJobRepository, UserRepository, CSVImporter, ExcelImporter, FieldRule, FieldType, Format (+39 more)

### Community 2 - "New"
Cohesion: 0.18
Nodes (21): Module, AuthConfig, RouterGroup, Service, RegisterAuthRoutes(), RegisterCaptchaRoute(), Engine, newRouterLimiter() (+13 more)

### Community 3 - "LocalStorage"
Cohesion: 0.06
Nodes (31): New(), newOSSStorage(), Context, Duration, ReadCloser, Reader, NewLocalStorage(), TestLocalStorageFilePersisted() (+23 more)

### Community 4 - "fakeRedis"
Cohesion: 0.08
Nodes (25): fakePipeline, fakeRedis, redisClientAdapter, RedisSessionStore, sessionPipeline, sessionRedis, SessionStore, BoolCmd (+17 more)

### Community 5 - "auth/apikey.go"
Cohesion: 0.16
Nodes (14): APIKey, APIKeyContextKey, APIKeyStore, APIKeyVerifier, dbAPIKeyStore, APIKeyFromContext(), ContextWithAPIKey(), Context (+6 more)

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
Cohesion: 0.06
Nodes (38): assertQueryParams(), readOpenAPI(), TestOpenAPIIncludesCRUDContract(), TestAPIKeyTableName(), TestHashKey(), TestImportJobTableNameAndStatus(), TestOAuthBindingTableName(), TestEntityValues() (+30 more)

### Community 10 - "New"
Cohesion: 0.32
Nodes (5): RoleHandler, Context, NewRoleHandler(), DB, New()

### Community 11 - "message.go"
Cohesion: 0.14
Nodes (14): BuildRoomChannel(), BuildUserChannel(), TestBuildRoomChannel(), TestBuildUserChannel(), TestNewMessage(), TestNewMessageMarshalError(), TestWSMessageDecodePayload(), TestWSMessageDecodePayloadError() (+6 more)

### Community 12 - "generator/module.go"
Cohesion: 0.20
Nodes (20): targetFile, templateData, camel(), GenerateModule(), GenerateModuleAt(), lowerCamel(), mkdirAllTracked(), nextMigrationNumber() (+12 more)

### Community 13 - "JobData"
Cohesion: 0.17
Nodes (10): Client, Context, Duration, NewRedisQueue(), Context, Duration, fakeConsumer, fakeProducer (+2 more)

### Community 14 - "PermissionService"
Cohesion: 0.20
Nodes (10): CreatePermissionRequest, PermissionService, UpdatePermissionRequest, PermissionResponse, Time, ToPermissionResponse(), ToPermissionResponses(), isDuplicateKey() (+2 more)

### Community 15 - "WorkerPool"
Cohesion: 0.15
Nodes (11): CancelFunc, GetWorker(), Context, Duration, MySQLStore, Consumer, Queue, WorkerConfig (+3 more)

### Community 16 - "AuthHandler"
Cohesion: 0.22
Nodes (11): CaptchaConfig, AuthHandler, loginRequest, refreshRequest, authContext(), Context, Duration, Service (+3 more)

### Community 17 - "mysqlAPIKeyRepository"
Cohesion: 0.29
Nodes (5): mysqlAPIKeyRepository, APIKey, Context, DB, NewMysqlAPIKeyRepository()

### Community 18 - "JobRegistry"
Cohesion: 0.15
Nodes (7): ComponentProvider, ErrorSource, EventBus, HTTPMiddlewareProvider, JobRegistry, Module, ProtectedHTTPMiddlewareProvider

### Community 19 - "Hub"
Cohesion: 0.17
Nodes (8): Channel, Context, RWMutex, Connection, Hub, Registration, WebSocket, WebSocketConfig

### Community 20 - "NewMQPublisher"
Cohesion: 0.16
Nodes (10): Context, Queue, NewMQPublisher(), Context, Duration, TestMQPublisher_Publish(), TestMQPublisher_PublishError(), errorQueue (+2 more)

### Community 21 - "Router"
Cohesion: 0.15
Nodes (6): fakeAuthzModule, fakeBusinessModule, Router, EventBus, HandlerFunc, TestBusinessRoutesRequireProtectedMiddleware()

### Community 22 - "AdminUserService"
Cohesion: 0.21
Nodes (9): AdminCreateUserRequest, AdminUpdateUserRequest, AdminUser, AdminUserService, ListUserFilter, userRole, Context, DB (+1 more)

### Community 23 - "AuthService"
Cohesion: 0.22
Nodes (9): AuthService, AuthServiceInterface, TokenPair, ErrAccountLocked(), Context, Duration, invalidCredentials(), normalizeUsername() (+1 more)

### Community 24 - "config/config.go"
Cohesion: 0.15
Nodes (24): AuditConfig, CacheConfig, CaptchaResult, EmailConfig, HTTPClientConfig, HTTPConfig, IDConfig, LogConfig (+16 more)

### Community 25 - "Worker"
Cohesion: 0.20
Nodes (10): Worker, Context, RWMutex, NewWorker(), Duration, testWorker(), TestWorkerFlushesConfiguredBatch(), TestWorkerRejectsWhenQueueFull() (+2 more)

### Community 26 - "fakeStorage"
Cohesion: 0.22
Nodes (5): fakeStorage, Context, Duration, ReadCloser, Reader

### Community 27 - "RedisCache"
Cohesion: 0.23
Nodes (7): Cache, RedisCache, Client, Context, Duration, NewRedisCache(), randomToken()

### Community 28 - "CronScheduler"
Cohesion: 0.18
Nodes (7): Cron, EntryID, Context, RWMutex, Time, CronScheduler, JobInfo

### Community 29 - "health.go"
Cohesion: 0.19
Nodes (15): Client, Context, DB, Duration, ResponseWriter, NewReadiness(), NewRedisChecker(), NewSQLChecker() (+7 more)

### Community 30 - "AuditLog"
Cohesion: 0.18
Nodes (9): fakeAuditRepository, fakeBatchRepository, AuditLog, Change, fakeQueue, Context, Context, Mutex (+1 more)

### Community 31 - "Fail"
Cohesion: 0.11
Nodes (13): AdminAPIKeyHandler, AdminConfigHandler, AdminJobHandler, UserHandler, Context, Context, Context, Context (+5 more)

### Community 32 - "Job"
Cohesion: 0.14
Nodes (9): Job, JobRepository, mysqlJobRepository, fakeJobRepo, Context, DB, NewMysqlJobRepository(), Time (+1 more)

### Community 33 - "Permission"
Cohesion: 0.25
Nodes (10): fakePermissionRepository, Permission, NewPermissionService(), Context, permissionAppCode(), TestPermissionServiceCreateMapsDuplicateNameToConflict(), TestPermissionServiceDeleteWrapsRepositoryError(), TestPermissionServiceListPassesPagination() (+2 more)

### Community 34 - "Context"
Cohesion: 0.11
Nodes (14): fakeOutboxUserRepo, recordingOutboxStore, appCode(), createOutboxUserService(), fakeUserRepository, Context, TestCreateWritesOutbox(), TestUpdateAndDeleteWriteOutbox() (+6 more)

### Community 35 - "OAuthService"
Cohesion: 0.20
Nodes (10): OAuthService, BindingRepository, Client, Context, DB, Duration, NewOAuthService(), refreshTTL() (+2 more)

### Community 36 - "newTaskHandler"
Cohesion: 0.24
Nodes (8): AdminTaskHandler, Context, NewAdminTaskHandler(), newTaskHandler(), TestAdminTaskHandlerHistory(), TestAdminTaskHandlerList(), TestAdminTaskHandlerToggle(), TestAdminTaskHandlerTrigger()

### Community 37 - "newLock"
Cohesion: 0.18
Nodes (16): RedisConfig, Client, newLock(), TestLock_AcquireRelease(), TestLock_ConcurrentAcquire(), TestLock_Extend(), TestLock_ReleaseOnlyOwnToken(), TestLock_WithLock() (+8 more)

### Community 38 - "InitSnowflake"
Cohesion: 0.07
Nodes (38): snowflakeModel, stringKeyModel, Field, Generator, snowflake, uuidGenerator, TestIndirect(), TestInitSnowflake_InvalidWorkerID() (+30 more)

### Community 39 - "Message"
Cohesion: 0.19
Nodes (9): Dispatcher, Channel, Context, Context, Channel, Message, SMS, fakeKafkaReader (+1 more)

### Community 40 - "NewWithStore"
Cohesion: 0.24
Nodes (13): NewMemoryStore(), TestSetEnabledNotFound(), TestSetEnabledToggle(), TestTriggerJobNotFound(), TestTriggerJobRunsCommand(), NewWithStore(), newTestLogger(), TestAddNamedFuncPersistsAndRuns() (+5 more)

### Community 41 - "New"
Cohesion: 0.19
Nodes (12): New(), TestAppError(), TestAppErrorWithCause(), TestFailDoesNotLeakInfrastructureDetails(), TestOKIncludesStableEnvelopeAndRequestID(), TestPageIncludesStableEnvelopeAndPagination(), TestCreatedUsesStandardEnvelope(), TestFailHidesInternalCauseAndIncludesRequestID() (+4 more)

### Community 42 - "User"
Cohesion: 0.12
Nodes (11): fakeUserRepo, User, fakeSessionStore, sessionRecord, Context, Duration, DeletedAt, Time (+3 more)

### Community 43 - "JWT"
Cohesion: 0.20
Nodes (10): Claims, JWT, Duration, New(), NewWithRotation(), TestJWTPopulatesTypedClaims(), TestParseRejectsWrongTokenType(), AuthMiddleware() (+2 more)

### Community 44 - "Logger"
Cohesion: 0.06
Nodes (43): AtomicLevel, gormLogger, Interface, Context, Duration, Time, isSensitiveField(), NewGormLogger() (+35 more)

### Community 45 - "DeadLetter"
Cohesion: 0.13
Nodes (10): DeadLetter, DeadLetterRepository, mysqlDeadLetterRepository, fakeDeadLetterRepo, Context, DB, NewMysqlDeadLetterRepository(), Time (+2 more)

### Community 46 - "JobHistory"
Cohesion: 0.19
Nodes (8): JobHistory, JobHistoryRepository, mysqlJobHistoryRepository, Context, DB, NewMysqlJobHistoryRepository(), Time, fakeHistoryRepo

### Community 47 - "Manager"
Cohesion: 0.14
Nodes (16): contextKey, Flag, Manager, AdminFeatureHandler, NewAdminFeatureHandler(), TestAdminFeatureHandlerList(), TestAdminFeatureHandlerUpdate(), FromContext() (+8 more)

### Community 48 - "AdminUserHandler"
Cohesion: 0.33
Nodes (4): AdminUserHandler, Context, NewAdminUserHandler(), paginationFromQuery()

### Community 50 - "NewChannelManager"
Cohesion: 0.26
Nodes (12): NewChannelManager(), TestChannelManagerGetChannelMissing(), TestChannelManagerGetSubscribers(), TestChannelManagerPreCreatedBroadcast(), TestChannelManagerSubscribeCreatesWithType(), TestChannelManagerSubscribeExisting(), TestChannelManagerUnsubscribe(), TestChannelManagerUnsubscribeAll() (+4 more)

### Community 51 - "Application"
Cohesion: 0.21
Nodes (9): Application, Component, forwardError(), Context, Duration, NewApplication(), TestApplicationReturnsComponentRuntimeError(), TestApplicationRollsBackStartedComponents() (+1 more)

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
Cohesion: 0.15
Nodes (23): CORSMiddleware(), DefaultLogConfig(), HandlerFunc, isAllowedOrigin(), Logger(), Recovery(), RequestID(), shouldLogBody() (+15 more)

### Community 56 - "NewEventBusPublisher"
Cohesion: 0.32
Nodes (6): Context, EventBus, RawMessage, NewEventBusPublisher(), EventBusPublisher, EventPayload

### Community 57 - "ws_integration_test.go"
Cohesion: 0.24
Nodes (28): NewMessage(), connID(), Time, mustDecodeTitle(), newWSFixture(), TestWSBroadcastAll(), TestWSChatBroadcast(), TestWSChatInvalidPayload() (+20 more)

### Community 58 - "oauth/application/service_test.go"
Cohesion: 0.23
Nodes (33): oauthStateKey(), dupSubjectProviders(), githubProviders(), Client, DB, Miniredis, newRedisClient(), newRepoResult() (+25 more)

### Community 59 - "AuditService"
Cohesion: 0.28
Nodes (7): AuditLogResponse, AuditService, Time, ToAuditLogResponse(), ToAuditLogResponses(), Context, serializeChanges()

### Community 60 - "securityRouter"
Cohesion: 0.23
Nodes (10): TestSecurityAddVary(), addVary(), Context, HandlerFunc, Security(), Engine, securityRouter(), TestSecurityHandlesAllowedPreflight() (+2 more)

### Community 61 - "RegisterRoleRoutes"
Cohesion: 0.20
Nodes (4): RouterGroup, RegisterRoleRoutes(), EventBus, Module

### Community 62 - "mysqlAuditRepository"
Cohesion: 0.39
Nodes (4): mysqlAuditRepository, deserializeChanges(), Context, DB

### Community 63 - "Data Model: Jimu 框架持久化实体"
Cohesion: 0.06
Nodes (31): 1. User（用户）, 2. Role（角色）, 3. Permission（权限）, 4. APIKey（API 密钥）, 5. AuditLog（审计日志）, 6. OAuthBinding（第三方绑定）, 7. OutboxEvent（Outbox 事件）, 8. ScheduledJob（定时任务定义） (+23 more)

### Community 64 - "config/config_test.go"
Cohesion: 0.22
Nodes (14): Load(), Config, TestDBPasswordOverride(), TestJWTSecretOverride(), TestLoad(), TestStorageConfigFieldMapping(), TestValidateHTTPMode(), TestValidateLogFormat() (+6 more)

### Community 65 - "NewUserRateLimiter"
Cohesion: 0.24
Nodes (14): defaultKeyFunc(), Client, Context, Duration, HandlerFunc, Time, NewUserRateLimiter(), TestDefaultKeyFuncPriority() (+6 more)

### Community 66 - "ImportJob"
Cohesion: 0.19
Nodes (7): ImportJob, mysqlImportJobRepository, fakeImportJobRepo, Time, Context, DB, NewMysqlImportJobRepository()

### Community 67 - "Bootstrap"
Cohesion: 0.36
Nodes (7): moduleLogger, registerRouter, main(), run(), Bootstrap(), registerEventBusBridge(), registerHTTP()

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
Cohesion: 0.20
Nodes (12): RouterGroup, RegisterPermissionRoutes(), Context, HandlerFunc, localeOf(), TestValidateJSONTranslatesFieldErrors(), translateValidationDetails(), translateValidationMessage() (+4 more)

### Community 73 - "NewMysqlRepository"
Cohesion: 0.22
Nodes (12): TestMysqlRepositoryMySQLIntegration(), NewMysqlRepository(), DB, newTestUser(), newUserTestDB(), TestMysqlRepositoryCreateAndFindByID(), TestMysqlRepositoryFindByUsername(), TestMysqlRepositoryListCountAndPagination() (+4 more)

### Community 74 - "dispatcher"
Cohesion: 0.22
Nodes (8): Channel, Context, dispatcher, RWMutex, NewDispatcher(), TestDispatcherDispatch(), TestDispatcherDispatchBatch(), Notification

### Community 75 - "KafkaQueue"
Cohesion: 0.28
Nodes (6): Context, Duration, NewKafkaQueue(), KafkaMessageReader, KafkaMessageWriter, KafkaQueue

### Community 76 - "RabbitMQQueue"
Cohesion: 0.28
Nodes (6): Context, Delivery, Duration, NewRabbitMQQueue(), RabbitMQChannel, RabbitMQQueue

### Community 77 - "Lock"
Cohesion: 0.33
Nodes (8): generateToken(), Client, Context, Duration, Time, NewLock(), AcquireResult, Lock

### Community 78 - "ClientHub"
Cohesion: 0.05
Nodes (23): RWMutex, NewChannel(), TestChannelSubscribeUnsubscribe(), TestNewChannel(), Conn, Context, HandlerFunc, RWMutex (+15 more)

### Community 79 - "routerLimiterRedis"
Cohesion: 0.29
Nodes (6): routerLimiterRedis, BoolSliceCmd, Cmd, Context, StringCmd, routerLimiterInt()

### Community 80 - "auth/application/service_test.go"
Cohesion: 0.33
Nodes (14): BenchmarkLogin(), benchUser(), B, NewAuthService(), appCode(), newFakeSessionStore(), newTestService(), TestLoginCreatesRefreshSession() (+6 more)

### Community 81 - "i18n.go"
Cohesion: 0.18
Nodes (7): HandlerFunc, Locale(), ParseAcceptLanguage(), TestParseAcceptLanguage(), TestTDefaultsToChinese(), TestTfFormatsArgs(), Tf()

### Community 82 - "MySQLStore"
Cohesion: 0.26
Nodes (6): Context, DB, NewMySQLStore(), TestMySQLStore_AddWithinTx(), TestMySQLStore_AddWithNilTx(), MySQLStore

### Community 83 - ".Consume"
Cohesion: 0.24
Nodes (9): Context, Delivery, TestRabbitMQQueue_ConsumeTimeout(), TestRabbitMQQueue_ConsumeUnmarshalError(), TestRabbitMQQueue_SubmitConsume(), TestRabbitMQQueueImplementsInterfaces(), Publishing, fakeRabbitMQChannel (+1 more)

### Community 84 - "DBConfig"
Cohesion: 0.31
Nodes (8): DBConfig, DB, mysqlEnvDBConfig(), mysqlReachable(), NewTestDB(), NewTestDBWithPool(), SkipUnlessMysql(), TestDB

### Community 85 - ".RegisterHTTP"
Cohesion: 0.23
Nodes (6): Module, Client, DB, EventBus, HandlerFunc, Service

### Community 86 - "NewContainer"
Cohesion: 0.15
Nodes (14): Container, Client, Config, Context, DB, EventBus, Service, TracerProvider (+6 more)

### Community 87 - "AdminTaskService"
Cohesion: 0.29
Nodes (5): AdminTaskService, TaskExecution, TaskInfo, Context, Time

### Community 88 - "SetupRouter"
Cohesion: 0.18
Nodes (17): ConfigureTrustedProxies(), formatAddr(), Engine, SetupRouter(), freeAddr(), TestConfigureTrustedProxies(), TestFormatAddr(), testLogger() (+9 more)

### Community 89 - "NewAuditService"
Cohesion: 0.60
Nodes (4): NewAuditService(), auditAppCode(), TestAuditServiceGetMapsNotFound(), TestAuditServiceListReturnsDTOAndPassesPagination()

### Community 90 - "LoginFailureTracker"
Cohesion: 0.33
Nodes (8): LockoutConfig, LoginFailureTracker, DefaultLockoutConfig(), Client, Context, Duration, lockKey(), NewLoginFailureTracker()

### Community 91 - "New"
Cohesion: 0.19
Nodes (11): buildProviders(), Client, DB, EventBus, New(), newTestModule(), TestBuildProvidersFiltersEnabled(), TestModuleNameAndContract() (+3 more)

### Community 92 - "OAuthBinding"
Cohesion: 0.18
Nodes (7): fakeBindingRepo, OAuthBinding, fakeBindingRepo, fakeSessionStore, Time, Context, Duration

### Community 93 - "NewManagementServer"
Cohesion: 0.22
Nodes (7): Handler, passingChecker, HealthRouter(), NewManagementServer(), Context, TestManagementRouterExposure(), TestNewManagementServer()

### Community 94 - "Implementation Plan: Jimu 后端框架能力规格"
Cohesion: 0.07
Nodes (26): Content Quality, Feature Readiness, Notes, Requirement Completeness, Specification Quality Checklist: Jimu 后端框架能力规格, Complexity Tracking, Constitution Check, Documentation (this feature) (+18 more)

### Community 95 - "New"
Cohesion: 0.15
Nodes (10): PermissionHandler, Context, NewPermissionHandler(), TestPermissionCreateReturnsCreated(), TestPermissionDeleteReturnsNoContent(), DB, EventBus, New() (+2 more)

### Community 96 - "AuditMiddleware"
Cohesion: 0.24
Nodes (9): Queue, AuditMiddleware(), Context, HandlerFunc, isManagementPath(), optionalString(), optionalUint64(), TestAuditMiddlewareAllowsAnonymousRequest() (+1 more)

### Community 97 - "adminAuthRouter"
Cohesion: 0.27
Nodes (9): AdminAuth(), HandlerFunc, adminAuthRouter(), Engine, TestAdminAuthAllowsAdminRole(), TestAdminAuthAllowsEnglishAdminRole(), TestAdminAuthRejectsNonAdminRole(), TestAdminAuthRequiresAuthentication() (+1 more)

### Community 98 - "IdempotencyMiddleware"
Cohesion: 0.09
Nodes (25): Int32, Buffer, ResponseWriter, newResponseBodyWriter(), sanitizeBody(), sanitizeJSONField(), TestResponseBodyWriterCapturesAndTruncates(), TestResponseBodyWriterWriteHeaderPassthrough() (+17 more)

### Community 99 - "DefaultCSRFConfig"
Cohesion: 0.18
Nodes (23): CSRF(), DefaultCSRFConfig(), generateToken(), Context, HandlerFunc, isSafeMethod(), setCSRFCookie(), csrfRouter() (+15 more)

### Community 100 - "RateLimiter"
Cohesion: 0.20
Nodes (16): GlobalRateLimit(), HandlerFunc, RWMutex, NewRateLimiter(), Engine, Request, ratelimitRequest(), ratelimitRouter() (+8 more)

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
Cohesion: 0.19
Nodes (9): Client, Queue, New(), TestNew_InvalidType(), TestNew_Redis(), Config, KafkaConfig, RabbitMQConfig (+1 more)

### Community 105 - "AdminMonitoringService"
Cohesion: 0.13
Nodes (17): AdminMonitoringService, HealthStatus, MemoryStats, SystemStatus, AdminMonitoringHandler, Client, Context, Time (+9 more)

### Community 106 - "ratelimit_user_test.go"
Cohesion: 0.53
Nodes (5): Engine, TestUserRateLimiterAllowsWithinLimit(), TestUserRateLimiterFailOpenOnRedisError(), TestUserRateLimiterRejectsOverLimit(), userRateRouter()

### Community 107 - "ConnectWithRetry"
Cohesion: 0.21
Nodes (14): configurePool(), ConnectWithRetry(), dsn(), Context, DB, open(), pingDB(), DB (+6 more)

### Community 108 - "Jimu API Swagger Definition"
Cohesion: 0.33
Nodes (10): Jimu API Swagger Definition, Audit Log Endpoints, Auth Flow Endpoints (login/refresh/logout/register), BearerAuth Security (API Key JWT), Captcha Endpoint, contract.ErrorResponse, OAuth Endpoints, contract.PageResponse (+2 more)

### Community 110 - "bindImportFile"
Cohesion: 0.18
Nodes (16): AdminImportHandler, DB, newSqliteDB(), bindImportFile(), Buffer, Context, NewAdminImportHandler(), DB (+8 more)

### Community 111 - "Role"
Cohesion: 0.05
Nodes (36): AssignPermissionsRequest, CreateRoleRequest, fakeRoleRepository, RoleResponse, RoleService, UpdateRoleRequest, Role, RoleRepository (+28 more)

### Community 112 - "fakeContainer"
Cohesion: 0.40
Nodes (9): bridgeFn(), registerOutboxWorkers(), fakeContainer(), newTestLogger(), TestBridgeWorkerConversionFailureErrors(), TestBridgeWorkerPublishesStrongTypeToBareTopic(), TestBridgeWorkerUnknownTypeErrors(), TestEventBusBridgePublishesToBareTopic() (+1 more)

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
Cohesion: 0.17
Nodes (12): BenchmarkWebhookSend(), B, Channel, Client, Context, NewWebhook(), TestWebhookSendBatch(), TestWebhookSendNilClient() (+4 more)

### Community 117 - "NewAdminConfigService"
Cohesion: 0.14
Nodes (20): AdminConfigService, Client, Context, EventBus, NewAdminConfigService(), Miniredis, newConfigTestService(), TestAdminConfigServiceConfigKey() (+12 more)

### Community 118 - "Limiter"
Cohesion: 0.19
Nodes (13): Limiter, Context, Duration, Scripter, limitKey(), NewLimiter(), newTestLimiter(), TestLimiterAllowsOnlyConfiguredWindow() (+5 more)

### Community 119 - "Now"
Cohesion: 0.12
Nodes (22): NewPresenceManager(), TestPresenceIsOnline(), TestPresenceIsTyping(), TestPresenceManagerAllPresences(), TestPresenceManagerHeartbeat(), TestPresenceManagerHeartbeatRestoresOnlineStatus(), TestPresenceManagerOfflineMissing(), TestPresenceManagerOnlineCountFiltersOffline() (+14 more)

### Community 120 - "Jimu Collaboration Rules (CLAUDE.md)"
Cohesion: 0.83
Nodes (4): Jimu Collaboration Rules (CLAUDE.md), Development Config (app.yaml), Production Config (app.prod.yaml), Configuration Reference Table

### Community 121 - "Wrap"
Cohesion: 0.25
Nodes (11): BatchDeleteRequest, BatchResult, CreateUserRequest, UpdateUserRequest, UserResponse, UserService, Time, ToUserResponse() (+3 more)

### Community 122 - "TestReadinessBoundsCheckerDuration"
Cohesion: 0.47
Nodes (4): Context, TestReadinessBoundsCheckerDuration(), TestReadinessStatus(), checkerFunc

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
Cohesion: 0.36
Nodes (5): Client, Config, NewWeChatProvider(), WeChatConfig, WeChatProvider

### Community 128 - "New"
Cohesion: 0.13
Nodes (14): NewUserService(), NewUserHandler(), Context, fakeUserRepository, TestUserCreateReturnsCreatedDTO(), TestUserDeleteReturnsNoContent(), TestUserListRejectsInvalidPaginationContract(), TestUserListUsesDefaultPaginationBeforeService() (+6 more)

### Community 130 - "validator/validator.go"
Cohesion: 0.39
Nodes (6): FieldLevel, Validate(), validateIDCard(), validateMobile(), validatePassword(), validateUsername()

### Community 132 - "newWSHandler"
Cohesion: 0.32
Nodes (7): AdminWSHandler, Context, NewAdminWSHandler(), newWSHandler(), TestAdminWSHandlerOnlineUsers(), TestAdminWSHandlerPresence(), TestAdminWSHandlerPush()

### Community 133 - "oauth/interfaces/handler_test.go"
Cohesion: 0.23
Nodes (17): OAuthHandler, NewOAuthHandler(), assertCode(), doRequest(), githubProvider(), Engine, newTestHandler(), newTestRouter() (+9 more)

### Community 134 - "Context"
Cohesion: 0.14
Nodes (7): fakeAPIKeyRepo, fakeEventBus, fakeImportJobRepo, APIKey, fakeUserRepository, Context, Mutex

### Community 135 - "NewEmail"
Cohesion: 0.16
Nodes (14): buildEmailHeaders(), Channel, Context, NewEmail(), Conn, newFakeSMTPServer(), TestBuildEmailHeaders(), TestEmailSendEmptyRecipient() (+6 more)

### Community 136 - "New"
Cohesion: 0.11
Nodes (30): fakeEventBus, testRole, fakeUserRepository, NewAdminUserService(), TestAdminUserServiceAssignRoles(), TestAdminUserServiceAssignRolesRoleQueryError(), TestAdminUserServiceCreateUser(), TestAdminUserServiceDisableUser() (+22 more)

### Community 137 - "Context"
Cohesion: 0.29
Nodes (6): fakeSessionStore, sessionRecord, Context, Duration, TestRefreshTTLPositive(), sessionRecord

### Community 138 - "SecurityHeadersFromConfig"
Cohesion: 0.27
Nodes (9): SecurityConfig, DefaultSecurityConfig(), TestSecurityHeadersMiddleware(), TestSecurityHeadersNilValuesSkipped(), TestSecurityHeadersRespectsCustomValues(), Context, HandlerFunc, SecurityHeadersFromConfig() (+1 more)

### Community 139 - "Redis Service"
Cohesion: 0.20
Nodes (12): Development Environment, Adminer Service, MariaDB Service, Redis Service, Server Service, Graphical Captcha, Docker & K8s Deployment, Redis Distributed Lock (+4 more)

### Community 140 - "AuthorizationMiddleware"
Cohesion: 0.18
Nodes (12): AuthorizationStore, DBAuthorizationStore, Enforcer, HandlerFunc, ProtectedMiddleware(), HandlerFunc, AuthorizationMiddleware(), Context (+4 more)

### Community 141 - "newMockGormDB"
Cohesion: 0.19
Nodes (15): MySQLBindingRepository, Context, DB, NewMySQLBindingRepository(), bindingRows(), DB, Sqlmock, newMockGormDB() (+7 more)

### Community 142 - "NewSMS"
Cohesion: 0.43
Nodes (5): NewSMS(), TestSMSAliyunReject(), TestSMSAliyunSend(), TestSMSUnknownProvider(), SMSConfig

### Community 143 - "queue/worker_test.go"
Cohesion: 0.39
Nodes (11): NewMySQLStore(), NewWorkerPool(), RegisterWorker(), fakeStore(), MySQLStore, newFakeJobRepo(), TestWorkerPoolConsumeJob(), TestWorkerPoolOutboxEventFailureWritesDeadLetter() (+3 more)

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
Cohesion: 0.22
Nodes (5): fakeProvider, fakeProvider, Context, Context, UserInfo

### Community 150 - "NewDBAPIKeyStore"
Cohesion: 0.33
Nodes (12): createKey(), APIKey, DB, newTestDB(), TestDBAPIKeyStore_GetByKeyHash(), TestDBAPIKeyStore_GetByKeyHashNotFound(), TestDBAPIKeyStore_UpdateLastUsed(), TestVerifyWithDBStore() (+4 more)

### Community 151 - "InitTracing"
Cohesion: 0.48
Nodes (6): DefaultTracingConfig(), Context, TracerProvider, InitTracing(), ShutdownTracing(), TracingConfig

### Community 153 - "Duration"
Cohesion: 0.43
Nodes (4): Duration, ExponentialBackoff, FixedRetry, RetryStrategy

### Community 154 - "AdminAuthMiddleware"
Cohesion: 0.40
Nodes (3): AdminAuthMiddleware(), HandlerFunc, TestAdminAuthMiddleware()

### Community 156 - "Metrics"
Cohesion: 0.33
Nodes (7): HandlerFunc, Metrics(), gatherHTTPMetric(), Engine, setupMetricsEngine(), TestMetricsLabelsUseRouteTemplate(), TestMetricsRecordsUnmatchedPath()

### Community 157 - "New"
Cohesion: 0.14
Nodes (12): Module, AuditHandler, Context, NewAuditHandler(), TestAuditGetInvalidIDReturnsStableBadRequest(), TestAuditGetReturnsLogDTO(), TestAuditListInvalidQueryReturnsStableBadRequest(), RouterGroup (+4 more)

### Community 159 - "Release Workflow"
Cohesion: 0.40
Nodes (5): Release Workflow, Jimu Changelog, Conventional Commits, Release Note Structure, Commit Style

### Community 160 - "Module Layering & contract.Module"
Cohesion: 0.40
Nodes (5): Principle III: Composable Modules, Module Layering & contract.Module, Module Generator CLI, Clean Architecture, Scaffold CLI

### Community 161 - "Page"
Cohesion: 0.27
Nodes (11): Context, HTTPStatus(), Created(), FailWithDetails(), Context, localeFrom(), Page(), requestID() (+3 more)

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

### Community 169 - "NewServer"
Cohesion: 0.19
Nodes (15): Client, Config, NewServer(), TestNewServerTLSInvalidFiles(), Context, Duration, Request, New() (+7 more)

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
Cohesion: 0.27
Nodes (9): DB, Enforcer, NewEnforcer(), NewPathEnforcer(), Enforcer, TestAuthorizationMiddlewareAllowsRolePolicy(), TestAuthorizationMiddlewareHidesStoreErrors(), TestAuthorizationMiddlewareRejectsMissingRole() (+1 more)

### Community 178 - "newMockGormDB"
Cohesion: 0.29
Nodes (11): Sqlmock, newMockGormDB(), DB, TestTransaction_Commit(), TestTransaction_Rollback(), TestWithTx_BeginError(), TestWithTx_Commit(), TestWithTx_PanicRollsBack() (+3 more)

### Community 210 - "mockTokenServer"
Cohesion: 0.22
Nodes (14): Client, mockClient(), mockTokenServer(), TestGitHubProviderExchange(), TestGoogleProviderExchange(), TestGoogleProviderExchangeBadUserInfo(), TestGoogleProviderExchangeTokenError(), TestWeChatProviderExchange() (+6 more)

### Community 212 - "Outbox"
Cohesion: 0.42
Nodes (5): Context, New(), Outbox, Publisher, Store

### Community 214 - "TestWebSocketNotification"
Cohesion: 0.33
Nodes (5): NewHub(), TestHubLifecycle(), TestWebSocketNotification(), waitHub(), mockConn

### Community 215 - "As"
Cohesion: 0.43
Nodes (4): AppError, ErrorInfo, AllErrorCodes(), As()

### Community 217 - "newHistoryTestDB"
Cohesion: 0.40
Nodes (4): TestMysqlDeadLetterRepositoryCRUD(), DB, newHistoryTestDB(), TestMysqlJobHistoryRepositoryCreateAndList()

### Community 218 - "newTestScheduler"
Cohesion: 0.62
Nodes (6): NewAdminTaskService(), newTestScheduler(), TestAdminTaskServiceGetHistory(), TestAdminTaskServiceListTasks(), TestAdminTaskServiceToggleTask(), TestAdminTaskServiceTriggerTask()

### Community 220 - "mockNotification"
Cohesion: 0.47
Nodes (3): Channel, Context, mockNotification

### Community 222 - "Policy"
Cohesion: 0.24
Nodes (6): fakeAuthorizationStore, Policy, fakeAuthzStore, Context, TestProtectedMiddlewareRequiresAccessToken(), Context

### Community 224 - "PermissionMiddleware"
Cohesion: 0.50
Nodes (3): Enforcer, HandlerFunc, PermissionMiddleware()

### Community 236 - "AuditRepository"
Cohesion: 0.32
Nodes (6): AuditRepository, AdminAuditHandler, DB, NewAdminAuditHandler(), TestAdminAuditHandlerList(), NewMysqlAuditRepository()

### Community 239 - "newRepoTestDB"
Cohesion: 0.53
Nodes (5): DB, newRepoTestDB(), TestMysqlAPIKeyRepository(), TestMysqlImportJobRepository(), TestMysqlJobRepository()

## Knowledge Gaps
- **202 isolated node(s):** `common.sh script`, `jimu`, `CaptchaResult`, `LogConfig`, `UserCreatedEvent` (+197 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **49 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `T()` connect `T` to `ImportService`, `New`, `LocalStorage`, `NewAdminAPIKeyService`, `message.go`, `generator/module.go`, `NewMQPublisher`, `Router`, `Worker`, `Permission`, `Context`, `OAuthService`, `newTaskHandler`, `newLock`, `InitSnowflake`, `NewWithStore`, `New`, `JWT`, `Logger`, `Manager`, `NewChannelManager`, `Application`, `RedisStore`, `upload_handler_test.go`, `middleware/middleware_test.go`, `ws_integration_test.go`, `oauth/application/service_test.go`, `securityRouter`, `config/config_test.go`, `NewUserRateLimiter`, `NewService`, `NewCleanupService`, `MySQLStore`, `ValidateJSON`, `NewMysqlRepository`, `dispatcher`, `ClientHub`, `auth/application/service_test.go`, `i18n.go`, `MySQLStore`, `.Consume`, `DBConfig`, `SetupRouter`, `NewAuditService`, `New`, `NewManagementServer`, `New`, `AuditMiddleware`, `adminAuthRouter`, `IdempotencyMiddleware`, `DefaultCSRFConfig`, `RateLimiter`, `New`, `AdminMonitoringService`, `ratelimit_user_test.go`, `ConnectWithRetry`, `bindImportFile`, `Role`, `fakeContainer`, `newRedisTestQueue`, `signature_test.go`, `NewWebhook`, `NewAdminConfigService`, `Limiter`, `Now`, `TestReadinessBoundsCheckerDuration`, `migrate_test.go`, `gzipRouter`, `New`, `newWSHandler`, `oauth/interfaces/handler_test.go`, `NewEmail`, `New`, `Context`, `SecurityHeadersFromConfig`, `newMockGormDB`, `NewSMS`, `queue/worker_test.go`, `New`, `setupTestCache`, `EventBus`, `Timeout`, `NewDBAPIKeyStore`, `AdminAuthMiddleware`, `Metrics`, `New`, `Page`, `NewServer`, `testEnforcer`, `newMockGormDB`, `mockTokenServer`, `TestWebSocketNotification`, `newHistoryTestDB`, `newTestScheduler`, `Policy`, `AuditRepository`, `newRepoTestDB`?**
  _High betweenness centrality (0.581) - this node is a cross-community bridge._
- **Why does `Now()` connect `Now` to `ImportService`, `auth/apikey.go`, `NewAdminAPIKeyService`, `New`, `Context`, `T`, `AuthorizationMiddleware`, `newMockGormDB`, `JobData`, `WorkerPool`, `New`, `mysqlAPIKeyRepository`, `NewDBAPIKeyStore`, `RedisCache`, `Metrics`, `CronScheduler`, `Fail`, `InitSnowflake`, `NewWithStore`, `JWT`, `Logger`, `DeadLetter`, `MySQLStore`, `upload_handler_test.go`, `middleware/middleware_test.go`, `ws_integration_test.go`, `NewUserRateLimiter`, `JobDef`, `NewService`, `NewCleanupService`, `NewMysqlRepository`, `Lock`, `ClientHub`, `MySQLStore`, `TestWebSocketNotification`, `SetupRouter`, `AuditMiddleware`, `DefaultCSRFConfig`, `AdminMonitoringService`, `newRedisTestQueue`, `signature_test.go`, `TestReadinessBoundsCheckerDuration`?**
  _High betweenness centrality (0.072) - this node is a cross-community bridge._
- **Why does `Wrap()` connect `Wrap` to `ImportService`, `OAuthService`, `NewAdminAPIKeyService`, `New`, `AuthorizationMiddleware`, `PermissionService`, `Role`, `AdminUserService`, `AuthService`, `upload_handler_test.go`, `As`, `AuditService`, `Fail`?**
  _High betweenness centrality (0.042) - this node is a cross-community bridge._
- **Are the 4 inferred relationships involving `T()` (e.g. with `translateValidationDetails()` and `ValidateJSON()`) actually correct?**
  _`T()` has 4 INFERRED edges - model-reasoned connections that need verification._
- **Are the 81 inferred relationships involving `New()` (e.g. with `.CreateKey()` and `.ToggleTask()`) actually correct?**
  _`New()` has 81 INFERRED edges - model-reasoned connections that need verification._
- **Are the 78 inferred relationships involving `Fail()` (e.g. with `.Generate()` and `.HandleDelete()`) actually correct?**
  _`Fail()` has 78 INFERRED edges - model-reasoned connections that need verification._
- **Are the 77 inferred relationships involving `Now()` (e.g. with `.CreateKey()` and `.Import()`) actually correct?**
  _`Now()` has 77 INFERRED edges - model-reasoned connections that need verification._