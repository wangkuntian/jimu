# Graph Report - jimu  (2026-08-14)

## Corpus Check
- 409 files · ~139,070 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3664 nodes · 8046 edges · 280 communities (214 shown, 66 thin omitted)
- Extraction: 82% EXTRACTED · 18% INFERRED · 0% AMBIGUOUS · INFERRED: 1453 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `c73fe526`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- RoleService
- oauth/application/service_test.go
- PermissionService
- New
- JWT
- SessionStore
- Server
- NewCSVExporter
- ws_integration_test.go
- Hub
- T
- DeadLetter
- Context
- api_contract_test.go
- Jimu Backend Framework README
- bindImportFile
- upload_handler_test.go
- RedisCache
- auth/apikey.go
- common.sh
- health.go
- Jimu Helm Chart Values
- Quickstart: 框架能力端到端验证
- migrate.go
- oauth/interfaces/handler_test.go
- config/config.go
- LocalStorage
- NewAdminAPIKeyService
- fakeRedis
- .RegisterHTTP
- New
- Implementation Plan: Jimu 后端框架能力规格
- NewAdminConfigService
- Wrap
- generator/module.go
- JobData
- mysqlRepository
- DefaultCSRFConfig
- User
- Fail
- Outbox
- Tasks: Jimu 后端框架能力规格
- circuit
- AuthHandler
- Context
- RedisQueue
- newResetService
- UserResponse
- NewEmail
- Context
- CronScheduler
- Manager
- NewWebhook
- fakeStorage
- AuditLog
- WorkerPool
- fakeUserRepository
- PresenceManager
- RegisterAuthRoutes
- middleware/middleware_test.go
- New
- newLock
- gorm_logger_test.go
- DBConfig
- rabbitmq_queue_test.go
- ClientHub
- JobRegistry
- AdminMonitoringService
- Context
- SetupRouter
- Client
- New
- Message
- dispatcher
- NewSnowflake
- gzipRouter
- AdminUserService
- RedisStore
- RabbitMQQueue
- OK
- queue/worker_test.go
- JobDef
- NewWithStore
- MySQLStore
- ChannelManager
- New
- TestDB
- Application
- newTestGRPCService
- mysqlAuditRepository
- EventBus
- Context
- ImportResult
- New
- OAuthService
- ValidateJSON
- NewMysqlRepository
- JobHistory
- NewUserRateLimiter
- HTTP API 契约
- NewService
- Router
- ImportService
- NewAuthService
- signature_test.go
- Lock
- message.go
- ListUsersRequest
- cleanup.go
- Redis Lock Impl
- Docker Compose Observability Stack
- mysqlRepository
- Page
- NewServer
- .Exchange
- newMockGormDB
- New
- KafkaQueue
- AuditService
- UserInfo
- AuditMiddleware
- LoginFailureTracker
- newMockGormDB
- NewCSVImporter
- routerLimiterRedis
- New
- New
- MySQLStore
- NewPresenceManager
- AdminTaskService
- Worker
- AdminJobHandler
- OAuthBinding
- .Validate
- idempotencyRouter
- adminAuthRouter
- responseBodyWriter
- mockNotification
- SMS
- Logger
- Jimu API Swagger Definition
- Server
- New
- Permission
- AGENTS.md
- mockTokenServer
- NewPermissionService
- userInfoService
- RegisterUserRoutes
- NewDBAPIKeyStore
- userinfo_grpc.pb.go
- Timeout
- TestWebSocketNotification
- GitHubProvider
- NewWeChatProvider
- Role
- mysqlAPIKeyRepository
- As
- validator/validator.go
- Registry
- AdminAPIKeyHandler
- New
- AdminUserHandler
- userinfo.pb.go
- tracing.go
- NewEventBusPublisher
- Module
- fakeRoleRepository
- ResetStore
- .validateCommon
- SecurityHeadersFromConfig
- newWSHandler
- newTestScheduler
- newRepoTestDB
- setupTestCache
- i18n.go
- Duration
- ListUsersResponse
- CaptchaHandler
- fakeAuditRepository
- NewHandler
- Now
- NewAdminUserService
- Policy
- NewManagementServer
- Metrics
- HTTPConfig
- DBCollector
- testEnforcer
- NewLogChannel
- Context
- fakeComponent
- Config Contract
- events.go
- fakeContainer
- newRedisTestQueue
- mysql/003_extensions.sql
- postgres/003_extensions.sql
- admin/module_test.go
- CLI 契约
- Release Workflow
- Security Policy
- Bootstrap
- Registry
- fakePermissionRepository
- NewRoleService
- newHistoryTestDB
- TestReadinessBoundsCheckerDuration
- Pagination
- mysql/001_core.sql
- postgres/001_core.sql
- Specification Quality Checklist: Jimu 后端框架能力规格
- GolangCI Lint Configuration
- Notification Abstraction (FR-028)
- openapi.go
- request.go
- importer_test.go
- workerPoolComponent
- PermissionMiddleware
- mysql/002_audit_outbox.sql
- postgres/002_audit_outbox.sql
- smoke_api_contract.sh
- test_runtime_security.sh
- RBAC (FR-010)
- LoginRequest
- GitHub Flow Branch Strategy
- backup.sh
- bench_ci.sh
- loadtest.sh
- restore.sh
- 4-Layer Module Architecture
- Unified Response Contract
- APIKey Entity
- AuditLog Entity
- ImportJob Entity
- OutboxEvent Entity
- Checklist Template
- Config Enum Startup Validation
- Goose Migration Convention
- Development Environment
- Grafana Dashboard Provider Config
- Bug Report Template
- Docker Images Update
- GitHub Actions Update
- Go Dependencies Update
- Feature Request Template
- Pull Request Template
- Container
- AuthorizationMiddleware
- RegisterPingServer
- S3Storage
- jimu
- Audit Logging
- Cache Abstraction
- Configuration Reference Table
- Docker & K8s Deployment
- Feature Flag
- Graceful Shutdown
- i18n Localization
- Management Server
- OAuth Login (Google/GitHub/WeChat)
- HTTP Security Boundary
- Technology Stack
- Account Lockout
- User Entity
- Field-Level Encryption (FR-041)
- gRPC Server (FR-039)
- Unified Outbound HTTP Client (FR-036)
- Jimu Backend Framework
- No Multi-Tenancy (FR-035)
- OAuth Multi-Provider (FR-012)
- Self-Service Password Reset (FR-040)
- Pluggable Queue (FR-022)
- Rate Limiting (FR-014)
- Read/Write Splitting (FR-020)

## God Nodes (most connected - your core abstractions)
1. `T()` - 715 edges
2. `New()` - 85 edges
3. `Now()` - 83 edges
4. `Fail()` - 82 edges
5. `User` - 71 edges
6. `OK()` - 55 edges
7. `New()` - 52 edges
8. `Wrap()` - 49 edges
9. `NewServer()` - 36 edges
10. `Message` - 36 edges

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
- **Observability Stack** — deploy_prometheus_prometheus_config, deploy_alertmanager_alertmanager_config, deploy_alert_rules_alert_rules, deploy_promtail_promtail_config, deploy_grafana_provisioning_datasources_prometheus_grafana_ds [EXTRACTED 0.95]
- **RBAC Entity Model** — specs_001_jimu_framework_spec_data_model_user, specs_001_jimu_framework_spec_data_model_role, specs_001_jimu_framework_spec_data_model_permission [EXTRACTED 0.95]
- **Release Gate Quality Checks** — specs_001_jimu_framework_spec_spec_release_gate, github_workflows_ci_security_scan, github_workflows_ci_coverage_gate [INFERRED 0.85]
- **Constitution Core Principles** — _specify_memory_constitution_business_agnostic, _specify_memory_constitution_api_stability, _specify_memory_constitution_composable_modules, _specify_memory_constitution_simplicity, _specify_memory_constitution_verification, _specify_memory_constitution_documentation [EXTRACTED 1.00]
- **Jimu Helm Chart Resource Set** — deploy_helm_chart, deploy_helm_values, deploy_helm_templates_configmap, deploy_helm_templates_deployment, deploy_helm_templates_hpa, deploy_helm_templates_ingress, deploy_helm_templates_networkpolicy, deploy_helm_templates_pdb, deploy_helm_templates_prometheusrule, deploy_helm_templates_service [EXTRACTED 1.00]
- **jimu-server Kubernetes Resource Set** — deploy_k8s_configmap, deploy_k8s_config_files, deploy_k8s_deployment, deploy_k8s_hpa, deploy_k8s_ingress, deploy_k8s_networkpolicy, deploy_k8s_pdb, deploy_k8s_prometheusrule, deploy_k8s_service [EXTRACTED 1.00]
- **Jimu Prometheus Alert Rules** — deploy_k8s_prometheusrule_jimuhigherrorrate, deploy_k8s_prometheusrule_jimuhighlatency, deploy_k8s_prometheusrule_jimupodcrashlooping, deploy_k8s_prometheusrule_jimupoddown, deploy_k8s_prometheusrule_jimudbpoolhigh, deploy_k8s_prometheusrule_jimureadinessfailing [EXTRACTED 1.00]

## Communities (280 total, 66 thin omitted)

### Community 0 - "RoleService"
Cohesion: 0.14
Nodes (15): AssignPermissionsRequest, CreateRoleRequest, RoleResponse, RoleService, UpdateRoleRequest, PermissionResponse, Time, TestRoleResponseUsesDTO() (+7 more)

### Community 1 - "oauth/application/service_test.go"
Cohesion: 0.23
Nodes (33): oauthStateKey(), dupSubjectProviders(), githubProviders(), Client, DB, Miniredis, newRedisClient(), newRepoResult() (+25 more)

### Community 2 - "PermissionService"
Cohesion: 0.20
Nodes (10): CreatePermissionRequest, PermissionService, UpdatePermissionRequest, PermissionResponse, Time, ToPermissionResponse(), ToPermissionResponses(), isDuplicateKey() (+2 more)

### Community 3 - "New"
Cohesion: 0.20
Nodes (13): Cipher, New(), TestBlindIndexDeterministicAndDistinct(), TestBlindIndexEmpty(), TestBlindIndexWithoutKey(), TestDecryptTamperedFails(), TestDecryptWrongKeyFails(), TestEmptyValuePassthrough() (+5 more)

### Community 4 - "JWT"
Cohesion: 0.16
Nodes (12): Claims, JWT, FuzzJWTParse(), F, Duration, New(), NewWithRotation(), TestJWTPopulatesTypedClaims() (+4 more)

### Community 5 - "SessionStore"
Cohesion: 0.22
Nodes (12): redisClientAdapter, RedisSessionStore, sessionPipeline, sessionRedis, SessionStore, Client, Context, Duration (+4 more)

### Community 6 - "Server"
Cohesion: 0.16
Nodes (15): ClientConn, Config, Server, Context, Listener, New(), dial(), Server (+7 more)

### Community 7 - "NewCSVExporter"
Cohesion: 0.15
Nodes (12): CSVExporter, ExcelExporter, Context, Writer, NewCSVExporter(), Context, Writer, NewExcelExporter() (+4 more)

### Community 8 - "ws_integration_test.go"
Cohesion: 0.17
Nodes (34): NewMessage(), connID(), Conn, Duration, Mutex, Server, Time, mustDecodeTitle() (+26 more)

### Community 9 - "Hub"
Cohesion: 0.17
Nodes (9): Channel, Context, RWMutex, NewWebSocket(), Connection, Hub, Registration, WebSocket (+1 more)

### Community 10 - "T"
Cohesion: 0.05
Nodes (54): Config, TestDBPasswordOverride(), TestJWTSecretOverride(), TestLoad(), TestStorageConfigFieldMapping(), TestValidateHTTPMode(), TestValidateLogFormat(), TestValidateLogLevel() (+46 more)

### Community 11 - "DeadLetter"
Cohesion: 0.17
Nodes (8): DeadLetter, DeadLetterRepository, mysqlDeadLetterRepository, Context, DB, NewMysqlDeadLetterRepository(), Time, fakeDeadRepo

### Community 12 - "Context"
Cohesion: 0.05
Nodes (26): fakeImportJobRepo, ImportJob, ImportJobRepository, Job, JobRepository, mysqlImportJobRepository, mysqlJobRepository, fakeAPIKeyRepo (+18 more)

### Community 13 - "api_contract_test.go"
Cohesion: 0.09
Nodes (45): apiResp, testAppDB, snowflakeModel, stringKeyModel, doJSON(), DB, Engine, RawMessage (+37 more)

### Community 14 - "Jimu Backend Framework README"
Cohesion: 0.05
Nodes (41): Principle II: API Stability & SemVer, Principle I: Business-Agnostic, Principle III: Composable Modules, Jimu Constitution, Principle VI: Documentation & Examples, Governance & Revision Process, Principle IV: Simplicity & Minimal Dependencies, Principle V: Verification & Quality (+33 more)

### Community 15 - "bindImportFile"
Cohesion: 0.17
Nodes (17): AdminImportHandler, DB, newSqliteDB(), bindImportFile(), Buffer, Context, Format, NewAdminImportHandler() (+9 more)

### Community 16 - "upload_handler_test.go"
Cohesion: 0.09
Nodes (38): FileHeader, fakeStorage, UploadConfig, UploadHandler, UploadResponse, Context, HandlerFunc, Reader (+30 more)

### Community 17 - "RedisCache"
Cohesion: 0.25
Nodes (6): RedisCache, Client, Context, Duration, NewRedisCache(), randomToken()

### Community 18 - "auth/apikey.go"
Cohesion: 0.16
Nodes (14): APIKey, APIKeyContextKey, APIKeyStore, APIKeyVerifier, dbAPIKeyStore, APIKeyFromContext(), ContextWithAPIKey(), Context (+6 more)

### Community 19 - "common.sh"
Cohesion: 0.08
Nodes (17): check-prerequisites.sh script, check_dir(), check_file(), get_feature_paths(), get_repo_root(), has_jq(), _persist_feature_json(), resolve_specify_init_dir() (+9 more)

### Community 20 - "health.go"
Cohesion: 0.19
Nodes (15): Client, Context, DB, Duration, ResponseWriter, NewReadiness(), NewRedisChecker(), NewSQLChecker() (+7 more)

### Community 21 - "Jimu Helm Chart Values"
Cohesion: 0.12
Nodes (34): Jimu Helm Chart, Helm ConfigMap Template, Helm Deployment Template, Helm HorizontalPodAutoscaler Template, Helm Ingress Template, Helm NetworkPolicy Template, Helm PodDisruptionBudget Template, Helm PrometheusRule Template (+26 more)

### Community 22 - "Quickstart: 框架能力端到端验证"
Cohesion: 0.15
Nodes (12): Quickstart: 框架能力端到端验证, V1 认证闭环（FR-008/009/010/011）, V2 运维与观测（FR-005/006/029/031）, V3 模块扩展（FR-032/033/契约）, V4 可选能力（FR-012/013/022/024/025/026/027）, V5 安全与限流（FR-014/015/016/017）, 前置条件, 环境搭建 (+4 more)

### Community 23 - "migrate.go"
Cohesion: 0.17
Nodes (19): AutoMigrate(), findUp(), DB, isDir(), Migrate(), MigrateWithRetry(), MigrationDir(), mysqlDSN() (+11 more)

### Community 24 - "oauth/interfaces/handler_test.go"
Cohesion: 0.18
Nodes (19): fakeProvider, OAuthHandler, NewOAuthHandler(), assertCode(), doRequest(), githubProvider(), Engine, ResponseRecorder (+11 more)

### Community 25 - "config/config.go"
Cohesion: 0.12
Nodes (30): main(), run(), AuditConfig, CacheConfig, CaptchaResult, EmailConfig, GRPCConfig, HTTPClientConfig (+22 more)

### Community 26 - "LocalStorage"
Cohesion: 0.09
Nodes (25): New(), newOSSStorage(), Context, Duration, ReadCloser, Reader, NewLocalStorage(), TestLocalStorageFilePersisted() (+17 more)

### Community 27 - "NewAdminAPIKeyService"
Cohesion: 0.12
Nodes (18): AdminAPIKeyService, CreateKeyInput, APIKey, APIKeyRepository, APIKey, Context, NewAdminAPIKeyService(), TestAdminAPIKeyServiceCreateKey() (+10 more)

### Community 28 - "fakeRedis"
Cohesion: 0.06
Nodes (42): fakePipeline, fakeRedis, Limiter, BoolCmd, Cmder, IntCmd, Context, Duration (+34 more)

### Community 29 - ".RegisterHTTP"
Cohesion: 0.14
Nodes (10): Module, AdminAuditHandler, DB, NewAdminAuditHandler(), TestAdminAuditHandlerList(), Client, DB, EventBus (+2 more)

### Community 30 - "New"
Cohesion: 0.19
Nodes (22): New(), basePermissions(), DB, RunSeed(), RunSeedWithCasbin(), SeedCasbinPolicies(), expectRunSeedQueries(), DB (+14 more)

### Community 31 - "Implementation Plan: Jimu 后端框架能力规格"
Cohesion: 0.22
Nodes (8): Complexity Tracking, Constitution Check, Documentation (this feature), Implementation Plan: Jimu 后端框架能力规格, Project Structure, Source Code (repository root), Summary, Technical Context

### Community 32 - "NewAdminConfigService"
Cohesion: 0.13
Nodes (21): AdminConfigService, AdminConfigHandler, Client, Context, EventBus, NewAdminConfigService(), Miniredis, newConfigTestService() (+13 more)

### Community 33 - "Wrap"
Cohesion: 0.20
Nodes (10): AuthService, AuthServiceInterface, TokenPair, ErrAccountLocked(), Context, Duration, invalidCredentials(), normalizeUsername() (+2 more)

### Community 34 - "generator/module.go"
Cohesion: 0.16
Nodes (25): targetFile, templateData, copyGoSum(), copyRootFile(), TestGeneratedModuleCompiles(), writeFileForTest(), writeStubPackages(), camel() (+17 more)

### Community 35 - "JobData"
Cohesion: 0.24
Nodes (6): Context, Duration, fakeConsumer, fakeJobRepo, fakeProducer, JobData

### Community 36 - "mysqlRepository"
Cohesion: 0.27
Nodes (3): Context, DB, mysqlRepository

### Community 37 - "DefaultCSRFConfig"
Cohesion: 0.18
Nodes (23): CSRF(), DefaultCSRFConfig(), generateToken(), Context, HandlerFunc, isSafeMethod(), setCSRFCookie(), csrfRouter() (+15 more)

### Community 38 - "User"
Cohesion: 0.12
Nodes (9): fakeOutboxUserRepo, recordingOutboxStore, User, fakeUserRepository, Context, TestCreateWritesOutbox(), TestUpdateAndDeleteWriteOutbox(), DeletedAt (+1 more)

### Community 39 - "Fail"
Cohesion: 0.35
Nodes (4): UserHandler, Context, Context, Fail()

### Community 40 - "Outbox"
Cohesion: 0.09
Nodes (22): Context, Queue, RawMessage, NewMQPublisher(), TestMQPublisher_Publish(), TestMQPublisher_PublishError(), traceFromMetadata(), Context (+14 more)

### Community 41 - "Tasks: Jimu 后端框架能力规格"
Cohesion: 0.08
Nodes (24): Dependencies & Execution Order, Format: `[ID] [P?] [Story] Description`, Implementation for User Story 1, Implementation for User Story 2, Implementation for User Story 3, Implementation for User Story 4, Implementation Strategy, Incremental Delivery (+16 more)

### Community 42 - "circuit"
Cohesion: 0.15
Nodes (15): circuit, circuitState, Client, Config, hostRateLimiter, Context, Duration, Limit (+7 more)

### Community 43 - "AuthHandler"
Cohesion: 0.33
Nodes (6): AuthHandler, authContext(), Context, Duration, normalizeUsername(), writeAuthRateLimitHeaders()

### Community 44 - "Context"
Cohesion: 0.17
Nodes (5): handlerNotifier, handlerSessionStore, handlerUserRepo, Context, Duration

### Community 45 - "RedisQueue"
Cohesion: 0.27
Nodes (5): Client, Context, Duration, NewRedisQueue(), RedisQueue

### Community 46 - "newResetService"
Cohesion: 0.23
Nodes (13): fakeDispatcher, fakeSessionStore, Client, Miniredis, Mutex, newResetRedis(), newResetService(), TestForgotPasswordHidesMissingUser() (+5 more)

### Community 47 - "UserResponse"
Cohesion: 0.20
Nodes (10): BatchDeleteRequest, BatchResult, CreateUserRequest, UpdateUserRequest, UserResponse, Time, ToUserResponse(), ToUserResponses() (+2 more)

### Community 48 - "NewEmail"
Cohesion: 0.18
Nodes (12): Channel, NewEmail(), Conn, Listener, newFakeSMTPServer(), TestBuildEmailHeaders(), TestEmailSendEmptyRecipient(), TestEmailSendMissingConfig() (+4 more)

### Community 49 - "Context"
Cohesion: 0.14
Nodes (6): fakeAPIKeyRepo, fakeEventBus, APIKey, fakeUserRepository, Context, Mutex

### Community 50 - "CronScheduler"
Cohesion: 0.17
Nodes (8): Cron, EntryID, Context, RWMutex, Time, CronScheduler, JobInfo, Store

### Community 51 - "Manager"
Cohesion: 0.14
Nodes (16): contextKey, Flag, Manager, AdminFeatureHandler, NewAdminFeatureHandler(), TestAdminFeatureHandlerList(), TestAdminFeatureHandlerUpdate(), FromContext() (+8 more)

### Community 52 - "NewWebhook"
Cohesion: 0.16
Nodes (15): BenchmarkWebhookSend(), B, Channel, Client, Context, NewWebhook(), signPayload(), TestWebhookNoSignatureWhenNoSecret() (+7 more)

### Community 53 - "fakeStorage"
Cohesion: 0.22
Nodes (5): fakeStorage, Context, Duration, ReadCloser, Reader

### Community 54 - "AuditLog"
Cohesion: 0.17
Nodes (9): fakeAuditRepository, fakeBatchRepository, AuditLog, Change, fakeQueue, Context, Context, Mutex (+1 more)

### Community 55 - "WorkerPool"
Cohesion: 0.15
Nodes (12): CancelFunc, TraceFromContext(), GetWorker(), Context, Duration, MySQLStore, Consumer, Queue (+4 more)

### Community 57 - "PresenceManager"
Cohesion: 0.19
Nodes (4): RWMutex, Time, Presence, PresenceManager

### Community 58 - "RegisterAuthRoutes"
Cohesion: 0.29
Nodes (14): RouterGroup, Service, RegisterAuthRoutes(), RegisterCaptchaRoute(), Engine, newRouterLimiter(), testAuthConfig(), TestLoginRateLimitUsesIPScope() (+6 more)

### Community 59 - "middleware/middleware_test.go"
Cohesion: 0.15
Nodes (23): CORSMiddleware(), DefaultLogConfig(), HandlerFunc, isAllowedOrigin(), Logger(), Recovery(), RequestID(), shouldLogBody() (+15 more)

### Community 60 - "New"
Cohesion: 0.21
Nodes (13): Module, AuthConfig, CaptchaConfig, Service, NewAuthHandler(), newHandlerService(), TestForgotPasswordHandler(), TestResetPasswordHandlerInvalidCode() (+5 more)

### Community 61 - "newLock"
Cohesion: 0.18
Nodes (16): RedisConfig, Client, newLock(), TestLock_AcquireRelease(), TestLock_ConcurrentAcquire(), TestLock_Extend(), TestLock_ReleaseOnlyOwnToken(), TestLock_WithLock() (+8 more)

### Community 62 - "gorm_logger_test.go"
Cohesion: 0.10
Nodes (28): gormLogger, Interface, Context, Duration, Time, isSensitiveField(), NewGormLogger(), sanitizeArgs() (+20 more)

### Community 63 - "DBConfig"
Cohesion: 0.27
Nodes (12): DBConfig, configurePool(), ConnectWithRetry(), dsn(), Context, DB, openByDriver(), openMySQL() (+4 more)

### Community 64 - "rabbitmq_queue_test.go"
Cohesion: 0.20
Nodes (12): Context, Delivery, newTestRabbitQueue(), TestRabbitMQQueue_ConsumeTimeout(), TestRabbitMQQueue_ConsumeUnmarshalError(), TestRabbitMQQueue_NackRequeues(), TestRabbitMQQueue_SubmitConsumeAck(), TestRabbitMQQueueImplementsInterfaces() (+4 more)

### Community 65 - "ClientHub"
Cohesion: 0.15
Nodes (8): RWMutex, mustEncode(), BuildUserChannel(), RawMessage, Time, broadcastMsg, ClientHub, WSMessage

### Community 66 - "JobRegistry"
Cohesion: 0.12
Nodes (6): fakeAuthzModule, fakeBusinessModule, JobRegistry, EventBus, HandlerFunc, TestBusinessRoutesRequireProtectedMiddleware()

### Community 67 - "AdminMonitoringService"
Cohesion: 0.13
Nodes (17): AdminMonitoringService, HealthStatus, MemoryStats, SystemStatus, AdminMonitoringHandler, Client, Context, Time (+9 more)

### Community 69 - "SetupRouter"
Cohesion: 0.16
Nodes (18): ConfigureTrustedProxies(), formatAddr(), Engine, SetupRouter(), freeAddr(), TestConfigureTrustedProxies(), TestFormatAddr(), testLogger() (+10 more)

### Community 70 - "Client"
Cohesion: 0.29
Nodes (6): Conn, Context, HandlerFunc, Time, WSHandler(), Client

### Community 71 - "New"
Cohesion: 0.21
Nodes (12): AuditHandler, NewAuditService(), auditAppCode(), TestAuditServiceGetMapsNotFound(), TestAuditServiceListReturnsDTOAndPassesPagination(), Context, NewAuditHandler(), TestAuditGetInvalidIDReturnsStableBadRequest() (+4 more)

### Community 72 - "Message"
Cohesion: 0.18
Nodes (9): Context, buildEmailHeaders(), Context, Dispatcher, Context, Channel, Message, fakeKafkaReader (+1 more)

### Community 73 - "dispatcher"
Cohesion: 0.15
Nodes (10): Channel, Channel, Channel, Context, dispatcher, RWMutex, NewDispatcher(), TestDispatcherDispatch() (+2 more)

### Community 74 - "NewSnowflake"
Cohesion: 0.11
Nodes (18): Generator, snowflake, uuidGenerator, BenchmarkSnowflakeNextID(), BenchmarkUUIDNextID(), B, FuzzSnowflakeWorkerID(), F (+10 more)

### Community 75 - "gzipRouter"
Cohesion: 0.17
Nodes (12): HandlerFunc, ResponseWriter, Writer, GzipCompression(), isAlreadyCompressed(), Engine, gzipRouter(), TestGzipCompressionEncodesBody() (+4 more)

### Community 76 - "AdminUserService"
Cohesion: 0.21
Nodes (9): AdminCreateUserRequest, AdminUpdateUserRequest, AdminUser, AdminUserService, ListUserFilter, userRole, Context, DB (+1 more)

### Community 77 - "RedisStore"
Cohesion: 0.22
Nodes (9): RedisStore, Service, Client, Context, Duration, NewRedisStore(), NewService(), TestGenerate() (+1 more)

### Community 78 - "RabbitMQQueue"
Cohesion: 0.26
Nodes (7): Context, Delivery, Duration, Mutex, NewRabbitMQQueue(), RabbitMQChannel, RabbitMQQueue

### Community 79 - "OK"
Cohesion: 0.18
Nodes (7): AdminTaskHandler, Context, Context, Context, NewAdminTaskHandler(), Context, OK()

### Community 80 - "queue/worker_test.go"
Cohesion: 0.34
Nodes (13): NewMySQLStore(), NewWorkerPool(), RegisterWorker(), fakeStore(), MySQLStore, newFakeJobRepo(), TestWorkerPoolConsumeJob(), TestWorkerPoolConsumeRestoresTrace() (+5 more)

### Community 81 - "JobDef"
Cohesion: 0.18
Nodes (8): Context, RWMutex, Context, Time, failStore, JobDef, MemoryStore, Store

### Community 82 - "NewWithStore"
Cohesion: 0.23
Nodes (14): NewMemoryStore(), TestObserveCountsPanicAsFailure(), TestObserveEmitsSuccessMetrics(), TestSetEnabledNotFound(), TestSetEnabledToggle(), TestTriggerJobNotFound(), TestTriggerJobRunsCommand(), NewWithStore() (+6 more)

### Community 83 - "MySQLStore"
Cohesion: 0.17
Nodes (10): Context, DB, DeletedAt, Time, NewMySQLStore(), DB, testDB(), TestMySQLStore() (+2 more)

### Community 84 - "ChannelManager"
Cohesion: 0.11
Nodes (18): RWMutex, NewChannel(), NewChannelManager(), TestChannelManagerGetChannelMissing(), TestChannelManagerGetSubscribers(), TestChannelManagerPreCreatedBroadcast(), TestChannelManagerSubscribeCreatesWithType(), TestChannelManagerSubscribeExisting() (+10 more)

### Community 85 - "New"
Cohesion: 0.19
Nodes (12): New(), TestAppError(), TestAppErrorWithCause(), TestFailDoesNotLeakInfrastructureDetails(), TestOKIncludesStableEnvelopeAndRequestID(), TestPageIncludesStableEnvelopeAndPagination(), TestCreatedUsesStandardEnvelope(), TestFailHidesInternalCauseAndIncludesRequestID() (+4 more)

### Community 86 - "TestDB"
Cohesion: 0.29
Nodes (9): dbReachable(), envDBConfig(), DB, NewTestDB(), NewTestDBWithPool(), openByDriver(), SkipUnlessDB(), SkipUnlessMysql() (+1 more)

### Community 87 - "Application"
Cohesion: 0.23
Nodes (9): Application, Component, forwardError(), Context, Duration, NewApplication(), TestApplicationReturnsComponentRuntimeError(), TestApplicationRollsBackStartedComponents() (+1 more)

### Community 88 - "newTestGRPCService"
Cohesion: 0.39
Nodes (6): ClientConnInterface, newTestGRPCService(), TestUserInfoService_GetUser(), TestUserInfoService_ListUsers(), NewUserInfoServiceClient(), UserInfoServiceClient

### Community 89 - "mysqlAuditRepository"
Cohesion: 0.36
Nodes (5): mysqlAuditRepository, deserializeChanges(), Context, DB, NewMysqlAuditRepository()

### Community 90 - "EventBus"
Cohesion: 0.19
Nodes (9): EventBus, Handler, RWMutex, New(), TestEventBus_Clear(), TestEventBus_HandlerPanicRecovered(), TestEventBus_MultipleHandlers(), TestEventBus_PublishAsync() (+1 more)

### Community 91 - "Context"
Cohesion: 0.16
Nodes (11): fakeProvider, Duration, refreshTTL(), fakeSessionStore, sessionRecord, Context, Duration, UserInfo (+3 more)

### Community 92 - "ImportResult"
Cohesion: 0.19
Nodes (8): ExcelImporter, ImportError, ImportResult, Context, Reader, NewExcelImporter(), Time, NewImportResult()

### Community 93 - "New"
Cohesion: 0.18
Nodes (9): PermissionHandler, Context, NewPermissionHandler(), TestPermissionCreateReturnsCreated(), TestPermissionDeleteReturnsNoContent(), DB, EventBus, New() (+1 more)

### Community 94 - "OAuthService"
Cohesion: 0.21
Nodes (9): OAuthService, BindingRepository, Client, Context, DB, UserInfo, NewOAuthService(), Provider (+1 more)

### Community 95 - "ValidateJSON"
Cohesion: 0.16
Nodes (14): RouterGroup, RegisterPermissionRoutes(), RouterGroup, RegisterRoleRoutes(), Context, HandlerFunc, localeOf(), TestValidateJSONTranslatesFieldErrors() (+6 more)

### Community 96 - "NewMysqlRepository"
Cohesion: 0.23
Nodes (15): TestMysqlRepositoryMySQLIntegration(), NewMysqlRepository(), DB, newTestUser(), newUserTestDB(), TestMysqlRepositoryCreateAndFindByID(), TestMysqlRepositoryFindByEmailHash(), TestMysqlRepositoryFindByPhoneHash() (+7 more)

### Community 97 - "JobHistory"
Cohesion: 0.17
Nodes (9): JobHistory, JobHistoryRepository, mysqlJobHistoryRepository, Context, DB, NewMysqlJobHistoryRepository(), Time, Mutex (+1 more)

### Community 98 - "NewUserRateLimiter"
Cohesion: 0.19
Nodes (19): defaultKeyFunc(), Client, Context, Duration, HandlerFunc, Time, NewUserRateLimiter(), Engine (+11 more)

### Community 99 - "HTTP API 契约"
Cohesion: 0.29
Nodes (6): Admin 模块, Auth 模块, HTTP API 契约, 中间件, 健康与观测（非业务）, 完整接口文档

### Community 100 - "NewService"
Cohesion: 0.16
Nodes (10): fakeRouter, Service, Client, Time, NewService(), TestService(), Engine, RouterGroup (+2 more)

### Community 101 - "Router"
Cohesion: 0.20
Nodes (7): ComponentProvider, ErrorSource, EventBus, HTTPMiddlewareProvider, Module, ProtectedHTTPMiddlewareProvider, Router

### Community 102 - "ImportService"
Cohesion: 0.17
Nodes (17): ImportService, DB, newSqliteDB(), Context, DB, Format, Reader, NewImportService() (+9 more)

### Community 103 - "NewAuthService"
Cohesion: 0.19
Nodes (19): BenchmarkLogin(), benchUser(), B, TestForgotPasswordNotConfigured(), NewAuthService(), appCode(), fakeSessionStore, sessionRecord (+11 more)

### Community 104 - "signature_test.go"
Cohesion: 0.19
Nodes (20): buildSignString(), DefaultSignatureConfig(), Context, Duration, HandlerFunc, hmacSign(), Signature(), SignRequest() (+12 more)

### Community 105 - "Lock"
Cohesion: 0.33
Nodes (8): generateToken(), Client, Context, Duration, Time, NewLock(), AcquireResult, Lock

### Community 106 - "message.go"
Cohesion: 0.29
Nodes (6): ChatPayload, NotificationPayload, PingPayload, PongPayload, PresencePayload, SubscribePayload

### Community 107 - "ListUsersRequest"
Cohesion: 0.14
Nodes (5): MessageState, SizeCache, UnknownFields, GetUserRequest, ListUsersRequest

### Community 108 - "cleanup.go"
Cohesion: 0.23
Nodes (11): CleanupConfig, CleanupResult, CleanupService, CleanupTable, DefaultCleanupConfig(), Context, DB, Time (+3 more)

### Community 109 - "Redis Lock Impl"
Cohesion: 0.24
Nodes (9): contact, contactPtr, DB, newEncryptionTestDB(), TestEncryptionHookBatchCreate(), TestEncryptionHookEmptySourceStoresNullAndCoexists(), TestEncryptionHookEncryptsOnWriteDecryptsOnRead(), TestEncryptionHookNoKeyStoresPlaintextButHashes() (+1 more)

### Community 110 - "Docker Compose Observability Stack"
Cohesion: 0.21
Nodes (13): Alert Rules (Jimu HTTP/Infra/Queue/Process), AlertManager Routing Config, Grafana Datasources (Prometheus+Loki), Prometheus Scrape Config, Promtail Log Collection Config, Docker Compose Observability Stack, CI Pipeline, Coverage Threshold Gate (70%) (+5 more)

### Community 111 - "mysqlRepository"
Cohesion: 0.20
Nodes (6): RoleRepository, rolePermission, Context, DB, mysqlRepository, NewMysqlRepository()

### Community 112 - "Page"
Cohesion: 0.36
Nodes (10): Created(), FailWithDetails(), Context, localeFrom(), NoContent(), Page(), requestID(), StatusForCode() (+2 more)

### Community 113 - "NewServer"
Cohesion: 0.38
Nodes (12): NewServer(), New(), TestCircuitHalfOpenOnlyAllowsProbe(), TestCircuitOpensAfterConsecutiveFailures(), TestCircuitRecoversAfterCooldown(), TestDoCancelsDuringRetryWait(), TestDoInjectsTraceParent(), TestDoNoRetryOn4xx() (+4 more)

### Community 115 - "newMockGormDB"
Cohesion: 0.24
Nodes (14): MySQLBindingRepository, DB, NewMySQLBindingRepository(), bindingRows(), DB, Sqlmock, newMockGormDB(), TestCreateError() (+6 more)

### Community 116 - "New"
Cohesion: 0.19
Nodes (9): Client, Queue, New(), TestNew_InvalidType(), TestNew_Redis(), Config, KafkaConfig, RabbitMQConfig (+1 more)

### Community 117 - "KafkaQueue"
Cohesion: 0.29
Nodes (6): Context, Duration, NewKafkaQueue(), KafkaMessageReader, KafkaMessageWriter, KafkaQueue

### Community 118 - "AuditService"
Cohesion: 0.30
Nodes (7): AuditLogResponse, AuditService, Time, ToAuditLogResponse(), ToAuditLogResponses(), Context, serializeChanges()

### Community 120 - "AuditMiddleware"
Cohesion: 0.31
Nodes (8): Queue, AuditMiddleware(), Context, HandlerFunc, isManagementPath(), optionalString(), optionalUint64(), TestAuditMiddlewareAllowsAnonymousRequest()

### Community 121 - "LoginFailureTracker"
Cohesion: 0.33
Nodes (8): LockoutConfig, LoginFailureTracker, DefaultLockoutConfig(), Client, Context, Duration, lockKey(), NewLoginFailureTracker()

### Community 122 - "newMockGormDB"
Cohesion: 0.12
Nodes (25): cleanupModel, noTableNameModel, NewCleanupService(), TestCleanupService_RunBatches(), TestCleanupService_RunCustomDeletedColumn(), TestCleanupService_RunError(), TestCleanupService_RunMultipleTables(), TestNewCleanupService_AppliesDefaults() (+17 more)

### Community 123 - "NewCSVImporter"
Cohesion: 0.24
Nodes (6): CSVImporter, Context, Reader, NewCSVImporter(), FuzzCSVImporterParse(), F

### Community 124 - "routerLimiterRedis"
Cohesion: 0.29
Nodes (6): routerLimiterRedis, BoolSliceCmd, Cmd, Context, StringCmd, routerLimiterInt()

### Community 125 - "New"
Cohesion: 0.17
Nodes (13): OAuthConfig, OAuthProviderConfig, buildProviders(), Client, DB, EventBus, New(), newTestModule() (+5 more)

### Community 126 - "New"
Cohesion: 0.12
Nodes (20): UserService, Cache, UserRepository, NewUserService(), appCode(), createOutboxUserService(), TestUserServiceBatchDelete(), TestUserServiceGetMapsNotFound() (+12 more)

### Community 127 - "MySQLStore"
Cohesion: 0.26
Nodes (6): Context, DB, NewMySQLStore(), TestMySQLStore_AddWithinTx(), TestMySQLStore_AddWithNilTx(), MySQLStore

### Community 128 - "NewPresenceManager"
Cohesion: 0.19
Nodes (14): NewPresenceManager(), TestPresenceIsOnline(), TestPresenceIsTyping(), TestPresenceManagerAllPresences(), TestPresenceManagerHeartbeat(), TestPresenceManagerHeartbeatRestoresOnlineStatus(), TestPresenceManagerOfflineMissing(), TestPresenceManagerOnlineCountFiltersOffline() (+6 more)

### Community 129 - "AdminTaskService"
Cohesion: 0.29
Nodes (5): AdminTaskService, TaskExecution, TaskInfo, Context, Time

### Community 130 - "Worker"
Cohesion: 0.20
Nodes (11): Worker, AuditRepository, Context, RWMutex, NewWorker(), Duration, testWorker(), TestWorkerFlushesConfiguredBatch() (+3 more)

### Community 132 - "OAuthBinding"
Cohesion: 0.15
Nodes (8): fakeBindingRepo, OAuthBinding, fakeBindingRepo, fakeSessionStore, Time, Context, Context, Duration

### Community 133 - ".Validate"
Cohesion: 0.33
Nodes (8): FieldRule, FieldType, ValidationRules, Validator, TestValidateUnique(), checkType(), Context, NewValidator()

### Community 134 - "idempotencyRouter"
Cohesion: 0.36
Nodes (10): Int32, Client, Engine, idempotencyRouter(), newRedisForTest(), TestIdempotencyCachedResponseSerializes(), TestIdempotencyCachesAndReplays(), TestIdempotencyDoesNotCacheFailure() (+2 more)

### Community 135 - "adminAuthRouter"
Cohesion: 0.27
Nodes (9): AdminAuth(), HandlerFunc, adminAuthRouter(), Engine, TestAdminAuthAllowsAdminRole(), TestAdminAuthAllowsEnglishAdminRole(), TestAdminAuthRejectsNonAdminRole(), TestAdminAuthRequiresAuthentication() (+1 more)

### Community 136 - "responseBodyWriter"
Cohesion: 0.17
Nodes (10): Buffer, ResponseWriter, newResponseBodyWriter(), sanitizeBody(), sanitizeJSONField(), TestResponseBodyWriterCapturesAndTruncates(), TestResponseBodyWriterWriteHeaderPassthrough(), TestSanitizeBody() (+2 more)

### Community 137 - "mockNotification"
Cohesion: 0.47
Nodes (3): Channel, Context, mockNotification

### Community 138 - "SMS"
Cohesion: 0.22
Nodes (8): Channel, Context, NewSMS(), TestSMSAliyunReject(), TestSMSAliyunSend(), TestSMSUnknownProvider(), SMS, SMSConfig

### Community 139 - "Logger"
Cohesion: 0.17
Nodes (11): AtomicLevel, Context, LogConfig, New(), TestFileOutput(), TestNewConsoleStdout(), TestNewJSONLevels(), TestSetLevel() (+3 more)

### Community 140 - "Jimu API Swagger Definition"
Cohesion: 0.33
Nodes (10): Jimu API Swagger Definition, Audit Log Endpoints, Auth Flow Endpoints (login/refresh/logout/register), BearerAuth Security (API Key JWT), Captcha Endpoint, contract.ErrorResponse, OAuth Endpoints, contract.PageResponse (+2 more)

### Community 142 - "New"
Cohesion: 0.18
Nodes (7): RoleHandler, Context, NewRoleHandler(), DB, EventBus, New(), Module

### Community 143 - "Permission"
Cohesion: 0.22
Nodes (7): Permission, PermissionRepository, mysqlPermissionRepository, Context, DB, NewMysqlPermissionRepository(), Time

### Community 144 - "AGENTS.md"
Cohesion: 0.14
Nodes (12): Commit Message 规范, graphify, Release Note 规范, 保护工作区, 分支策略, 开发前必读, 文档维护, 架构约束 (+4 more)

### Community 145 - "mockTokenServer"
Cohesion: 0.20
Nodes (15): Client, Server, mockClient(), mockTokenServer(), TestGitHubProviderExchange(), TestGoogleProviderExchange(), TestGoogleProviderExchangeBadUserInfo(), TestGoogleProviderExchangeTokenError() (+7 more)

### Community 146 - "NewPermissionService"
Cohesion: 0.30
Nodes (8): fakePermissionRepository, NewPermissionService(), Context, permissionAppCode(), TestPermissionServiceCreateMapsDuplicateNameToConflict(), TestPermissionServiceDeleteWrapsRepositoryError(), TestPermissionServiceListPassesPagination(), TestPermissionServiceUpdateMapsNotFound()

### Community 147 - "userInfoService"
Cohesion: 0.24
Nodes (6): userInfoService, DB, Context, DB, UserInfo, NewUserInfoGRPCService()

### Community 148 - "RegisterUserRoutes"
Cohesion: 0.20
Nodes (7): RouterGroup, RegisterUserRoutes(), Client, Duration, HandlerFunc, IdempotencyMiddleware(), cachedResponse

### Community 149 - "NewDBAPIKeyStore"
Cohesion: 0.33
Nodes (12): createKey(), APIKey, DB, newTestDB(), TestDBAPIKeyStore_GetByKeyHash(), TestDBAPIKeyStore_GetByKeyHashNotFound(), TestDBAPIKeyStore_UpdateLastUsed(), TestVerifyWithDBStore() (+4 more)

### Community 150 - "userinfo_grpc.pb.go"
Cohesion: 0.20
Nodes (10): Context, UserInfo, RegisterUserInfoServiceServer(), _UserInfoService_GetUser_Handler(), _UserInfoService_ListUsers_Handler(), ServiceRegistrar, UnaryServerInterceptor, UnimplementedUserInfoServiceServer (+2 more)

### Community 151 - "Timeout"
Cohesion: 0.22
Nodes (6): Duration, HandlerFunc, TestTimeoutOnlyPropagatesContextDeadline(), Timeout(), RouterGroup, RegisterSwagger()

### Community 152 - "TestWebSocketNotification"
Cohesion: 0.33
Nodes (5): NewHub(), TestHubLifecycle(), TestWebSocketNotification(), waitHub(), mockConn

### Community 153 - "GitHubProvider"
Cohesion: 0.24
Nodes (7): Client, Config, Context, UserInfo, NewGitHubProvider(), GitHubConfig, GitHubProvider

### Community 154 - "NewWeChatProvider"
Cohesion: 0.24
Nodes (7): Client, Config, Context, UserInfo, NewWeChatProvider(), WeChatConfig, WeChatProvider

### Community 155 - "Role"
Cohesion: 0.28
Nodes (5): fakeRoleRepository, Role, Context, DeletedAt, Time

### Community 156 - "mysqlAPIKeyRepository"
Cohesion: 0.29
Nodes (5): mysqlAPIKeyRepository, APIKey, Context, DB, NewMysqlAPIKeyRepository()

### Community 157 - "As"
Cohesion: 0.36
Nodes (5): AppError, ErrorInfo, AllErrorCodes(), As(), HTTPStatus()

### Community 158 - "validator/validator.go"
Cohesion: 0.25
Nodes (8): FieldLevel, FuzzValidateRules(), F, Validate(), validateIDCard(), validateMobile(), validatePassword(), validateUsername()

### Community 159 - "Registry"
Cohesion: 0.54
Nodes (5): Format, Importer, Registry, NewRegistry(), TestRegistryGetUnsupported()

### Community 161 - "New"
Cohesion: 0.14
Nodes (25): fakeUserRepository, NewAdminJobHandler(), TestAdminJobHandlerGet(), TestAdminJobHandlerList(), TestAdminJobHandlerListDeadLetters(), TestAdminJobHandlerResolveDeadLetter(), TestAdminJobHandlerRetry(), TestAdminJobHandlerSubmit() (+17 more)

### Community 162 - "AdminUserHandler"
Cohesion: 0.26
Nodes (5): AdminUserHandler, Context, Context, NewAdminUserHandler(), paginationFromQuery()

### Community 163 - "userinfo.pb.go"
Cohesion: 0.29
Nodes (3): file_proto_jimu_v1_userinfo_proto_init(), file_proto_jimu_v1_userinfo_proto_rawDescGZIP(), init()

### Community 164 - "tracing.go"
Cohesion: 0.43
Nodes (7): ContextWithTrace(), DefaultTracingConfig(), Context, TracerProvider, InitTracing(), ShutdownTracing(), TracingConfig

### Community 165 - "NewEventBusPublisher"
Cohesion: 0.32
Nodes (6): Context, EventBus, RawMessage, NewEventBusPublisher(), EventBusPublisher, EventPayload

### Community 166 - "Module"
Cohesion: 0.17
Nodes (5): Module, RouterGroup, RegisterAuditRoutes(), EventBus, HandlerFunc

### Community 167 - "fakeRoleRepository"
Cohesion: 0.27
Nodes (4): fakeRoleRepository, Context, TestRoleCreateReturnsCreated(), TestRoleDeleteReturnsNoContent()

### Community 168 - "ResetStore"
Cohesion: 0.39
Nodes (5): ResetStore, Client, Context, Duration, NewResetStore()

### Community 170 - "SecurityHeadersFromConfig"
Cohesion: 0.27
Nodes (9): SecurityConfig, DefaultSecurityConfig(), TestSecurityHeadersMiddleware(), TestSecurityHeadersNilValuesSkipped(), TestSecurityHeadersRespectsCustomValues(), Context, HandlerFunc, SecurityHeadersFromConfig() (+1 more)

### Community 171 - "newWSHandler"
Cohesion: 0.32
Nodes (7): AdminWSHandler, Context, NewAdminWSHandler(), newWSHandler(), TestAdminWSHandlerOnlineUsers(), TestAdminWSHandlerPresence(), TestAdminWSHandlerPush()

### Community 172 - "newTestScheduler"
Cohesion: 0.62
Nodes (6): NewAdminTaskService(), newTestScheduler(), TestAdminTaskServiceGetHistory(), TestAdminTaskServiceListTasks(), TestAdminTaskServiceToggleTask(), TestAdminTaskServiceTriggerTask()

### Community 173 - "newRepoTestDB"
Cohesion: 0.53
Nodes (5): DB, newRepoTestDB(), TestMysqlAPIKeyRepository(), TestMysqlImportJobRepository(), TestMysqlJobRepository()

### Community 174 - "setupTestCache"
Cohesion: 0.52
Nodes (6): setupTestCache(), TestRedisCache_Delete(), TestRedisCache_DeletePattern(), TestRedisCache_GetOrSet(), TestRedisCache_GetOrSetStampede(), TestRedisCache_SetAndGet()

### Community 175 - "i18n.go"
Cohesion: 0.18
Nodes (7): HandlerFunc, Locale(), ParseAcceptLanguage(), TestParseAcceptLanguage(), TestTDefaultsToChinese(), TestTfFormatsArgs(), Tf()

### Community 176 - "Duration"
Cohesion: 0.43
Nodes (4): Duration, ExponentialBackoff, FixedRetry, RetryStrategy

### Community 178 - "CaptchaHandler"
Cohesion: 0.83
Nodes (3): CaptchaHandler, Service, NewCaptchaHandler()

### Community 180 - "NewHandler"
Cohesion: 0.32
Nodes (5): Handler, Context, Service, NewHandler(), TestHandlerGetErrorCodes()

### Community 181 - "Now"
Cohesion: 0.29
Nodes (4): Context, Time, Now(), MySQLStore

### Community 182 - "NewAdminUserService"
Cohesion: 0.29
Nodes (9): testRole, NewAdminUserService(), TestAdminUserServiceAssignRoles(), TestAdminUserServiceAssignRolesRoleQueryError(), TestAdminUserServiceCreateUser(), TestAdminUserServiceDisableUser(), TestAdminUserServiceGetUser(), TestAdminUserServiceListUsers() (+1 more)

### Community 183 - "Policy"
Cohesion: 0.24
Nodes (6): fakeAuthorizationStore, Policy, fakeAuthzStore, Context, TestProtectedMiddlewareRequiresAccessToken(), Context

### Community 184 - "NewManagementServer"
Cohesion: 0.20
Nodes (8): Handler, passingChecker, Server, HealthRouter(), NewManagementServer(), Context, TestManagementRouterExposure(), TestNewManagementServer()

### Community 185 - "Metrics"
Cohesion: 0.33
Nodes (7): HandlerFunc, Metrics(), gatherHTTPMetric(), Engine, setupMetricsEngine(), TestMetricsLabelsUseRouteTemplate(), TestMetricsRecordsUnmatchedPath()

### Community 186 - "HTTPConfig"
Cohesion: 0.20
Nodes (12): HTTPConfig, TLSConfig, TestSecurityAddVary(), addVary(), Context, HandlerFunc, Security(), Engine (+4 more)

### Community 187 - "DBCollector"
Cohesion: 0.47
Nodes (4): CollectRuntime(), DB, NewDBCollector(), DBCollector

### Community 188 - "testEnforcer"
Cohesion: 0.27
Nodes (9): DB, Enforcer, NewEnforcer(), NewPathEnforcer(), Enforcer, TestAuthorizationMiddlewareAllowsRolePolicy(), TestAuthorizationMiddlewareHidesStoreErrors(), TestAuthorizationMiddlewareRejectsMissingRole() (+1 more)

### Community 189 - "NewLogChannel"
Cohesion: 0.29
Nodes (6): Channel, Context, NewLogChannel(), TestLogChannelSendBatchNoError(), TestLogChannelSendNoError(), LogChannel

### Community 190 - "Context"
Cohesion: 0.31
Nodes (4): Context, Duration, errorQueue, fakeQueue

### Community 192 - "Config Contract"
Cohesion: 0.50
Nodes (5): Development Config (app.yaml), Production Config (app.prod.yaml), Config Contract, Config Enum Validation (FR-002), File-Based Secret Injection (FR-003)

### Community 193 - "events.go"
Cohesion: 0.40
Nodes (4): UserCreatedEvent, UserDeletedEvent, UserLoggedInEvent, UserUpdatedEvent

### Community 194 - "fakeContainer"
Cohesion: 0.44
Nodes (8): bridgeFn(), fakeContainer(), newTestLogger(), TestBridgeWorkerConversionFailureErrors(), TestBridgeWorkerPublishesStrongTypeToBareTopic(), TestBridgeWorkerUnknownTypeErrors(), TestEventBusBridgePublishesToBareTopic(), TestRegisterOutboxWorkersRegistersAll()

### Community 195 - "newRedisTestQueue"
Cohesion: 0.39
Nodes (7): Client, newRedisTestQueue(), TestQueueContract_SubmitConsume(), TestRedisQueueAckRemovesFromProcessing(), TestRedisQueueImplementsInterfaces(), TestRedisQueueNackRequeues(), TestRedisQueueRequeueExpired()

### Community 196 - "mysql/003_extensions.sql"
Cohesion: 0.25
Nodes (7): api_keys, dead_letters, import_jobs, job_history, jobs, scheduled_jobs, user_oauth_bindings

### Community 197 - "postgres/003_extensions.sql"
Cohesion: 0.25
Nodes (7): api_keys, dead_letters, import_jobs, job_history, jobs, scheduled_jobs, user_oauth_bindings

### Community 198 - "admin/module_test.go"
Cohesion: 0.29
Nodes (4): fakeEventBus, TestModuleInitWSIdempotent(), TestModuleNameAndNew(), TestModuleWSHandler()

### Community 199 - "CLI 契约"
Cohesion: 0.40
Nodes (4): CLI 契约, 命令表, 种子数据, 迁移命名

### Community 200 - "Release Workflow"
Cohesion: 0.50
Nodes (4): Release Workflow, Conventional Commits, Release Note Structure, Commit Style

### Community 201 - "Security Policy"
Cohesion: 0.50
Nodes (4): No .env & Secrets Hook, Password Strength Rule, Security Policy, Vulnerability Reporting Process

### Community 202 - "Bootstrap"
Cohesion: 0.52
Nodes (6): moduleLogger, registerRouter, Bootstrap(), registerEventBusBridge(), registerHTTP(), registerOutboxWorkers()

### Community 203 - "Registry"
Cohesion: 0.62
Nodes (4): Exporter, Format, Registry, NewRegistry()

### Community 205 - "NewRoleService"
Cohesion: 0.57
Nodes (6): NewRoleService(), roleAppCode(), TestRoleServiceCreateMapsDuplicateNameToConflict(), TestRoleServiceDeleteWrapsRepositoryError(), TestRoleServiceListPassesPagination(), TestRoleServiceUpdateMapsNotFound()

### Community 206 - "newHistoryTestDB"
Cohesion: 0.40
Nodes (4): TestMysqlDeadLetterRepositoryCRUD(), DB, newHistoryTestDB(), TestMysqlJobHistoryRepositoryCreateAndList()

### Community 207 - "TestReadinessBoundsCheckerDuration"
Cohesion: 0.47
Nodes (4): Context, TestReadinessBoundsCheckerDuration(), TestReadinessStatus(), checkerFunc

### Community 209 - "mysql/001_core.sql"
Cohesion: 0.33
Nodes (5): permissions, role_permissions, roles, user_roles, users

### Community 210 - "postgres/001_core.sql"
Cohesion: 0.33
Nodes (5): permissions, role_permissions, roles, user_roles, users

### Community 211 - "Specification Quality Checklist: Jimu 后端框架能力规格"
Cohesion: 0.33
Nodes (5): Content Quality, Feature Readiness, Notes, Requirement Completeness, Specification Quality Checklist: Jimu 后端框架能力规格

### Community 212 - "GolangCI Lint Configuration"
Cohesion: 0.67
Nodes (3): GolangCI Lint Configuration, Lint Rule Set, Pre-commit Hooks Configuration

### Community 213 - "Notification Abstraction (FR-028)"
Cohesion: 0.67
Nodes (3): Unreleased Capabilities, Aliyun SMS Implementation (D2), Notification Abstraction (FR-028)

### Community 216 - "request.go"
Cohesion: 0.40
Nodes (4): forgotPasswordRequest, loginRequest, refreshRequest, resetPasswordRequest

### Community 217 - "importer_test.go"
Cohesion: 0.60
Nodes (4): csvReader(), Reader, TestCSVParseAndValidate(), TestCSVParseEmptyFile()

### Community 219 - "PermissionMiddleware"
Cohesion: 0.50
Nodes (3): Enforcer, HandlerFunc, PermissionMiddleware()

### Community 225 - "RBAC (FR-010)"
Cohesion: 0.67
Nodes (3): Permission Entity, Role Entity, RBAC (FR-010)

### Community 276 - "Container"
Cohesion: 0.22
Nodes (10): Container, Client, Config, Context, DB, EventBus, Server, Service (+2 more)

### Community 293 - "AuthorizationMiddleware"
Cohesion: 0.18
Nodes (12): AuthorizationStore, DBAuthorizationStore, Enforcer, HandlerFunc, ProtectedMiddleware(), HandlerFunc, AuthorizationMiddleware(), Context (+4 more)

### Community 295 - "RegisterPingServer"
Cohesion: 0.29
Nodes (6): PingServer, pingService, Context, Server, RegisterPingServer(), StringValue

### Community 308 - "S3Storage"
Cohesion: 0.20
Nodes (6): Client, Context, Duration, ReadCloser, Reader, S3Storage

## Knowledge Gaps
- **224 isolated node(s):** `common.sh script`, `jimu`, `CaptchaResult`, `LogConfig`, `UserCreatedEvent` (+219 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **66 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `T()` connect `T` to `RoleService`, `oauth/application/service_test.go`, `New`, `JWT`, `Server`, `NewCSVExporter`, `ws_integration_test.go`, `api_contract_test.go`, `bindImportFile`, `upload_handler_test.go`, `migrate.go`, `oauth/interfaces/handler_test.go`, `LocalStorage`, `NewAdminAPIKeyService`, `fakeRedis`, `.RegisterHTTP`, `New`, `NewAdminConfigService`, `generator/module.go`, `DefaultCSRFConfig`, `User`, `Outbox`, `newResetService`, `UserResponse`, `NewEmail`, `Manager`, `NewWebhook`, `RegisterAuthRoutes`, `middleware/middleware_test.go`, `New`, `newLock`, `gorm_logger_test.go`, `rabbitmq_queue_test.go`, `JobRegistry`, `AdminMonitoringService`, `SetupRouter`, `New`, `dispatcher`, `NewSnowflake`, `gzipRouter`, `RedisStore`, `queue/worker_test.go`, `NewWithStore`, `MySQLStore`, `ChannelManager`, `New`, `TestDB`, `Application`, `newTestGRPCService`, `EventBus`, `Context`, `New`, `ValidateJSON`, `NewMysqlRepository`, `NewUserRateLimiter`, `NewService`, `ImportService`, `NewAuthService`, `signature_test.go`, `cleanup.go`, `Redis Lock Impl`, `Page`, `NewServer`, `newMockGormDB`, `New`, `AuditMiddleware`, `newMockGormDB`, `New`, `New`, `MySQLStore`, `NewPresenceManager`, `Worker`, `.Validate`, `idempotencyRouter`, `adminAuthRouter`, `responseBodyWriter`, `SMS`, `Logger`, `mockTokenServer`, `NewPermissionService`, `NewDBAPIKeyStore`, `Timeout`, `TestWebSocketNotification`, `Registry`, `New`, `fakeRoleRepository`, `SecurityHeadersFromConfig`, `newWSHandler`, `newTestScheduler`, `newRepoTestDB`, `setupTestCache`, `i18n.go`, `NewHandler`, `NewAdminUserService`, `Policy`, `NewManagementServer`, `Metrics`, `HTTPConfig`, `testEnforcer`, `NewLogChannel`, `fakeContainer`, `newRedisTestQueue`, `admin/module_test.go`, `NewRoleService`, `newHistoryTestDB`, `TestReadinessBoundsCheckerDuration`, `importer_test.go`?**
  _High betweenness centrality (0.581) - this node is a cross-community bridge._
- **Why does `Now()` connect `Now` to `NewPresenceManager`, `JWT`, `.Validate`, `ws_integration_test.go`, `T`, `DeadLetter`, `upload_handler_test.go`, `RedisCache`, `auth/apikey.go`, `NewDBAPIKeyStore`, `TestWebSocketNotification`, `NewAdminAPIKeyService`, `mysqlAPIKeyRepository`, `New`, `Wrap`, `AuthorizationMiddleware`, `DefaultCSRFConfig`, `Fail`, `circuit`, `RedisQueue`, `CronScheduler`, `NewWebhook`, `NewAdminUserService`, `WorkerPool`, `Metrics`, `PresenceManager`, `middleware/middleware_test.go`, `gorm_logger_test.go`, `ClientHub`, `AdminMonitoringService`, `newRedisTestQueue`, `SetupRouter`, `Client`, `NewSnowflake`, `TestReadinessBoundsCheckerDuration`, `JobDef`, `NewWithStore`, `Context`, `ImportResult`, `NewMysqlRepository`, `NewUserRateLimiter`, `NewService`, `ImportService`, `signature_test.go`, `Lock`, `cleanup.go`, `newMockGormDB`, `AuditMiddleware`, `MySQLStore`?**
  _High betweenness centrality (0.085) - this node is a cross-community bridge._
- **Why does `Wrap()` connect `Wrap` to `RoleService`, `PermissionService`, `AdminJobHandler`, `AuthorizationMiddleware`, `ImportService`, `AdminUserService`, `UserResponse`, `upload_handler_test.go`, `New`, `AuditService`, `NewAdminAPIKeyService`, `As`, `OAuthService`?**
  _High betweenness centrality (0.055) - this node is a cross-community bridge._
- **Are the 4 inferred relationships involving `T()` (e.g. with `translateValidationDetails()` and `ValidateJSON()`) actually correct?**
  _`T()` has 4 INFERRED edges - model-reasoned connections that need verification._
- **Are the 81 inferred relationships involving `New()` (e.g. with `.CreateKey()` and `.ToggleTask()`) actually correct?**
  _`New()` has 81 INFERRED edges - model-reasoned connections that need verification._
- **Are the 81 inferred relationships involving `Now()` (e.g. with `.CreateKey()` and `.generateResetCode()`) actually correct?**
  _`Now()` has 81 INFERRED edges - model-reasoned connections that need verification._
- **Are the 80 inferred relationships involving `Fail()` (e.g. with `.Generate()` and `.HandleDelete()`) actually correct?**
  _`Fail()` has 80 INFERRED edges - model-reasoned connections that need verification._