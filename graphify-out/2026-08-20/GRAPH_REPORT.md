# Graph Report - jimu  (2026-08-20)

## Corpus Check
- 402 files · ~132,358 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3744 nodes · 8280 edges · 258 communities (225 shown, 33 thin omitted)
- Extraction: 82% EXTRACTED · 18% INFERRED · 0% AMBIGUOUS · INFERRED: 1505 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `0fad5204`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- RoleService
- T
- InitSnowflake
- RunSeed
- Event
- upload_handler_test.go
- RateLimiter
- SessionStore
- Data Model: Jimu 框架持久化实体
- User
- Implementation Plan: Jimu 后端框架能力规格
- ws_integration_test.go
- New
- config/config.go
- Tasks: Jimu 后端框架能力规格
- oauth/application/service_test.go
- gorm_logger_test.go
- Context
- NewAdminConfigService
- AuthHandler
- WorkerPool
- AdminMonitoringService
- OAuthService
- Manager
- DefaultCSRFConfig
- middleware/middleware_test.go
- Context
- migrate.go
- NewCleanupService
- JobData
- newImportHandler
- NewUserRateLimiter
- Hub
- Wrap
- JWT
- generator/module.go
- circuit
- signature_test.go
- NewEmail
- OK
- Message
- ImportJob
- auth/apikey.go
- RegisterAuthRoutes
- New
- UserService
- NewWebhook
- Module
- AuditLog
- New
- CronScheduler
- Context
- Page
- NewWithStore
- NewAdminAPIKeyService
- Context
- Context
- api_contract_test.go
- Job
- mockTokenServer
- health.go
- Context
- responseBodyWriter
- New
- oauth/interfaces/handler_test.go
- NewAuthService
- rabbitmq_queue_test.go
- PresenceManager
- NewPresenceManager
- dispatcher
- AuthorizationMiddleware
- New
- fakeRedis
- DBConfig
- NewCSVExporter
- New
- SetupRouter
- ChannelManager
- NewService
- AdminUserService
- Worker
- Server
- RedisStore
- Context
- DeadLetter
- ImportResult
- newMockGormDB
- IdempotencyMiddleware
- gzipRouter
- NewSMS
- NewWorkerPool
- Permission
- MySQLStore
- newResetService
- New
- TestDB
- ListUsersRequest
- Application
- fakeAuthzModule
- JobHistory
- OAuthBinding
- NewMysqlRepository
- NewDBAPIKeyStore
- ImportService
- newMockGormDB
- InitTracing
- RabbitMQQueue
- Lock
- RedisCache
- Limiter
- fakeStorage
- Container
- AuditLogResponse
- NewManagementServer
- mysqlAPIKeyRepository
- mysqlRepository
- NewServer
- New
- KafkaQueue
- AuditService
- Now
- New
- newEncryptionTestDB
- LoginFailureTracker
- NewAdminUserService
- 统一响应契约
- userinfo_grpc.pb.go
- newWSHandler
- routerLimiterRedis
- i18n.go
- ValidateJSON
- newLock
- mysqlAuditRepository
- validator/validator.go
- NewLocalStorage
- NewSnowflake
- Logger
- AGENTS.md
- AdminUserHandler
- adminAuthRouter
- bindImportFile
- GitHubProvider
- NewWeChatProvider
- testConn
- AdminTaskService
- SecurityHeadersFromConfig
- Router
- NewCSVImporter
- .Validate
- AuditMiddleware
- NewDBCollector
- Policy
- TestWebSocketNotification
- fakeContainer
- CLAUDE.md
- setupMetricsEngine
- Timeout
- NewLogChannel
- ListUsersResponse
- UserInfo
- ResetStore
- 贡献指南
- EventBus
- .List
- Registry
- API 示例
- MySQLStore
- userinfo.pb.go
- NewEventBusPublisher
- RoleHandler
- newRedisTestQueue
- mysql/003_extensions.sql
- postgres/003_extensions.sql
- Bootstrap
- contract/module.go
- Registry
- startTestGRPCServer
- newTestScheduler
- RegisterEncryptionHooks
- Jimu Backend Framework
- setupTestCache
- GoogleProvider
- Duration
- 安全政策
- .GetUser
- Server
- message.go
- [Unreleased]
- AdminAPIKeyHandler
- newHistoryTestDB
- Security
- newRepoTestDB
- mockNotification
- TestReadinessBoundsCheckerDuration
- PermissionService
- mysql/001_core.sql
- postgres/001_core.sql
- bug_report.md
- fakeComponent
- events.go
- 快速开始
- 分支策略
- newMockS3Storage
- newRateLimitHandler
- CaptchaHandler
- .validateCommon
- PermissionMiddleware
- PULL_REQUEST_TEMPLATE.md
- openapi.go
- AdminJobHandler
- mysql/002_audit_outbox.sql
- postgres/002_audit_outbox.sql
- smoke_api_contract.sh
- test_backup_restore.sh
- test_runtime_security.sh
- LoginRequest
- backup.sh
- bench_ci.sh
- loadtest.sh
- restore.sh
- jimu
- AdminWSHandler
- As
- 配置契约
- Module 契约
- 编码约束
- 配置约束
- feature_request.md
- CLI 契约
- 架构约束
- securityRouter
- 配置说明
- TestGeneratedModuleCompiles
- AdminAuthMiddleware
- fakeEventBus
- RegisterRoleRoutes
- newTestGRPCService
- wsFixture
- NewPathEnforcer
- Module
- EventBus
- fakeAuditRepository
- JobRegistry
- HandlerFunc
- Pagination
- Fail
- request.go
- JobDef
- Service
- TestOpenAPIIncludesCRUDContract
- Duration
- Scripter
- Sqlmock

## God Nodes (most connected - your core abstractions)
1. `T()` - 762 edges
2. `New()` - 95 edges
3. `Fail()` - 82 edges
4. `Now()` - 82 edges
5. `User` - 72 edges
6. `OK()` - 55 edges
7. `New()` - 52 edges
8. `Wrap()` - 49 edges
9. `NewServer()` - 37 edges
10. `Message` - 36 edges

## Surprising Connections (you probably didn't know these)
- `run()` --calls--> `Bootstrap()`  [INFERRED]
  cmd/server/main.go → internal/app/bootstrap.go
- `run()` --calls--> `NewContainer()`  [INFERRED]
  cmd/server/main.go → internal/app/container.go
- `run()` --calls--> `NewWithRotation()`  [INFERRED]
  cmd/server/main.go → internal/platform/auth/jwt.go
- `run()` --calls--> `Load()`  [INFERRED]
  cmd/server/main.go → internal/config/config.go
- `run()` --calls--> `Watch()`  [INFERRED]
  cmd/server/main.go → internal/config/config.go

## Import Cycles
- None detected.

## Communities (258 total, 33 thin omitted)

### Community 0 - "RoleService"
Cohesion: 0.14
Nodes (15): AssignPermissionsRequest, CreateRoleRequest, RoleResponse, RoleService, UpdateRoleRequest, PermissionResponse, Time, TestRoleResponseUsesDTO() (+7 more)

### Community 1 - "T"
Cohesion: 0.04
Nodes (57): Config, TestDBPasswordOverride(), TestJWTSecretOverride(), TestLoad(), TestStorageConfigFieldMapping(), TestValidateHTTPMode(), TestValidateLogFormat(), TestValidateLogLevel() (+49 more)

### Community 2 - "InitSnowflake"
Cohesion: 0.14
Nodes (22): snowflakeModel, stringKeyModel, TestIndirect(), TestInitSnowflake_InvalidWorkerID(), TestIsIntegerID(), TestRegisterSnowflakeHook_NilDB(), TestSnowflakeHook_NonIntegerPK(), TestSnowflakeHook_PointerSlice() (+14 more)

### Community 3 - "RunSeed"
Cohesion: 0.20
Nodes (24): basePermissions(), DB, RunSeed(), RunSeedWithCasbin(), SeedCasbinPolicies(), expectRunSeedQueries(), DB, T (+16 more)

### Community 4 - "Event"
Cohesion: 0.06
Nodes (31): Context, TestOutboxQueueWorkerEndToEnd(), Context, Queue, RawMessage, NewMQPublisher(), TestMQPublisher_Publish(), TestMQPublisher_PublishError() (+23 more)

### Community 5 - "upload_handler_test.go"
Cohesion: 0.09
Nodes (38): FileHeader, fakeStorage, UploadConfig, UploadHandler, UploadResponse, Context, HandlerFunc, Reader (+30 more)

### Community 6 - "RateLimiter"
Cohesion: 0.20
Nodes (16): GlobalRateLimit(), HandlerFunc, Limit, RWMutex, NewRateLimiter(), Engine, Request, ratelimitRequest() (+8 more)

### Community 7 - "SessionStore"
Cohesion: 0.22
Nodes (12): redisClientAdapter, RedisSessionStore, sessionPipeline, sessionRedis, SessionStore, Client, Context, Duration (+4 more)

### Community 8 - "Data Model: Jimu 框架持久化实体"
Cohesion: 0.06
Nodes (31): 1. User（用户）, 2. Role（角色）, 3. Permission（权限）, 4. APIKey（API 密钥）, 5. AuditLog（审计日志）, 6. OAuthBinding（第三方绑定）, 7. OutboxEvent（Outbox 事件）, 8. ScheduledJob（定时任务定义） (+23 more)

### Community 9 - "User"
Cohesion: 0.11
Nodes (15): fakeOutboxUserRepo, recordingOutboxStore, User, appCode(), createOutboxUserService(), fakeUserRepository, Context, TestCreateWritesOutbox() (+7 more)

### Community 10 - "Implementation Plan: Jimu 后端框架能力规格"
Cohesion: 0.07
Nodes (26): Content Quality, Feature Readiness, Notes, Requirement Completeness, Specification Quality Checklist: Jimu 后端框架能力规格, Complexity Tracking, Constitution Check, Documentation (this feature) (+18 more)

### Community 11 - "ws_integration_test.go"
Cohesion: 0.26
Nodes (27): NewMessage(), TestNewMessageMarshalError(), connID(), mustDecodeTitle(), newWSFixture(), TestClientChannels(), TestWSBroadcastAll(), TestWSChatBroadcast() (+19 more)

### Community 12 - "New"
Cohesion: 0.17
Nodes (13): OAuthConfig, OAuthProviderConfig, buildProviders(), Client, DB, EventBus, New(), newTestModule() (+5 more)

### Community 13 - "config/config.go"
Cohesion: 0.11
Nodes (33): main(), run(), AuditConfig, CacheConfig, CaptchaConfig, CaptchaResult, EmailConfig, GRPCConfig (+25 more)

### Community 14 - "Tasks: Jimu 后端框架能力规格"
Cohesion: 0.08
Nodes (24): Dependencies & Execution Order, Format: `[ID] [P?] [Story] Description`, Implementation for User Story 1, Implementation for User Story 2, Implementation for User Story 3, Implementation for User Story 4, Implementation Strategy, Incremental Delivery (+16 more)

### Community 15 - "oauth/application/service_test.go"
Cohesion: 0.23
Nodes (33): oauthStateKey(), dupSubjectProviders(), githubProviders(), Client, DB, Miniredis, newRedisClient(), newRepoResult() (+25 more)

### Community 16 - "gorm_logger_test.go"
Cohesion: 0.11
Nodes (27): gormLogger, Interface, Context, Duration, Time, isSensitiveField(), NewGormLogger(), sanitizeArgs() (+19 more)

### Community 17 - "Context"
Cohesion: 0.11
Nodes (9): fakeAPIKeyRepo, fakeDeadLetterRepo, fakeEventBus, testRole, testUserRole, APIKey, Context, fakeUserRepository (+1 more)

### Community 18 - "NewAdminConfigService"
Cohesion: 0.14
Nodes (20): AdminConfigService, Client, Context, EventBus, NewAdminConfigService(), Miniredis, newConfigTestService(), TestAdminConfigServiceConfigKey() (+12 more)

### Community 19 - "AuthHandler"
Cohesion: 0.33
Nodes (6): AuthHandler, authContext(), Context, Duration, normalizeUsername(), writeAuthRateLimitHeaders()

### Community 20 - "WorkerPool"
Cohesion: 0.15
Nodes (12): CancelFunc, TraceFromContext(), GetWorker(), Context, Duration, MySQLStore, Consumer, Queue (+4 more)

### Community 21 - "AdminMonitoringService"
Cohesion: 0.13
Nodes (17): AdminMonitoringService, HealthStatus, MemoryStats, SystemStatus, AdminMonitoringHandler, Client, Context, Time (+9 more)

### Community 22 - "OAuthService"
Cohesion: 0.21
Nodes (9): OAuthService, BindingRepository, Client, Context, DB, UserInfo, NewOAuthService(), Provider (+1 more)

### Community 23 - "Manager"
Cohesion: 0.14
Nodes (16): contextKey, Flag, Manager, AdminFeatureHandler, NewAdminFeatureHandler(), TestAdminFeatureHandlerList(), TestAdminFeatureHandlerUpdate(), FromContext() (+8 more)

### Community 24 - "DefaultCSRFConfig"
Cohesion: 0.18
Nodes (23): CSRF(), DefaultCSRFConfig(), generateToken(), Context, HandlerFunc, isSafeMethod(), setCSRFCookie(), csrfRouter() (+15 more)

### Community 25 - "middleware/middleware_test.go"
Cohesion: 0.15
Nodes (23): CORSMiddleware(), DefaultLogConfig(), HandlerFunc, isAllowedOrigin(), Logger(), Recovery(), RequestID(), shouldLogBody() (+15 more)

### Community 26 - "Context"
Cohesion: 0.20
Nodes (5): Context, Duration, fakeConsumer, fakeJobRepo, fakeProducer

### Community 27 - "migrate.go"
Cohesion: 0.16
Nodes (18): AutoMigrate(), findUp(), DB, isDir(), Migrate(), MigrateWithRetry(), MigrationDir(), PostgresMigrationDir() (+10 more)

### Community 28 - "NewCleanupService"
Cohesion: 0.14
Nodes (19): CleanupConfig, cleanupModel, CleanupResult, CleanupService, CleanupTable, noTableNameModel, DefaultCleanupConfig(), Context (+11 more)

### Community 29 - "JobData"
Cohesion: 0.17
Nodes (10): Context, Duration, Client, Context, Duration, NewRedisQueue(), errorQueue, fakeQueue (+2 more)

### Community 30 - "newImportHandler"
Cohesion: 0.28
Nodes (12): DB, newSqliteDB(), NewAdminImportHandler(), DB, Request, multipartRequest(), newImportHandler(), TestAdminImportHandlerGet() (+4 more)

### Community 31 - "NewUserRateLimiter"
Cohesion: 0.19
Nodes (19): defaultKeyFunc(), Client, Context, Duration, HandlerFunc, Time, NewUserRateLimiter(), Engine (+11 more)

### Community 32 - "Hub"
Cohesion: 0.17
Nodes (8): Channel, Context, RWMutex, Connection, Hub, Registration, WebSocket, WebSocketConfig

### Community 33 - "Wrap"
Cohesion: 0.20
Nodes (10): AuthService, AuthServiceInterface, TokenPair, ErrAccountLocked(), Context, Duration, invalidCredentials(), normalizeUsername() (+2 more)

### Community 34 - "JWT"
Cohesion: 0.16
Nodes (12): Claims, JWT, FuzzJWTParse(), F, Duration, New(), NewWithRotation(), TestJWTPopulatesTypedClaims() (+4 more)

### Community 35 - "generator/module.go"
Cohesion: 0.20
Nodes (20): targetFile, templateData, camel(), GenerateModule(), GenerateModuleAt(), lowerCamel(), mkdirAllTracked(), nextMigrationNumber() (+12 more)

### Community 36 - "circuit"
Cohesion: 0.16
Nodes (15): circuit, circuitState, Client, Config, hostRateLimiter, Context, Duration, Limit (+7 more)

### Community 37 - "signature_test.go"
Cohesion: 0.19
Nodes (20): buildSignString(), DefaultSignatureConfig(), Context, Duration, HandlerFunc, hmacSign(), Signature(), SignRequest() (+12 more)

### Community 38 - "NewEmail"
Cohesion: 0.16
Nodes (14): buildEmailHeaders(), Channel, Context, NewEmail(), Conn, Listener, newFakeSMTPServer(), TestBuildEmailHeaders() (+6 more)

### Community 39 - "OK"
Cohesion: 0.16
Nodes (8): AdminConfigHandler, AdminTaskHandler, Context, Context, Context, NewAdminTaskHandler(), Context, OK()

### Community 40 - "Message"
Cohesion: 0.19
Nodes (9): Dispatcher, Channel, Context, Context, Channel, Message, SMS, fakeKafkaReader (+1 more)

### Community 41 - "ImportJob"
Cohesion: 0.16
Nodes (8): fakeImportJobRepo, ImportJob, mysqlImportJobRepository, fakeImportJobRepo, Time, Context, DB, NewMysqlImportJobRepository()

### Community 42 - "auth/apikey.go"
Cohesion: 0.16
Nodes (14): APIKey, APIKeyContextKey, APIKeyStore, APIKeyVerifier, dbAPIKeyStore, APIKeyFromContext(), ContextWithAPIKey(), Context (+6 more)

### Community 43 - "RegisterAuthRoutes"
Cohesion: 0.29
Nodes (14): RouterGroup, Service, RegisterAuthRoutes(), RegisterCaptchaRoute(), Engine, newRouterLimiter(), testAuthConfig(), TestLoginRateLimitUsesIPScope() (+6 more)

### Community 44 - "New"
Cohesion: 0.18
Nodes (21): fakeUserRepository, NewAdminJobHandler(), TestAdminJobHandlerGet(), TestAdminJobHandlerList(), TestAdminJobHandlerListDeadLetters(), TestAdminJobHandlerResolveDeadLetter(), TestAdminJobHandlerRetry(), TestAdminJobHandlerSubmit() (+13 more)

### Community 45 - "UserService"
Cohesion: 0.22
Nodes (10): BatchDeleteRequest, BatchResult, CreateUserRequest, UpdateUserRequest, UserResponse, UserService, Time, ToUserResponse() (+2 more)

### Community 46 - "NewWebhook"
Cohesion: 0.16
Nodes (15): BenchmarkWebhookSend(), B, Channel, Client, Context, NewWebhook(), signPayload(), TestWebhookNoSignatureWhenNoSecret() (+7 more)

### Community 47 - "Module"
Cohesion: 0.11
Nodes (15): Module, ChannelManager, ClientHub, CronScheduler, EventBus, HandlerFunc, Client, DB (+7 more)

### Community 48 - "AuditLog"
Cohesion: 0.21
Nodes (8): fakeAuditRepository, fakeBatchRepository, AuditLog, fakeQueue, Context, Context, Mutex, Time

### Community 49 - "New"
Cohesion: 0.16
Nodes (22): errDeleteRepository, errFindRepository, NewUserService(), NewUserHandler(), Context, fakeUserRepository, TestUserBatchDeleteReturnsOK(), TestUserCreateReturnsCreatedDTO() (+14 more)

### Community 50 - "CronScheduler"
Cohesion: 0.17
Nodes (8): Cron, EntryID, Context, RWMutex, Time, CronScheduler, JobInfo, Store

### Community 51 - "Context"
Cohesion: 0.17
Nodes (5): handlerNotifier, handlerSessionStore, handlerUserRepo, Context, Duration

### Community 52 - "Page"
Cohesion: 0.20
Nodes (12): PermissionHandler, Context, Created(), FailWithDetails(), Context, localeFrom(), NoContent(), Page() (+4 more)

### Community 53 - "NewWithStore"
Cohesion: 0.23
Nodes (14): NewMemoryStore(), TestObserveCountsPanicAsFailure(), TestObserveEmitsSuccessMetrics(), TestSetEnabledNotFound(), TestSetEnabledToggle(), TestTriggerJobNotFound(), TestTriggerJobRunsCommand(), NewWithStore() (+6 more)

### Community 54 - "NewAdminAPIKeyService"
Cohesion: 0.10
Nodes (20): AdminAPIKeyService, CreateKeyInput, APIKey, APIKeyRepository, APIKey, Context, NewAdminAPIKeyService(), TestAdminAPIKeyServiceCreateKey() (+12 more)

### Community 55 - "Context"
Cohesion: 0.16
Nodes (11): fakeProvider, Duration, refreshTTL(), fakeSessionStore, sessionRecord, Context, Duration, UserInfo (+3 more)

### Community 56 - "Context"
Cohesion: 0.14
Nodes (6): fakeAPIKeyRepo, fakeEventBus, APIKey, fakeUserRepository, Context, Mutex

### Community 57 - "api_contract_test.go"
Cohesion: 0.30
Nodes (18): apiResp, testAppDB, doJSON(), DB, Engine, RawMessage, ResponseRecorder, login() (+10 more)

### Community 58 - "Job"
Cohesion: 0.18
Nodes (8): Job, JobRepository, mysqlJobRepository, fakeJobRepo, Context, DB, NewMysqlJobRepository(), Time

### Community 59 - "mockTokenServer"
Cohesion: 0.31
Nodes (12): Client, Server, mockClient(), mockTokenServer(), TestGitHubProviderExchange(), TestGoogleProviderExchange(), TestGoogleProviderExchangeBadUserInfo(), TestGoogleProviderExchangeTokenError() (+4 more)

### Community 60 - "health.go"
Cohesion: 0.19
Nodes (15): Client, Context, DB, Duration, ResponseWriter, NewReadiness(), NewRedisChecker(), NewSQLChecker() (+7 more)

### Community 61 - "Context"
Cohesion: 0.20
Nodes (5): fakeUserRepo, fakeSessionStore, sessionRecord, Context, Duration

### Community 62 - "responseBodyWriter"
Cohesion: 0.17
Nodes (10): Buffer, ResponseWriter, newResponseBodyWriter(), sanitizeBody(), sanitizeJSONField(), TestResponseBodyWriterCapturesAndTruncates(), TestResponseBodyWriterWriteHeaderPassthrough(), TestSanitizeBody() (+2 more)

### Community 63 - "New"
Cohesion: 0.20
Nodes (13): Cipher, New(), TestBlindIndexDeterministicAndDistinct(), TestBlindIndexEmpty(), TestBlindIndexWithoutKey(), TestDecryptTamperedFails(), TestDecryptWrongKeyFails(), TestEmptyValuePassthrough() (+5 more)

### Community 64 - "oauth/interfaces/handler_test.go"
Cohesion: 0.18
Nodes (19): fakeProvider, OAuthHandler, NewOAuthHandler(), assertCode(), doRequest(), githubProvider(), Engine, ResponseRecorder (+11 more)

### Community 65 - "NewAuthService"
Cohesion: 0.31
Nodes (16): BenchmarkLogin(), benchUser(), B, TestForgotPasswordNotConfigured(), NewAuthService(), appCode(), newFakeSessionStore(), newTestService() (+8 more)

### Community 66 - "rabbitmq_queue_test.go"
Cohesion: 0.20
Nodes (12): Context, Delivery, newTestRabbitQueue(), TestRabbitMQQueue_ConsumeTimeout(), TestRabbitMQQueue_ConsumeUnmarshalError(), TestRabbitMQQueue_NackRequeues(), TestRabbitMQQueue_SubmitConsumeAck(), TestRabbitMQQueueImplementsInterfaces() (+4 more)

### Community 67 - "PresenceManager"
Cohesion: 0.16
Nodes (4): RWMutex, Time, Presence, PresenceManager

### Community 68 - "NewPresenceManager"
Cohesion: 0.19
Nodes (14): NewPresenceManager(), TestPresenceIsOnline(), TestPresenceIsTyping(), TestPresenceManagerAllPresences(), TestPresenceManagerHeartbeat(), TestPresenceManagerHeartbeatRestoresOnlineStatus(), TestPresenceManagerOfflineMissing(), TestPresenceManagerOnlineCountFiltersOffline() (+6 more)

### Community 69 - "dispatcher"
Cohesion: 0.15
Nodes (10): Channel, Channel, Channel, Context, dispatcher, RWMutex, NewDispatcher(), TestDispatcherDispatch() (+2 more)

### Community 70 - "AuthorizationMiddleware"
Cohesion: 0.14
Nodes (17): AuthorizationStore, DBAuthorizationStore, Enforcer, HandlerFunc, ProtectedMiddleware(), HandlerFunc, AuthorizationMiddleware(), Context (+9 more)

### Community 71 - "New"
Cohesion: 0.21
Nodes (12): Module, AuthConfig, Service, NewAuthHandler(), newHandlerService(), TestForgotPasswordHandler(), TestResetPasswordHandlerInvalidCode(), Client (+4 more)

### Community 72 - "fakeRedis"
Cohesion: 0.12
Nodes (13): fakePipeline, fakeRedis, BoolCmd, Cmder, IntCmd, BoolSliceCmd, Cmd, Context (+5 more)

### Community 73 - "DBConfig"
Cohesion: 0.17
Nodes (20): DBConfig, mysqlDSN(), sqlDriverAndDSN(), configurePool(), ConnectWithRetry(), dsn(), Context, DB (+12 more)

### Community 74 - "NewCSVExporter"
Cohesion: 0.15
Nodes (12): CSVExporter, ExcelExporter, Context, Writer, NewCSVExporter(), Context, Writer, NewExcelExporter() (+4 more)

### Community 75 - "New"
Cohesion: 0.17
Nodes (10): Module, AuditHandler, NewAuditHandler(), TestAuditGetInvalidIDReturnsStableBadRequest(), TestAuditGetReturnsLogDTO(), TestAuditListInvalidQueryReturnsStableBadRequest(), DB, EventBus (+2 more)

### Community 76 - "SetupRouter"
Cohesion: 0.13
Nodes (20): HandlerFunc, Metrics(), ConfigureTrustedProxies(), formatAddr(), Engine, SetupRouter(), freeAddr(), TestConfigureTrustedProxies() (+12 more)

### Community 77 - "ChannelManager"
Cohesion: 0.11
Nodes (18): RWMutex, NewChannel(), NewChannelManager(), TestChannelManagerGetChannelMissing(), TestChannelManagerGetSubscribers(), TestChannelManagerPreCreatedBroadcast(), TestChannelManagerSubscribeCreatesWithType(), TestChannelManagerSubscribeExisting() (+10 more)

### Community 78 - "NewService"
Cohesion: 0.11
Nodes (15): fakeRouter, Service, Handler, Client, Time, NewService(), TestService(), Context (+7 more)

### Community 79 - "AdminUserService"
Cohesion: 0.21
Nodes (9): AdminCreateUserRequest, AdminUpdateUserRequest, AdminUser, AdminUserService, ListUserFilter, userRole, Context, DB (+1 more)

### Community 80 - "Worker"
Cohesion: 0.20
Nodes (10): Worker, Context, RWMutex, NewWorker(), Duration, testWorker(), TestWorkerFlushesConfiguredBatch(), TestWorkerRejectsWhenQueueFull() (+2 more)

### Community 81 - "Server"
Cohesion: 0.10
Nodes (21): Config, PingServer, pingService, Server, Context, Server, RegisterPingServer(), Context (+13 more)

### Community 82 - "RedisStore"
Cohesion: 0.22
Nodes (9): RedisStore, Service, Client, Context, Duration, NewRedisStore(), NewService(), TestGenerate() (+1 more)

### Community 83 - "Context"
Cohesion: 0.33
Nodes (5): Context, UserInfo, _UserInfoService_GetUser_Handler(), _UserInfoService_ListUsers_Handler(), UnaryServerInterceptor

### Community 84 - "DeadLetter"
Cohesion: 0.16
Nodes (9): DeadLetter, DeadLetterRepository, mysqlDeadLetterRepository, Context, DB, NewMysqlDeadLetterRepository(), Time, Mutex (+1 more)

### Community 85 - "ImportResult"
Cohesion: 0.19
Nodes (8): ExcelImporter, ImportError, ImportResult, Context, Reader, NewExcelImporter(), Time, NewImportResult()

### Community 86 - "newMockGormDB"
Cohesion: 0.24
Nodes (14): MySQLBindingRepository, DB, NewMySQLBindingRepository(), bindingRows(), DB, Sqlmock, newMockGormDB(), TestCreateError() (+6 more)

### Community 87 - "IdempotencyMiddleware"
Cohesion: 0.20
Nodes (15): Int32, Client, Duration, HandlerFunc, IdempotencyMiddleware(), Client, Engine, idempotencyRouter() (+7 more)

### Community 88 - "gzipRouter"
Cohesion: 0.17
Nodes (12): HandlerFunc, ResponseWriter, Writer, GzipCompression(), isAlreadyCompressed(), Engine, gzipRouter(), TestGzipCompressionEncodesBody() (+4 more)

### Community 89 - "NewSMS"
Cohesion: 0.43
Nodes (5): NewSMS(), TestSMSAliyunReject(), TestSMSAliyunSend(), TestSMSUnknownProvider(), SMSConfig

### Community 90 - "NewWorkerPool"
Cohesion: 0.32
Nodes (13): NewMySQLStore(), NewWorkerPool(), RegisterWorker(), fakeStore(), MySQLStore, newFakeJobRepo(), TestWorkerPoolConsumeJob(), TestWorkerPoolConsumeRestoresTrace() (+5 more)

### Community 91 - "Permission"
Cohesion: 0.09
Nodes (30): fakePermissionRepository, Permission, PermissionRepository, mysqlPermissionRepository, errDeletePermissionRepository, fakePermissionRepository, NewPermissionService(), Context (+22 more)

### Community 92 - "MySQLStore"
Cohesion: 0.17
Nodes (10): Context, DB, DeletedAt, Time, NewMySQLStore(), DB, testDB(), TestMySQLStore() (+2 more)

### Community 93 - "newResetService"
Cohesion: 0.18
Nodes (14): fakeDispatcher, fakeSessionStore, Client, Context, Miniredis, Mutex, newResetRedis(), newResetService() (+6 more)

### Community 94 - "New"
Cohesion: 0.19
Nodes (12): New(), TestAppError(), TestAppErrorWithCause(), TestFailDoesNotLeakInfrastructureDetails(), TestOKIncludesStableEnvelopeAndRequestID(), TestPageIncludesStableEnvelopeAndPagination(), TestCreatedUsesStandardEnvelope(), TestFailHidesInternalCauseAndIncludesRequestID() (+4 more)

### Community 95 - "TestDB"
Cohesion: 0.28
Nodes (11): dbReachable(), defaultDBPort(), envDBConfig(), DB, NewTestDB(), NewTestDBWithPool(), openByDriver(), SkipUnlessDB() (+3 more)

### Community 96 - "ListUsersRequest"
Cohesion: 0.14
Nodes (5): MessageState, SizeCache, UnknownFields, GetUserRequest, ListUsersRequest

### Community 97 - "Application"
Cohesion: 0.21
Nodes (9): Application, Component, forwardError(), Context, Duration, NewApplication(), TestApplicationReturnsComponentRuntimeError(), TestApplicationRollsBackStartedComponents() (+1 more)

### Community 98 - "fakeAuthzModule"
Cohesion: 0.16
Nodes (5): fakeAuthzModule, fakeBusinessModule, EventBus, HandlerFunc, TestBusinessRoutesRequireProtectedMiddleware()

### Community 99 - "JobHistory"
Cohesion: 0.19
Nodes (8): JobHistory, JobHistoryRepository, mysqlJobHistoryRepository, Context, DB, NewMysqlJobHistoryRepository(), Time, fakeHistoryRepo

### Community 100 - "OAuthBinding"
Cohesion: 0.15
Nodes (8): fakeBindingRepo, OAuthBinding, fakeBindingRepo, fakeSessionStore, Time, Context, Context, Duration

### Community 101 - "NewMysqlRepository"
Cohesion: 0.23
Nodes (15): TestMysqlRepositoryMySQLIntegration(), NewMysqlRepository(), DB, newTestUser(), newUserTestDB(), TestMysqlRepositoryCreateAndFindByID(), TestMysqlRepositoryFindByEmailHash(), TestMysqlRepositoryFindByPhoneHash() (+7 more)

### Community 102 - "NewDBAPIKeyStore"
Cohesion: 0.33
Nodes (12): createKey(), APIKey, DB, newTestDB(), TestDBAPIKeyStore_GetByKeyHash(), TestDBAPIKeyStore_GetByKeyHashNotFound(), TestDBAPIKeyStore_UpdateLastUsed(), TestVerifyWithDBStore() (+4 more)

### Community 103 - "ImportService"
Cohesion: 0.15
Nodes (19): ImportService, ImportJobRepository, UserRepository, DB, newSqliteDB(), Context, DB, Format (+11 more)

### Community 104 - "newMockGormDB"
Cohesion: 0.25
Nodes (13): DB, Sqlmock, newMockGormDB(), TestConfigurePool(), DB, TestTransaction_Commit(), TestTransaction_Rollback(), TestWithTx_BeginError() (+5 more)

### Community 105 - "InitTracing"
Cohesion: 0.25
Nodes (12): ContextWithTrace(), DefaultTracingConfig(), Context, TracerProvider, InitTracing(), ShutdownTracing(), TestContextWithTraceEmptyPassthrough(), TestDefaultTracingConfig() (+4 more)

### Community 106 - "RabbitMQQueue"
Cohesion: 0.26
Nodes (7): Context, Delivery, Duration, Mutex, NewRabbitMQQueue(), RabbitMQChannel, RabbitMQQueue

### Community 107 - "Lock"
Cohesion: 0.33
Nodes (8): generateToken(), Client, Context, Duration, Time, NewLock(), AcquireResult, Lock

### Community 108 - "RedisCache"
Cohesion: 0.23
Nodes (7): Cache, RedisCache, Client, Context, Duration, NewRedisCache(), randomToken()

### Community 109 - "Limiter"
Cohesion: 0.25
Nodes (10): Limiter, NewLimiter(), newTestLimiter(), TestLimiterAllowsOnlyConfiguredWindow(), TestLimiterDoesNotExposeUsernameInRedisKeys(), TestLimiterFailsClosedInProductionMode(), TestLimiterScopesAreIsolated(), newFakeRedis() (+2 more)

### Community 110 - "fakeStorage"
Cohesion: 0.22
Nodes (5): fakeStorage, Context, Duration, ReadCloser, Reader

### Community 111 - "Container"
Cohesion: 0.22
Nodes (10): Container, Client, Config, Context, DB, EventBus, Server, Service (+2 more)

### Community 112 - "AuditLogResponse"
Cohesion: 0.46
Nodes (5): AuditLogResponse, Time, ToAuditLogResponse(), ToAuditLogResponses(), Context

### Community 113 - "NewManagementServer"
Cohesion: 0.20
Nodes (8): Handler, passingChecker, Server, HealthRouter(), NewManagementServer(), Context, TestManagementRouterExposure(), TestNewManagementServer()

### Community 114 - "mysqlAPIKeyRepository"
Cohesion: 0.29
Nodes (5): mysqlAPIKeyRepository, APIKey, Context, DB, NewMysqlAPIKeyRepository()

### Community 115 - "mysqlRepository"
Cohesion: 0.27
Nodes (3): Context, DB, mysqlRepository

### Community 116 - "NewServer"
Cohesion: 0.38
Nodes (12): NewServer(), New(), TestCircuitHalfOpenOnlyAllowsProbe(), TestCircuitOpensAfterConsecutiveFailures(), TestCircuitRecoversAfterCooldown(), TestDoCancelsDuringRetryWait(), TestDoInjectsTraceParent(), TestDoNoRetryOn4xx() (+4 more)

### Community 117 - "New"
Cohesion: 0.19
Nodes (9): Client, Queue, New(), TestNew_InvalidType(), TestNew_Redis(), Config, KafkaConfig, RabbitMQConfig (+1 more)

### Community 118 - "KafkaQueue"
Cohesion: 0.29
Nodes (6): Context, Duration, NewKafkaQueue(), KafkaMessageReader, KafkaMessageWriter, KafkaQueue

### Community 119 - "AuditService"
Cohesion: 0.23
Nodes (8): AuditService, AuditRepository, Change, NewAuditService(), serializeChanges(), auditAppCode(), TestAuditServiceGetMapsNotFound(), TestAuditServiceListReturnsDTOAndPassesPagination()

### Community 120 - "Now"
Cohesion: 0.11
Nodes (17): Conn, Context, HandlerFunc, RWMutex, Time, mustEncode(), WSHandler(), BuildUserChannel() (+9 more)

### Community 121 - "New"
Cohesion: 0.07
Nodes (36): fakeRoleRepository, Role, RoleRepository, rolePermission, errDeleteRoleRepository, fakeRoleRepository, NewRoleService(), Context (+28 more)

### Community 122 - "newEncryptionTestDB"
Cohesion: 0.24
Nodes (9): contact, contactPtr, DB, newEncryptionTestDB(), TestEncryptionHookBatchCreate(), TestEncryptionHookEmptySourceStoresNullAndCoexists(), TestEncryptionHookEncryptsOnWriteDecryptsOnRead(), TestEncryptionHookNoKeyStoresPlaintextButHashes() (+1 more)

### Community 123 - "LoginFailureTracker"
Cohesion: 0.33
Nodes (8): LockoutConfig, LoginFailureTracker, DefaultLockoutConfig(), Client, Context, Duration, lockKey(), NewLoginFailureTracker()

### Community 124 - "NewAdminUserService"
Cohesion: 0.29
Nodes (9): testRole, NewAdminUserService(), TestAdminUserServiceAssignRoles(), TestAdminUserServiceAssignRolesRoleQueryError(), TestAdminUserServiceCreateUser(), TestAdminUserServiceDisableUser(), TestAdminUserServiceGetUser(), TestAdminUserServiceListUsers() (+1 more)

### Community 125 - "统一响应契约"
Cohesion: 0.13
Nodes (13): Admin 模块, Auth 模块, HTTP API 契约, 中间件, 健康与观测（非业务）, 完整接口文档, 分页, 国际化 (+5 more)

### Community 126 - "userinfo_grpc.pb.go"
Cohesion: 0.20
Nodes (9): userInfoService, DB, DB, NewUserInfoGRPCService(), RegisterUserInfoServiceServer(), ServiceRegistrar, UnimplementedUserInfoServiceServer, UnsafeUserInfoServiceServer (+1 more)

### Community 127 - "newWSHandler"
Cohesion: 0.73
Nodes (5): NewAdminWSHandler(), newWSHandler(), TestAdminWSHandlerOnlineUsers(), TestAdminWSHandlerPresence(), TestAdminWSHandlerPush()

### Community 128 - "routerLimiterRedis"
Cohesion: 0.29
Nodes (6): routerLimiterRedis, BoolSliceCmd, Cmd, Context, StringCmd, routerLimiterInt()

### Community 129 - "i18n.go"
Cohesion: 0.18
Nodes (7): HandlerFunc, Locale(), ParseAcceptLanguage(), TestParseAcceptLanguage(), TestTDefaultsToChinese(), TestTfFormatsArgs(), Tf()

### Community 130 - "ValidateJSON"
Cohesion: 0.20
Nodes (12): RouterGroup, RegisterUserRoutes(), Context, HandlerFunc, localeOf(), TestValidateJSONTranslatesFieldErrors(), translateValidationDetails(), translateValidationMessage() (+4 more)

### Community 131 - "newLock"
Cohesion: 0.18
Nodes (16): RedisConfig, Client, newLock(), TestLock_AcquireRelease(), TestLock_ConcurrentAcquire(), TestLock_Extend(), TestLock_ReleaseOnlyOwnToken(), TestLock_WithLock() (+8 more)

### Community 132 - "mysqlAuditRepository"
Cohesion: 0.36
Nodes (5): mysqlAuditRepository, deserializeChanges(), Context, DB, NewMysqlAuditRepository()

### Community 133 - "validator/validator.go"
Cohesion: 0.25
Nodes (8): FieldLevel, FuzzValidateRules(), F, Validate(), validateIDCard(), validateMobile(), validatePassword(), validateUsername()

### Community 134 - "NewLocalStorage"
Cohesion: 0.06
Nodes (36): New(), newOSSStorage(), TestLocalStorageDeleteMissingIsNoop(), TestLocalStorageDownloadNotFound(), TestLocalStoragePathTraversal(), TestLocalStoragePresignedUploadUnsupported(), TestLocalStoragePresignedURL(), TestLocalStorageSizeMismatch() (+28 more)

### Community 135 - "NewSnowflake"
Cohesion: 0.11
Nodes (18): Generator, snowflake, uuidGenerator, BenchmarkSnowflakeNextID(), BenchmarkUUIDNextID(), B, FuzzSnowflakeWorkerID(), F (+10 more)

### Community 136 - "Logger"
Cohesion: 0.17
Nodes (11): AtomicLevel, Context, LogConfig, New(), TestFileOutput(), TestNewConsoleStdout(), TestNewJSONLevels(), TestSetLevel() (+3 more)

### Community 137 - "AGENTS.md"
Cohesion: 0.14
Nodes (12): Commit Message 规范, graphify, Release Note 规范, 保护工作区, 分支策略, 开发前必读, 文档维护, 架构约束 (+4 more)

### Community 138 - "AdminUserHandler"
Cohesion: 0.29
Nodes (5): AdminUserHandler, Context, NewAdminUserHandler(), paginationFromQuery(), TestPaginationFromQuery()

### Community 139 - "adminAuthRouter"
Cohesion: 0.27
Nodes (9): AdminAuth(), HandlerFunc, adminAuthRouter(), Engine, TestAdminAuthAllowsAdminRole(), TestAdminAuthAllowsEnglishAdminRole(), TestAdminAuthRejectsNonAdminRole(), TestAdminAuthRequiresAuthentication() (+1 more)

### Community 140 - "bindImportFile"
Cohesion: 0.38
Nodes (5): AdminImportHandler, bindImportFile(), Buffer, Context, Format

### Community 141 - "GitHubProvider"
Cohesion: 0.24
Nodes (7): Client, Config, Context, UserInfo, NewGitHubProvider(), GitHubConfig, GitHubProvider

### Community 142 - "NewWeChatProvider"
Cohesion: 0.24
Nodes (7): Client, Config, Context, UserInfo, NewWeChatProvider(), WeChatConfig, WeChatProvider

### Community 143 - "testConn"
Cohesion: 0.29
Nodes (6): Conn, Duration, Mutex, newTestConn(), PresencePayload, testConn

### Community 144 - "AdminTaskService"
Cohesion: 0.29
Nodes (5): AdminTaskService, TaskExecution, TaskInfo, Context, Time

### Community 145 - "SecurityHeadersFromConfig"
Cohesion: 0.27
Nodes (9): SecurityConfig, DefaultSecurityConfig(), TestSecurityHeadersMiddleware(), TestSecurityHeadersNilValuesSkipped(), TestSecurityHeadersRespectsCustomValues(), Context, HandlerFunc, SecurityHeadersFromConfig() (+1 more)

### Community 146 - "Router"
Cohesion: 0.20
Nodes (5): Router, RouterGroup, RegisterAuditRoutes(), RouterGroup, RegisterPermissionRoutes()

### Community 147 - "NewCSVImporter"
Cohesion: 0.21
Nodes (9): CSVImporter, Reader, NewCSVImporter(), FuzzCSVImporterParse(), F, csvReader(), Reader, TestCSVParseAndValidate() (+1 more)

### Community 148 - ".Validate"
Cohesion: 0.27
Nodes (9): FieldRule, FieldType, ValidationRules, Validator, Context, TestValidateUnique(), checkType(), Context (+1 more)

### Community 149 - "AuditMiddleware"
Cohesion: 0.31
Nodes (8): Queue, AuditMiddleware(), Context, HandlerFunc, isManagementPath(), optionalString(), optionalUint64(), TestAuditMiddlewareAllowsAnonymousRequest()

### Community 150 - "NewDBCollector"
Cohesion: 0.29
Nodes (7): CollectRuntime(), DB, NewDBCollector(), TestCollectRuntime(), TestDBCollectorCollect(), TestDBCollectorCollectNilDB(), DBCollector

### Community 151 - "Policy"
Cohesion: 0.31
Nodes (5): fakeAuthorizationStore, Policy, fakeAuthzStore, Context, Context

### Community 152 - "TestWebSocketNotification"
Cohesion: 0.29
Nodes (6): NewHub(), NewWebSocket(), TestHubLifecycle(), TestWebSocketNotification(), waitHub(), mockConn

### Community 153 - "fakeContainer"
Cohesion: 0.44
Nodes (8): bridgeFn(), fakeContainer(), newTestLogger(), TestBridgeWorkerConversionFailureErrors(), TestBridgeWorkerPublishesStrongTypeToBareTopic(), TestBridgeWorkerUnknownTypeErrors(), TestEventBusBridgePublishesToBareTopic(), TestRegisterOutboxWorkersRegistersAll()

### Community 154 - "CLAUDE.md"
Cohesion: 0.18
Nodes (8): Commit Message 规范, graphify, Release Note 规范, 保护工作区, 回复格式, 开发前必读, 文档维护, 简单优先

### Community 155 - "setupMetricsEngine"
Cohesion: 0.60
Nodes (5): gatherHTTPMetric(), Engine, setupMetricsEngine(), TestMetricsLabelsUseRouteTemplate(), TestMetricsRecordsUnmatchedPath()

### Community 156 - "Timeout"
Cohesion: 0.22
Nodes (6): Duration, HandlerFunc, TestTimeoutOnlyPropagatesContextDeadline(), Timeout(), RouterGroup, RegisterSwagger()

### Community 157 - "NewLogChannel"
Cohesion: 0.29
Nodes (6): Channel, Context, NewLogChannel(), TestLogChannelSendBatchNoError(), TestLogChannelSendNoError(), LogChannel

### Community 160 - "ResetStore"
Cohesion: 0.39
Nodes (5): ResetStore, Client, Context, Duration, NewResetStore()

### Community 161 - "贡献指南"
Cohesion: 0.18
Nodes (10): Commit 规范, Pull Request, 代码规范, 分支策略, 开发环境, 报告问题, 模块开发, 测试 (+2 more)

### Community 162 - "EventBus"
Cohesion: 0.19
Nodes (9): EventBus, Handler, RWMutex, New(), TestEventBus_Clear(), TestEventBus_HandlerPanicRecovered(), TestEventBus_MultipleHandlers(), TestEventBus_PublishAsync() (+1 more)

### Community 163 - ".List"
Cohesion: 0.29
Nodes (5): AdminAuditHandler, Context, DB, NewAdminAuditHandler(), TestAdminAuditHandlerList()

### Community 164 - "Registry"
Cohesion: 0.54
Nodes (5): Format, Importer, Registry, NewRegistry(), TestRegistryGetUnsupported()

### Community 165 - "API 示例"
Cohesion: 0.17
Nodes (12): API 示例, Metrics, OAuth 登录, 健康检查, 创建用户, 刷新 Token, 忘记密码（发送验证码）, 查看认证限流状态 (+4 more)

### Community 167 - "userinfo.pb.go"
Cohesion: 0.29
Nodes (3): file_proto_jimu_v1_userinfo_proto_init(), file_proto_jimu_v1_userinfo_proto_rawDescGZIP(), init()

### Community 168 - "NewEventBusPublisher"
Cohesion: 0.32
Nodes (6): Context, EventBus, RawMessage, NewEventBusPublisher(), EventBusPublisher, EventPayload

### Community 170 - "newRedisTestQueue"
Cohesion: 0.39
Nodes (7): Client, newRedisTestQueue(), TestQueueContract_SubmitConsume(), TestRedisQueueAckRemovesFromProcessing(), TestRedisQueueImplementsInterfaces(), TestRedisQueueNackRequeues(), TestRedisQueueRequeueExpired()

### Community 171 - "mysql/003_extensions.sql"
Cohesion: 0.25
Nodes (7): api_keys, dead_letters, import_jobs, job_history, jobs, scheduled_jobs, user_oauth_bindings

### Community 172 - "postgres/003_extensions.sql"
Cohesion: 0.25
Nodes (7): api_keys, dead_letters, import_jobs, job_history, jobs, scheduled_jobs, user_oauth_bindings

### Community 173 - "Bootstrap"
Cohesion: 0.29
Nodes (8): moduleLogger, registerRouter, workerPoolComponent, Bootstrap(), Context, registerEventBusBridge(), registerHTTP(), registerOutboxWorkers()

### Community 174 - "contract/module.go"
Cohesion: 0.29
Nodes (6): ComponentProvider, ErrorSource, EventBus, HTTPMiddlewareProvider, Module, ProtectedHTTPMiddlewareProvider

### Community 175 - "Registry"
Cohesion: 0.62
Nodes (4): Exporter, Format, Registry, NewRegistry()

### Community 176 - "startTestGRPCServer"
Cohesion: 0.43
Nodes (6): ClientConn, Server, startTestGRPCServer(), TestGRPCHealthCheck(), TestGRPCPingEcho(), TestGRPCReflection()

### Community 177 - "newTestScheduler"
Cohesion: 0.62
Nodes (6): NewAdminTaskService(), newTestScheduler(), TestAdminTaskServiceGetHistory(), TestAdminTaskServiceListTasks(), TestAdminTaskServiceToggleTask(), TestAdminTaskServiceTriggerTask()

### Community 178 - "RegisterEncryptionHooks"
Cohesion: 0.57
Nodes (7): applyBlindIndexFields(), applyEncryptedFields(), DB, Field, Value, RegisterEncryptionHooks(), walkElements()

### Community 179 - "Jimu Backend Framework"
Cohesion: 0.20
Nodes (10): CLI 工具, Docker 部署, Jimu Backend Framework, K8s 部署, License, Makefile 命令, 技术栈, 特性 (+2 more)

### Community 180 - "setupTestCache"
Cohesion: 0.52
Nodes (6): setupTestCache(), TestRedisCache_Delete(), TestRedisCache_DeletePattern(), TestRedisCache_GetOrSet(), TestRedisCache_GetOrSetStampede(), TestRedisCache_SetAndGet()

### Community 181 - "GoogleProvider"
Cohesion: 0.25
Nodes (5): Client, Config, Context, UserInfo, GoogleProvider

### Community 182 - "Duration"
Cohesion: 0.43
Nodes (4): Duration, ExponentialBackoff, FixedRetry, RetryStrategy

### Community 183 - "安全政策"
Cohesion: 0.22
Nodes (8): 响应时间, 如何报告, 安全政策, 安全最佳实践, 已知限制, 报告安全问题, 披露政策, 支持的版本

### Community 186 - "message.go"
Cohesion: 0.25
Nodes (7): BuildRoomChannel(), TestBuildRoomChannel(), ChatPayload, NotificationPayload, PingPayload, PongPayload, SubscribePayload

### Community 187 - "[Unreleased]"
Cohesion: 0.29
Nodes (6): Changelog, [Unreleased], 修复, 变更, 提交历史（pre-tag）, 新增

### Community 189 - "newHistoryTestDB"
Cohesion: 0.40
Nodes (4): TestMysqlDeadLetterRepositoryCRUD(), DB, newHistoryTestDB(), TestMysqlJobHistoryRepositoryCreateAndList()

### Community 190 - "Security"
Cohesion: 0.40
Nodes (5): TestSecurityAddVary(), addVary(), Context, HandlerFunc, Security()

### Community 191 - "newRepoTestDB"
Cohesion: 0.53
Nodes (5): DB, newRepoTestDB(), TestMysqlAPIKeyRepository(), TestMysqlImportJobRepository(), TestMysqlJobRepository()

### Community 192 - "mockNotification"
Cohesion: 0.47
Nodes (3): Channel, Context, mockNotification

### Community 193 - "TestReadinessBoundsCheckerDuration"
Cohesion: 0.47
Nodes (4): Context, TestReadinessBoundsCheckerDuration(), TestReadinessStatus(), checkerFunc

### Community 194 - "PermissionService"
Cohesion: 0.20
Nodes (10): CreatePermissionRequest, PermissionService, UpdatePermissionRequest, PermissionResponse, Time, ToPermissionResponse(), ToPermissionResponses(), isDuplicateKey() (+2 more)

### Community 195 - "mysql/001_core.sql"
Cohesion: 0.33
Nodes (5): permissions, role_permissions, roles, user_roles, users

### Community 196 - "postgres/001_core.sql"
Cohesion: 0.33
Nodes (5): permissions, role_permissions, roles, user_roles, users

### Community 197 - "bug_report.md"
Cohesion: 0.29
Nodes (6): 复现步骤, 实际行为, 描述, 期望行为, 环境, 附加信息

### Community 199 - "events.go"
Cohesion: 0.40
Nodes (4): UserCreatedEvent, UserDeletedEvent, UserLoggedInEvent, UserUpdatedEvent

### Community 200 - "快速开始"
Cohesion: 0.29
Nodes (7): 前置条件, 可观测性（可选）, 安装, 快速开始, 方式一：本地运行, 方式二：Docker Compose 一键启动, 配置

### Community 201 - "分支策略"
Cohesion: 0.33
Nodes (6): Tag 与发布, 分支模型, 分支策略, 合并与 PR, 命名约定, 回滚

### Community 202 - "newMockS3Storage"
Cohesion: 0.70
Nodes (4): newMockS3Storage(), TestS3DeleteMissingIsNoop(), TestS3ExistsAndSizeMissing(), TestS3UploadDownloadRoundTrip()

### Community 203 - "newRateLimitHandler"
Cohesion: 0.19
Nodes (13): Duration, AdminRateLimitHandler, Client, Context, NewAdminRateLimitHandler(), Client, T, newRateLimitHandler() (+5 more)

### Community 204 - "CaptchaHandler"
Cohesion: 0.83
Nodes (3): CaptchaHandler, Service, NewCaptchaHandler()

### Community 206 - "PermissionMiddleware"
Cohesion: 0.50
Nodes (3): Enforcer, HandlerFunc, PermissionMiddleware()

### Community 207 - "PULL_REQUEST_TEMPLATE.md"
Cohesion: 0.33
Nodes (5): 变更类型, 变更说明, 检查清单, 相关 Issue, 风险与注意事项

### Community 226 - "As"
Cohesion: 0.36
Nodes (5): AppError, ErrorInfo, AllErrorCodes(), As(), HTTPStatus()

### Community 227 - "配置契约"
Cohesion: 0.33
Nodes (5): 关键配置组, 加载优先级, 敏感值注入（Docker Secrets）, 枚举约束（非法值启动报错）, 配置契约

### Community 228 - "Module 契约"
Cohesion: 0.33
Nodes (5): Module 契约, 中间件挂载, 分层结构, 模块接口, 组件生命周期

### Community 229 - "编码约束"
Cohesion: 0.40
Nodes (5): CLI, 数据库, 日志, 统一响应, 编码约束

### Community 230 - "配置约束"
Cohesion: 0.40
Nodes (5): 多环境配置, 本地数据库集成测试, 枚举值, 环境变量, 配置约束

### Community 231 - "feature_request.md"
Cohesion: 0.40
Nodes (4): 备选方案, 方案, 背景, 附加信息

### Community 232 - "CLI 契约"
Cohesion: 0.40
Nodes (4): CLI 契约, 命令表, 种子数据, 迁移命名

### Community 233 - "架构约束"
Cohesion: 0.50
Nodes (4): 架构约束, 模块注册, 模块结构, 设计边界（非目标）

### Community 234 - "securityRouter"
Cohesion: 0.53
Nodes (5): Engine, securityRouter(), TestSecurityHandlesAllowedPreflight(), TestSecurityRejectsOversizedBody(), TestSecurityUsesOriginAllowList()

### Community 235 - "配置说明"
Cohesion: 0.40
Nodes (5): 多环境配置, 环境变量, 配置说明, 配置项, 静态加密（Data at Rest）

### Community 236 - "TestGeneratedModuleCompiles"
Cohesion: 0.73
Nodes (5): copyGoSum(), copyRootFile(), TestGeneratedModuleCompiles(), writeFileForTest(), writeStubPackages()

### Community 238 - "AdminAuthMiddleware"
Cohesion: 0.40
Nodes (3): AdminAuthMiddleware(), HandlerFunc, TestAdminAuthMiddleware()

### Community 241 - "newTestGRPCService"
Cohesion: 0.48
Nodes (6): ClientConnInterface, newTestGRPCService(), TestUserInfoService_GetUser(), TestUserInfoService_ListUsers(), NewUserInfoServiceClient(), UserInfoServiceClient

### Community 242 - "wsFixture"
Cohesion: 0.50
Nodes (3): Server, Time, wsFixture

### Community 243 - "NewPathEnforcer"
Cohesion: 0.38
Nodes (5): TestProtectedMiddlewareRequiresAccessToken(), DB, Enforcer, NewEnforcer(), NewPathEnforcer()

### Community 244 - "Module"
Cohesion: 0.33
Nodes (3): Client, EventBus, Module

### Community 247 - "JobRegistry"
Cohesion: 0.18
Nodes (3): JobRegistry, EventBus, Module

### Community 250 - "Fail"
Cohesion: 0.24
Nodes (5): UserHandler, Context, Context, Context, Fail()

### Community 251 - "request.go"
Cohesion: 0.40
Nodes (4): forgotPasswordRequest, loginRequest, refreshRequest, resetPasswordRequest

### Community 252 - "JobDef"
Cohesion: 0.18
Nodes (8): Context, RWMutex, Context, Time, failStore, JobDef, MemoryStore, Store

### Community 254 - "TestOpenAPIIncludesCRUDContract"
Cohesion: 0.83
Nodes (3): assertQueryParams(), readOpenAPI(), TestOpenAPIIncludesCRUDContract()

## Knowledge Gaps
- **262 isolated node(s):** `新增`, `变更`, `修复`, `提交历史（pre-tag）`, `特性` (+257 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **33 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `T()` connect `T` to `RoleService`, `InitSnowflake`, `Event`, `upload_handler_test.go`, `RateLimiter`, `User`, `ws_integration_test.go`, `New`, `oauth/application/service_test.go`, `gorm_logger_test.go`, `NewAdminConfigService`, `AdminMonitoringService`, `Manager`, `DefaultCSRFConfig`, `middleware/middleware_test.go`, `migrate.go`, `NewCleanupService`, `newImportHandler`, `NewUserRateLimiter`, `JWT`, `generator/module.go`, `signature_test.go`, `NewEmail`, `RegisterAuthRoutes`, `New`, `NewWebhook`, `New`, `Page`, `NewWithStore`, `NewAdminAPIKeyService`, `Context`, `api_contract_test.go`, `mockTokenServer`, `responseBodyWriter`, `New`, `oauth/interfaces/handler_test.go`, `NewAuthService`, `rabbitmq_queue_test.go`, `NewPresenceManager`, `dispatcher`, `AuthorizationMiddleware`, `New`, `DBConfig`, `NewCSVExporter`, `New`, `SetupRouter`, `ChannelManager`, `NewService`, `Worker`, `Server`, `RedisStore`, `newMockGormDB`, `IdempotencyMiddleware`, `gzipRouter`, `NewSMS`, `NewWorkerPool`, `Permission`, `MySQLStore`, `newResetService`, `New`, `TestDB`, `Application`, `fakeAuthzModule`, `NewMysqlRepository`, `NewDBAPIKeyStore`, `ImportService`, `newMockGormDB`, `InitTracing`, `Limiter`, `NewManagementServer`, `NewServer`, `New`, `AuditService`, `Now`, `New`, `newEncryptionTestDB`, `NewAdminUserService`, `newWSHandler`, `i18n.go`, `ValidateJSON`, `newLock`, `NewLocalStorage`, `NewSnowflake`, `Logger`, `AdminUserHandler`, `adminAuthRouter`, `testConn`, `SecurityHeadersFromConfig`, `NewCSVImporter`, `.Validate`, `AuditMiddleware`, `NewDBCollector`, `TestWebSocketNotification`, `fakeContainer`, `setupMetricsEngine`, `Timeout`, `NewLogChannel`, `EventBus`, `.List`, `Registry`, `newRedisTestQueue`, `startTestGRPCServer`, `newTestScheduler`, `setupTestCache`, `message.go`, `newHistoryTestDB`, `Security`, `newRepoTestDB`, `TestReadinessBoundsCheckerDuration`, `newMockS3Storage`, `securityRouter`, `TestGeneratedModuleCompiles`, `AdminAuthMiddleware`, `newTestGRPCService`, `NewPathEnforcer`, `TestOpenAPIIncludesCRUDContract`?**
  _High betweenness centrality (0.662) - this node is a cross-community bridge._
- **Why does `Now()` connect `Now` to `T`, `Event`, `upload_handler_test.go`, `NewSnowflake`, `ws_integration_test.go`, `testConn`, `gorm_logger_test.go`, `.Validate`, `AdminMonitoringService`, `AuditMiddleware`, `WorkerPool`, `DefaultCSRFConfig`, `middleware/middleware_test.go`, `TestWebSocketNotification`, `NewCleanupService`, `JobData`, `NewUserRateLimiter`, `Wrap`, `JWT`, `circuit`, `signature_test.go`, `MySQLStore`, `auth/apikey.go`, `newRedisTestQueue`, `NewWebhook`, `CronScheduler`, `NewWithStore`, `NewAdminAPIKeyService`, `Context`, `TestReadinessBoundsCheckerDuration`, `PresenceManager`, `NewPresenceManager`, `AuthorizationMiddleware`, `SetupRouter`, `NewService`, `DeadLetter`, `ImportResult`, `newMockGormDB`, `NewMysqlRepository`, `NewDBAPIKeyStore`, `ImportService`, `Lock`, `RedisCache`, `mysqlAPIKeyRepository`, `JobDef`, `Fail`, `NewAdminUserService`?**
  _High betweenness centrality (0.068) - this node is a cross-community bridge._
- **Why does `User` connect `User` to `Wrap`, `NewAuthService`, `NewMysqlRepository`, `UserService`, `New`, `Context`, `mysqlRepository`, `Context`, `Context`, `Context`?**
  _High betweenness centrality (0.040) - this node is a cross-community bridge._
- **Are the 4 inferred relationships involving `T()` (e.g. with `translateValidationDetails()` and `ValidateJSON()`) actually correct?**
  _`T()` has 4 INFERRED edges - model-reasoned connections that need verification._
- **Are the 91 inferred relationships involving `New()` (e.g. with `.CreateKey()` and `.ToggleTask()`) actually correct?**
  _`New()` has 91 INFERRED edges - model-reasoned connections that need verification._
- **Are the 80 inferred relationships involving `Fail()` (e.g. with `.Generate()` and `.HandleDelete()`) actually correct?**
  _`Fail()` has 80 INFERRED edges - model-reasoned connections that need verification._
- **Are the 80 inferred relationships involving `Now()` (e.g. with `.CreateKey()` and `.generateResetCode()`) actually correct?**
  _`Now()` has 80 INFERRED edges - model-reasoned connections that need verification._