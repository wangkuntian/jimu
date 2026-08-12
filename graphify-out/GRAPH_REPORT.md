# Graph Report - jimu  (2026-08-12)

## Corpus Check
- 322 files · ~102,289 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2593 nodes · 5159 edges · 219 communities (170 shown, 49 thin omitted)
- Extraction: 84% EXTRACTED · 16% INFERRED · 0% AMBIGUOUS · INFERRED: 819 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `b98c1b3e`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- Role
- ImportResult
- New
- LocalStorage
- fakeRedis
- auth/apikey.go
- common.sh
- Jimu Helm Chart Values
- AdminAPIKeyService
- T
- Page
- WSMessage
- generator/module.go
- JobData
- PresenceManager
- WorkerPool
- AuthHandler
- UserService
- RegisterAuthRoutes
- Hub
- NewMQPublisher
- fakeAuthzModule
- AdminUserService
- Wrap
- config/config.go
- Worker
- session.go
- RedisCache
- CronScheduler
- health.go
- AuditLog
- OK
- Context
- New
- Context
- .Login
- mysqlRepository
- newLock
- id.go
- Message
- NewWithStore
- New
- User
- JWT
- gormLogger
- DeadLetter
- JobHistory
- Manager
- AdminUserHandler
- Fail
- ChannelManager
- Application
- RedisStore
- Jimu Backend Framework README
- .upload
- SetupRouter
- NewEventBusPublisher
- AdminFeatureHandler
- Bootstrap
- AuditService
- securityRouter
- MySQLStore
- mysqlAuditRepository
- Data Model: Jimu 框架持久化实体
- config_test.go
- ratelimit_user.go
- fakeUserRepository
- queue/worker_test.go
- JobDef
- fakeRouter
- cleanup.go
- MySQLStore
- ValidateJSON
- NewMysqlRepository
- dispatcher
- KafkaQueue
- RabbitMQQueue
- Lock
- Client
- routerLimiterRedis
- auth/application/service_test.go
- i18n.go
- MySQLStore
- .Consume
- TestDB
- .RegisterHTTP
- NewContainer
- AdminTaskService
- AuditRepository
- New
- LoginFailureTracker
- New
- OAuthBinding
- TestManagementRouterExposure
- Implementation Plan: Jimu 后端框架能力规格
- Permission
- AuditMiddleware
- adminAuthRouter
- responseBodyWriter
- CSRF
- RateLimiter
- NewLogChannel
- CI Workflow
- Jimu Constitution
- contract/module.go
- AdminMonitoringService
- Logger
- DBConfig
- Jimu API Swagger Definition
- NewServer
- bindImportFile
- RoleService
- fakeContainer
- Tasks: Jimu 后端框架能力规格
- newRedisTestQueue
- signature.go
- Webhook
- AdminConfigService
- Limiter
- Service
- Server Service
- openTestDB
- As
- 统一响应契约
- New
- migrate.go
- gzipResponseWriter
- WeChatProvider
- Outbox
- fakeAuditRepository
- validator/validator.go
- AdminAPIKeyHandler
- AdminWSHandler
- OAuthService
- RegisterUserRoutes
- NewEmail
- New
- UserInfo
- SecurityConfig
- Redis Service
- RegisterSnowflakeHook
- AdminConfigHandler
- NewSMS
- testWorker
- TestReadinessBoundsCheckerDuration
- setupTestCache
- EventBus
- Timeout
- GitHubProvider
- GoogleProvider
- TestGeneratedModuleCompiles
- InitTracing
- New
- Duration
- newHistoryTestDB
- JobRegistry
- Metrics
- Module
- Pagination
- Release Workflow
- Module Layering & contract.Module
- fakeComponent
- 配置契约
- Module 契约
- workerPoolComponent
- events.go
- Prometheus Service
- CLI 契约
- Router
- RegisterRoleRoutes
- 010_create_jobs.sql
- Security Policy
- 001_create_users.sql
- CaptchaHandler
- .validateCommon
- 002_create_roles.sql
- transaction.go
- 003_create_permissions.sql
- openapi.go
- 004_create_user_roles.sql
- TestValidateJSONTranslatesFieldErrors
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

## God Nodes (most connected - your core abstractions)
1. `T()` - 252 edges
2. `Fail()` - 80 edges
3. `Client` - 55 edges
4. `OK()` - 53 edges
5. `Now()` - 52 edges
6. `Wrap()` - 47 edges
7. `New()` - 37 edges
8. `User` - 37 edges
9. `JobData` - 33 edges
10. `AuditLog` - 31 edges

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

## Communities (219 total, 49 thin omitted)

### Community 0 - "Role"
Cohesion: 0.08
Nodes (22): fakeRoleRepository, Role, RoleRepository, rolePermission, fakeRoleRepository, NewRoleService(), Context, roleAppCode() (+14 more)

### Community 1 - "ImportResult"
Cohesion: 0.05
Nodes (42): ImportService, ImportJob, ImportJobRepository, CSVImporter, ExcelImporter, FieldRule, FieldType, Format (+34 more)

### Community 2 - "New"
Cohesion: 0.48
Nodes (11): Engine, newRouterLimiter(), testAuthConfig(), TestLoginRateLimitUsesIPScope(), TestLoginRateLimitUsesNormalizedUsernameScope(), TestLogoutRouteRequiresAccessToken(), TestRefreshRouteStaysPublic(), TestRegisterRateLimitUsesIPScope() (+3 more)

### Community 3 - "LocalStorage"
Cohesion: 0.06
Nodes (30): New(), newOSSStorage(), Context, Duration, ReadCloser, Reader, NewLocalStorage(), TestLocalStorageFilePersisted() (+22 more)

### Community 4 - "fakeRedis"
Cohesion: 0.12
Nodes (13): fakePipeline, fakeRedis, BoolCmd, Cmder, IntCmd, BoolSliceCmd, Cmd, Context (+5 more)

### Community 5 - "auth/apikey.go"
Cohesion: 0.12
Nodes (26): APIKey, APIKeyContextKey, APIKeyStore, APIKeyVerifier, dbAPIKeyStore, APIKeyFromContext(), ContextWithAPIKey(), createKey() (+18 more)

### Community 6 - "common.sh"
Cohesion: 0.08
Nodes (17): check-prerequisites.sh script, check_dir(), check_file(), get_feature_paths(), get_repo_root(), has_jq(), _persist_feature_json(), resolve_specify_init_dir() (+9 more)

### Community 7 - "Jimu Helm Chart Values"
Cohesion: 0.12
Nodes (35): Grafana Prometheus Datasource, Jimu Helm Chart, Helm ConfigMap Template, Helm Deployment Template, Helm HorizontalPodAutoscaler Template, Helm Ingress Template, Helm NetworkPolicy Template, Helm PodDisruptionBudget Template (+27 more)

### Community 8 - "AdminAPIKeyService"
Cohesion: 0.11
Nodes (14): AdminAPIKeyService, CreateKeyInput, APIKey, APIKeyRepository, mysqlAPIKeyRepository, APIKey, Context, NewAdminAPIKeyService() (+6 more)

### Community 9 - "T"
Cohesion: 0.09
Nodes (24): assertQueryParams(), readOpenAPI(), TestOpenAPIIncludesCRUDContract(), TestLocaleParsesAcceptLanguage(), TestGoogleAuthURL(), TestProvidersImplementInterface(), TestKafkaQueue_ConsumeUnmarshalError(), TestKafkaQueue_SubmitConsume() (+16 more)

### Community 10 - "Page"
Cohesion: 0.40
Nodes (9): Created(), FailWithDetails(), Context, localeFrom(), Page(), requestID(), StatusForCode(), Body (+1 more)

### Community 11 - "WSMessage"
Cohesion: 0.14
Nodes (10): RawMessage, Time, NewMessage(), ChatPayload, NotificationPayload, PingPayload, PongPayload, PresencePayload (+2 more)

### Community 12 - "generator/module.go"
Cohesion: 0.20
Nodes (20): targetFile, templateData, camel(), GenerateModule(), GenerateModuleAt(), lowerCamel(), mkdirAllTracked(), nextMigrationNumber() (+12 more)

### Community 13 - "JobData"
Cohesion: 0.17
Nodes (8): Context, Duration, NewRedisQueue(), Duration, fakeConsumer, fakeProducer, JobData, RedisQueue

### Community 14 - "PresenceManager"
Cohesion: 0.13
Nodes (8): RWMutex, Time, NewPresenceManager(), TestPresenceGet(), TestPresenceOnlineOffline(), TestPresenceOnlineUsers(), Presence, PresenceManager

### Community 15 - "WorkerPool"
Cohesion: 0.15
Nodes (11): CancelFunc, GetWorker(), Context, Duration, MySQLStore, Consumer, Queue, WorkerConfig (+3 more)

### Community 16 - "AuthHandler"
Cohesion: 0.28
Nodes (8): AuthHandler, loginRequest, refreshRequest, authContext(), Context, Duration, normalizeUsername(), writeAuthRateLimitHeaders()

### Community 17 - "UserService"
Cohesion: 0.22
Nodes (10): BatchDeleteRequest, BatchResult, CreateUserRequest, UpdateUserRequest, UserResponse, UserService, Time, ToUserResponse() (+2 more)

### Community 18 - "RegisterAuthRoutes"
Cohesion: 0.17
Nodes (12): Module, AuthConfig, CaptchaConfig, Service, NewAuthHandler(), RouterGroup, Service, RegisterAuthRoutes() (+4 more)

### Community 19 - "Hub"
Cohesion: 0.17
Nodes (9): Channel, Context, RWMutex, NewWebSocket(), Connection, Hub, Registration, WebSocket (+1 more)

### Community 20 - "NewMQPublisher"
Cohesion: 0.16
Nodes (10): Context, Queue, NewMQPublisher(), Context, Duration, TestMQPublisher_Publish(), TestMQPublisher_PublishError(), errorQueue (+2 more)

### Community 21 - "fakeAuthzModule"
Cohesion: 0.18
Nodes (5): fakeAuthzModule, fakeBusinessModule, EventBus, HandlerFunc, TestBusinessRoutesRequireProtectedMiddleware()

### Community 22 - "AdminUserService"
Cohesion: 0.18
Nodes (11): AdminCreateUserRequest, AdminUpdateUserRequest, AdminUser, AdminUserService, ListUserFilter, userRole, UserRepository, Context (+3 more)

### Community 23 - "Wrap"
Cohesion: 0.23
Nodes (10): AuthService, AuthServiceInterface, TokenPair, ErrAccountLocked(), Context, Duration, invalidCredentials(), normalizeUsername() (+2 more)

### Community 24 - "config/config.go"
Cohesion: 0.16
Nodes (23): CacheConfig, CaptchaResult, EmailConfig, HTTPConfig, IDConfig, LogConfig, ManagementConfig, OAuthConfig (+15 more)

### Community 25 - "Worker"
Cohesion: 0.29
Nodes (6): Worker, AuditConfig, Context, RWMutex, NewWorker(), Once

### Community 26 - "session.go"
Cohesion: 0.24
Nodes (11): redisClientAdapter, RedisSessionStore, sessionPipeline, sessionRedis, SessionStore, Context, Duration, Scripter (+3 more)

### Community 27 - "RedisCache"
Cohesion: 0.24
Nodes (6): Cache, RedisCache, Context, Duration, NewRedisCache(), randomToken()

### Community 28 - "CronScheduler"
Cohesion: 0.17
Nodes (8): Cron, EntryID, Context, RWMutex, Time, CronScheduler, JobInfo, Store

### Community 29 - "health.go"
Cohesion: 0.20
Nodes (14): Context, DB, Duration, ResponseWriter, NewReadiness(), NewRedisChecker(), NewSQLChecker(), RegisterHealth() (+6 more)

### Community 30 - "AuditLog"
Cohesion: 0.21
Nodes (8): fakeAuditRepository, fakeBatchRepository, AuditLog, fakeQueue, Context, Context, Mutex, Time

### Community 31 - "OK"
Cohesion: 0.18
Nodes (8): AdminMonitoringHandler, AdminTaskHandler, Context, NewAdminMonitoringHandler(), Context, NewAdminTaskHandler(), Context, OK()

### Community 32 - "Context"
Cohesion: 0.20
Nodes (8): Job, mysqlJobRepository, Context, DB, NewMysqlJobRepository(), Time, Context, fakeJobRepo

### Community 33 - "New"
Cohesion: 0.19
Nodes (11): NewUserService(), NewUserHandler(), TestUserCreateReturnsCreatedDTO(), TestUserDeleteReturnsNoContent(), TestUserListRejectsInvalidPaginationContract(), TestUserListUsesDefaultPaginationBeforeService(), Config, DB (+3 more)

### Community 34 - "Context"
Cohesion: 0.11
Nodes (14): fakeOutboxUserRepo, fakeUserRepository, recordingOutboxStore, appCode(), createOutboxUserService(), Context, TestCreateWritesOutbox(), TestUpdateAndDeleteWriteOutbox() (+6 more)

### Community 35 - ".Login"
Cohesion: 0.33
Nodes (4): Context, Duration, oauthStateKey(), refreshTTL()

### Community 36 - "mysqlRepository"
Cohesion: 0.31
Nodes (3): Context, DB, mysqlRepository

### Community 37 - "newLock"
Cohesion: 0.20
Nodes (14): RedisConfig, newLock(), TestLock_AcquireRelease(), TestLock_ConcurrentAcquire(), TestLock_Extend(), TestLock_ReleaseOnlyOwnToken(), TestLock_WithLock(), ConnectWithRetry() (+6 more)

### Community 38 - "id.go"
Cohesion: 0.16
Nodes (13): Generator, snowflake, uuidGenerator, binaryID(), Mutex, NewSnowflake(), NewUUIDGenerator(), TestNewSnowflakeValidatesWorkerID() (+5 more)

### Community 39 - "Message"
Cohesion: 0.24
Nodes (7): Channel, Context, Context, Message, SMS, fakeKafkaReader, fakeKafkaWriter

### Community 40 - "NewWithStore"
Cohesion: 0.27
Nodes (12): NewMemoryStore(), TestSetEnabledNotFound(), TestSetEnabledToggle(), TestTriggerJobNotFound(), TestTriggerJobRunsCommand(), NewWithStore(), newTestLogger(), TestAddNamedFuncPersistsAndRuns() (+4 more)

### Community 41 - "New"
Cohesion: 0.19
Nodes (12): New(), TestAppError(), TestAppErrorWithCause(), TestFailDoesNotLeakInfrastructureDetails(), TestOKIncludesStableEnvelopeAndRequestID(), TestPageIncludesStableEnvelopeAndPagination(), TestCreatedUsesStandardEnvelope(), TestFailHidesInternalCauseAndIncludesRequestID() (+4 more)

### Community 42 - "User"
Cohesion: 0.17
Nodes (8): fakeSessionStore, fakeUserRepo, sessionRecord, User, Context, Duration, DeletedAt, Time

### Community 43 - "JWT"
Cohesion: 0.06
Nodes (40): AuthorizationStore, Claims, DBAuthorizationStore, fakeAuthorizationStore, JWT, Policy, fakeAuthzStore, Enforcer (+32 more)

### Community 44 - "gormLogger"
Cohesion: 0.21
Nodes (10): gormLogger, Interface, Context, Duration, Time, isSensitiveField(), NewGormLogger(), sanitizeArgs() (+2 more)

### Community 45 - "DeadLetter"
Cohesion: 0.17
Nodes (8): DeadLetter, mysqlDeadLetterRepository, Context, DB, NewMysqlDeadLetterRepository(), Time, Mutex, fakeDeadRepo

### Community 46 - "JobHistory"
Cohesion: 0.19
Nodes (8): JobHistory, JobHistoryRepository, mysqlJobHistoryRepository, Context, DB, NewMysqlJobHistoryRepository(), Time, fakeHistoryRepo

### Community 47 - "Manager"
Cohesion: 0.21
Nodes (9): contextKey, Flag, Manager, FromContext(), Context, RWMutex, hashUserID(), NewManager() (+1 more)

### Community 48 - "AdminUserHandler"
Cohesion: 0.26
Nodes (5): AdminUserHandler, Context, Context, NewAdminUserHandler(), paginationFromQuery()

### Community 49 - "Fail"
Cohesion: 0.19
Nodes (7): PermissionHandler, UserHandler, Context, Context, Context, Context, Fail()

### Community 50 - "ChannelManager"
Cohesion: 0.19
Nodes (5): RWMutex, NewChannel(), NewChannelManager(), Channel, ChannelManager

### Community 51 - "Application"
Cohesion: 0.17
Nodes (11): Application, main(), run(), Component, forwardError(), Context, Duration, NewApplication() (+3 more)

### Community 52 - "RedisStore"
Cohesion: 0.22
Nodes (8): RedisStore, Service, Context, Duration, NewRedisStore(), NewService(), TestGenerate(), TestVerify()

### Community 53 - "Jimu Backend Framework README"
Cohesion: 0.15
Nodes (14): Unreleased Feature Set, Error Code Ranges, Snowflake ID Primary Keys, Uniform Response Format, Contributing Guide, Snowflake Distributed ID, Event Bus, Health Checks (+6 more)

### Community 54 - ".upload"
Cohesion: 0.26
Nodes (10): FileHeader, UploadConfig, UploadHandler, UploadResponse, Context, HandlerFunc, Reader, isAllowedType() (+2 more)

### Community 55 - "SetupRouter"
Cohesion: 0.33
Nodes (11): CORSMiddleware(), DefaultLogConfig(), HandlerFunc, isAllowedOrigin(), Logger(), Recovery(), RequestID(), shouldLogBody() (+3 more)

### Community 56 - "NewEventBusPublisher"
Cohesion: 0.32
Nodes (6): Context, EventBus, RawMessage, NewEventBusPublisher(), EventBusPublisher, EventPayload

### Community 57 - "AdminFeatureHandler"
Cohesion: 0.47
Nodes (3): AdminFeatureHandler, Context, NewAdminFeatureHandler()

### Community 58 - "Bootstrap"
Cohesion: 0.31
Nodes (9): moduleLogger, registerRouter, Handler, Bootstrap(), registerEventBusBridge(), registerHTTP(), registerOutboxWorkers(), HealthRouter() (+1 more)

### Community 59 - "AuditService"
Cohesion: 0.23
Nodes (8): AuditLogResponse, AuditService, Change, Time, ToAuditLogResponse(), ToAuditLogResponses(), Context, serializeChanges()

### Community 60 - "securityRouter"
Cohesion: 0.25
Nodes (9): addVary(), Context, HandlerFunc, Security(), Engine, securityRouter(), TestSecurityHandlesAllowedPreflight(), TestSecurityRejectsOversizedBody() (+1 more)

### Community 62 - "mysqlAuditRepository"
Cohesion: 0.46
Nodes (3): mysqlAuditRepository, deserializeChanges(), Context

### Community 63 - "Data Model: Jimu 框架持久化实体"
Cohesion: 0.06
Nodes (31): 1. User（用户）, 2. Role（角色）, 3. Permission（权限）, 4. APIKey（API 密钥）, 5. AuditLog（审计日志）, 6. OAuthBinding（第三方绑定）, 7. OutboxEvent（Outbox 事件）, 8. ScheduledJob（定时任务定义） (+23 more)

### Community 64 - "config_test.go"
Cohesion: 0.22
Nodes (13): Config, TestDBPasswordOverride(), TestJWTSecretOverride(), TestLoad(), TestStorageConfigFieldMapping(), TestValidateHTTPMode(), TestValidateLogFormat(), TestValidateLogLevel() (+5 more)

### Community 65 - "ratelimit_user.go"
Cohesion: 0.29
Nodes (11): defaultKeyFunc(), Context, Duration, HandlerFunc, Time, NewUserRateLimiter(), UserRateLimitMiddleware(), WithKeyFunc() (+3 more)

### Community 67 - "queue/worker_test.go"
Cohesion: 0.33
Nodes (11): NewMySQLStore(), NewWorkerPool(), RegisterWorker(), fakeStore(), MySQLStore, newFakeJobRepo(), TestWorkerPoolConsumeJob(), TestWorkerPoolOutboxEventFailureWritesDeadLetter() (+3 more)

### Community 68 - "JobDef"
Cohesion: 0.18
Nodes (8): Context, RWMutex, Context, Time, failStore, JobDef, MemoryStore, Store

### Community 69 - "fakeRouter"
Cohesion: 0.38
Nodes (5): fakeRouter, Engine, RouterGroup, newFakeRouter(), TestAdminRoutesUseAPIV1Prefix()

### Community 70 - "cleanup.go"
Cohesion: 0.29
Nodes (10): CleanupConfig, CleanupResult, CleanupService, CleanupTable, DefaultCleanupConfig(), Context, DB, Time (+2 more)

### Community 71 - "MySQLStore"
Cohesion: 0.17
Nodes (10): Context, DB, DeletedAt, Time, NewMySQLStore(), DB, testDB(), TestMySQLStore() (+2 more)

### Community 72 - "ValidateJSON"
Cohesion: 0.44
Nodes (8): Context, HandlerFunc, localeOf(), translateValidationDetails(), translateValidationMessage(), ValidateJSON(), ValidateQuery(), fieldError

### Community 73 - "NewMysqlRepository"
Cohesion: 0.22
Nodes (12): TestMysqlRepositoryMySQLIntegration(), NewMysqlRepository(), DB, newTestUser(), newUserTestDB(), TestMysqlRepositoryCreateAndFindByID(), TestMysqlRepositoryFindByUsername(), TestMysqlRepositoryListCountAndPagination() (+4 more)

### Community 74 - "dispatcher"
Cohesion: 0.21
Nodes (8): Channel, Context, dispatcher, RWMutex, NewDispatcher(), Dispatcher, Channel, Notification

### Community 75 - "KafkaQueue"
Cohesion: 0.28
Nodes (6): Context, Duration, NewKafkaQueue(), KafkaMessageReader, KafkaMessageWriter, KafkaQueue

### Community 76 - "RabbitMQQueue"
Cohesion: 0.28
Nodes (6): Context, Delivery, Duration, NewRabbitMQQueue(), RabbitMQChannel, RabbitMQQueue

### Community 77 - "Lock"
Cohesion: 0.36
Nodes (7): generateToken(), Context, Duration, Time, NewLock(), AcquireResult, Lock

### Community 78 - "Client"
Cohesion: 0.14
Nodes (14): Conn, Context, HandlerFunc, RWMutex, Time, mustEncode(), NewClientHub(), WSHandler() (+6 more)

### Community 79 - "routerLimiterRedis"
Cohesion: 0.29
Nodes (6): routerLimiterRedis, BoolSliceCmd, Cmd, Context, StringCmd, routerLimiterInt()

### Community 80 - "auth/application/service_test.go"
Cohesion: 0.50
Nodes (11): NewAuthService(), appCode(), newFakeSessionStore(), newTestService(), TestLoginCreatesRefreshSession(), TestLoginHidesCredentialFailures(), TestLoginRejectsDisabledUser(), TestLogoutRevokesSessions() (+3 more)

### Community 81 - "i18n.go"
Cohesion: 0.25
Nodes (4): HandlerFunc, Locale(), ParseAcceptLanguage(), TestParseAcceptLanguage()

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
Cohesion: 0.16
Nodes (9): Module, Handler, Context, Service, NewHandler(), DB, EventBus, HandlerFunc (+1 more)

### Community 86 - "NewContainer"
Cohesion: 0.16
Nodes (13): Container, Config, Context, DB, EventBus, Service, TracerProvider, NewContainer() (+5 more)

### Community 87 - "AdminTaskService"
Cohesion: 0.27
Nodes (6): AdminTaskService, TaskExecution, TaskInfo, Context, Time, NewAdminTaskService()

### Community 88 - "AuditRepository"
Cohesion: 0.38
Nodes (6): AuditRepository, AdminAuditHandler, DB, NewAdminAuditHandler(), DB, NewMysqlAuditRepository()

### Community 89 - "New"
Cohesion: 0.24
Nodes (11): AuditHandler, NewAuditService(), auditAppCode(), TestAuditServiceGetMapsNotFound(), TestAuditServiceListReturnsDTOAndPassesPagination(), NewAuditHandler(), TestAuditGetInvalidIDReturnsStableBadRequest(), TestAuditGetReturnsLogDTO() (+3 more)

### Community 90 - "LoginFailureTracker"
Cohesion: 0.36
Nodes (7): LockoutConfig, LoginFailureTracker, DefaultLockoutConfig(), Context, Duration, lockKey(), NewLoginFailureTracker()

### Community 91 - "New"
Cohesion: 0.28
Nodes (5): buildProviders(), DB, EventBus, New(), Module

### Community 92 - "OAuthBinding"
Cohesion: 0.25
Nodes (6): OAuthBinding, MySQLBindingRepository, Time, Context, DB, NewMySQLBindingRepository()

### Community 93 - "TestManagementRouterExposure"
Cohesion: 0.40
Nodes (3): passingChecker, Context, TestManagementRouterExposure()

### Community 94 - "Implementation Plan: Jimu 后端框架能力规格"
Cohesion: 0.07
Nodes (26): Content Quality, Feature Readiness, Notes, Requirement Completeness, Specification Quality Checklist: Jimu 后端框架能力规格, Complexity Tracking, Constitution Check, Documentation (this feature) (+18 more)

### Community 95 - "Permission"
Cohesion: 0.05
Nodes (40): CreatePermissionRequest, fakePermissionRepository, PermissionService, UpdatePermissionRequest, Permission, PermissionRepository, mysqlPermissionRepository, fakePermissionRepository (+32 more)

### Community 96 - "AuditMiddleware"
Cohesion: 0.31
Nodes (8): Queue, AuditMiddleware(), Context, HandlerFunc, isManagementPath(), optionalString(), optionalUint64(), TestAuditMiddlewareAllowsAnonymousRequest()

### Community 97 - "adminAuthRouter"
Cohesion: 0.27
Nodes (9): AdminAuth(), HandlerFunc, adminAuthRouter(), Engine, TestAdminAuthAllowsAdminRole(), TestAdminAuthAllowsEnglishAdminRole(), TestAdminAuthRejectsNonAdminRole(), TestAdminAuthRequiresAuthentication() (+1 more)

### Community 98 - "responseBodyWriter"
Cohesion: 0.24
Nodes (6): Buffer, ResponseWriter, newResponseBodyWriter(), sanitizeBody(), sanitizeJSONField(), responseBodyWriter

### Community 99 - "CSRF"
Cohesion: 0.42
Nodes (8): CSRF(), DefaultCSRFConfig(), generateToken(), Context, HandlerFunc, isSafeMethod(), setCSRFCookie(), CSRFConfig

### Community 100 - "RateLimiter"
Cohesion: 0.36
Nodes (6): GlobalRateLimit(), HandlerFunc, RWMutex, NewRateLimiter(), Limit, RateLimiter

### Community 101 - "NewLogChannel"
Cohesion: 0.29
Nodes (6): Channel, Context, NewLogChannel(), TestLogChannelSendBatchNoError(), TestLogChannelSendNoError(), LogChannel

### Community 102 - "CI Workflow"
Cohesion: 0.27
Nodes (11): CI Build Job, CI Docker Build Job, CI Lint Job, CI Security Scan Job, CI Image Smoke Test Job, CI Test Job, CI Workflow, GolangCI Lint Configuration (+3 more)

### Community 103 - "Jimu Constitution"
Cohesion: 0.13
Nodes (15): Principle II: API Stability & SemVer, Principle I: Business-Agnostic, Jimu Constitution, Principle VI: Documentation & Examples, Governance & Revision Process, Principle IV: Simplicity & Minimal Dependencies, Principle V: Verification & Quality, Constitution Template (+7 more)

### Community 104 - "contract/module.go"
Cohesion: 0.29
Nodes (6): ComponentProvider, ErrorSource, EventBus, HTTPMiddlewareProvider, Module, ProtectedHTTPMiddlewareProvider

### Community 105 - "AdminMonitoringService"
Cohesion: 0.31
Nodes (7): AdminMonitoringService, HealthStatus, MemoryStats, SystemStatus, Context, Time, NewAdminMonitoringService()

### Community 106 - "Logger"
Cohesion: 0.24
Nodes (6): AtomicLevel, Context, LogConfig, New(), Logger, SugaredLogger

### Community 107 - "DBConfig"
Cohesion: 0.47
Nodes (9): DBConfig, configurePool(), ConnectWithRetry(), dsn(), Context, DB, New(), open() (+1 more)

### Community 108 - "Jimu API Swagger Definition"
Cohesion: 0.33
Nodes (10): Jimu API Swagger Definition, Audit Log Endpoints, Auth Flow Endpoints (login/refresh/logout/register), BearerAuth Security (API Key JWT), Captcha Endpoint, contract.ErrorResponse, OAuth Endpoints, contract.PageResponse (+2 more)

### Community 109 - "NewServer"
Cohesion: 0.29
Nodes (6): Server, ConfigureTrustedProxies(), formatAddr(), Context, Engine, NewServer()

### Community 110 - "bindImportFile"
Cohesion: 0.40
Nodes (5): AdminImportHandler, bindImportFile(), Buffer, Context, NewAdminImportHandler()

### Community 111 - "RoleService"
Cohesion: 0.15
Nodes (15): AssignPermissionsRequest, CreateRoleRequest, RoleResponse, RoleService, UpdateRoleRequest, PermissionResponse, Time, TestRoleResponseUsesDTO() (+7 more)

### Community 112 - "fakeContainer"
Cohesion: 0.44
Nodes (8): bridgeFn(), fakeContainer(), newTestLogger(), TestBridgeWorkerConversionFailureErrors(), TestBridgeWorkerPublishesStrongTypeToBareTopic(), TestBridgeWorkerUnknownTypeErrors(), TestEventBusBridgePublishesToBareTopic(), TestRegisterOutboxWorkersRegistersAll()

### Community 113 - "Tasks: Jimu 后端框架能力规格"
Cohesion: 0.08
Nodes (24): Dependencies & Execution Order, Format: `[ID] [P?] [Story] Description`, Implementation for User Story 1, Implementation for User Story 2, Implementation for User Story 3, Implementation for User Story 4, Implementation Strategy, Incremental Delivery (+16 more)

### Community 114 - "newRedisTestQueue"
Cohesion: 0.48
Nodes (6): newRedisTestQueue(), TestQueueContract_SubmitConsume(), TestRedisQueueAckRemovesFromProcessing(), TestRedisQueueImplementsInterfaces(), TestRedisQueueNackRequeues(), TestRedisQueueRequeueExpired()

### Community 115 - "signature.go"
Cohesion: 0.33
Nodes (9): buildSignString(), DefaultSignatureConfig(), Context, Duration, HandlerFunc, hmacSign(), Signature(), SignRequest() (+1 more)

### Community 116 - "Webhook"
Cohesion: 0.33
Nodes (6): Channel, Context, Duration, NewWebhook(), Webhook, WebhookConfig

### Community 117 - "AdminConfigService"
Cohesion: 0.36
Nodes (4): AdminConfigService, Context, EventBus, NewAdminConfigService()

### Community 118 - "Limiter"
Cohesion: 0.19
Nodes (13): Limiter, Context, Duration, Scripter, limitKey(), NewLimiter(), newTestLimiter(), TestLimiterAllowsOnlyConfiguredWindow() (+5 more)

### Community 119 - "Service"
Cohesion: 0.40
Nodes (3): Service, Time, NewService()

### Community 120 - "Server Service"
Cohesion: 0.28
Nodes (9): Jimu Collaboration Rules (CLAUDE.md), Development Config (app.yaml), Production Config (app.prod.yaml), Development Environment, Adminer Service, MariaDB Service, Server Service, Configuration Reference Table (+1 more)

### Community 121 - "openTestDB"
Cohesion: 0.39
Nodes (8): snowflakeModel, InitSnowflake(), DB, openTestDB(), TestSnowflakeHook_AssignsID(), TestSnowflakeHook_BatchCreate(), TestSnowflakeHook_NoopWithoutInit(), TestSnowflakeHook_NoOverwriteExistingID()

### Community 122 - "As"
Cohesion: 0.33
Nodes (6): AppError, ErrorInfo, AllErrorCodes(), As(), HTTPStatus(), IsCode()

### Community 123 - "统一响应契约"
Cohesion: 0.13
Nodes (13): Admin 模块, Auth 模块, HTTP API 契约, 中间件, 健康与观测（非业务）, 完整接口文档, 分页, 国际化 (+5 more)

### Community 124 - "New"
Cohesion: 0.18
Nodes (7): RoleHandler, Context, DB, EventBus, New(), NoContent(), Module

### Community 125 - "migrate.go"
Cohesion: 0.39
Nodes (8): AutoMigrate(), findUp(), DB, isDir(), Migrate(), MigrateWithRetry(), MigrationDir(), runMigration()

### Community 126 - "gzipResponseWriter"
Cohesion: 0.28
Nodes (6): HandlerFunc, ResponseWriter, GzipCompression(), isAlreadyCompressed(), gzipResponseWriter, Writer

### Community 127 - "WeChatProvider"
Cohesion: 0.38
Nodes (4): Config, NewWeChatProvider(), WeChatConfig, WeChatProvider

### Community 128 - "Outbox"
Cohesion: 0.42
Nodes (5): Context, New(), Outbox, Publisher, Store

### Community 130 - "validator/validator.go"
Cohesion: 0.39
Nodes (6): FieldLevel, Validate(), validateIDCard(), validateMobile(), validatePassword(), validateUsername()

### Community 131 - "AdminAPIKeyHandler"
Cohesion: 0.43
Nodes (3): AdminAPIKeyHandler, Context, NewAdminAPIKeyHandler()

### Community 132 - "AdminWSHandler"
Cohesion: 0.43
Nodes (3): AdminWSHandler, Context, NewAdminWSHandler()

### Community 133 - "OAuthService"
Cohesion: 0.21
Nodes (9): OAuthService, BindingRepository, OAuthHandler, DB, NewOAuthService(), NewOAuthHandler(), RouterGroup, RegisterOAuthRoutes() (+1 more)

### Community 134 - "RegisterUserRoutes"
Cohesion: 0.22
Nodes (6): RouterGroup, RegisterUserRoutes(), Duration, HandlerFunc, IdempotencyMiddleware(), cachedResponse

### Community 135 - "NewEmail"
Cohesion: 0.16
Nodes (14): buildEmailHeaders(), Channel, Context, NewEmail(), Conn, newFakeSMTPServer(), TestBuildEmailHeaders(), TestEmailSendEmptyRecipient() (+6 more)

### Community 136 - "New"
Cohesion: 0.19
Nodes (8): DeadLetterRepository, JobRepository, AdminJobHandler, Context, NewAdminJobHandler(), AdminAuthMiddleware(), HandlerFunc, New()

### Community 137 - "UserInfo"
Cohesion: 0.25
Nodes (4): Context, Context, Context, UserInfo

### Community 138 - "SecurityConfig"
Cohesion: 0.43
Nodes (6): SecurityConfig, DefaultSecurityConfig(), Context, HandlerFunc, SecurityHeadersFromConfig(), writeSecurityHeaders()

### Community 139 - "Redis Service"
Cohesion: 0.33
Nodes (7): Redis Service, Graphical Captcha, Redis Distributed Lock, Rate Limiting, Cron Scheduler, Unified Auth (JWT + Casbin + API Key), Account Lockout

### Community 140 - "RegisterSnowflakeHook"
Cohesion: 0.38
Nodes (6): Field, DB, indirect(), isIntegerID(), RegisterSnowflakeHook(), Value

### Community 141 - "AdminConfigHandler"
Cohesion: 0.43
Nodes (3): AdminConfigHandler, Context, NewAdminConfigHandler()

### Community 142 - "NewSMS"
Cohesion: 0.43
Nodes (5): NewSMS(), TestSMSAliyunReject(), TestSMSAliyunSend(), TestSMSUnknownProvider(), SMSConfig

### Community 143 - "testWorker"
Cohesion: 0.53
Nodes (5): Duration, testWorker(), TestWorkerFlushesConfiguredBatch(), TestWorkerRejectsWhenQueueFull(), TestWorkerStopDrainsAcceptedRecords()

### Community 144 - "TestReadinessBoundsCheckerDuration"
Cohesion: 0.47
Nodes (4): Context, TestReadinessBoundsCheckerDuration(), TestReadinessStatus(), checkerFunc

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
Cohesion: 0.38
Nodes (4): Config, NewGitHubProvider(), GitHubConfig, GitHubProvider

### Community 149 - "GoogleProvider"
Cohesion: 0.38
Nodes (4): Config, NewGoogleProvider(), GoogleConfig, GoogleProvider

### Community 150 - "TestGeneratedModuleCompiles"
Cohesion: 0.73
Nodes (5): copyGoSum(), copyRootFile(), TestGeneratedModuleCompiles(), writeFileForTest(), writeStubPackages()

### Community 151 - "InitTracing"
Cohesion: 0.48
Nodes (6): DefaultTracingConfig(), Context, TracerProvider, InitTracing(), ShutdownTracing(), TracingConfig

### Community 152 - "New"
Cohesion: 0.21
Nodes (8): Queue, New(), TestNew_InvalidType(), TestNew_Redis(), Config, KafkaConfig, RabbitMQConfig, Type

### Community 153 - "Duration"
Cohesion: 0.43
Nodes (4): Duration, ExponentialBackoff, FixedRetry, RetryStrategy

### Community 154 - "newHistoryTestDB"
Cohesion: 0.40
Nodes (4): TestMysqlDeadLetterRepositoryCRUD(), DB, newHistoryTestDB(), TestMysqlJobHistoryRepositoryCreateAndList()

### Community 156 - "Metrics"
Cohesion: 0.33
Nodes (7): HandlerFunc, Metrics(), gatherHTTPMetric(), Engine, setupMetricsEngine(), TestMetricsLabelsUseRouteTemplate(), TestMetricsRecordsUnmatchedPath()

### Community 157 - "Module"
Cohesion: 0.17
Nodes (5): Module, RouterGroup, RegisterAuditRoutes(), EventBus, HandlerFunc

### Community 159 - "Release Workflow"
Cohesion: 0.40
Nodes (5): Release Workflow, Jimu Changelog, Conventional Commits, Release Note Structure, Commit Style

### Community 160 - "Module Layering & contract.Module"
Cohesion: 0.40
Nodes (5): Principle III: Composable Modules, Module Layering & contract.Module, Module Generator CLI, Clean Architecture, Scaffold CLI

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

### Community 168 - "Router"
Cohesion: 0.33
Nodes (3): Router, RouterGroup, RegisterPermissionRoutes()

### Community 171 - "010_create_jobs.sql"
Cohesion: 0.50
Nodes (3): dead_letters, job_history, jobs

### Community 172 - "Security Policy"
Cohesion: 0.50
Nodes (4): No .env & Secrets Hook, Password Strength Rule, Security Policy, Vulnerability Reporting Process

### Community 174 - "CaptchaHandler"
Cohesion: 0.83
Nodes (3): CaptchaHandler, Service, NewCaptchaHandler()

### Community 178 - "transaction.go"
Cohesion: 0.67
Nodes (3): DB, Transaction(), WithTx()

## Knowledge Gaps
- **200 isolated node(s):** `common.sh script`, `jimu`, `CaptchaResult`, `LogConfig`, `UserCreatedEvent` (+195 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **49 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `T()` connect `T` to `Role`, `ImportResult`, `New`, `LocalStorage`, `auth/apikey.go`, `NewEmail`, `Page`, `generator/module.go`, `NewSMS`, `testWorker`, `TestReadinessBoundsCheckerDuration`, `setupTestCache`, `EventBus`, `Timeout`, `NewMQPublisher`, `fakeAuthzModule`, `PresenceManager`, `TestGeneratedModuleCompiles`, `New`, `newHistoryTestDB`, `Metrics`, `New`, `Context`, `newLock`, `id.go`, `NewWithStore`, `New`, `JWT`, `Application`, `RedisStore`, `TestValidateJSONTranslatesFieldErrors`, `securityRouter`, `config_test.go`, `queue/worker_test.go`, `fakeRouter`, `MySQLStore`, `ValidateJSON`, `NewMysqlRepository`, `auth/application/service_test.go`, `i18n.go`, `MySQLStore`, `.Consume`, `TestDB`, `New`, `TestManagementRouterExposure`, `Permission`, `AuditMiddleware`, `adminAuthRouter`, `NewLogChannel`, `RoleService`, `fakeContainer`, `newRedisTestQueue`, `Limiter`, `openTestDB`?**
  _High betweenness centrality (0.346) - this node is a cross-community bridge._
- **Why does `Now()` connect `Client` to `ImportResult`, `auth/apikey.go`, `AdminAPIKeyService`, `WSMessage`, `JobData`, `PresenceManager`, `WorkerPool`, `TestReadinessBoundsCheckerDuration`, `RedisCache`, `Metrics`, `CronScheduler`, `id.go`, `NewWithStore`, `JWT`, `DeadLetter`, `Fail`, `.upload`, `SetupRouter`, `MySQLStore`, `ratelimit_user.go`, `JobDef`, `cleanup.go`, `NewMysqlRepository`, `Lock`, `MySQLStore`, `AuditMiddleware`, `CSRF`, `AdminMonitoringService`, `newRedisTestQueue`, `Service`?**
  _High betweenness centrality (0.114) - this node is a cross-community bridge._
- **Why does `Client` connect `Client` to `New`, `LocalStorage`, `OAuthService`, `RegisterUserRoutes`, `New`, `JobData`, `New`, `session.go`, `RedisCache`, `health.go`, `New`, `newLock`, `RedisStore`, `ratelimit_user.go`, `Lock`, `.RegisterHTTP`, `NewContainer`, `LoginFailureTracker`, `New`, `AdminMonitoringService`, `newRedisTestQueue`, `Webhook`, `AdminConfigService`, `Service`?**
  _High betweenness centrality (0.109) - this node is a cross-community bridge._
- **Are the 4 inferred relationships involving `T()` (e.g. with `translateValidationDetails()` and `ValidateJSON()`) actually correct?**
  _`T()` has 4 INFERRED edges - model-reasoned connections that need verification._
- **Are the 78 inferred relationships involving `Fail()` (e.g. with `.Generate()` and `.HandleDelete()`) actually correct?**
  _`Fail()` has 78 INFERRED edges - model-reasoned connections that need verification._
- **Are the 50 inferred relationships involving `OK()` (e.g. with `.Generate()` and `.HandleDelete()`) actually correct?**
  _`OK()` has 50 INFERRED edges - model-reasoned connections that need verification._
- **Are the 50 inferred relationships involving `Now()` (e.g. with `.CreateKey()` and `.Import()`) actually correct?**
  _`Now()` has 50 INFERRED edges - model-reasoned connections that need verification._