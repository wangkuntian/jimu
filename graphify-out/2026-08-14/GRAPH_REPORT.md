# Graph Report - .  (2026-08-14)

## Corpus Check
- 30 files · ~138,614 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3901 nodes · 7622 edges · 338 communities (214 shown, 124 thin omitted)
- Extraction: 87% EXTRACTED · 13% INFERRED · 0% AMBIGUOUS · INFERRED: 985 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- Admin User Management
- OAuth Binding Repos
- User Service DTOs
- gRPC Server
- Storage Abstraction
- WebSocket Hub
- Notification Dispatcher
- Queue Dead Letter
- Auth Redis Pipeline
- Authorization Store
- E2E API Contract
- OAuth Service
- Spec Constitution
- Admin Import Handler
- APIKey Repos
- Storage HTTP Fake
- OpenAPI Contract Test
- Outbox User Repo
- Gorm Logger
- Specify Scripts
- Management Server
- APIKey Auth
- Auth Module
- Helm Chart
- Spec Data Model
- APIKey Repos (app)
- DB Config/Migrate
- OAuth Binding (interfaces)
- Scheduler Memory Store
- User Repo Fake
- Admin APIKey Service
- Admin Tasks Service
- Queue Worker Test
- Spec Checklists
- Admin Config Service
- Auth Service
- Admin Module
- Config Structs
- Role Service
- CSRF Middleware
- WebSocket Handler
- Spec Tasks
- Observability Tracing
- HTTP Client Circuit
- Handler Notifier
- Dispatcher Fake
- JWT Auth
- OAuth Config
- Rate Limit Middleware
- Email Notification
- Outbox MQ Publisher
- App Container
- Role Repo Fake
- DB Cleanup
- Feature Flag
- Code Generator
- Webhook Notification
- Event Bus Fake
- WebSocket Presence
- Job Registry
- Permission Service
- Audit Worker
- Middleware Tests
- Role/Permission Routers
- Queue Worker
- App Container
- Redis Cache
- Redis Lock
- Field Encryption
- Auth Handler
- Auth Service Test
- MySQL DB
- DB Seed
- RabbitMQ Queue
- Admin Monitoring
- OAuth/User Handlers
- HTTP Server Test
- Audit Service
- Permission Repo Fake
- Snowflake ID
- Admin Job Handler
- User Handler Test
- Gzip Middleware
- WebSocket Conn
- Audit Repo Fake
- Captcha Redis Store
- RabbitMQ Connection
- Config Load
- Scheduler MySQL Store
- WebSocket Channels
- Shared Errors
- Test DB Util
- App Application
- AuthZ Module Fake
- gRPC UserInfo PB
- Event Bus
- Upload Handler
- User MySQL Repo
- Metrics Middleware
- User Rate Limit
- Spec API Contract
- Admin Router Fake
- Import Service
- Audit MySQL Repo
- Role/Permission MySQL
- Role Repo (interfaces)
- Permission Handler
- Signature Test
- Kafka Queue
- Redis Lock Impl
- UserInfo PB Messages
- Module Contract
- Import Job Repo
- DB Cleanup Service
- DB Contact Model
- Observability Deploy
- gRPC UserInfo Service
- Admin Audit Handler
- OAuth MySQL Repo
- HTTP Client Test
- OAuth Google Provider
- Queue Factory
- Password Reset Store
- Login Failure Tracker
- CSV Importer
- Audit Handler
- Permission Repo (interfaces)
- Audit Queue Fake
- Router Redis Limiter
- Locale/i18n Middleware
- Outbox MySQL Store
- Redis Queue
- Presence State Test
- Admin Task Service
- OAuth Binding Domain
- Importer Validation
- Idempotency Middleware
- Admin Auth Middleware
- Body Recorder
- Dispatcher Test
- WebSocket Channels Test
- OpenAPI Swagger Docs
- HTTP Server
- Permission MySQL Repo
- User Rate Limit Test
- Signature Middleware
- OAuth Exchange Test
- internal_platform
- internal_platform
- user module
- userinfopb unimplementeduserinfoservices
- internal_platform
- internal_platform
- internal_platform
- internal_platform
- internal_platform
- modules
- shared
- shared
- importer registry
- interfaces adminapikeyhandler
- modules
- internal_platform
- internal_platform
- internal_platform
- internal_platform
- internal_platform
- modules
- modules
- shared
- internal_platform
- interfaces adminwshandler
- modules
- modules
- internal_platform
- internal_platform
- internal_platform
- userinfopb listusersresponse
- internal_platform
- interfaces fakeauditrepository
- modules
- modules
- modules
- modules
- internal_platform
- internal_platform
- internal_platform
- internal_platform
- shared
- specs 001
- specs 001
- app fakecomponent
- specs 001
- contract usercreatedevent
- modules
- internal_platform
- internal_platform
- internal_platform
- internal_platform
- specs 001
- github
- pre
- contract
- modules
- modules
- modules
- modules
- modules
- internal_platform
- shared
- migrations 010
- migrations postgres
- golangci
- specs 001
- contract errorresponse
- modules
- modules
- modules
- modules
- internal_platform
- internal_platform
- shared
- scripts smoke
- scripts test
- specs 001
- application loginrequest
- claude github
- migrations 001
- migrations 002
- migrations 003
- migrations 004
- migrations 005
- migrations 006
- migrations 007
- migrations 008
- migrations 009
- migrations 011
- migrations 012
- migrations 013
- migrations 014
- migrations postgres
- migrations postgres
- migrations postgres
- migrations postgres
- migrations postgres
- migrations postgres
- migrations postgres
- migrations postgres
- migrations postgres
- migrations postgres
- migrations postgres
- migrations postgres
- migrations postgres
- scripts backup
- scripts bench
- scripts loadtest
- scripts restore
- specs 001
- specs 001
- specs 001
- specs 001
- specs 001
- specs 001
- specify
- claude config
- claude goose
- contributing dev
- deploy grafana
- github bug
- github dependabot
- github dependabot
- github dependabot
- github feature
- github pr
- app
- app
- config
- modules
- modules
- modules
- modules
- modules
- modules
- modules
- modules
- modules
- modules
- modules
- modules
- modules
- internal_platform
- internal_platform
- internal_platform
- internal_platform
- internal_platform
- internal_platform
- internal_platform
- internal_platform
- internal_platform
- internal_platform
- internal_platform
- internal_platform
- internal_platform
- internal_platform
- internal_platform
- internal_platform
- internal_platform
- internal_platform
- pkg jimu
- readme audit
- readme cache
- readme config
- readme deployment
- readme feature
- readme graceful
- readme i18n
- readme management
- readme oauth
- readme security
- readme tech
- responserecorder
- security account
- specs 001
- specs 001
- specs 001
- specs 001
- specs 001
- specs 001
- specs 001
- specs 001
- specs 001
- specs 001
- specs 001
- store

## God Nodes (most connected - your core abstractions)
1. `T()` - 200 edges
2. `User` - 71 edges
3. `Fail()` - 67 edges
4. `Wrap()` - 49 edges
5. `New()` - 48 edges
6. `OK()` - 45 edges
7. `Now()` - 42 edges
8. `Client` - 37 edges
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
- **Observability Stack** — deploy_prometheus_prometheus_config, deploy_alertmanager_alertmanager_config, deploy_alert_rules_alert_rules, deploy_promtail_promtail_config, deploy_grafana_provisioning_datasources_prometheus_grafana_ds [EXTRACTED 0.95]
- **RBAC Entity Model** — specs_001_jimu_framework_spec_data_model_user, specs_001_jimu_framework_spec_data_model_role, specs_001_jimu_framework_spec_data_model_permission [EXTRACTED 0.95]
- **Release Gate Quality Checks** — specs_001_jimu_framework_spec_spec_release_gate, github_workflows_ci_security_scan, github_workflows_ci_coverage_gate [INFERRED 0.85]
- **Constitution Core Principles** — _specify_memory_constitution_business_agnostic, _specify_memory_constitution_api_stability, _specify_memory_constitution_composable_modules, _specify_memory_constitution_simplicity, _specify_memory_constitution_verification, _specify_memory_constitution_documentation [EXTRACTED 1.00]
- **Jimu Helm Chart Resource Set** — deploy_helm_chart, deploy_helm_values, deploy_helm_templates_configmap, deploy_helm_templates_deployment, deploy_helm_templates_hpa, deploy_helm_templates_ingress, deploy_helm_templates_networkpolicy, deploy_helm_templates_pdb, deploy_helm_templates_prometheusrule, deploy_helm_templates_service [EXTRACTED 1.00]
- **jimu-server Kubernetes Resource Set** — deploy_k8s_configmap, deploy_k8s_config_files, deploy_k8s_deployment, deploy_k8s_hpa, deploy_k8s_ingress, deploy_k8s_networkpolicy, deploy_k8s_pdb, deploy_k8s_prometheusrule, deploy_k8s_service [EXTRACTED 1.00]
- **Jimu Prometheus Alert Rules** — deploy_k8s_prometheusrule_jimuhigherrorrate, deploy_k8s_prometheusrule_jimuhighlatency, deploy_k8s_prometheusrule_jimupodcrashlooping, deploy_k8s_prometheusrule_jimupoddown, deploy_k8s_prometheusrule_jimudbpoolhigh, deploy_k8s_prometheusrule_jimureadinessfailing [EXTRACTED 1.00]

## Communities (338 total, 124 thin omitted)

### Community 0 - "Admin User Management"
Cohesion: 0.05
Nodes (37): AssignPermissionsRequest, CreateRoleRequest, fakeRoleRepository, RoleResponse, RoleService, UpdateRoleRequest, Role, RoleRepository (+29 more)

### Community 1 - "OAuth Binding Repos"
Cohesion: 0.08
Nodes (65): fakeBindingRepo, BindingRepository, dupSubjectProviders(), githubProviders(), fakeSessionStore, sessionRecord, Context, DB (+57 more)

### Community 2 - "User Service DTOs"
Cohesion: 0.05
Nodes (37): CreatePermissionRequest, fakePermissionRepository, PermissionService, UpdatePermissionRequest, Permission, PermissionRepository, mysqlPermissionRepository, fakePermissionRepository (+29 more)

### Community 3 - "gRPC Server"
Cohesion: 0.05
Nodes (52): Container, moduleLogger, registerRouter, workerPoolComponent, Application, main(), run(), DBCollector (+44 more)

### Community 4 - "Storage Abstraction"
Cohesion: 0.05
Nodes (40): AuthorizationStore, Claims, DBAuthorizationStore, fakeAuthorizationStore, JWT, Policy, Enforcer, fakeAuthzStore (+32 more)

### Community 5 - "WebSocket Hub"
Cohesion: 0.06
Nodes (32): OAuthService, redisClientAdapter, RedisSessionStore, sessionPipeline, sessionRedis, SessionStore, BindingRepository, OAuthHandler (+24 more)

### Community 6 - "Notification Dispatcher"
Cohesion: 0.07
Nodes (35): ClientConn, Config, PingServer, pingService, Server, Context, RegisterPingServer(), Context (+27 more)

### Community 7 - "Queue Dead Letter"
Cohesion: 0.07
Nodes (40): testRole, CSVExporter, ExcelExporter, Exporter, Format, Registry, DB, T (+32 more)

### Community 8 - "Auth Redis Pipeline"
Cohesion: 0.14
Nodes (43): ChannelManager, Conn, connID(), ClientHub, Duration, JWT, Mutex, PresenceManager (+35 more)

### Community 9 - "Authorization Store"
Cohesion: 0.07
Nodes (28): Channel, Context, dispatcher, RWMutex, NewDispatcher(), Channel, Context, NewLogChannel() (+20 more)

### Community 10 - "E2E API Contract"
Cohesion: 0.06
Nodes (34): TestMysqlDeadLetterRepositoryCRUD(), DB, newHistoryTestDB(), TestMysqlJobHistoryRepositoryCreateAndList(), HandlerFunc, Locale(), TestLocaleParsesAcceptLanguage(), TestKafkaQueue_ConsumeUnmarshalError() (+26 more)

### Community 11 - "OAuth Service"
Cohesion: 0.07
Nodes (23): DeadLetter, DeadLetterRepository, Job, JobHistory, JobHistoryRepository, mysqlDeadLetterRepository, mysqlJobHistoryRepository, mysqlJobRepository (+15 more)

### Community 12 - "Spec Constitution"
Cohesion: 0.09
Nodes (14): APIKey, fakeAPIKeyRepo, fakeDeadLetterRepo, fakeEventBus, fakeImportJobRepo, fakeJobRepo, testRole, testUserRole (+6 more)

### Community 13 - "Admin Import Handler"
Cohesion: 0.11
Nodes (40): apiResp, testAppDB, snowflakeModel, stringKeyModel, doJSON(), DB, Engine, RawMessage (+32 more)

### Community 14 - "APIKey Repos"
Cohesion: 0.05
Nodes (41): Principle II: API Stability & SemVer, Principle I: Business-Agnostic, Principle III: Composable Modules, Jimu Constitution, Principle VI: Documentation & Examples, Governance & Revision Process, Principle IV: Simplicity & Minimal Dependencies, Principle V: Verification & Quality (+33 more)

### Community 15 - "Storage HTTP Fake"
Cohesion: 0.10
Nodes (32): AdminUserHandler, fakeUserRepository, AdminImportHandler, T, TestAdminAuditHandlerList(), DB, T, newSqliteDB() (+24 more)

### Community 16 - "OpenAPI Contract Test"
Cohesion: 0.13
Nodes (30): fakeStorage, Context, Duration, Engine, ReadCloser, Reader, Request, T (+22 more)

### Community 17 - "Outbox User Repo"
Cohesion: 0.11
Nodes (17): BatchDeleteRequest, BatchResult, CreateUserRequest, UpdateUserRequest, UserResponse, UserService, Cache, RedisCache (+9 more)

### Community 18 - "Gorm Logger"
Cohesion: 0.11
Nodes (28): APIKeyContextKey, APIKeyStore, APIKeyVerifier, dbAPIKeyStore, T, TestAPIKeyTableName(), TestHashKey(), APIKeyFromContext() (+20 more)

### Community 19 - "Specify Scripts"
Cohesion: 0.08
Nodes (17): check-prerequisites.sh script, check_dir(), check_file(), get_feature_paths(), get_repo_root(), has_jq(), _persist_feature_json(), resolve_specify_init_dir() (+9 more)

### Community 20 - "Management Server"
Cohesion: 0.09
Nodes (26): Handler, passingChecker, Server, HealthRouter(), NewManagementServer(), Context, TestManagementRouterExposure(), Client (+18 more)

### Community 21 - "APIKey Auth"
Cohesion: 0.12
Nodes (34): Jimu Helm Chart, Helm ConfigMap Template, Helm Deployment Template, Helm HorizontalPodAutoscaler Template, Helm Ingress Template, Helm NetworkPolicy Template, Helm PodDisruptionBudget Template, Helm PrometheusRule Template (+26 more)

### Community 22 - "Auth Module"
Cohesion: 0.06
Nodes (31): 1. User（用户）, 2. Role（角色）, 3. Permission（权限）, 4. APIKey（API 密钥）, 5. AuditLog（审计日志）, 6. OAuthBinding（第三方绑定）, 7. OutboxEvent（Outbox 事件）, 8. ScheduledJob（定时任务定义） (+23 more)

### Community 23 - "Helm Chart"
Cohesion: 0.13
Nodes (27): DBConfig, AutoMigrate(), findUp(), DB, Logger, isDir(), Migrate(), MigrateWithRetry() (+19 more)

### Community 24 - "Spec Data Model"
Cohesion: 0.16
Nodes (22): fakeBindingRepo, fakeProvider, fakeSessionStore, assertCode(), doRequest(), githubProvider(), Context, Duration (+14 more)

### Community 25 - "APIKey Repos (app)"
Cohesion: 0.13
Nodes (30): CacheConfig, CaptchaResult, EmailConfig, GRPCConfig, HTTPClientConfig, HTTPConfig, IDConfig, LogConfig (+22 more)

### Community 26 - "DB Config/Migrate"
Cohesion: 0.10
Nodes (20): New(), newOSSStorage(), Config, Context, Duration, ReadCloser, Reader, Storage (+12 more)

### Community 27 - "OAuth Binding (interfaces)"
Cohesion: 0.11
Nodes (14): AdminAPIKeyService, CreateKeyInput, APIKey, APIKeyRepository, mysqlAPIKeyRepository, APIKey, Context, NewAdminAPIKeyService() (+6 more)

### Community 28 - "Scheduler Memory Store"
Cohesion: 0.12
Nodes (13): fakePipeline, fakeRedis, BoolCmd, Cmder, IntCmd, BoolSliceCmd, Cmd, Context (+5 more)

### Community 29 - "User Repo Fake"
Cohesion: 0.12
Nodes (15): Module, AdminUserService, AdminUserHandler, DB, AdminAuthMiddleware(), HandlerFunc, Context, NewAdminUserHandler() (+7 more)

### Community 30 - "Admin APIKey Service"
Cohesion: 0.17
Nodes (27): New(), Sqlmock, newMockGormDB(), expectRunSeedQueries(), DB, Sqlmock, T, newSeedSqliteDB() (+19 more)

### Community 31 - "Admin Tasks Service"
Cohesion: 0.07
Nodes (26): Content Quality, Feature Readiness, Notes, Requirement Completeness, Specification Quality Checklist: Jimu 后端框架能力规格, Complexity Tracking, Constitution Check, Documentation (this feature) (+18 more)

### Community 32 - "Queue Worker Test"
Cohesion: 0.15
Nodes (20): AdminConfigHandler, AdminConfigService, Context, EventBus, NewAdminConfigService(), Miniredis, T, newConfigTestService() (+12 more)

### Community 33 - "Spec Checklists"
Cohesion: 0.15
Nodes (16): AuthService, Claims, AuthServiceInterface, TokenPair, ErrAccountLocked(), Context, Dispatcher, Duration (+8 more)

### Community 34 - "Admin Config Service"
Cohesion: 0.16
Nodes (25): targetFile, templateData, copyGoSum(), copyRootFile(), TestGeneratedModuleCompiles(), writeFileForTest(), writeStubPackages(), camel() (+17 more)

### Community 35 - "Auth Service"
Cohesion: 0.14
Nodes (11): Context, DeadLetter, Duration, Job, Mutex, JobHistory, fakeConsumer, fakeDeadRepo (+3 more)

### Community 36 - "Admin Module"
Cohesion: 0.14
Nodes (7): DeletedAt, User, mysqlRepository, fakeUserRepository, Time, Context, DB

### Community 37 - "Config Structs"
Cohesion: 0.21
Nodes (24): CSRF(), DefaultCSRFConfig(), generateToken(), Context, HandlerFunc, isSafeMethod(), setCSRFCookie(), csrfRouter() (+16 more)

### Community 38 - "Role Service"
Cohesion: 0.14
Nodes (4): fakeOutboxUserRepo, recordingOutboxStore, fakeUserRepository, Context

### Community 39 - "CSRF Middleware"
Cohesion: 0.18
Nodes (9): RoleHandler, UserHandler, Context, Context, DB, New(), Context, NewUserHandler() (+1 more)

### Community 40 - "WebSocket Handler"
Cohesion: 0.15
Nodes (15): Context, RawMessage, Time, mergeTrace(), New(), Context, T, TestMQPublisherPropagatesTrace() (+7 more)

### Community 41 - "Spec Tasks"
Cohesion: 0.08
Nodes (24): Dependencies & Execution Order, Format: `[ID] [P?] [Story] Description`, Implementation for User Story 1, Implementation for User Story 2, Implementation for User Story 3, Implementation for User Story 4, Implementation Strategy, Incremental Delivery (+16 more)

### Community 42 - "Observability Tracing"
Cohesion: 0.14
Nodes (16): circuit, circuitState, Client, Config, hostRateLimiter, Context, Duration, Limit (+8 more)

### Community 43 - "HTTP Client Circuit"
Cohesion: 0.18
Nodes (13): AuthHandler, forgotPasswordRequest, loginRequest, refreshRequest, resetPasswordRequest, authContext(), Context, Duration (+5 more)

### Community 44 - "Handler Notifier"
Cohesion: 0.13
Nodes (8): handlerNotifier, handlerSessionStore, handlerUserRepo, Channel, Context, Duration, Message, Notification

### Community 45 - "Dispatcher Fake"
Cohesion: 0.16
Nodes (9): Context, Duration, Context, Duration, NewRedisQueue(), errorQueue, fakeQueue, JobData (+1 more)

### Community 46 - "JWT Auth"
Cohesion: 0.17
Nodes (18): fakeDispatcher, fakeSessionStore, Channel, Context, Message, Miniredis, Mutex, Notification (+10 more)

### Community 47 - "OAuth Config"
Cohesion: 0.19
Nodes (18): GlobalRateLimit(), HandlerFunc, Limit, Limiter, RWMutex, NewRateLimiter(), Engine, Request (+10 more)

### Community 48 - "Rate Limit Middleware"
Cohesion: 0.16
Nodes (15): buildEmailHeaders(), Channel, Context, Message, NewEmail(), Conn, Listener, newFakeSMTPServer() (+7 more)

### Community 49 - "Email Notification"
Cohesion: 0.15
Nodes (6): fakeAPIKeyRepo, fakeEventBus, fakeImportJobRepo, Context, ImportJob, Mutex

### Community 50 - "Outbox MQ Publisher"
Cohesion: 0.17
Nodes (9): Cron, EntryID, Context, Lock, Logger, RWMutex, Time, CronScheduler (+1 more)

### Community 51 - "App Container"
Cohesion: 0.15
Nodes (12): contextKey, Flag, Manager, AdminFeatureHandler, Context, NewAdminFeatureHandler(), FromContext(), Context (+4 more)

### Community 52 - "Role Repo Fake"
Cohesion: 0.17
Nodes (16): BenchmarkWebhookSend(), B, Channel, Context, Message, NewWebhook(), signPayload(), T (+8 more)

### Community 53 - "DB Cleanup"
Cohesion: 0.13
Nodes (10): fakeEventBus, fakeStorage, Context, Duration, ReadCloser, Reader, T, TestModuleInitWSIdempotent() (+2 more)

### Community 54 - "Feature Flag"
Cohesion: 0.17
Nodes (9): fakeAuditRepository, fakeBatchRepository, AuditLog, Change, fakeQueue, Context, Context, Mutex (+1 more)

### Community 55 - "Code Generator"
Cohesion: 0.17
Nodes (12): CancelFunc, TraceFromContext(), GetWorker(), Context, Duration, Job, MySQLStore, WorkerConfig (+4 more)

### Community 56 - "Webhook Notification"
Cohesion: 0.18
Nodes (10): Context, fakeUserRepository, T, TestUserCreateReturnsCreatedDTO(), TestUserDeleteReturnsNoContent(), TestUserListRejectsInvalidPaginationContract(), TestUserListUsesDefaultPaginationBeforeService(), Config (+2 more)

### Community 57 - "Event Bus Fake"
Cohesion: 0.15
Nodes (8): RWMutex, Time, NewPresenceManager(), TestPresenceGet(), TestPresenceOnlineOffline(), TestPresenceOnlineUsers(), Presence, PresenceManager

### Community 58 - "WebSocket Presence"
Cohesion: 0.22
Nodes (17): JWT, Limiter, Service, RegisterAuthRoutes(), RegisterCaptchaRoute(), Engine, newRouterLimiter(), testAuthConfig() (+9 more)

### Community 59 - "Job Registry"
Cohesion: 0.19
Nodes (19): CORSConfig, corsRouter(), Engine, T, TestCORSAllowAllOrigins(), TestCORSAllowedOriginEcho(), TestCORSDisallowedOrigin(), TestCORSPreflight() (+11 more)

### Community 60 - "Permission Service"
Cohesion: 0.17
Nodes (14): Module, AuthConfig, CaptchaConfig, T, newHandlerService(), TestForgotPasswordHandler(), TestResetPasswordHandlerInvalidCode(), DB (+6 more)

### Community 61 - "Audit Worker"
Cohesion: 0.18
Nodes (16): RedisConfig, Client, newLock(), TestLock_AcquireRelease(), TestLock_ConcurrentAcquire(), TestLock_Extend(), TestLock_ReleaseOnlyOwnToken(), TestLock_WithLock() (+8 more)

### Community 62 - "Middleware Tests"
Cohesion: 0.25
Nodes (18): Buffer, Logger, T, newBufferLogger(), newGormLogger(), TestGormLogger_ErrorRedactsSensitive(), TestGormLogger_InfoRedactsSensitive(), TestGormLogger_LogMode() (+10 more)

### Community 63 - "Role/Permission Routers"
Cohesion: 0.22
Nodes (17): configurePool(), ConnectWithRetry(), dsn(), Context, DB, Logger, openByDriver(), openMySQL() (+9 more)

### Community 64 - "Queue Worker"
Cohesion: 0.21
Nodes (13): Context, Delivery, T, newTestRabbitQueue(), TestRabbitMQQueue_ConsumeTimeout(), TestRabbitMQQueue_ConsumeUnmarshalError(), TestRabbitMQQueue_NackRequeues(), TestRabbitMQQueue_SubmitConsumeAck() (+5 more)

### Community 65 - "App Container"
Cohesion: 0.19
Nodes (5): mustEncode(), NewClientHub(), BuildUserChannel(), broadcastMsg, ClientHub

### Community 66 - "Redis Cache"
Cohesion: 0.13
Nodes (6): fakeAuthzModule, fakeBusinessModule, JobRegistry, EventBus, HandlerFunc, TestBusinessRoutesRequireProtectedMiddleware()

### Community 67 - "Redis Lock"
Cohesion: 0.18
Nodes (11): AdminMonitoringService, HealthStatus, MemoryStats, SystemStatus, AdminMonitoringHandler, Client, Context, Time (+3 more)

### Community 68 - "Field Encryption"
Cohesion: 0.20
Nodes (5): fakeUserRepo, fakeSessionStore, sessionRecord, Context, Duration

### Community 69 - "Auth Handler"
Cohesion: 0.24
Nodes (17): freeAddr(), Logger, T, TestConfigureTrustedProxies(), TestFormatAddr(), testLogger(), TestNewManagementServer(), TestNewServerPlain() (+9 more)

### Community 70 - "Auth Service Test"
Cohesion: 0.21
Nodes (9): Conn, Context, HandlerFunc, RWMutex, Time, WSHandler(), Time, Now() (+1 more)

### Community 71 - "MySQL DB"
Cohesion: 0.18
Nodes (10): Module, AuditHandler, Context, NewAuditHandler(), TestAuditGetInvalidIDReturnsStableBadRequest(), TestAuditGetReturnsLogDTO(), TestAuditListInvalidQueryReturnsStableBadRequest(), DB (+2 more)

### Community 72 - "DB Seed"
Cohesion: 0.19
Nodes (13): Limiter, Context, Duration, Scripter, limitKey(), NewLimiter(), newTestLimiter(), TestLimiterAllowsOnlyConfiguredWindow() (+5 more)

### Community 73 - "RabbitMQ Queue"
Cohesion: 0.20
Nodes (11): gormLogger, Interface, Context, Duration, Logger, Time, isSensitiveField(), NewGormLogger() (+3 more)

### Community 74 - "Admin Monitoring"
Cohesion: 0.16
Nodes (13): Generator, snowflake, uuidGenerator, binaryID(), Mutex, NewSnowflake(), NewUUIDGenerator(), TestNewSnowflakeValidatesWorkerID() (+5 more)

### Community 75 - "OAuth/User Handlers"
Cohesion: 0.18
Nodes (13): HandlerFunc, Writer, GzipCompression(), isAlreadyCompressed(), Engine, T, gzipRouter(), TestGzipCompressionEncodesBody() (+5 more)

### Community 76 - "HTTP Server Test"
Cohesion: 0.18
Nodes (8): AdminCreateUserRequest, AdminUpdateUserRequest, AdminUser, ListUserFilter, userRole, Context, Pagination, int8Ptr()

### Community 77 - "Audit Service"
Cohesion: 0.22
Nodes (9): RedisStore, Service, Client, Context, Duration, NewRedisStore(), NewService(), TestGenerate() (+1 more)

### Community 78 - "Permission Repo Fake"
Cohesion: 0.22
Nodes (9): Connection, Context, Delivery, Duration, Mutex, NewRabbitMQQueue(), RabbitMQChannel, RabbitMQQueue (+1 more)

### Community 79 - "Snowflake ID"
Cohesion: 0.23
Nodes (7): AdminConfigHandler, AdminTaskHandler, Context, NewAdminConfigHandler(), Context, NewAdminTaskHandler(), OK()

### Community 80 - "Admin Job Handler"
Cohesion: 0.34
Nodes (13): NewWorkerPool(), RegisterWorker(), fakeStore(), MySQLStore, T, newFakeJobRepo(), TestWorkerPoolConsumeJob(), TestWorkerPoolConsumeRestoresTrace() (+5 more)

### Community 81 - "User Handler Test"
Cohesion: 0.18
Nodes (8): Context, RWMutex, Context, Time, failStore, JobDef, MemoryStore, Store

### Community 82 - "Gzip Middleware"
Cohesion: 0.27
Nodes (12): NewMemoryStore(), TestSetEnabledNotFound(), TestSetEnabledToggle(), TestTriggerJobNotFound(), TestTriggerJobRunsCommand(), NewWithStore(), newTestLogger(), TestAddNamedFuncPersistsAndRuns() (+4 more)

### Community 83 - "WebSocket Conn"
Cohesion: 0.17
Nodes (10): Context, DB, DeletedAt, Time, NewMySQLStore(), DB, testDB(), TestMySQLStore() (+2 more)

### Community 84 - "Audit Repo Fake"
Cohesion: 0.19
Nodes (5): RWMutex, NewChannel(), NewChannelManager(), Channel, ChannelManager

### Community 85 - "Captcha Redis Store"
Cohesion: 0.19
Nodes (12): New(), TestAppError(), TestAppErrorWithCause(), TestFailDoesNotLeakInfrastructureDetails(), TestOKIncludesStableEnvelopeAndRequestID(), TestPageIncludesStableEnvelopeAndPagination(), TestCreatedUsesStandardEnvelope(), TestFailHidesInternalCauseAndIncludesRequestID() (+4 more)

### Community 86 - "RabbitMQ Connection"
Cohesion: 0.27
Nodes (10): dbReachable(), envDBConfig(), DB, T, NewTestDB(), NewTestDBWithPool(), openByDriver(), SkipUnlessDB() (+2 more)

### Community 87 - "Config Load"
Cohesion: 0.21
Nodes (9): Application, Component, forwardError(), Context, Duration, NewApplication(), TestApplicationReturnsComponentRuntimeError(), TestApplicationRollsBackStartedComponents() (+1 more)

### Community 88 - "Scheduler MySQL Store"
Cohesion: 0.22
Nodes (11): ClientConnInterface, T, newTestGRPCService(), TestUserInfoService_GetUser(), TestUserInfoService_ListUsers(), NewUserInfoServiceClient(), RegisterUserInfoServiceServer(), ServiceRegistrar (+3 more)

### Community 89 - "WebSocket Channels"
Cohesion: 0.22
Nodes (9): AuditRepository, mysqlAuditRepository, AdminAuditHandler, DB, NewAdminAuditHandler(), deserializeChanges(), Context, DB (+1 more)

### Community 90 - "Shared Errors"
Cohesion: 0.19
Nodes (9): EventBus, Handler, RWMutex, New(), TestEventBus_Clear(), TestEventBus_HandlerPanicRecovered(), TestEventBus_MultipleHandlers(), TestEventBus_PublishAsync() (+1 more)

### Community 91 - "Test DB Util"
Cohesion: 0.26
Nodes (10): FileHeader, UploadConfig, UploadHandler, UploadResponse, Context, HandlerFunc, Reader, isAllowedType() (+2 more)

### Community 92 - "App Application"
Cohesion: 0.19
Nodes (8): ExcelImporter, ImportError, ImportResult, Context, Reader, NewExcelImporter(), Time, NewImportResult()

### Community 93 - "AuthZ Module Fake"
Cohesion: 0.21
Nodes (6): PermissionHandler, Context, DB, EventBus, New(), Module

### Community 94 - "gRPC UserInfo PB"
Cohesion: 0.30
Nodes (14): Config, T, TestDBPasswordOverride(), TestJWTSecretOverride(), TestLoad(), TestStorageConfigFieldMapping(), TestValidateHTTPMode(), TestValidateLogFormat() (+6 more)

### Community 95 - "Event Bus"
Cohesion: 0.22
Nodes (12): RouterGroup, RegisterPermissionRoutes(), Context, HandlerFunc, localeOf(), TestValidateJSONTranslatesFieldErrors(), translateValidationDetails(), translateValidationMessage() (+4 more)

### Community 96 - "Upload Handler"
Cohesion: 0.40
Nodes (13): TestMysqlRepositoryMySQLIntegration(), NewMysqlRepository(), DB, T, newTestUser(), newUserTestDB(), TestMysqlRepositoryCreateAndFindByID(), TestMysqlRepositoryFindByEmailHash() (+5 more)

### Community 97 - "User MySQL Repo"
Cohesion: 0.24
Nodes (13): HandlerFunc, Metrics(), CORSMiddleware(), DefaultLogConfig(), HandlerFunc, isAllowedOrigin(), Logger(), Recovery() (+5 more)

### Community 98 - "Metrics Middleware"
Cohesion: 0.28
Nodes (12): defaultKeyFunc(), Client, Context, Duration, HandlerFunc, Time, NewUserRateLimiter(), UserRateLimitMiddleware() (+4 more)

### Community 99 - "User Rate Limit"
Cohesion: 0.13
Nodes (13): Admin 模块, Auth 模块, HTTP API 契约, 中间件, 健康与观测（非业务）, 完整接口文档, 分页, 国际化 (+5 more)

### Community 100 - "Spec API Contract"
Cohesion: 0.19
Nodes (9): fakeRouter, Service, Client, Time, NewService(), Engine, RouterGroup, newFakeRouter() (+1 more)

### Community 101 - "Admin Router Fake"
Cohesion: 0.14
Nodes (9): ComponentProvider, ErrorSource, EventBus, HTTPMiddlewareProvider, Module, ProtectedHTTPMiddlewareProvider, Router, RouterGroup (+1 more)

### Community 102 - "Import Service"
Cohesion: 0.26
Nodes (8): ImportService, UserRepository, Context, DB, Format, Reader, NewImportService(), rulesFor()

### Community 103 - "Audit MySQL Repo"
Cohesion: 0.48
Nodes (13): appCode(), SessionStore, T, newFakeSessionStore(), newTestService(), TestLoginCreatesRefreshSession(), TestLoginHidesCredentialFailures(), TestLoginRejectsDisabledUser() (+5 more)

### Community 104 - "Role/Permission MySQL"
Cohesion: 0.30
Nodes (13): Engine, T, signatureRouter(), TestBuildSignString(), TestDefaultSignatureConfig(), TestHmacSign(), TestSignatureExpired(), TestSignatureInvalidSignature() (+5 more)

### Community 105 - "Role Repo (interfaces)"
Cohesion: 0.33
Nodes (8): generateToken(), Client, Context, Duration, Time, NewLock(), AcquireResult, Lock

### Community 106 - "Permission Handler"
Cohesion: 0.15
Nodes (10): RawMessage, Time, NewMessage(), ChatPayload, NotificationPayload, PingPayload, PongPayload, PresencePayload (+2 more)

### Community 107 - "Signature Test"
Cohesion: 0.16
Nodes (5): MessageState, SizeCache, UnknownFields, GetUserRequest, ListUsersRequest

### Community 108 - "Kafka Queue"
Cohesion: 0.29
Nodes (10): CleanupConfig, CleanupResult, CleanupService, CleanupTable, DefaultCleanupConfig(), Context, DB, Time (+2 more)

### Community 109 - "Redis Lock Impl"
Cohesion: 0.28
Nodes (10): contact, contactPtr, DB, T, newEncryptionTestDB(), TestEncryptionHookBatchCreate(), TestEncryptionHookEmptySourceStoresNullAndCoexists(), TestEncryptionHookEncryptsOnWriteDecryptsOnRead() (+2 more)

### Community 110 - "UserInfo PB Messages"
Cohesion: 0.21
Nodes (13): Alert Rules (Jimu HTTP/Infra/Queue/Process), AlertManager Routing Config, Grafana Datasources (Prometheus+Loki), Prometheus Scrape Config, Promtail Log Collection Config, Docker Compose Observability Stack, CI Pipeline, Coverage Threshold Gate (70%) (+5 more)

### Community 111 - "Module Contract"
Cohesion: 0.23
Nodes (7): ImportJob, ImportJobRepository, mysqlImportJobRepository, Time, Context, DB, NewMysqlImportJobRepository()

### Community 112 - "Import Job Repo"
Cohesion: 0.28
Nodes (11): Context, Created(), FailWithDetails(), Context, localeFrom(), NoContent(), Page(), requestID() (+3 more)

### Community 113 - "DB Cleanup Service"
Cohesion: 0.38
Nodes (12): New(), T, TestCircuitHalfOpenOnlyAllowsProbe(), TestCircuitOpensAfterConsecutiveFailures(), TestCircuitRecoversAfterCooldown(), TestDoCancelsDuringRetryWait(), TestDoInjectsTraceParent(), TestDoNoRetryOn4xx() (+4 more)

### Community 114 - "DB Contact Model"
Cohesion: 0.19
Nodes (8): Config, Context, NewGoogleProvider(), T, TestGoogleAuthURL(), TestProvidersImplementInterface(), GoogleConfig, GoogleProvider

### Community 115 - "Observability Deploy"
Cohesion: 0.21
Nodes (9): Context, RawMessage, NewMQPublisher(), TestMQPublisher_Publish(), TestMQPublisher_PublishError(), traceFromMetadata(), MQPublisher, Consumer (+1 more)

### Community 116 - "gRPC UserInfo Service"
Cohesion: 0.19
Nodes (9): Client, Queue, New(), TestNew_InvalidType(), TestNew_Redis(), Config, KafkaConfig, RabbitMQConfig (+1 more)

### Community 117 - "Admin Audit Handler"
Cohesion: 0.28
Nodes (7): Context, Duration, NewKafkaQueue(), KafkaConfig, KafkaMessageReader, KafkaMessageWriter, KafkaQueue

### Community 118 - "OAuth MySQL Repo"
Cohesion: 0.30
Nodes (7): AuditLogResponse, AuditService, Time, ToAuditLogResponse(), ToAuditLogResponses(), Context, serializeChanges()

### Community 120 - "OAuth Google Provider"
Cohesion: 0.24
Nodes (9): Queue, AuditMiddleware(), Context, HandlerFunc, isManagementPath(), optionalString(), optionalUint64(), TestAuditMiddlewareAllowsAnonymousRequest() (+1 more)

### Community 121 - "Queue Factory"
Cohesion: 0.33
Nodes (8): LockoutConfig, LoginFailureTracker, DefaultLockoutConfig(), Client, Context, Duration, lockKey(), NewLoginFailureTracker()

### Community 122 - "Password Reset Store"
Cohesion: 0.26
Nodes (10): cleanupModel, noTableNameModel, T, TestCleanupService_RunBatches(), TestCleanupService_RunCustomDeletedColumn(), TestCleanupService_RunError(), TestCleanupService_RunMultipleTables(), TestDefaultCleanupConfig() (+2 more)

### Community 123 - "Login Failure Tracker"
Cohesion: 0.23
Nodes (9): CSVImporter, Context, Reader, NewCSVImporter(), csvReader(), Reader, TestCSVParseAndValidate(), TestCSVParseEmptyFile() (+1 more)

### Community 124 - "CSV Importer"
Cohesion: 0.29
Nodes (6): routerLimiterRedis, BoolSliceCmd, Cmd, Context, StringCmd, routerLimiterInt()

### Community 125 - "Audit Handler"
Cohesion: 0.17
Nodes (7): buildProviders(), EventBus, JobRegistry, OAuthService, Provider, Router, Module

### Community 126 - "Permission Repo (interfaces)"
Cohesion: 0.35
Nodes (10): NewUserService(), appCode(), createOutboxUserService(), T, TestCreateWritesOutbox(), TestUpdateAndDeleteWriteOutbox(), TestUserResponseDoesNotContainPassword(), TestUserServiceBatchDelete() (+2 more)

### Community 127 - "Audit Queue Fake"
Cohesion: 0.26
Nodes (6): Context, DB, NewMySQLStore(), TestMySQLStore_AddWithinTx(), TestMySQLStore_AddWithNilTx(), MySQLStore

### Community 128 - "Router Redis Limiter"
Cohesion: 0.30
Nodes (11): T, TestPresenceIsOnline(), TestPresenceIsTyping(), TestPresenceManagerAllPresences(), TestPresenceManagerHeartbeat(), TestPresenceManagerHeartbeatRestoresOnlineStatus(), TestPresenceManagerOfflineMissing(), TestPresenceManagerOnlineCountFiltersOffline() (+3 more)

### Community 129 - "Locale/i18n Middleware"
Cohesion: 0.27
Nodes (6): AdminTaskService, TaskExecution, TaskInfo, Context, Time, NewAdminTaskService()

### Community 130 - "Outbox MySQL Store"
Cohesion: 0.29
Nodes (6): Worker, AuditConfig, Context, RWMutex, NewWorker(), Once

### Community 131 - "Redis Queue"
Cohesion: 0.33
Nodes (4): JobRepository, AdminJobHandler, Context, NewAdminJobHandler()

### Community 132 - "Presence State Test"
Cohesion: 0.25
Nodes (6): OAuthBinding, MySQLBindingRepository, Time, Context, DB, NewMySQLBindingRepository()

### Community 133 - "Admin Task Service"
Cohesion: 0.33
Nodes (7): FieldRule, FieldType, ValidationRules, Validator, checkType(), Context, NewValidator()

### Community 134 - "OAuth Binding Domain"
Cohesion: 0.44
Nodes (10): Int32, Engine, T, idempotencyRouter(), newRedisForTest(), TestIdempotencyCachedResponseSerializes(), TestIdempotencyCachesAndReplays(), TestIdempotencyDoesNotCacheFailure() (+2 more)

### Community 135 - "Importer Validation"
Cohesion: 0.27
Nodes (9): AdminAuth(), HandlerFunc, adminAuthRouter(), Engine, TestAdminAuthAllowsAdminRole(), TestAdminAuthAllowsEnglishAdminRole(), TestAdminAuthRejectsNonAdminRole(), TestAdminAuthRequiresAuthentication() (+1 more)

### Community 136 - "Idempotency Middleware"
Cohesion: 0.24
Nodes (6): Buffer, ResponseWriter, newResponseBodyWriter(), sanitizeBody(), sanitizeJSONField(), responseBodyWriter

### Community 137 - "Admin Auth Middleware"
Cohesion: 0.25
Nodes (7): Channel, Context, Message, T, TestDispatcherDispatch(), TestDispatcherDispatchBatch(), mockNotification

### Community 138 - "Body Recorder"
Cohesion: 0.33
Nodes (10): T, TestChannelManagerGetChannelMissing(), TestChannelManagerGetSubscribers(), TestChannelManagerPreCreatedBroadcast(), TestChannelManagerSubscribeCreatesWithType(), TestChannelManagerSubscribeExisting(), TestChannelManagerUnsubscribe(), TestChannelManagerUnsubscribeAll() (+2 more)

### Community 139 - "Dispatcher Test"
Cohesion: 0.24
Nodes (6): AtomicLevel, Context, LogConfig, New(), Logger, SugaredLogger

### Community 140 - "WebSocket Channels Test"
Cohesion: 0.33
Nodes (10): Jimu API Swagger Definition, Audit Log Endpoints, Auth Flow Endpoints (login/refresh/logout/register), BearerAuth Security (API Key JWT), Captcha Endpoint, contract.ErrorResponse, OAuth Endpoints, contract.PageResponse (+2 more)

### Community 141 - "OpenAPI Swagger Docs"
Cohesion: 0.29
Nodes (6): Server, ConfigureTrustedProxies(), formatAddr(), Context, Engine, NewServer()

### Community 142 - "HTTP Server"
Cohesion: 0.20
Nodes (4): RouterGroup, RegisterRoleRoutes(), EventBus, Module

### Community 143 - "Permission MySQL Repo"
Cohesion: 0.36
Nodes (9): Engine, T, TestDefaultKeyFuncPriority(), TestUserRateLimiterAllowsWithinLimit(), TestUserRateLimiterFailOpenOnRedisError(), TestUserRateLimiterOptions(), TestUserRateLimiterRejectsOverLimit(), userRateRouter() (+1 more)

### Community 144 - "User Rate Limit Test"
Cohesion: 0.33
Nodes (9): buildSignString(), DefaultSignatureConfig(), Context, Duration, HandlerFunc, hmacSign(), Signature(), SignRequest() (+1 more)

### Community 145 - "Signature Middleware"
Cohesion: 0.60
Nodes (9): T, mockClient(), mockTokenServer(), TestGitHubProviderExchange(), TestGoogleProviderExchange(), TestGoogleProviderExchangeBadUserInfo(), TestGoogleProviderExchangeTokenError(), TestWeChatProviderExchange() (+1 more)

### Community 146 - "OAuth Exchange Test"
Cohesion: 0.39
Nodes (8): AdminWSHandler, ClientHub, PresenceManager, T, newWSHandler(), TestAdminWSHandlerOnlineUsers(), TestAdminWSHandlerPresence(), TestAdminWSHandlerPush()

### Community 147 - "internal_platform"
Cohesion: 0.28
Nodes (5): userInfoService, DB, Context, DB, NewUserInfoGRPCService()

### Community 148 - "internal_platform"
Cohesion: 0.22
Nodes (7): RouterGroup, RegisterUserRoutes(), Client, Duration, HandlerFunc, IdempotencyMiddleware(), cachedResponse

### Community 149 - "user module"
Cohesion: 0.22
Nodes (4): EventBus, JobRegistry, Router, Module

### Community 150 - "userinfopb unimplementeduserinfoservices"
Cohesion: 0.33
Nodes (5): Context, _UserInfoService_GetUser_Handler(), _UserInfoService_ListUsers_Handler(), UnaryServerInterceptor, UnimplementedUserInfoServiceServer

### Community 151 - "internal_platform"
Cohesion: 0.22
Nodes (6): Duration, HandlerFunc, TestTimeoutOnlyPropagatesContextDeadline(), Timeout(), RouterGroup, RegisterSwagger()

### Community 152 - "internal_platform"
Cohesion: 0.36
Nodes (5): T, TestHubLifecycle(), TestWebSocketNotification(), waitHub(), mockConn

### Community 153 - "internal_platform"
Cohesion: 0.28
Nodes (5): Config, Context, NewGitHubProvider(), GitHubConfig, GitHubProvider

### Community 154 - "internal_platform"
Cohesion: 0.28
Nodes (5): Config, Context, NewWeChatProvider(), WeChatConfig, WeChatProvider

### Community 155 - "internal_platform"
Cohesion: 0.39
Nodes (8): T, TestBuildRoomChannel(), TestBuildUserChannel(), TestNewMessage(), TestNewMessageMarshalError(), TestWSMessageDecodePayload(), TestWSMessageDecodePayloadError(), TestWSMessageEncodeRoundTrip()

### Community 156 - "modules"
Cohesion: 0.54
Nodes (7): AdminTaskHandler, T, newTaskHandler(), TestAdminTaskHandlerHistory(), TestAdminTaskHandlerList(), TestAdminTaskHandlerToggle(), TestAdminTaskHandlerTrigger()

### Community 157 - "shared"
Cohesion: 0.36
Nodes (5): AppError, ErrorInfo, AllErrorCodes(), As(), HTTPStatus()

### Community 158 - "shared"
Cohesion: 0.39
Nodes (6): FieldLevel, Validate(), validateIDCard(), validateMobile(), validatePassword(), validateUsername()

### Community 159 - "importer registry"
Cohesion: 0.54
Nodes (5): Format, Importer, Registry, NewRegistry(), TestRegistryGetUnsupported()

### Community 160 - "interfaces adminapikeyhandler"
Cohesion: 0.43
Nodes (3): AdminAPIKeyHandler, Context, NewAdminAPIKeyHandler()

### Community 161 - "modules"
Cohesion: 0.43
Nodes (7): T, TestAdminJobHandlerGet(), TestAdminJobHandlerList(), TestAdminJobHandlerListDeadLetters(), TestAdminJobHandlerResolveDeadLetter(), TestAdminJobHandlerRetry(), TestAdminJobHandlerSubmit()

### Community 162 - "internal_platform"
Cohesion: 0.57
Nodes (7): applyBlindIndexFields(), applyEncryptedFields(), DB, Field, Value, RegisterEncryptionHooks(), walkElements()

### Community 163 - "internal_platform"
Cohesion: 0.29
Nodes (3): file_proto_jimu_v1_userinfo_proto_init(), file_proto_jimu_v1_userinfo_proto_rawDescGZIP(), init()

### Community 164 - "internal_platform"
Cohesion: 0.43
Nodes (7): ContextWithTrace(), DefaultTracingConfig(), Context, TracerProvider, InitTracing(), ShutdownTracing(), TracingConfig

### Community 165 - "internal_platform"
Cohesion: 0.32
Nodes (6): Context, EventBus, RawMessage, NewEventBusPublisher(), EventBusPublisher, EventPayload

### Community 166 - "internal_platform"
Cohesion: 0.32
Nodes (6): NewLocalStorage(), TestLocalStorageFilePersisted(), TestLocalStorageUploadDownloadDelete(), TestLocalStorageURL(), TestNewS3RequiresBucket(), TestNewUnsupportedType()

### Community 167 - "modules"
Cohesion: 0.57
Nodes (6): AdminMonitoringHandler, T, newMonitoringHandler(), TestAdminMonitoringHandlerHealth(), TestAdminMonitoringHandlerMetrics(), TestAdminMonitoringHandlerStatus()

### Community 168 - "modules"
Cohesion: 0.43
Nodes (4): ResetStore, Context, Duration, NewResetStore()

### Community 169 - "shared"
Cohesion: 0.33
Nodes (4): contains(), Config, FuzzValidateRules(), F

### Community 170 - "internal_platform"
Cohesion: 0.43
Nodes (6): SecurityConfig, DefaultSecurityConfig(), Context, HandlerFunc, SecurityHeadersFromConfig(), writeSecurityHeaders()

### Community 171 - "interfaces adminwshandler"
Cohesion: 0.43
Nodes (3): AdminWSHandler, Context, NewAdminWSHandler()

### Community 172 - "modules"
Cohesion: 0.67
Nodes (6): T, newTestScheduler(), TestAdminTaskServiceGetHistory(), TestAdminTaskServiceListTasks(), TestAdminTaskServiceToggleTask(), TestAdminTaskServiceTriggerTask()

### Community 173 - "modules"
Cohesion: 0.57
Nodes (6): DB, T, newRepoTestDB(), TestMysqlAPIKeyRepository(), TestMysqlImportJobRepository(), TestMysqlJobRepository()

### Community 174 - "internal_platform"
Cohesion: 0.52
Nodes (6): setupTestCache(), TestRedisCache_Delete(), TestRedisCache_DeletePattern(), TestRedisCache_GetOrSet(), TestRedisCache_GetOrSetStampede(), TestRedisCache_SetAndGet()

### Community 175 - "internal_platform"
Cohesion: 0.48
Nodes (6): T, TestFileOutput(), TestNewConsoleStdout(), TestNewJSONLevels(), TestSetLevel(), TestWithContext()

### Community 176 - "internal_platform"
Cohesion: 0.43
Nodes (4): Duration, ExponentialBackoff, FixedRetry, RetryStrategy

### Community 178 - "internal_platform"
Cohesion: 0.47
Nodes (4): CaptchaHandler, Context, Service, NewCaptchaHandler()

### Community 180 - "modules"
Cohesion: 0.47
Nodes (4): Handler, Context, Service, NewHandler()

### Community 181 - "modules"
Cohesion: 0.53
Nodes (5): T, TestAdminAPIKeyServiceCreateKey(), TestAdminAPIKeyServiceGetKey(), TestAdminAPIKeyServiceListKeys(), TestAdminAPIKeyServiceRevokeKey()

### Community 182 - "modules"
Cohesion: 0.53
Nodes (5): T, TestAdminAPIKeyHandlerCreate(), TestAdminAPIKeyHandlerGet(), TestAdminAPIKeyHandlerList(), TestAdminAPIKeyHandlerRevoke()

### Community 183 - "modules"
Cohesion: 0.53
Nodes (5): Duration, testWorker(), TestWorkerFlushesConfiguredBatch(), TestWorkerRejectsWhenQueueFull(), TestWorkerStopDrainsAcceptedRecords()

### Community 184 - "internal_platform"
Cohesion: 0.53
Nodes (5): T, TestResponseBodyWriterCapturesAndTruncates(), TestResponseBodyWriterWriteHeaderPassthrough(), TestSanitizeBody(), TestSanitizeJSONFieldEdgeCases()

### Community 185 - "internal_platform"
Cohesion: 0.60
Nodes (5): gatherHTTPMetric(), Engine, setupMetricsEngine(), TestMetricsLabelsUseRouteTemplate(), TestMetricsRecordsUnmatchedPath()

### Community 186 - "internal_platform"
Cohesion: 0.53
Nodes (5): Engine, securityRouter(), TestSecurityHandlesAllowedPreflight(), TestSecurityRejectsOversizedBody(), TestSecurityUsesOriginAllowList()

### Community 187 - "internal_platform"
Cohesion: 0.47
Nodes (3): DB, NewDBCollector(), DBCollector

### Community 188 - "shared"
Cohesion: 0.40
Nodes (3): Ptr(), RandomEmail(), RandomString()

### Community 189 - "specs 001"
Cohesion: 0.33
Nodes (5): 关键配置组, 加载优先级, 敏感值注入（Docker Secrets）, 枚举约束（非法值启动报错）, 配置契约

### Community 190 - "specs 001"
Cohesion: 0.33
Nodes (5): Module 契约, 中间件挂载, 分层结构, 模块接口, 组件生命周期

### Community 192 - "specs 001"
Cohesion: 0.50
Nodes (5): Development Config (app.yaml), Production Config (app.prod.yaml), Config Contract, Config Enum Validation (FR-002), File-Based Secret Injection (FR-003)

### Community 193 - "contract usercreatedevent"
Cohesion: 0.40
Nodes (4): UserCreatedEvent, UserDeletedEvent, UserLoggedInEvent, UserUpdatedEvent

### Community 194 - "modules"
Cohesion: 0.60
Nodes (4): NewAuditService(), auditAppCode(), TestAuditServiceGetMapsNotFound(), TestAuditServiceListReturnsDTOAndPassesPagination()

### Community 195 - "internal_platform"
Cohesion: 0.60
Nodes (4): T, TestFlagIsEnabled(), TestHashUserIDStable(), TestManagerLifecycle()

### Community 197 - "internal_platform"
Cohesion: 0.50
Nodes (4): addVary(), Context, HandlerFunc, Security()

### Community 198 - "internal_platform"
Cohesion: 0.60
Nodes (4): T, TestEntityValues(), TestJobStatusConstants(), TestTableNames()

### Community 199 - "specs 001"
Cohesion: 0.40
Nodes (4): CLI 契约, 命令表, 种子数据, 迁移命名

### Community 200 - "github"
Cohesion: 0.50
Nodes (4): Release Workflow, Conventional Commits, Release Note Structure, Commit Style

### Community 201 - "pre"
Cohesion: 0.50
Nodes (4): No .env & Secrets Hook, Password Strength Rule, Security Policy, Vulnerability Reporting Process

### Community 202 - "contract"
Cohesion: 0.83
Nodes (3): assertQueryParams(), readOpenAPI(), TestOpenAPIIncludesCRUDContract()

### Community 203 - "modules"
Cohesion: 0.67
Nodes (3): T, TestAdminMonitoringServiceGetHealth(), TestAdminMonitoringServiceGetStatus()

### Community 204 - "modules"
Cohesion: 0.67
Nodes (3): T, TestAdminFeatureHandlerList(), TestAdminFeatureHandlerUpdate()

### Community 205 - "modules"
Cohesion: 0.67
Nodes (3): BenchmarkLogin(), benchUser(), B

### Community 206 - "modules"
Cohesion: 0.67
Nodes (3): T, TestEntityValues(), TestTableNames()

### Community 207 - "modules"
Cohesion: 0.67
Nodes (3): T, TestUserFields(), TestUserTableName()

### Community 208 - "internal_platform"
Cohesion: 0.67
Nodes (3): DB, Transaction(), WithTx()

### Community 209 - "shared"
Cohesion: 0.67
Nodes (3): BenchmarkSnowflakeNextID(), BenchmarkUUIDNextID(), B

### Community 210 - "migrations 010"
Cohesion: 0.50
Nodes (3): dead_letters, job_history, jobs

### Community 211 - "migrations postgres"
Cohesion: 0.50
Nodes (3): dead_letters, job_history, jobs

### Community 212 - "golangci"
Cohesion: 0.67
Nodes (3): GolangCI Lint Configuration, Lint Rule Set, Pre-commit Hooks Configuration

### Community 213 - "specs 001"
Cohesion: 0.67
Nodes (3): Unreleased Capabilities, Aliyun SMS Implementation (D2), Notification Abstraction (FR-028)

### Community 225 - "specs 001"
Cohesion: 0.67
Nodes (3): Permission Entity, Role Entity, RBAC (FR-010)

## Knowledge Gaps
- **255 isolated node(s):** `common.sh script`, `ErrorSource`, `ComponentProvider`, `HTTPMiddlewareProvider`, `ProtectedHTTPMiddlewareProvider` (+250 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **124 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `T()` connect `E2E API Contract` to `Admin User Management`, `User Service DTOs`, `gRPC Server`, `Storage Abstraction`, `Importer Validation`, `Authorization Store`, `Admin Import Handler`, `Gorm Logger`, `Management Server`, `internal_platform`, `importer registry`, `Admin Config Service`, `internal_platform`, `internal_platform`, `Rate Limit Middleware`, `modules`, `internal_platform`, `WebSocket Presence`, `internal_platform`, `Event Bus Fake`, `Audit Worker`, `shared`, `Redis Cache`, `modules`, `MySQL DB`, `DB Seed`, `contract`, `Admin Monitoring`, `Audit Service`, `Gzip Middleware`, `WebSocket Conn`, `Captcha Redis Store`, `Config Load`, `Shared Errors`, `Event Bus`, `Upload Handler`, `Spec API Contract`, `Import Job Repo`, `Observability Deploy`, `gRPC UserInfo Service`, `OAuth Google Provider`, `Login Failure Tracker`, `Audit Queue Fake`?**
  _High betweenness centrality (0.270) - this node is a cross-community bridge._
- **Why does `Client` connect `Observability Tracing` to `Queue Worker Test`, `OAuth Binding Repos`, `gRPC Server`, `DB Config/Migrate`, `OAuth Binding Domain`, `modules`, `Dispatcher Fake`, `JWT Auth`, `Outbox User Repo`, `DB Cleanup Service`, `Signature Middleware`, `Role Repo Fake`, `DB Contact Model`, `user module`, `internal_platform`, `internal_platform`, `Permission Service`, `Audit Handler`?**
  _High betweenness centrality (0.102) - this node is a cross-community bridge._
- **Why does `Now()` connect `Auth Service Test` to `Storage Abstraction`, `Admin Task Service`, `E2E API Contract`, `OAuth Service`, `Gorm Logger`, `Management Server`, `OAuth Binding (interfaces)`, `CSRF Middleware`, `Event Bus Fake`, `shared`, `App Container`, `Redis Lock`, `Admin Monitoring`, `User Handler Test`, `Gzip Middleware`, `Test DB Util`, `App Application`, `User MySQL Repo`, `Metrics Middleware`, `Spec API Contract`, `Import Service`, `Role Repo (interfaces)`, `Permission Handler`, `Kafka Queue`, `OAuth Google Provider`, `Audit Queue Fake`?**
  _High betweenness centrality (0.086) - this node is a cross-community bridge._
- **Are the 65 inferred relationships involving `Fail()` (e.g. with `.Generate()` and `.HandleDelete()`) actually correct?**
  _`Fail()` has 65 INFERRED edges - model-reasoned connections that need verification._
- **Are the 47 inferred relationships involving `Wrap()` (e.g. with `.CreateKey()` and `.AssignRoles()`) actually correct?**
  _`Wrap()` has 47 INFERRED edges - model-reasoned connections that need verification._
- **Are the 41 inferred relationships involving `New()` (e.g. with `.ForgotPassword()` and `.Logout()`) actually correct?**
  _`New()` has 41 INFERRED edges - model-reasoned connections that need verification._
- **What connects `common.sh script`, `ErrorSource`, `ComponentProvider` to the rest of the system?**
  _255 weakly-connected nodes found - possible documentation gaps or missing edges._