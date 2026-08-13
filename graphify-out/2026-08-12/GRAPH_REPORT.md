# Graph Report - jimu  (2026-08-12)

## Corpus Check
- 388 files · ~122,944 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3298 nodes · 7211 edges · 242 communities (194 shown, 48 thin omitted)
- Extraction: 82% EXTRACTED · 18% INFERRED · 0% AMBIGUOUS · INFERRED: 1322 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `45202e5b`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- NewSnowflake
- ImportResult
- RegisterAuthRoutes
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
- PresenceManager
- WorkerPool
- AuthHandler
- mysqlAPIKeyRepository
- New
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
- importer_test.go
- User
- OAuthService
- mysqlRepository
- newLock
- InitSnowflake
- Message
- NewWithStore
- New
- Context
- JWT
- gorm_logger_test.go
- DeadLetter
- JobHistory
- Manager
- AdminUserHandler
- Fail
- ChannelManager
- Application
- RedisStore
- Jimu Backend Framework README
- upload_handler_test.go
- middleware/middleware_test.go
- NewEventBusPublisher
- ws_integration_test.go
- oauth/application/service_test.go
- AuditService
- Security
- MySQLStore
- mysqlAuditRepository
- Data Model: Jimu 框架持久化实体
- config/config_test.go
- NewUserRateLimiter
- ImportJob
- RoleService
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
- TestDB
- .RegisterHTTP
- NewContainer
- AdminTaskService
- SetupRouter
- fakeAuditRepository
- LoginFailureTracker
- New
- OAuthBinding
- NewManagementServer
- Implementation Plan: Jimu 后端框架能力规格
- Permission
- AuditMiddleware
- adminAuthRouter
- IdempotencyMiddleware
- DefaultCSRFConfig
- RateLimiter
- NewLogChannel
- CI Workflow
- Jimu Constitution
- NewUserService
- AdminMonitoringService
- Logger
- New
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
- Limiter
- Now
- Server Service
- UserService
- .upload
- 统一响应契约
- Module
- migrate_test.go
- gzipRouter
- NewWeChatProvider
- NewUserHandler
- ImportService
- validator/validator.go
- newSqliteDB
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
- Context
- RunSeed
- setupTestCache
- EventBus
- Timeout
- UserInfo
- NewGoogleProvider
- NewDBAPIKeyStore
- InitTracing
- newFakeSMTPServer
- Duration
- NewAdminUserService
- .Validate
- setupMetricsEngine
- New
- fakeStorage
- Release Workflow
- Module Layering & contract.Module
- Page
- 配置契约
- Module 契约
- workerPoolComponent
- events.go
- Prometheus Service
- CLI 契约
- testConn
- NewServer
- Context
- 010_create_jobs.sql
- Security Policy
- 001_create_users.sql
- CaptchaHandler
- .validateCommon
- 002_create_roles.sql
- ProtectedMiddleware
- newMockGormDB
- 003_create_permissions.sql
- openapi.go
- 004_create_user_roles.sql
- Format
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
- AdminJobHandler
- Outbox
- generator/module_test.go
- TestWebSocketNotification
- As
- Client
- newHistoryTestDB
- newTestScheduler
- NewHandler
- mockNotification
- AdminAPIKeyHandler
- Policy
- fakeEventBus
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
- TestGeneratedModuleCompiles
- newRepoTestDB
- admin/module_test.go
- Pagination

## God Nodes (most connected - your core abstractions)
1. `T()` - 636 edges
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

## Communities (242 total, 48 thin omitted)

### Community 0 - "NewSnowflake"
Cohesion: 0.13
Nodes (16): Generator, snowflake, uuidGenerator, BenchmarkSnowflakeNextID(), BenchmarkUUIDNextID(), B, binaryID(), Mutex (+8 more)

### Community 1 - "ImportResult"
Cohesion: 0.19
Nodes (8): CSVImporter, ImportError, ImportResult, Context, Reader, NewCSVImporter(), Time, NewImportResult()

### Community 2 - "RegisterAuthRoutes"
Cohesion: 0.29
Nodes (14): RouterGroup, Service, RegisterAuthRoutes(), RegisterCaptchaRoute(), Engine, newRouterLimiter(), testAuthConfig(), TestLoginRateLimitUsesIPScope() (+6 more)

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
Cohesion: 0.11
Nodes (20): AdminAPIKeyService, CreateKeyInput, APIKey, APIKeyRepository, APIKey, Context, NewAdminAPIKeyService(), TestAdminAPIKeyServiceCreateKey() (+12 more)

### Community 9 - "T"
Cohesion: 0.06
Nodes (35): assertQueryParams(), readOpenAPI(), TestOpenAPIIncludesCRUDContract(), TestImportJobTableNameAndStatus(), TestOAuthBindingTableName(), TestEntityValues(), TestTableNames(), TestUserFields() (+27 more)

### Community 10 - "New"
Cohesion: 0.15
Nodes (9): RoleHandler, Context, NewRoleHandler(), TestRoleCreateReturnsCreated(), TestRoleDeleteReturnsNoContent(), DB, EventBus, New() (+1 more)

### Community 11 - "message.go"
Cohesion: 0.14
Nodes (14): BuildRoomChannel(), BuildUserChannel(), TestBuildRoomChannel(), TestBuildUserChannel(), TestNewMessage(), TestNewMessageMarshalError(), TestWSMessageDecodePayload(), TestWSMessageDecodePayloadError() (+6 more)

### Community 12 - "generator/module.go"
Cohesion: 0.33
Nodes (12): targetFile, templateData, camel(), GenerateModule(), GenerateModuleAt(), lowerCamel(), mkdirAllTracked(), nextMigrationNumber() (+4 more)

### Community 13 - "JobData"
Cohesion: 0.17
Nodes (10): Context, Duration, Client, Context, Duration, NewRedisQueue(), errorQueue, fakeQueue (+2 more)

### Community 14 - "PresenceManager"
Cohesion: 0.18
Nodes (4): RWMutex, Time, Presence, PresenceManager

### Community 15 - "WorkerPool"
Cohesion: 0.11
Nodes (18): CancelFunc, Client, Queue, New(), TestNew_InvalidType(), TestNew_Redis(), GetWorker(), Context (+10 more)

### Community 16 - "AuthHandler"
Cohesion: 0.24
Nodes (9): AuthHandler, loginRequest, refreshRequest, authContext(), Context, Duration, Service, normalizeUsername() (+1 more)

### Community 17 - "mysqlAPIKeyRepository"
Cohesion: 0.29
Nodes (5): mysqlAPIKeyRepository, APIKey, Context, DB, NewMysqlAPIKeyRepository()

### Community 18 - "New"
Cohesion: 0.26
Nodes (9): Module, AuthConfig, CaptchaConfig, NewAuthHandler(), Client, DB, EventBus, Service (+1 more)

### Community 19 - "Hub"
Cohesion: 0.17
Nodes (9): Channel, Context, RWMutex, NewWebSocket(), Connection, Hub, Registration, WebSocket (+1 more)

### Community 20 - "NewMQPublisher"
Cohesion: 0.31
Nodes (6): Context, Queue, NewMQPublisher(), TestMQPublisher_Publish(), TestMQPublisher_PublishError(), MQPublisher

### Community 21 - "JobRegistry"
Cohesion: 0.07
Nodes (16): fakeAuthzModule, fakeBusinessModule, Component, ComponentProvider, ErrorSource, EventBus, HTTPMiddlewareProvider, JobRegistry (+8 more)

### Community 22 - "AdminUserService"
Cohesion: 0.21
Nodes (9): AdminCreateUserRequest, AdminUpdateUserRequest, AdminUser, AdminUserService, ListUserFilter, userRole, Context, DB (+1 more)

### Community 23 - "Wrap"
Cohesion: 0.23
Nodes (10): AuthService, AuthServiceInterface, TokenPair, ErrAccountLocked(), Context, Duration, invalidCredentials(), normalizeUsername() (+2 more)

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
Cohesion: 0.25
Nodes (6): RedisCache, Client, Context, Duration, NewRedisCache(), randomToken()

### Community 28 - "CronScheduler"
Cohesion: 0.17
Nodes (8): Cron, EntryID, Context, RWMutex, Time, CronScheduler, JobInfo, Store

### Community 29 - "health.go"
Cohesion: 0.14
Nodes (19): Client, Context, DB, Duration, ResponseWriter, NewReadiness(), NewRedisChecker(), NewSQLChecker() (+11 more)

### Community 30 - "AuditLog"
Cohesion: 0.20
Nodes (8): fakeBatchRepository, AuditLog, fakeAuditRepository, fakeQueue, Context, Mutex, Time, Context

### Community 31 - "OK"
Cohesion: 0.16
Nodes (8): AdminConfigHandler, AdminTaskHandler, Context, Context, Context, NewAdminTaskHandler(), Context, OK()

### Community 32 - "Job"
Cohesion: 0.18
Nodes (8): Job, JobRepository, mysqlJobRepository, fakeJobRepo, Context, DB, NewMysqlJobRepository(), Time

### Community 33 - "importer_test.go"
Cohesion: 0.47
Nodes (5): csvReader(), Reader, TestCSVParseAndValidate(), TestCSVParseEmptyFile(), TestValidateUnique()

### Community 34 - "User"
Cohesion: 0.20
Nodes (7): fakeOutboxUserRepo, User, fakeUserRepository, Context, TestUpdateAndDeleteWriteOutbox(), DeletedAt, Time

### Community 35 - "OAuthService"
Cohesion: 0.20
Nodes (10): OAuthService, BindingRepository, Client, Context, DB, Duration, NewOAuthService(), refreshTTL() (+2 more)

### Community 36 - "mysqlRepository"
Cohesion: 0.31
Nodes (3): Context, DB, mysqlRepository

### Community 37 - "newLock"
Cohesion: 0.18
Nodes (16): RedisConfig, Client, newLock(), TestLock_AcquireRelease(), TestLock_ConcurrentAcquire(), TestLock_Extend(), TestLock_ReleaseOnlyOwnToken(), TestLock_WithLock() (+8 more)

### Community 38 - "InitSnowflake"
Cohesion: 0.14
Nodes (22): snowflakeModel, stringKeyModel, Field, TestIndirect(), TestInitSnowflake_InvalidWorkerID(), TestIsIntegerID(), TestRegisterSnowflakeHook_NilDB(), TestSnowflakeHook_NonIntegerPK() (+14 more)

### Community 39 - "Message"
Cohesion: 0.19
Nodes (9): Dispatcher, Channel, Context, Context, Channel, Message, SMS, fakeKafkaReader (+1 more)

### Community 40 - "NewWithStore"
Cohesion: 0.27
Nodes (12): NewMemoryStore(), TestSetEnabledNotFound(), TestSetEnabledToggle(), TestTriggerJobNotFound(), TestTriggerJobRunsCommand(), NewWithStore(), newTestLogger(), TestAddNamedFuncPersistsAndRuns() (+4 more)

### Community 41 - "New"
Cohesion: 0.19
Nodes (12): New(), TestAppError(), TestAppErrorWithCause(), TestFailDoesNotLeakInfrastructureDetails(), TestOKIncludesStableEnvelopeAndRequestID(), TestPageIncludesStableEnvelopeAndPagination(), TestCreatedUsesStandardEnvelope(), TestFailHidesInternalCauseAndIncludesRequestID() (+4 more)

### Community 42 - "Context"
Cohesion: 0.22
Nodes (5): fakeUserRepo, fakeSessionStore, sessionRecord, Context, Duration

### Community 43 - "JWT"
Cohesion: 0.20
Nodes (10): Claims, JWT, Duration, New(), NewWithRotation(), TestJWTPopulatesTypedClaims(), TestParseRejectsWrongTokenType(), AuthMiddleware() (+2 more)

### Community 44 - "gorm_logger_test.go"
Cohesion: 0.10
Nodes (28): gormLogger, Interface, Context, Duration, Time, isSensitiveField(), NewGormLogger(), sanitizeArgs() (+20 more)

### Community 45 - "DeadLetter"
Cohesion: 0.17
Nodes (8): DeadLetter, DeadLetterRepository, mysqlDeadLetterRepository, Context, DB, NewMysqlDeadLetterRepository(), Time, fakeDeadRepo

### Community 46 - "JobHistory"
Cohesion: 0.17
Nodes (9): JobHistory, JobHistoryRepository, mysqlJobHistoryRepository, Context, DB, NewMysqlJobHistoryRepository(), Time, Mutex (+1 more)

### Community 47 - "Manager"
Cohesion: 0.14
Nodes (16): contextKey, Flag, Manager, AdminFeatureHandler, NewAdminFeatureHandler(), TestAdminFeatureHandlerList(), TestAdminFeatureHandlerUpdate(), FromContext() (+8 more)

### Community 48 - "AdminUserHandler"
Cohesion: 0.26
Nodes (5): AdminUserHandler, Context, Context, NewAdminUserHandler(), paginationFromQuery()

### Community 49 - "Fail"
Cohesion: 0.28
Nodes (7): UserHandler, Context, Context, Config, DB, New(), Fail()

### Community 50 - "ChannelManager"
Cohesion: 0.10
Nodes (20): RWMutex, NewChannel(), NewChannelManager(), TestChannelManagerGetChannelMissing(), TestChannelManagerGetSubscribers(), TestChannelManagerPreCreatedBroadcast(), TestChannelManagerSubscribeCreatesWithType(), TestChannelManagerSubscribeExisting() (+12 more)

### Community 51 - "Application"
Cohesion: 0.15
Nodes (12): Application, fakeComponent, main(), run(), forwardError(), Context, Duration, NewApplication() (+4 more)

### Community 52 - "RedisStore"
Cohesion: 0.22
Nodes (9): RedisStore, Service, Client, Context, Duration, NewRedisStore(), NewService(), TestGenerate() (+1 more)

### Community 53 - "Jimu Backend Framework README"
Cohesion: 0.15
Nodes (14): Unreleased Feature Set, Error Code Ranges, Snowflake ID Primary Keys, Uniform Response Format, Contributing Guide, Snowflake Distributed ID, Event Bus, Health Checks (+6 more)

### Community 54 - "upload_handler_test.go"
Cohesion: 0.27
Nodes (22): NewUploadHandler(), Engine, Request, multipartFileRequestWithType(), newFakeStorage(), TestHandleDeleteMissingKey(), TestHandleDeleteStorageError(), TestHandleDeleteSuccess() (+14 more)

### Community 55 - "middleware/middleware_test.go"
Cohesion: 0.15
Nodes (23): CORSMiddleware(), DefaultLogConfig(), HandlerFunc, isAllowedOrigin(), Logger(), Recovery(), RequestID(), shouldLogBody() (+15 more)

### Community 56 - "NewEventBusPublisher"
Cohesion: 0.32
Nodes (6): Context, EventBus, RawMessage, NewEventBusPublisher(), EventBusPublisher, EventPayload

### Community 57 - "ws_integration_test.go"
Cohesion: 0.26
Nodes (27): NewMessage(), connID(), mustDecodeTitle(), newWSFixture(), TestClientChannels(), TestWSBroadcastAll(), TestWSChatBroadcast(), TestWSChatInvalidPayload() (+19 more)

### Community 58 - "oauth/application/service_test.go"
Cohesion: 0.23
Nodes (33): oauthStateKey(), dupSubjectProviders(), githubProviders(), Client, DB, Miniredis, newRedisClient(), newRepoResult() (+25 more)

### Community 59 - "AuditService"
Cohesion: 0.21
Nodes (10): AuditLogResponse, AuditService, AuditRepository, Change, Time, ToAuditLogResponse(), ToAuditLogResponses(), Context (+2 more)

### Community 60 - "Security"
Cohesion: 0.40
Nodes (5): TestSecurityAddVary(), addVary(), Context, HandlerFunc, Security()

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
Cohesion: 0.19
Nodes (7): fakeImportJobRepo, ImportJob, mysqlImportJobRepository, Time, Context, DB, NewMysqlImportJobRepository()

### Community 67 - "RoleService"
Cohesion: 0.15
Nodes (15): AssignPermissionsRequest, CreateRoleRequest, RoleResponse, RoleService, UpdateRoleRequest, PermissionResponse, Time, TestRoleResponseUsesDTO() (+7 more)

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
Cohesion: 0.16
Nodes (14): RouterGroup, RegisterRoleRoutes(), RouterGroup, RegisterUserRoutes(), Context, HandlerFunc, localeOf(), TestValidateJSONTranslatesFieldErrors() (+6 more)

### Community 73 - "NewMysqlRepository"
Cohesion: 0.22
Nodes (12): TestMysqlRepositoryMySQLIntegration(), NewMysqlRepository(), DB, newTestUser(), newUserTestDB(), TestMysqlRepositoryCreateAndFindByID(), TestMysqlRepositoryFindByUsername(), TestMysqlRepositoryListCountAndPagination() (+4 more)

### Community 74 - "dispatcher"
Cohesion: 0.22
Nodes (8): Channel, Context, dispatcher, RWMutex, NewDispatcher(), TestDispatcherDispatch(), TestDispatcherDispatchBatch(), Notification

### Community 75 - "KafkaQueue"
Cohesion: 0.23
Nodes (7): Context, Duration, NewKafkaQueue(), KafkaConfig, KafkaMessageReader, KafkaMessageWriter, KafkaQueue

### Community 76 - "RabbitMQQueue"
Cohesion: 0.23
Nodes (7): Context, Delivery, Duration, NewRabbitMQQueue(), RabbitMQChannel, RabbitMQConfig, RabbitMQQueue

### Community 77 - "Lock"
Cohesion: 0.33
Nodes (8): generateToken(), Client, Context, Duration, Time, NewLock(), AcquireResult, Lock

### Community 78 - "ClientHub"
Cohesion: 0.12
Nodes (13): Conn, Context, HandlerFunc, RWMutex, Time, mustEncode(), WSHandler(), RawMessage (+5 more)

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
Cohesion: 0.14
Nodes (19): HandlerFunc, Metrics(), ConfigureTrustedProxies(), formatAddr(), Engine, SetupRouter(), freeAddr(), TestConfigureTrustedProxies() (+11 more)

### Community 89 - "fakeAuditRepository"
Cohesion: 0.31
Nodes (5): fakeAuditRepository, auditAppCode(), Context, TestAuditServiceGetMapsNotFound(), TestAuditServiceListReturnsDTOAndPassesPagination()

### Community 90 - "LoginFailureTracker"
Cohesion: 0.33
Nodes (8): LockoutConfig, LoginFailureTracker, DefaultLockoutConfig(), Client, Context, Duration, lockKey(), NewLoginFailureTracker()

### Community 91 - "New"
Cohesion: 0.14
Nodes (13): RouterGroup, RegisterOAuthRoutes(), buildProviders(), Client, DB, EventBus, New(), newTestModule() (+5 more)

### Community 92 - "OAuthBinding"
Cohesion: 0.14
Nodes (9): fakeBindingRepo, OAuthBinding, fakeBindingRepo, fakeSessionStore, Time, Context, newTestBinding(), Context (+1 more)

### Community 93 - "NewManagementServer"
Cohesion: 0.22
Nodes (7): Handler, passingChecker, HealthRouter(), NewManagementServer(), Context, TestManagementRouterExposure(), TestNewManagementServer()

### Community 94 - "Implementation Plan: Jimu 后端框架能力规格"
Cohesion: 0.07
Nodes (26): Content Quality, Feature Readiness, Notes, Requirement Completeness, Specification Quality Checklist: Jimu 后端框架能力规格, Complexity Tracking, Constitution Check, Documentation (this feature) (+18 more)

### Community 95 - "Permission"
Cohesion: 0.05
Nodes (38): CreatePermissionRequest, fakePermissionRepository, PermissionService, UpdatePermissionRequest, Permission, PermissionRepository, mysqlPermissionRepository, fakePermissionRepository (+30 more)

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

### Community 101 - "NewLogChannel"
Cohesion: 0.29
Nodes (6): Channel, Context, NewLogChannel(), TestLogChannelSendBatchNoError(), TestLogChannelSendNoError(), LogChannel

### Community 102 - "CI Workflow"
Cohesion: 0.27
Nodes (11): CI Build Job, CI Docker Build Job, CI Lint Job, CI Security Scan Job, CI Image Smoke Test Job, CI Test Job, CI Workflow, GolangCI Lint Configuration (+3 more)

### Community 103 - "Jimu Constitution"
Cohesion: 0.13
Nodes (15): Principle II: API Stability & SemVer, Principle I: Business-Agnostic, Jimu Constitution, Principle VI: Documentation & Examples, Governance & Revision Process, Principle IV: Simplicity & Minimal Dependencies, Principle V: Verification & Quality, Constitution Template (+7 more)

### Community 104 - "NewUserService"
Cohesion: 0.14
Nodes (12): recordingOutboxStore, Cache, NewUserService(), appCode(), createOutboxUserService(), TestCreateWritesOutbox(), TestUserResponseDoesNotContainPassword(), TestUserServiceGetMapsNotFound() (+4 more)

### Community 105 - "AdminMonitoringService"
Cohesion: 0.13
Nodes (17): AdminMonitoringService, HealthStatus, MemoryStats, SystemStatus, AdminMonitoringHandler, Client, Context, Time (+9 more)

### Community 106 - "Logger"
Cohesion: 0.17
Nodes (11): AtomicLevel, Context, LogConfig, New(), TestFileOutput(), TestNewConsoleStdout(), TestNewJSONLevels(), TestSetLevel() (+3 more)

### Community 107 - "New"
Cohesion: 0.28
Nodes (13): DBConfig, configurePool(), ConnectWithRetry(), dsn(), Context, DB, New(), open() (+5 more)

### Community 108 - "Jimu API Swagger Definition"
Cohesion: 0.33
Nodes (10): Jimu API Swagger Definition, Audit Log Endpoints, Auth Flow Endpoints (login/refresh/logout/register), BearerAuth Security (API Key JWT), Captcha Endpoint, contract.ErrorResponse, OAuth Endpoints, contract.PageResponse (+2 more)

### Community 110 - "bindImportFile"
Cohesion: 0.18
Nodes (16): AdminImportHandler, DB, newSqliteDB(), bindImportFile(), Buffer, Context, NewAdminImportHandler(), DB (+8 more)

### Community 111 - "Role"
Cohesion: 0.09
Nodes (19): fakeRoleRepository, Role, RoleRepository, rolePermission, fakeRoleRepository, NewRoleService(), Context, roleAppCode() (+11 more)

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
Cohesion: 0.17
Nodes (12): BenchmarkWebhookSend(), B, Channel, Client, Context, NewWebhook(), TestWebhookSendBatch(), TestWebhookSendNilClient() (+4 more)

### Community 117 - "NewAdminConfigService"
Cohesion: 0.14
Nodes (20): AdminConfigService, Client, Context, EventBus, NewAdminConfigService(), Miniredis, newConfigTestService(), TestAdminConfigServiceConfigKey() (+12 more)

### Community 118 - "Limiter"
Cohesion: 0.19
Nodes (13): Limiter, Context, Duration, Scripter, limitKey(), NewLimiter(), newTestLimiter(), TestLimiterAllowsOnlyConfiguredWindow() (+5 more)

### Community 119 - "Now"
Cohesion: 0.18
Nodes (16): NewPresenceManager(), TestPresenceIsOnline(), TestPresenceIsTyping(), TestPresenceManagerAllPresences(), TestPresenceManagerHeartbeat(), TestPresenceManagerHeartbeatRestoresOnlineStatus(), TestPresenceManagerOfflineMissing(), TestPresenceManagerOnlineCountFiltersOffline() (+8 more)

### Community 120 - "Server Service"
Cohesion: 0.28
Nodes (9): Jimu Collaboration Rules (CLAUDE.md), Development Config (app.yaml), Production Config (app.prod.yaml), Development Environment, Adminer Service, MariaDB Service, Server Service, Configuration Reference Table (+1 more)

### Community 121 - "UserService"
Cohesion: 0.24
Nodes (10): BatchDeleteRequest, BatchResult, CreateUserRequest, UpdateUserRequest, UserResponse, UserService, Time, ToUserResponse() (+2 more)

### Community 122 - ".upload"
Cohesion: 0.22
Nodes (11): FileHeader, UploadConfig, UploadHandler, UploadResponse, Context, HandlerFunc, Reader, isAllowedType() (+3 more)

### Community 123 - "统一响应契约"
Cohesion: 0.13
Nodes (13): Admin 模块, Auth 模块, HTTP API 契约, 中间件, 健康与观测（非业务）, 完整接口文档, 分页, 国际化 (+5 more)

### Community 124 - "Module"
Cohesion: 0.29
Nodes (3): Client, EventBus, Module

### Community 125 - "migrate_test.go"
Cohesion: 0.20
Nodes (16): AutoMigrate(), findUp(), DB, isDir(), Migrate(), MigrateWithRetry(), MigrationDir(), runMigration() (+8 more)

### Community 126 - "gzipRouter"
Cohesion: 0.17
Nodes (12): HandlerFunc, ResponseWriter, GzipCompression(), isAlreadyCompressed(), Engine, gzipRouter(), TestGzipCompressionEncodesBody(), TestGzipCompressionNoAcceptEncoding() (+4 more)

### Community 127 - "NewWeChatProvider"
Cohesion: 0.27
Nodes (6): Client, Config, Context, NewWeChatProvider(), WeChatConfig, WeChatProvider

### Community 128 - "NewUserHandler"
Cohesion: 0.24
Nodes (7): NewUserHandler(), Context, fakeUserRepository, TestUserCreateReturnsCreatedDTO(), TestUserDeleteReturnsNoContent(), TestUserListRejectsInvalidPaginationContract(), TestUserListUsesDefaultPaginationBeforeService()

### Community 129 - "ImportService"
Cohesion: 0.28
Nodes (8): ImportService, ImportJobRepository, Context, DB, Reader, NewImportService(), rulesFor(), TestRulesFor()

### Community 130 - "validator/validator.go"
Cohesion: 0.39
Nodes (6): FieldLevel, Validate(), validateIDCard(), validateMobile(), validatePassword(), validateUsername()

### Community 131 - "newSqliteDB"
Cohesion: 0.38
Nodes (9): DB, newSqliteDB(), DB, newImportService(), TestImportServiceGetImportJob(), TestImportServiceImport(), TestImportServiceInsertUser(), TestImportServicePreview() (+1 more)

### Community 132 - "newWSHandler"
Cohesion: 0.32
Nodes (7): AdminWSHandler, Context, NewAdminWSHandler(), newWSHandler(), TestAdminWSHandlerOnlineUsers(), TestAdminWSHandlerPresence(), TestAdminWSHandlerPush()

### Community 133 - "oauth/interfaces/handler_test.go"
Cohesion: 0.32
Nodes (15): OAuthHandler, NewOAuthHandler(), assertCode(), doRequest(), githubProvider(), Engine, newTestHandler(), newTestRouter() (+7 more)

### Community 134 - "Context"
Cohesion: 0.25
Nodes (4): fakeAPIKeyRepo, APIKey, fakeUserRepository, Context

### Community 135 - "NewEmail"
Cohesion: 0.22
Nodes (10): buildEmailHeaders(), Channel, Context, NewEmail(), TestBuildEmailHeaders(), TestEmailSendEmptyRecipient(), TestEmailSendMissingConfig(), TestEmailSendSMTPIntegration() (+2 more)

### Community 136 - "New"
Cohesion: 0.14
Nodes (25): fakeUserRepository, NewAdminJobHandler(), TestAdminJobHandlerGet(), TestAdminJobHandlerList(), TestAdminJobHandlerListDeadLetters(), TestAdminJobHandlerResolveDeadLetter(), TestAdminJobHandlerRetry(), TestAdminJobHandlerSubmit() (+17 more)

### Community 137 - "Context"
Cohesion: 0.29
Nodes (6): fakeSessionStore, sessionRecord, Context, Duration, TestRefreshTTLPositive(), sessionRecord

### Community 138 - "SecurityHeadersFromConfig"
Cohesion: 0.27
Nodes (9): SecurityConfig, DefaultSecurityConfig(), TestSecurityHeadersMiddleware(), TestSecurityHeadersNilValuesSkipped(), TestSecurityHeadersRespectsCustomValues(), Context, HandlerFunc, SecurityHeadersFromConfig() (+1 more)

### Community 139 - "Redis Service"
Cohesion: 0.33
Nodes (7): Redis Service, Graphical Captcha, Redis Distributed Lock, Rate Limiting, Cron Scheduler, Unified Auth (JWT + Casbin + API Key), Account Lockout

### Community 140 - "AuthorizationMiddleware"
Cohesion: 0.22
Nodes (12): DBAuthorizationStore, AuthorizationMiddleware(), Context, DB, Enforcer, HandlerFunc, NewDBAuthorizationStore(), Enforcer (+4 more)

### Community 141 - "newMockGormDB"
Cohesion: 0.24
Nodes (14): MySQLBindingRepository, DB, NewMySQLBindingRepository(), bindingRows(), DB, Sqlmock, newMockGormDB(), TestCreateError() (+6 more)

### Community 142 - "NewSMS"
Cohesion: 0.43
Nodes (5): NewSMS(), TestSMSAliyunReject(), TestSMSAliyunSend(), TestSMSUnknownProvider(), SMSConfig

### Community 143 - "Context"
Cohesion: 0.14
Nodes (16): NewMySQLStore(), NewWorkerPool(), RegisterWorker(), fakeStore(), Context, Duration, MySQLStore, newFakeJobRepo() (+8 more)

### Community 144 - "RunSeed"
Cohesion: 0.22
Nodes (15): basePermissions(), DB, RunSeed(), RunSeedWithCasbin(), SeedCasbinPolicies(), expectRunSeedQueries(), Sqlmock, repoRoot() (+7 more)

### Community 145 - "setupTestCache"
Cohesion: 0.52
Nodes (6): setupTestCache(), TestRedisCache_Delete(), TestRedisCache_DeletePattern(), TestRedisCache_GetOrSet(), TestRedisCache_GetOrSetStampede(), TestRedisCache_SetAndGet()

### Community 146 - "EventBus"
Cohesion: 0.19
Nodes (9): EventBus, Handler, RWMutex, New(), TestEventBus_Clear(), TestEventBus_HandlerPanicRecovered(), TestEventBus_MultipleHandlers(), TestEventBus_PublishAsync() (+1 more)

### Community 147 - "Timeout"
Cohesion: 0.22
Nodes (6): Duration, HandlerFunc, TestTimeoutOnlyPropagatesContextDeadline(), Timeout(), RouterGroup, RegisterSwagger()

### Community 148 - "UserInfo"
Cohesion: 0.15
Nodes (9): fakeProvider, fakeProvider, Client, Config, Context, NewGitHubProvider(), GitHubConfig, GitHubProvider (+1 more)

### Community 149 - "NewGoogleProvider"
Cohesion: 0.27
Nodes (6): Client, Config, Context, NewGoogleProvider(), GoogleConfig, GoogleProvider

### Community 150 - "NewDBAPIKeyStore"
Cohesion: 0.33
Nodes (12): createKey(), APIKey, DB, newTestDB(), TestDBAPIKeyStore_GetByKeyHash(), TestDBAPIKeyStore_GetByKeyHashNotFound(), TestDBAPIKeyStore_UpdateLastUsed(), TestVerifyWithDBStore() (+4 more)

### Community 151 - "InitTracing"
Cohesion: 0.48
Nodes (6): DefaultTracingConfig(), Context, TracerProvider, InitTracing(), ShutdownTracing(), TracingConfig

### Community 152 - "newFakeSMTPServer"
Cohesion: 0.47
Nodes (4): Conn, newFakeSMTPServer(), Listener, fakeSMTPServer

### Community 153 - "Duration"
Cohesion: 0.43
Nodes (4): Duration, ExponentialBackoff, FixedRetry, RetryStrategy

### Community 154 - "NewAdminUserService"
Cohesion: 0.23
Nodes (10): testRole, UserRepository, NewAdminUserService(), TestAdminUserServiceAssignRoles(), TestAdminUserServiceAssignRolesRoleQueryError(), TestAdminUserServiceCreateUser(), TestAdminUserServiceDisableUser(), TestAdminUserServiceGetUser() (+2 more)

### Community 155 - ".Validate"
Cohesion: 0.20
Nodes (11): ExcelImporter, FieldRule, FieldType, ValidationRules, Validator, Context, Reader, NewExcelImporter() (+3 more)

### Community 156 - "setupMetricsEngine"
Cohesion: 0.60
Nodes (5): gatherHTTPMetric(), Engine, setupMetricsEngine(), TestMetricsLabelsUseRouteTemplate(), TestMetricsRecordsUnmatchedPath()

### Community 157 - "New"
Cohesion: 0.18
Nodes (10): Module, AuditHandler, Context, NewAuditHandler(), TestAuditGetInvalidIDReturnsStableBadRequest(), TestAuditGetReturnsLogDTO(), TestAuditListInvalidQueryReturnsStableBadRequest(), DB (+2 more)

### Community 158 - "fakeStorage"
Cohesion: 0.22
Nodes (5): fakeStorage, Context, Duration, ReadCloser, Reader

### Community 159 - "Release Workflow"
Cohesion: 0.40
Nodes (5): Release Workflow, Jimu Changelog, Conventional Commits, Release Note Structure, Commit Style

### Community 160 - "Module Layering & contract.Module"
Cohesion: 0.40
Nodes (5): Principle III: Composable Modules, Module Layering & contract.Module, Module Generator CLI, Clean Architecture, Scaffold CLI

### Community 161 - "Page"
Cohesion: 0.36
Nodes (10): Created(), FailWithDetails(), Context, localeFrom(), NoContent(), Page(), requestID(), StatusForCode() (+2 more)

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

### Community 168 - "testConn"
Cohesion: 0.29
Nodes (6): Conn, Duration, Mutex, newTestConn(), PresencePayload, testConn

### Community 169 - "NewServer"
Cohesion: 0.35
Nodes (9): Config, NewServer(), TestNewServerPlain(), New(), TestDoCancelsDuringRetryWait(), TestDoInjectsTraceParent(), TestDoNoRetryOn4xx(), TestDoRetriesOn5xx() (+1 more)

### Community 170 - "Context"
Cohesion: 0.11
Nodes (10): fakeAPIKeyRepo, fakeDeadLetterRepo, fakeEventBus, fakeImportJobRepo, testRole, testUserRole, APIKey, Context (+2 more)

### Community 171 - "010_create_jobs.sql"
Cohesion: 0.50
Nodes (3): dead_letters, job_history, jobs

### Community 172 - "Security Policy"
Cohesion: 0.50
Nodes (4): No .env & Secrets Hook, Password Strength Rule, Security Policy, Vulnerability Reporting Process

### Community 174 - "CaptchaHandler"
Cohesion: 0.83
Nodes (3): CaptchaHandler, Service, NewCaptchaHandler()

### Community 177 - "ProtectedMiddleware"
Cohesion: 0.18
Nodes (10): AuthorizationStore, Enforcer, HandlerFunc, ProtectedMiddleware(), TestProtectedMiddlewareRequiresAccessToken(), HandlerFunc, DB, Enforcer (+2 more)

### Community 178 - "newMockGormDB"
Cohesion: 0.25
Nodes (13): DB, Sqlmock, newMockGormDB(), TestConfigurePool(), DB, TestTransaction_Commit(), TestTransaction_Rollback(), TestWithTx_BeginError() (+5 more)

### Community 183 - "Format"
Cohesion: 0.54
Nodes (5): Format, Importer, Registry, NewRegistry(), TestRegistryGetUnsupported()

### Community 210 - "mockTokenServer"
Cohesion: 0.47
Nodes (9): Client, mockClient(), mockTokenServer(), TestGitHubProviderExchange(), TestGoogleProviderExchange(), TestGoogleProviderExchangeBadUserInfo(), TestGoogleProviderExchangeTokenError(), TestWeChatProviderExchange() (+1 more)

### Community 212 - "Outbox"
Cohesion: 0.42
Nodes (5): Context, New(), Outbox, Publisher, Store

### Community 213 - "generator/module_test.go"
Cohesion: 0.44
Nodes (8): assertNoGeneratedFiles(), newTestRepository(), requiredFiles(), TestGenerateModuleCreatesCompleteCRUD(), TestGenerateModuleDoesNotOverwriteExistingTarget(), TestGenerateModuleRejectsInvalidNames(), TestGenerateModuleRollsBackWriteFailure(), writeMigration()

### Community 214 - "TestWebSocketNotification"
Cohesion: 0.33
Nodes (5): NewHub(), TestHubLifecycle(), TestWebSocketNotification(), waitHub(), mockConn

### Community 215 - "As"
Cohesion: 0.36
Nodes (5): AppError, ErrorInfo, AllErrorCodes(), As(), HTTPStatus()

### Community 216 - "Client"
Cohesion: 0.32
Nodes (6): Client, Context, Duration, Request, Response, Tracer

### Community 217 - "newHistoryTestDB"
Cohesion: 0.40
Nodes (4): TestMysqlDeadLetterRepositoryCRUD(), DB, newHistoryTestDB(), TestMysqlJobHistoryRepositoryCreateAndList()

### Community 218 - "newTestScheduler"
Cohesion: 0.62
Nodes (6): NewAdminTaskService(), newTestScheduler(), TestAdminTaskServiceGetHistory(), TestAdminTaskServiceListTasks(), TestAdminTaskServiceToggleTask(), TestAdminTaskServiceTriggerTask()

### Community 219 - "NewHandler"
Cohesion: 0.32
Nodes (5): Handler, Context, Service, NewHandler(), TestHandlerGetErrorCodes()

### Community 220 - "mockNotification"
Cohesion: 0.47
Nodes (3): Channel, Context, mockNotification

### Community 222 - "Policy"
Cohesion: 0.31
Nodes (5): fakeAuthorizationStore, Policy, fakeAuthzStore, Context, Context

### Community 224 - "PermissionMiddleware"
Cohesion: 0.50
Nodes (3): Enforcer, HandlerFunc, PermissionMiddleware()

### Community 236 - "NewAdminAuditHandler"
Cohesion: 0.40
Nodes (4): AdminAuditHandler, DB, NewAdminAuditHandler(), TestAdminAuditHandlerList()

### Community 238 - "TestGeneratedModuleCompiles"
Cohesion: 0.73
Nodes (5): copyGoSum(), copyRootFile(), TestGeneratedModuleCompiles(), writeFileForTest(), writeStubPackages()

### Community 239 - "newRepoTestDB"
Cohesion: 0.53
Nodes (5): DB, newRepoTestDB(), TestMysqlAPIKeyRepository(), TestMysqlImportJobRepository(), TestMysqlJobRepository()

### Community 240 - "admin/module_test.go"
Cohesion: 0.29
Nodes (4): fakeEventBus, TestModuleInitWSIdempotent(), TestModuleNameAndNew(), TestModuleWSHandler()

## Knowledge Gaps
- **202 isolated node(s):** `common.sh script`, `jimu`, `CaptchaResult`, `LogConfig`, `UserCreatedEvent` (+197 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **48 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `T()` connect `T` to `NewSnowflake`, `RegisterAuthRoutes`, `LocalStorage`, `NewAdminAPIKeyService`, `New`, `message.go`, `WorkerPool`, `NewMQPublisher`, `JobRegistry`, `Worker`, `health.go`, `importer_test.go`, `User`, `OAuthService`, `newLock`, `InitSnowflake`, `NewWithStore`, `New`, `JWT`, `gorm_logger_test.go`, `Manager`, `ChannelManager`, `Application`, `RedisStore`, `upload_handler_test.go`, `middleware/middleware_test.go`, `ws_integration_test.go`, `oauth/application/service_test.go`, `Security`, `config/config_test.go`, `NewUserRateLimiter`, `RoleService`, `NewService`, `NewCleanupService`, `MySQLStore`, `ValidateJSON`, `NewMysqlRepository`, `dispatcher`, `auth/application/service_test.go`, `i18n.go`, `MySQLStore`, `.Consume`, `TestDB`, `SetupRouter`, `fakeAuditRepository`, `New`, `NewManagementServer`, `Permission`, `AuditMiddleware`, `adminAuthRouter`, `IdempotencyMiddleware`, `DefaultCSRFConfig`, `RateLimiter`, `NewLogChannel`, `NewUserService`, `AdminMonitoringService`, `Logger`, `New`, `bindImportFile`, `Role`, `Bootstrap`, `newRedisTestQueue`, `signature_test.go`, `NewWebhook`, `NewAdminConfigService`, `Limiter`, `Now`, `.upload`, `migrate_test.go`, `gzipRouter`, `NewUserHandler`, `ImportService`, `newSqliteDB`, `newWSHandler`, `oauth/interfaces/handler_test.go`, `NewEmail`, `New`, `Context`, `SecurityHeadersFromConfig`, `AuthorizationMiddleware`, `newMockGormDB`, `NewSMS`, `Context`, `RunSeed`, `setupTestCache`, `EventBus`, `Timeout`, `NewDBAPIKeyStore`, `newFakeSMTPServer`, `NewAdminUserService`, `setupMetricsEngine`, `New`, `Page`, `testConn`, `NewServer`, `ProtectedMiddleware`, `newMockGormDB`, `Format`, `mockTokenServer`, `generator/module_test.go`, `TestWebSocketNotification`, `newHistoryTestDB`, `newTestScheduler`, `NewHandler`, `NewAdminAuditHandler`, `TestGeneratedModuleCompiles`, `newRepoTestDB`, `admin/module_test.go`?**
  _High betweenness centrality (0.575) - this node is a cross-community bridge._
- **Why does `Now()` connect `Now` to `NewSnowflake`, `ImportService`, `ImportResult`, `auth/apikey.go`, `NewAdminAPIKeyService`, `Context`, `T`, `AuthorizationMiddleware`, `newMockGormDB`, `JobData`, `WorkerPool`, `PresenceManager`, `mysqlAPIKeyRepository`, `NewDBAPIKeyStore`, `NewAdminUserService`, `.Validate`, `RedisCache`, `health.go`, `CronScheduler`, `NewWithStore`, `testConn`, `JWT`, `gorm_logger_test.go`, `DeadLetter`, `Fail`, `middleware/middleware_test.go`, `ws_integration_test.go`, `MySQLStore`, `NewUserRateLimiter`, `JobDef`, `NewService`, `NewCleanupService`, `NewMysqlRepository`, `Lock`, `ClientHub`, `MySQLStore`, `TestWebSocketNotification`, `SetupRouter`, `OAuthBinding`, `AuditMiddleware`, `DefaultCSRFConfig`, `AdminMonitoringService`, `newRedisTestQueue`, `signature_test.go`, `.upload`?**
  _High betweenness centrality (0.086) - this node is a cross-community bridge._
- **Why does `Wrap()` connect `Wrap` to `ImportService`, `OAuthService`, `RoleService`, `NewAdminAPIKeyService`, `New`, `AuthorizationMiddleware`, `AdminJobHandler`, `AdminUserService`, `As`, `UserService`, `.upload`, `AuditService`, `Permission`?**
  _High betweenness centrality (0.046) - this node is a cross-community bridge._
- **Are the 4 inferred relationships involving `T()` (e.g. with `translateValidationDetails()` and `ValidateJSON()`) actually correct?**
  _`T()` has 4 INFERRED edges - model-reasoned connections that need verification._
- **Are the 81 inferred relationships involving `New()` (e.g. with `.CreateKey()` and `.ToggleTask()`) actually correct?**
  _`New()` has 81 INFERRED edges - model-reasoned connections that need verification._
- **Are the 78 inferred relationships involving `Fail()` (e.g. with `.Generate()` and `.HandleDelete()`) actually correct?**
  _`Fail()` has 78 INFERRED edges - model-reasoned connections that need verification._
- **Are the 77 inferred relationships involving `Now()` (e.g. with `.CreateKey()` and `.Import()`) actually correct?**
  _`Now()` has 77 INFERRED edges - model-reasoned connections that need verification._