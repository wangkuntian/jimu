# Graph Report - jimu  (2026-08-21)

## Corpus Check
- 407 files · ~134,856 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3839 nodes · 8367 edges · 273 communities (225 shown, 48 thin omitted)
- Extraction: 82% EXTRACTED · 18% INFERRED · 0% AMBIGUOUS · INFERRED: 1485 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `0fde0fcf`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- RoleService
- T
- api_contract_test.go
- New
- NewSnowflake
- upload_handler_test.go
- ReadCloser
- New
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
- T
- AdminMonitoringService
- OAuthService
- Manager
- DefaultCSRFConfig
- middleware/middleware_test.go
- ChannelManager
- migrate.go
- NewCleanupService
- Context
- bindImportFile
- NewUserRateLimiter
- Hub
- Wrap
- JWT
- generator/module.go
- circuit
- signature_test.go
- Email
- OK
- SMS
- ImportJob
- auth/apikey.go
- New
- UserHandler
- New
- NewWebhook
- Module
- AuditLog
- RedisCache
- CronScheduler
- Context
- Page
- NewWithStore
- NewAdminAPIKeyService
- UserService
- Context
- Role
- Job
- mockTokenServer
- health.go
- Context
- responseBodyWriter
- New
- oauth/interfaces/handler_test.go
- NewAuthService
- rabbitmq_queue_test.go
- newMockGormDB
- Now
- dispatcher
- AuthorizationMiddleware
- RegisterAuthRoutes
- Limiter
- DBConfig
- NewCSVExporter
- New
- SetupRouter
- NewChannelManager
- NewService
- Pagination
- Worker
- AdminUserService
- RedisStore
- Context
- DeadLetter
- ImportResult
- newMockGormDB
- message.go
- gzipRouter
- Event
- fakeUserRepository
- newMonitoringHandler
- MySQLStore
- Message
- New
- TestDB
- ListUsersRequest
- Application
- fakeAuthzModule
- JobHistory
- OAuthBinding
- NewMysqlRepository
- NewDBAPIKeyStore
- IdempotencyMiddleware
- RabbitMQQueue
- New
- Client
- Lock
- fakeRoleRepository
- Message
- New
- Container
- RedisQueue
- NewManagementServer
- HandlerFunc
- mysqlRepository
- NewServer
- PresenceManager
- KafkaQueue
- Permission
- ClientHub
- mysqlRepository
- newEncryptionTestDB
- LoginFailureTracker
- New
- 统一响应契约
- userinfo_grpc.pb.go
- AdminWSHandler
- routerLimiterRedis
- i18n.go
- ValidateJSON
- newLock
- mysqlAuditRepository
- validator/validator.go
- fakeRedis
- ImportService
- Logger
- AGENTS.md
- New
- adminAuthRouter
- interfaces/fakes_test.go
- GitHubProvider
- NewWeChatProvider
- newResetService
- AdminTaskService
- SecurityHeadersFromConfig
- Module
- NewCSVImporter
- .Validate
- AuditMiddleware
- NewDBCollector
- Policy
- TestWebSocketNotification
- fakeContainer
- CLAUDE.md
- Metrics
- Timeout
- NewPermissionService
- ListUsersResponse
- UserInfo
- ResetStore
- 贡献指南
- EventBus
- NewMQPublisher
- Registry
- API 示例
- MySQLStore
- userinfo.pb.go
- NewEventBusPublisher
- AuditRepository
- JobData
- mysql/003_extensions.sql
- postgres/003_extensions.sql
- Bootstrap
- JobRegistry
- Registry
- startTestGRPCServer
- newTestScheduler
- Module
- Jimu Backend Framework
- setupTestCache
- Context
- Duration
- 安全政策
- .GetUser
- Server
- mysqlAPIKeyRepository
- [v0.1.0] - 2026-08-21
- Fail
- NewAdminUserService
- Security
- newRepoTestDB
- mockNotification
- TestReadinessBoundsCheckerDuration
- request.go
- mysql/001_core.sql
- postgres/001_core.sql
- bug_report.md
- ClamAVScanner
- events.go
- 快速开始
- 分支策略
- NewLocalStorage
- newRateLimitHandler
- CaptchaHandler
- .validateCommon
- PermissionMiddleware
- PULL_REQUEST_TEMPLATE.md
- openapi.go
- AdminAPIKeyHandler
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
- As
- AdminJobHandler
- 配置契约
- Module 契约
- 编码约束
- 配置约束
- feature_request.md
- CLI 契约
- 架构约束
- .Exchange
- 配置说明
- Router
- workerPoolComponent
- mysqlPermissionRepository
- RoleHandler
- newTestGRPCService
- newWSHandler
- NewPathEnforcer
- newHistoryTestDB
- securityRouter
- Client
- CronScheduler
- DB
- Config
- Server
- JobDef
- TracerProvider
- LogConfig
- Engine
- Request
- BenchmarkExecuteJobDedup
- ClamAVConfig
- EventBus
- JobRegistry
- Manager
- Service
- Reader
- T
- fakePermissionRepository
- AuditService
- run
- v0.1.0
- queue.go
- Mutex
- T
- newRedisTestQueue
- newMockS3Storage

## God Nodes (most connected - your core abstractions)
1. `T()` - 746 edges
2. `New()` - 97 edges
3. `Now()` - 82 edges
4. `Fail()` - 80 edges
5. `User` - 72 edges
6. `OK()` - 53 edges
7. `New()` - 52 edges
8. `Wrap()` - 48 edges
9. `NewServer()` - 37 edges
10. `Message` - 33 edges

## Surprising Connections (you probably didn't know these)
- `run()` --calls--> `NewContainer()`  [INFERRED]
  cmd/server/main.go → internal/app/container.go
- `run()` --calls--> `Load()`  [INFERRED]
  cmd/server/main.go → internal/config/config.go
- `run()` --calls--> `Watch()`  [INFERRED]
  cmd/server/main.go → internal/config/config.go
- `run()` --calls--> `New()`  [INFERRED]
  cmd/server/main.go → internal/modules/admin/module.go
- `assertNoGeneratedFiles()` --references--> `T()`  [EXTRACTED]
  tools/generator/module_test.go → internal/shared/i18n/i18n.go

## Import Cycles
- None detected.

## Communities (273 total, 48 thin omitted)

### Community 0 - "RoleService"
Cohesion: 0.14
Nodes (15): AssignPermissionsRequest, CreateRoleRequest, RoleResponse, RoleService, UpdateRoleRequest, PermissionResponse, Time, TestRoleResponseUsesDTO() (+7 more)

### Community 1 - "T"
Cohesion: 0.05
Nodes (55): Config, TestDBPasswordOverride(), TestJWTSecretOverride(), TestLoad(), TestStorageConfigFieldMapping(), TestValidateHTTPMode(), TestValidateLogFormat(), TestValidateLogLevel() (+47 more)

### Community 2 - "api_contract_test.go"
Cohesion: 0.09
Nodes (47): snowflakeModel, stringKeyModel, apiResp, testAppDB, doJSON(), DB, Engine, RawMessage (+39 more)

### Community 3 - "New"
Cohesion: 0.19
Nodes (23): New(), basePermissions(), DB, RunSeed(), RunSeedWithCasbin(), SeedCasbinPolicies(), expectRunSeedQueries(), DB (+15 more)

### Community 4 - "NewSnowflake"
Cohesion: 0.11
Nodes (18): Generator, snowflake, uuidGenerator, BenchmarkSnowflakeNextID(), BenchmarkUUIDNextID(), B, FuzzSnowflakeWorkerID(), F (+10 more)

### Community 5 - "upload_handler_test.go"
Cohesion: 0.07
Nodes (56): Context, Engine, FileHeader, HandlerFunc, fakeScanner, fakeStorage, noopScanner, UploadConfig (+48 more)

### Community 7 - "New"
Cohesion: 0.20
Nodes (21): NewUserService(), NewUserHandler(), TestUserBatchDeleteReturnsOK(), TestUserCreateReturnsCreatedDTO(), TestUserDeleteReturnsNoContent(), TestUserDeleteServiceError(), TestUserExportCSVRejectsInvalidSort(), TestUserExportCSVReturnsOK() (+13 more)

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
Cohesion: 0.17
Nodes (34): NewMessage(), connID(), Conn, Duration, Mutex, Server, Time, mustDecodeTitle() (+26 more)

### Community 12 - "New"
Cohesion: 0.14
Nodes (13): RouterGroup, RegisterOAuthRoutes(), buildProviders(), Client, DB, EventBus, New(), newTestModule() (+5 more)

### Community 13 - "config/config.go"
Cohesion: 0.11
Nodes (35): AuditConfig, CacheConfig, CaptchaResult, ClamAVConfig, Config, EmailConfig, GRPCConfig, HTTPClientConfig (+27 more)

### Community 14 - "Tasks: Jimu 后端框架能力规格"
Cohesion: 0.08
Nodes (24): Dependencies & Execution Order, Format: `[ID] [P?] [Story] Description`, Implementation for User Story 1, Implementation for User Story 2, Implementation for User Story 3, Implementation for User Story 4, Implementation Strategy, Incremental Delivery (+16 more)

### Community 15 - "oauth/application/service_test.go"
Cohesion: 0.23
Nodes (33): oauthStateKey(), dupSubjectProviders(), githubProviders(), Client, DB, Miniredis, newRedisClient(), newRepoResult() (+25 more)

### Community 16 - "gorm_logger_test.go"
Cohesion: 0.11
Nodes (26): gormLogger, Interface, Context, Duration, Time, isSensitiveField(), NewGormLogger(), sanitizeArgs() (+18 more)

### Community 17 - "Context"
Cohesion: 0.21
Nodes (4): fakeAPIKeyRepo, APIKey, Context, fakeUserRepository

### Community 18 - "NewAdminConfigService"
Cohesion: 0.14
Nodes (20): AdminConfigService, Client, Context, EventBus, NewAdminConfigService(), Miniredis, newConfigTestService(), TestAdminConfigServiceConfigKey() (+12 more)

### Community 19 - "AuthHandler"
Cohesion: 0.33
Nodes (6): AuthHandler, authContext(), Context, Duration, normalizeUsername(), writeAuthRateLimitHeaders()

### Community 20 - "T"
Cohesion: 0.05
Nodes (59): CancelFunc, Consumer, DeadLetter, assertNoErr(), clamavAddr(), skipUnlessClamAV(), TestClamAVScannerIntegrationClean(), TestClamAVScannerIntegrationEICAR() (+51 more)

### Community 21 - "AdminMonitoringService"
Cohesion: 0.15
Nodes (13): AdminMonitoringService, HealthStatus, MemoryStats, SystemStatus, AdminMonitoringHandler, Client, Context, Time (+5 more)

### Community 22 - "OAuthService"
Cohesion: 0.21
Nodes (9): OAuthService, BindingRepository, Client, Context, DB, UserInfo, NewOAuthService(), Provider (+1 more)

### Community 23 - "Manager"
Cohesion: 0.07
Nodes (25): fakeEventBus, fakeStorage, contextKey, Flag, Manager, AdminFeatureHandler, NewAdminFeatureHandler(), TestAdminFeatureHandlerList() (+17 more)

### Community 24 - "DefaultCSRFConfig"
Cohesion: 0.18
Nodes (23): CSRF(), DefaultCSRFConfig(), generateToken(), Context, HandlerFunc, isSafeMethod(), setCSRFCookie(), csrfRouter() (+15 more)

### Community 25 - "middleware/middleware_test.go"
Cohesion: 0.15
Nodes (23): CORSMiddleware(), DefaultLogConfig(), HandlerFunc, isAllowedOrigin(), Logger(), Recovery(), RequestID(), shouldLogBody() (+15 more)

### Community 26 - "ChannelManager"
Cohesion: 0.21
Nodes (3): RWMutex, Channel, ChannelManager

### Community 27 - "migrate.go"
Cohesion: 0.14
Nodes (21): AutoMigrate(), findUp(), DB, isDir(), Migrate(), MigrateWithRetry(), MigrationDir(), mysqlDSN() (+13 more)

### Community 28 - "NewCleanupService"
Cohesion: 0.14
Nodes (19): CleanupConfig, cleanupModel, CleanupResult, CleanupService, CleanupTable, noTableNameModel, DefaultCleanupConfig(), Context (+11 more)

### Community 29 - "Context"
Cohesion: 0.16
Nodes (11): fakeProvider, Duration, refreshTTL(), fakeSessionStore, sessionRecord, Context, Duration, UserInfo (+3 more)

### Community 30 - "bindImportFile"
Cohesion: 0.17
Nodes (17): AdminImportHandler, DB, newSqliteDB(), bindImportFile(), Buffer, Context, Format, NewAdminImportHandler() (+9 more)

### Community 31 - "NewUserRateLimiter"
Cohesion: 0.19
Nodes (19): defaultKeyFunc(), Client, Context, Duration, HandlerFunc, Time, NewUserRateLimiter(), Engine (+11 more)

### Community 32 - "Hub"
Cohesion: 0.17
Nodes (9): Channel, Context, RWMutex, NewWebSocket(), Connection, Hub, Registration, WebSocket (+1 more)

### Community 33 - "Wrap"
Cohesion: 0.18
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

### Community 38 - "Email"
Cohesion: 0.16
Nodes (14): buildEmailHeaders(), Channel, Context, NewEmail(), Conn, Listener, newFakeSMTPServer(), TestBuildEmailHeaders() (+6 more)

### Community 39 - "OK"
Cohesion: 0.14
Nodes (9): AdminConfigHandler, AdminTaskHandler, Context, Context, Context, Context, NewAdminTaskHandler(), Context (+1 more)

### Community 40 - "SMS"
Cohesion: 0.22
Nodes (8): Channel, Context, NewSMS(), TestSMSAliyunReject(), TestSMSAliyunSend(), TestSMSUnknownProvider(), SMS, SMSConfig

### Community 41 - "ImportJob"
Cohesion: 0.19
Nodes (7): ImportJob, mysqlImportJobRepository, fakeImportJobRepo, Time, Context, DB, NewMysqlImportJobRepository()

### Community 42 - "auth/apikey.go"
Cohesion: 0.16
Nodes (14): APIKey, APIKeyContextKey, APIKeyStore, APIKeyVerifier, dbAPIKeyStore, APIKeyFromContext(), ContextWithAPIKey(), Context (+6 more)

### Community 43 - "New"
Cohesion: 0.21
Nodes (13): Module, AuthConfig, CaptchaConfig, Service, NewAuthHandler(), newHandlerService(), TestForgotPasswordHandler(), TestResetPasswordHandlerInvalidCode() (+5 more)

### Community 44 - "UserHandler"
Cohesion: 0.27
Nodes (3): UserHandler, Context, NoContent()

### Community 45 - "New"
Cohesion: 0.38
Nodes (15): NewRoleService(), NewRoleHandler(), TestRoleAssignPermissionsInvalidID(), TestRoleAssignPermissionsReturnsOK(), TestRoleCreateReturnsCreated(), TestRoleDeleteReturnsNoContent(), TestRoleDeleteServiceError(), TestRoleGetInvalidID() (+7 more)

### Community 46 - "NewWebhook"
Cohesion: 0.16
Nodes (15): BenchmarkWebhookSend(), B, Channel, Client, Context, NewWebhook(), signPayload(), TestWebhookNoSignatureWhenNoSecret() (+7 more)

### Community 47 - "Module"
Cohesion: 0.11
Nodes (15): Module, ChannelManager, ClientHub, Client, CronScheduler, DB, EventBus, HandlerFunc (+7 more)

### Community 48 - "AuditLog"
Cohesion: 0.16
Nodes (10): fakeAuditRepository, fakeBatchRepository, AuditLog, fakeAuditRepository, fakeQueue, Context, Context, Mutex (+2 more)

### Community 49 - "RedisCache"
Cohesion: 0.23
Nodes (7): Cache, RedisCache, Client, Context, Duration, NewRedisCache(), randomToken()

### Community 50 - "CronScheduler"
Cohesion: 0.17
Nodes (8): Cron, EntryID, Context, RWMutex, Time, CronScheduler, JobInfo, Store

### Community 51 - "Context"
Cohesion: 0.17
Nodes (5): handlerNotifier, handlerSessionStore, handlerUserRepo, Context, Duration

### Community 52 - "Page"
Cohesion: 0.40
Nodes (9): Created(), FailWithDetails(), Context, localeFrom(), Page(), requestID(), StatusForCode(), Body (+1 more)

### Community 53 - "NewWithStore"
Cohesion: 0.23
Nodes (14): NewMemoryStore(), TestObserveCountsPanicAsFailure(), TestObserveEmitsSuccessMetrics(), TestSetEnabledNotFound(), TestSetEnabledToggle(), TestTriggerJobNotFound(), TestTriggerJobRunsCommand(), NewWithStore() (+6 more)

### Community 54 - "NewAdminAPIKeyService"
Cohesion: 0.11
Nodes (20): AdminAPIKeyService, CreateKeyInput, APIKey, APIKeyRepository, APIKey, Context, NewAdminAPIKeyService(), TestAdminAPIKeyServiceCreateKey() (+12 more)

### Community 55 - "UserService"
Cohesion: 0.22
Nodes (10): BatchDeleteRequest, BatchResult, CreateUserRequest, UpdateUserRequest, UserResponse, UserService, Time, ToUserResponse() (+2 more)

### Community 56 - "Context"
Cohesion: 0.13
Nodes (7): fakeAPIKeyRepo, fakeEventBus, fakeImportJobRepo, APIKey, fakeUserRepository, Context, Mutex

### Community 57 - "Role"
Cohesion: 0.20
Nodes (10): fakeRoleRepository, Role, Context, roleAppCode(), TestRoleServiceCreateMapsDuplicateNameToConflict(), TestRoleServiceDeleteWrapsRepositoryError(), TestRoleServiceListPassesPagination(), TestRoleServiceUpdateMapsNotFound() (+2 more)

### Community 58 - "Job"
Cohesion: 0.20
Nodes (7): Job, mysqlJobRepository, fakeJobRepo, Context, DB, NewMysqlJobRepository(), Time

### Community 59 - "mockTokenServer"
Cohesion: 0.20
Nodes (15): Client, Server, mockClient(), mockTokenServer(), TestGitHubProviderExchange(), TestGoogleProviderExchange(), TestGoogleProviderExchangeBadUserInfo(), TestGoogleProviderExchangeTokenError() (+7 more)

### Community 60 - "health.go"
Cohesion: 0.19
Nodes (15): Client, Context, DB, Duration, ResponseWriter, NewReadiness(), NewRedisChecker(), NewSQLChecker() (+7 more)

### Community 62 - "responseBodyWriter"
Cohesion: 0.17
Nodes (10): Buffer, ResponseWriter, newResponseBodyWriter(), sanitizeBody(), sanitizeJSONField(), TestResponseBodyWriterCapturesAndTruncates(), TestResponseBodyWriterWriteHeaderPassthrough(), TestSanitizeBody() (+2 more)

### Community 63 - "New"
Cohesion: 0.20
Nodes (13): Cipher, New(), TestBlindIndexDeterministicAndDistinct(), TestBlindIndexEmpty(), TestBlindIndexWithoutKey(), TestDecryptTamperedFails(), TestDecryptWrongKeyFails(), TestEmptyValuePassthrough() (+5 more)

### Community 64 - "oauth/interfaces/handler_test.go"
Cohesion: 0.32
Nodes (15): OAuthHandler, NewOAuthHandler(), assertCode(), doRequest(), githubProvider(), Engine, ResponseRecorder, newTestHandler() (+7 more)

### Community 65 - "NewAuthService"
Cohesion: 0.19
Nodes (19): BenchmarkLogin(), benchUser(), B, TestForgotPasswordNotConfigured(), NewAuthService(), appCode(), fakeSessionStore, sessionRecord (+11 more)

### Community 66 - "rabbitmq_queue_test.go"
Cohesion: 0.20
Nodes (12): Context, Delivery, newTestRabbitQueue(), TestRabbitMQQueue_ConsumeTimeout(), TestRabbitMQQueue_ConsumeUnmarshalError(), TestRabbitMQQueue_NackRequeues(), TestRabbitMQQueue_SubmitConsumeAck(), TestRabbitMQQueueImplementsInterfaces() (+4 more)

### Community 67 - "newMockGormDB"
Cohesion: 0.25
Nodes (13): DB, Sqlmock, newMockGormDB(), TestConfigurePool(), DB, TestTransaction_Commit(), TestTransaction_Rollback(), TestWithTx_BeginError() (+5 more)

### Community 68 - "Now"
Cohesion: 0.15
Nodes (20): NewClientHub(), NewPresenceManager(), TestPresenceIsOnline(), TestPresenceIsTyping(), TestPresenceManagerAllPresences(), TestPresenceManagerHeartbeat(), TestPresenceManagerHeartbeatRestoresOnlineStatus(), TestPresenceManagerOfflineMissing() (+12 more)

### Community 69 - "dispatcher"
Cohesion: 0.15
Nodes (10): Channel, Channel, Channel, Context, dispatcher, RWMutex, NewDispatcher(), TestDispatcherDispatch() (+2 more)

### Community 70 - "AuthorizationMiddleware"
Cohesion: 0.14
Nodes (17): AuthorizationStore, DBAuthorizationStore, Enforcer, HandlerFunc, ProtectedMiddleware(), HandlerFunc, AuthorizationMiddleware(), Context (+9 more)

### Community 71 - "RegisterAuthRoutes"
Cohesion: 0.29
Nodes (14): RouterGroup, Service, RegisterAuthRoutes(), RegisterCaptchaRoute(), Engine, newRouterLimiter(), testAuthConfig(), TestLoginRateLimitUsesIPScope() (+6 more)

### Community 72 - "Limiter"
Cohesion: 0.10
Nodes (29): Limiter, Context, Duration, Scripter, LimitKey(), NewLimiter(), newTestLimiter(), TestLimiterAllowsOnlyConfiguredWindow() (+21 more)

### Community 73 - "DBConfig"
Cohesion: 0.17
Nodes (17): DBConfig, configurePool(), ConnectWithRetry(), dsn(), Context, DB, openByDriver(), openMySQL() (+9 more)

### Community 74 - "NewCSVExporter"
Cohesion: 0.15
Nodes (12): CSVExporter, ExcelExporter, Context, Writer, NewCSVExporter(), Context, Writer, NewExcelExporter() (+4 more)

### Community 75 - "New"
Cohesion: 0.21
Nodes (12): AuditHandler, NewAuditService(), auditAppCode(), TestAuditServiceGetMapsNotFound(), TestAuditServiceListReturnsDTOAndPassesPagination(), Context, NewAuditHandler(), TestAuditGetInvalidIDReturnsStableBadRequest() (+4 more)

### Community 76 - "SetupRouter"
Cohesion: 0.22
Nodes (14): ConfigureTrustedProxies(), Engine, SetupRouter(), freeAddr(), TestConfigureTrustedProxies(), testLogger(), TestNewServerPlain(), TestNewServerTLSInvalidFiles() (+6 more)

### Community 77 - "NewChannelManager"
Cohesion: 0.26
Nodes (11): NewChannel(), NewChannelManager(), TestChannelManagerGetChannelMissing(), TestChannelManagerGetSubscribers(), TestChannelManagerPreCreatedBroadcast(), TestChannelManagerSubscribeCreatesWithType(), TestChannelManagerSubscribeExisting(), TestChannelManagerUnsubscribe() (+3 more)

### Community 78 - "NewService"
Cohesion: 0.11
Nodes (15): fakeRouter, Service, Handler, Client, Time, NewService(), TestService(), Context (+7 more)

### Community 79 - "Pagination"
Cohesion: 0.14
Nodes (11): CreatePermissionRequest, UpdatePermissionRequest, PermissionResponse, Time, ToPermissionResponse(), ToPermissionResponses(), isDuplicateKey(), Context (+3 more)

### Community 80 - "Worker"
Cohesion: 0.20
Nodes (10): Worker, Context, RWMutex, NewWorker(), Duration, testWorker(), TestWorkerFlushesConfiguredBatch(), TestWorkerRejectsWhenQueueFull() (+2 more)

### Community 81 - "AdminUserService"
Cohesion: 0.21
Nodes (9): AdminCreateUserRequest, AdminUpdateUserRequest, AdminUser, AdminUserService, ListUserFilter, userRole, Context, DB (+1 more)

### Community 82 - "RedisStore"
Cohesion: 0.22
Nodes (9): RedisStore, Service, Client, Context, Duration, NewRedisStore(), NewService(), TestGenerate() (+1 more)

### Community 83 - "Context"
Cohesion: 0.33
Nodes (5): Context, UserInfo, _UserInfoService_GetUser_Handler(), _UserInfoService_ListUsers_Handler(), UnaryServerInterceptor

### Community 84 - "DeadLetter"
Cohesion: 0.17
Nodes (8): DeadLetter, DeadLetterRepository, mysqlDeadLetterRepository, fakeDeadLetterRepo, Context, DB, NewMysqlDeadLetterRepository(), Time

### Community 85 - "ImportResult"
Cohesion: 0.19
Nodes (8): ExcelImporter, ImportError, ImportResult, Context, Reader, NewExcelImporter(), Time, NewImportResult()

### Community 86 - "newMockGormDB"
Cohesion: 0.24
Nodes (14): MySQLBindingRepository, DB, NewMySQLBindingRepository(), bindingRows(), DB, Sqlmock, newMockGormDB(), TestCreateError() (+6 more)

### Community 87 - "message.go"
Cohesion: 0.29
Nodes (6): ChatPayload, NotificationPayload, PingPayload, PongPayload, PresencePayload, SubscribePayload

### Community 88 - "gzipRouter"
Cohesion: 0.17
Nodes (12): HandlerFunc, ResponseWriter, Writer, GzipCompression(), isAlreadyCompressed(), Engine, gzipRouter(), TestGzipCompressionEncodesBody() (+4 more)

### Community 89 - "Event"
Cohesion: 0.06
Nodes (36): ContextWithTrace(), DefaultTracingConfig(), Context, TracerProvider, InitTracing(), ShutdownTracing(), TestContextWithTraceEmptyPassthrough(), TestDefaultTracingConfig() (+28 more)

### Community 90 - "fakeUserRepository"
Cohesion: 0.23
Nodes (4): errDeleteRepository, errFindRepository, Context, fakeUserRepository

### Community 91 - "newMonitoringHandler"
Cohesion: 0.70
Nodes (4): newMonitoringHandler(), TestAdminMonitoringHandlerHealth(), TestAdminMonitoringHandlerMetrics(), TestAdminMonitoringHandlerStatus()

### Community 92 - "MySQLStore"
Cohesion: 0.17
Nodes (10): Context, DB, DeletedAt, Time, NewMySQLStore(), DB, testDB(), TestMySQLStore() (+2 more)

### Community 93 - "Message"
Cohesion: 0.31
Nodes (6): fakeDispatcher, Context, Mutex, Dispatcher, Channel, Message

### Community 94 - "New"
Cohesion: 0.19
Nodes (12): New(), TestAppError(), TestAppErrorWithCause(), TestFailDoesNotLeakInfrastructureDetails(), TestOKIncludesStableEnvelopeAndRequestID(), TestPageIncludesStableEnvelopeAndPagination(), TestCreatedUsesStandardEnvelope(), TestFailHidesInternalCauseAndIncludesRequestID() (+4 more)

### Community 95 - "TestDB"
Cohesion: 0.31
Nodes (11): dbReachable(), defaultDBPort(), envDBConfig(), DB, NewTestDB(), NewTestDBWithPool(), openByDriver(), SkipUnlessDB() (+3 more)

### Community 96 - "ListUsersRequest"
Cohesion: 0.14
Nodes (5): MessageState, SizeCache, UnknownFields, GetUserRequest, ListUsersRequest

### Community 97 - "Application"
Cohesion: 0.16
Nodes (11): Application, fakeComponent, Component, forwardError(), Context, Duration, NewApplication(), Context (+3 more)

### Community 98 - "fakeAuthzModule"
Cohesion: 0.15
Nodes (5): fakeAuthzModule, fakeBusinessModule, EventBus, HandlerFunc, TestBusinessRoutesRequireProtectedMiddleware()

### Community 99 - "JobHistory"
Cohesion: 0.23
Nodes (7): JobHistory, JobHistoryRepository, mysqlJobHistoryRepository, Context, DB, NewMysqlJobHistoryRepository(), Time

### Community 100 - "OAuthBinding"
Cohesion: 0.22
Nodes (5): fakeBindingRepo, OAuthBinding, fakeBindingRepo, Time, Context

### Community 101 - "NewMysqlRepository"
Cohesion: 0.23
Nodes (15): TestMysqlRepositoryMySQLIntegration(), NewMysqlRepository(), DB, newTestUser(), newUserTestDB(), TestMysqlRepositoryCreateAndFindByID(), TestMysqlRepositoryFindByEmailHash(), TestMysqlRepositoryFindByPhoneHash() (+7 more)

### Community 102 - "NewDBAPIKeyStore"
Cohesion: 0.33
Nodes (12): createKey(), APIKey, DB, newTestDB(), TestDBAPIKeyStore_GetByKeyHash(), TestDBAPIKeyStore_GetByKeyHashNotFound(), TestDBAPIKeyStore_UpdateLastUsed(), TestVerifyWithDBStore() (+4 more)

### Community 103 - "IdempotencyMiddleware"
Cohesion: 0.20
Nodes (15): Int32, Client, Duration, HandlerFunc, IdempotencyMiddleware(), Client, Engine, idempotencyRouter() (+7 more)

### Community 104 - "RabbitMQQueue"
Cohesion: 0.26
Nodes (7): Context, Delivery, Duration, Mutex, NewRabbitMQQueue(), RabbitMQChannel, RabbitMQQueue

### Community 105 - "New"
Cohesion: 0.11
Nodes (20): New(), newOSSStorage(), Client, Config, Context, Duration, ReadCloser, Reader (+12 more)

### Community 106 - "Client"
Cohesion: 0.26
Nodes (7): Conn, Context, HandlerFunc, RWMutex, Time, WSHandler(), Client

### Community 107 - "Lock"
Cohesion: 0.33
Nodes (8): generateToken(), Client, Context, Duration, Time, NewLock(), AcquireResult, Lock

### Community 108 - "fakeRoleRepository"
Cohesion: 0.31
Nodes (3): errDeleteRoleRepository, fakeRoleRepository, Context

### Community 110 - "New"
Cohesion: 0.19
Nodes (9): Client, Queue, New(), TestNew_InvalidType(), TestNew_Redis(), Config, KafkaConfig, RabbitMQConfig (+1 more)

### Community 111 - "Container"
Cohesion: 0.09
Nodes (23): APIKeyVerifier, Container, Cipher, Client, CronScheduler, DB, DBCollector, Dispatcher (+15 more)

### Community 112 - "RedisQueue"
Cohesion: 0.27
Nodes (5): Client, Context, Duration, NewRedisQueue(), RedisQueue

### Community 113 - "NewManagementServer"
Cohesion: 0.20
Nodes (8): Handler, passingChecker, Server, HealthRouter(), NewManagementServer(), Context, TestManagementRouterExposure(), TestNewManagementServer()

### Community 115 - "mysqlRepository"
Cohesion: 0.27
Nodes (3): Context, DB, mysqlRepository

### Community 116 - "NewServer"
Cohesion: 0.38
Nodes (12): NewServer(), New(), TestCircuitHalfOpenOnlyAllowsProbe(), TestCircuitOpensAfterConsecutiveFailures(), TestCircuitRecoversAfterCooldown(), TestDoCancelsDuringRetryWait(), TestDoInjectsTraceParent(), TestDoNoRetryOn4xx() (+4 more)

### Community 117 - "PresenceManager"
Cohesion: 0.16
Nodes (4): RWMutex, Time, Presence, PresenceManager

### Community 118 - "KafkaQueue"
Cohesion: 0.22
Nodes (10): Context, Duration, JobData, Message, Mutex, NewKafkaQueue(), KafkaConfig, KafkaMessageReader (+2 more)

### Community 119 - "Permission"
Cohesion: 0.23
Nodes (9): fakePermissionRepository, Permission, Context, permissionAppCode(), TestPermissionServiceCreateMapsDuplicateNameToConflict(), TestPermissionServiceDeleteWrapsRepositoryError(), TestPermissionServiceListPassesPagination(), TestPermissionServiceUpdateMapsNotFound() (+1 more)

### Community 120 - "ClientHub"
Cohesion: 0.16
Nodes (7): mustEncode(), BuildUserChannel(), RawMessage, Time, broadcastMsg, ClientHub, WSMessage

### Community 121 - "mysqlRepository"
Cohesion: 0.20
Nodes (6): RoleRepository, rolePermission, Context, DB, mysqlRepository, NewMysqlRepository()

### Community 122 - "newEncryptionTestDB"
Cohesion: 0.24
Nodes (9): contact, contactPtr, DB, newEncryptionTestDB(), TestEncryptionHookBatchCreate(), TestEncryptionHookEmptySourceStoresNullAndCoexists(), TestEncryptionHookEncryptsOnWriteDecryptsOnRead(), TestEncryptionHookNoKeyStoresPlaintextButHashes() (+1 more)

### Community 123 - "LoginFailureTracker"
Cohesion: 0.33
Nodes (8): LockoutConfig, LoginFailureTracker, DefaultLockoutConfig(), Client, Context, Duration, lockKey(), NewLoginFailureTracker()

### Community 124 - "New"
Cohesion: 0.23
Nodes (6): PermissionHandler, Context, DB, EventBus, New(), Module

### Community 125 - "统一响应契约"
Cohesion: 0.13
Nodes (13): Admin 模块, Auth 模块, HTTP API 契约, 中间件, 健康与观测（非业务）, 完整接口文档, 分页, 国际化 (+5 more)

### Community 126 - "userinfo_grpc.pb.go"
Cohesion: 0.20
Nodes (9): userInfoService, DB, DB, NewUserInfoGRPCService(), RegisterUserInfoServiceServer(), ServiceRegistrar, UnimplementedUserInfoServiceServer, UnsafeUserInfoServiceServer (+1 more)

### Community 128 - "routerLimiterRedis"
Cohesion: 0.29
Nodes (6): routerLimiterRedis, BoolSliceCmd, Cmd, Context, StringCmd, routerLimiterInt()

### Community 129 - "i18n.go"
Cohesion: 0.18
Nodes (7): HandlerFunc, Locale(), ParseAcceptLanguage(), TestParseAcceptLanguage(), TestTDefaultsToChinese(), TestTfFormatsArgs(), Tf()

### Community 130 - "ValidateJSON"
Cohesion: 0.15
Nodes (14): RouterGroup, RegisterRoleRoutes(), RouterGroup, RegisterUserRoutes(), Context, HandlerFunc, localeOf(), TestValidateJSONTranslatesFieldErrors() (+6 more)

### Community 131 - "newLock"
Cohesion: 0.18
Nodes (16): RedisConfig, Client, newLock(), TestLock_AcquireRelease(), TestLock_ConcurrentAcquire(), TestLock_Extend(), TestLock_ReleaseOnlyOwnToken(), TestLock_WithLock() (+8 more)

### Community 132 - "mysqlAuditRepository"
Cohesion: 0.46
Nodes (3): mysqlAuditRepository, deserializeChanges(), Context

### Community 133 - "validator/validator.go"
Cohesion: 0.25
Nodes (8): FieldLevel, FuzzValidateRules(), F, Validate(), validateIDCard(), validateMobile(), validatePassword(), validateUsername()

### Community 134 - "fakeRedis"
Cohesion: 0.06
Nodes (30): fakePipeline, fakeRedis, redisClientAdapter, RedisSessionStore, sessionPipeline, sessionRedis, SessionStore, BoolCmd (+22 more)

### Community 135 - "ImportService"
Cohesion: 0.15
Nodes (19): ImportService, ImportJobRepository, UserRepository, DB, newSqliteDB(), Context, DB, Format (+11 more)

### Community 136 - "Logger"
Cohesion: 0.05
Nodes (38): AtomicLevel, Config, PingServer, pingService, Server, Context, Server, RegisterPingServer() (+30 more)

### Community 137 - "AGENTS.md"
Cohesion: 0.14
Nodes (12): Commit Message 规范, graphify, Release Note 规范, 保护工作区, 分支策略, 开发前必读, 文档维护, 架构约束 (+4 more)

### Community 138 - "New"
Cohesion: 0.13
Nodes (25): fakeUserRepository, NewAdminJobHandler(), TestAdminJobHandlerGet(), TestAdminJobHandlerList(), TestAdminJobHandlerListDeadLetters(), TestAdminJobHandlerResolveDeadLetter(), TestAdminJobHandlerRetry(), TestAdminJobHandlerSubmit() (+17 more)

### Community 139 - "adminAuthRouter"
Cohesion: 0.27
Nodes (9): AdminAuth(), HandlerFunc, adminAuthRouter(), Engine, TestAdminAuthAllowsAdminRole(), TestAdminAuthAllowsEnglishAdminRole(), TestAdminAuthRejectsNonAdminRole(), TestAdminAuthRequiresAuthentication() (+1 more)

### Community 140 - "interfaces/fakes_test.go"
Cohesion: 0.22
Nodes (4): fakeEventBus, testRole, testUserRole, Mutex

### Community 141 - "GitHubProvider"
Cohesion: 0.24
Nodes (7): Client, Config, Context, UserInfo, NewGitHubProvider(), GitHubConfig, GitHubProvider

### Community 142 - "NewWeChatProvider"
Cohesion: 0.24
Nodes (7): Client, Config, Context, UserInfo, NewWeChatProvider(), WeChatConfig, WeChatProvider

### Community 143 - "newResetService"
Cohesion: 0.27
Nodes (11): fakeSessionStore, Client, Miniredis, newResetRedis(), newResetService(), TestForgotPasswordHidesMissingUser(), TestForgotPasswordSendsCode(), TestGenerateResetCodeReal() (+3 more)

### Community 144 - "AdminTaskService"
Cohesion: 0.29
Nodes (5): AdminTaskService, TaskExecution, TaskInfo, Context, Time

### Community 145 - "SecurityHeadersFromConfig"
Cohesion: 0.27
Nodes (9): SecurityConfig, DefaultSecurityConfig(), TestSecurityHeadersMiddleware(), TestSecurityHeadersNilValuesSkipped(), TestSecurityHeadersRespectsCustomValues(), Context, HandlerFunc, SecurityHeadersFromConfig() (+1 more)

### Community 146 - "Module"
Cohesion: 0.25
Nodes (3): Module, EventBus, HandlerFunc

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
Cohesion: 0.33
Nodes (5): NewHub(), TestHubLifecycle(), TestWebSocketNotification(), waitHub(), mockConn

### Community 153 - "fakeContainer"
Cohesion: 0.44
Nodes (8): bridgeFn(), fakeContainer(), newTestLogger(), TestBridgeWorkerConversionFailureErrors(), TestBridgeWorkerPublishesStrongTypeToBareTopic(), TestBridgeWorkerUnknownTypeErrors(), TestEventBusBridgePublishesToBareTopic(), TestRegisterOutboxWorkersRegistersAll()

### Community 154 - "CLAUDE.md"
Cohesion: 0.18
Nodes (8): Commit Message 规范, graphify, Release Note 规范, 保护工作区, 回复格式, 开发前必读, 文档维护, 简单优先

### Community 155 - "Metrics"
Cohesion: 0.33
Nodes (7): HandlerFunc, Metrics(), gatherHTTPMetric(), Engine, setupMetricsEngine(), TestMetricsLabelsUseRouteTemplate(), TestMetricsRecordsUnmatchedPath()

### Community 156 - "Timeout"
Cohesion: 0.22
Nodes (6): Duration, HandlerFunc, TestTimeoutOnlyPropagatesContextDeadline(), Timeout(), RouterGroup, RegisterSwagger()

### Community 157 - "NewPermissionService"
Cohesion: 0.29
Nodes (11): PermissionService, PermissionRepository, NewPermissionService(), NewPermissionHandler(), TestPermissionDeleteReturnsNoContent(), TestPermissionDeleteServiceError(), TestPermissionGetInvalidID(), TestPermissionGetReturnsOK() (+3 more)

### Community 160 - "ResetStore"
Cohesion: 0.39
Nodes (5): ResetStore, Client, Context, Duration, NewResetStore()

### Community 161 - "贡献指南"
Cohesion: 0.18
Nodes (10): Commit 规范, Pull Request, 代码规范, 分支策略, 开发环境, 报告问题, 模块开发, 测试 (+2 more)

### Community 162 - "EventBus"
Cohesion: 0.19
Nodes (9): EventBus, Handler, RWMutex, New(), TestEventBus_Clear(), TestEventBus_HandlerPanicRecovered(), TestEventBus_MultipleHandlers(), TestEventBus_PublishAsync() (+1 more)

### Community 163 - "NewMQPublisher"
Cohesion: 0.25
Nodes (8): Context, Queue, RawMessage, NewMQPublisher(), TestMQPublisher_Publish(), TestMQPublisher_PublishError(), traceFromMetadata(), MQPublisher

### Community 164 - "Registry"
Cohesion: 0.54
Nodes (5): Format, Importer, Registry, NewRegistry(), TestRegistryGetUnsupported()

### Community 165 - "API 示例"
Cohesion: 0.17
Nodes (12): API 示例, Metrics, OAuth 登录, 健康检查, 创建用户, 刷新 Token, 忘记密码（发送验证码）, 查看认证限流状态 (+4 more)

### Community 166 - "MySQLStore"
Cohesion: 0.33
Nodes (4): JobRepository, Context, NewMySQLStore(), MySQLStore

### Community 167 - "userinfo.pb.go"
Cohesion: 0.29
Nodes (3): file_proto_jimu_v1_userinfo_proto_init(), file_proto_jimu_v1_userinfo_proto_rawDescGZIP(), init()

### Community 168 - "NewEventBusPublisher"
Cohesion: 0.32
Nodes (6): Context, EventBus, RawMessage, NewEventBusPublisher(), EventBusPublisher, EventPayload

### Community 169 - "AuditRepository"
Cohesion: 0.28
Nodes (7): AuditRepository, AdminAuditHandler, DB, NewAdminAuditHandler(), TestAdminAuditHandlerList(), DB, NewMysqlAuditRepository()

### Community 170 - "JobData"
Cohesion: 0.35
Nodes (5): Context, Duration, errorQueue, fakeQueue, JobData

### Community 171 - "mysql/003_extensions.sql"
Cohesion: 0.25
Nodes (7): api_keys, dead_letters, import_jobs, job_history, jobs, scheduled_jobs, user_oauth_bindings

### Community 172 - "postgres/003_extensions.sql"
Cohesion: 0.25
Nodes (7): api_keys, dead_letters, import_jobs, job_history, jobs, scheduled_jobs, user_oauth_bindings

### Community 173 - "Bootstrap"
Cohesion: 0.52
Nodes (6): moduleLogger, registerRouter, Bootstrap(), registerEventBusBridge(), registerHTTP(), registerOutboxWorkers()

### Community 174 - "JobRegistry"
Cohesion: 0.17
Nodes (7): ComponentProvider, ErrorSource, EventBus, HTTPMiddlewareProvider, JobRegistry, Module, ProtectedHTTPMiddlewareProvider

### Community 175 - "Registry"
Cohesion: 0.62
Nodes (4): Exporter, Format, Registry, NewRegistry()

### Community 176 - "startTestGRPCServer"
Cohesion: 0.43
Nodes (6): ClientConn, Server, startTestGRPCServer(), TestGRPCHealthCheck(), TestGRPCPingEcho(), TestGRPCReflection()

### Community 177 - "newTestScheduler"
Cohesion: 0.62
Nodes (6): NewAdminTaskService(), newTestScheduler(), TestAdminTaskServiceGetHistory(), TestAdminTaskServiceListTasks(), TestAdminTaskServiceToggleTask(), TestAdminTaskServiceTriggerTask()

### Community 179 - "Jimu Backend Framework"
Cohesion: 0.20
Nodes (10): CLI 工具, Docker 部署, Jimu Backend Framework, K8s 部署, License, Makefile 命令, 技术栈, 特性 (+2 more)

### Community 180 - "setupTestCache"
Cohesion: 0.52
Nodes (6): setupTestCache(), TestRedisCache_Delete(), TestRedisCache_DeletePattern(), TestRedisCache_GetOrSet(), TestRedisCache_GetOrSetStampede(), TestRedisCache_SetAndGet()

### Community 181 - "Context"
Cohesion: 0.27
Nodes (5): fakeProvider, fakeSessionStore, Context, Duration, UserInfo

### Community 182 - "Duration"
Cohesion: 0.43
Nodes (4): Duration, ExponentialBackoff, FixedRetry, RetryStrategy

### Community 183 - "安全政策"
Cohesion: 0.22
Nodes (8): 响应时间, 如何报告, 安全政策, 安全最佳实践, 已知限制, 报告安全问题, 披露政策, 支持的版本

### Community 185 - "Server"
Cohesion: 0.29
Nodes (4): Server, formatAddr(), Context, TestFormatAddr()

### Community 186 - "mysqlAPIKeyRepository"
Cohesion: 0.29
Nodes (5): mysqlAPIKeyRepository, APIKey, Context, DB, NewMysqlAPIKeyRepository()

### Community 187 - "[v0.1.0] - 2026-08-21"
Cohesion: 0.29
Nodes (6): Changelog, [Unreleased], [v0.1.0] - 2026-08-21, 修复, 变更, 新增

### Community 188 - "Fail"
Cohesion: 0.24
Nodes (7): AdminUserHandler, Context, Context, NewAdminUserHandler(), paginationFromQuery(), Context, Fail()

### Community 189 - "NewAdminUserService"
Cohesion: 0.29
Nodes (9): testRole, NewAdminUserService(), TestAdminUserServiceAssignRoles(), TestAdminUserServiceAssignRolesRoleQueryError(), TestAdminUserServiceCreateUser(), TestAdminUserServiceDisableUser(), TestAdminUserServiceGetUser(), TestAdminUserServiceListUsers() (+1 more)

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

### Community 194 - "request.go"
Cohesion: 0.40
Nodes (4): forgotPasswordRequest, loginRequest, refreshRequest, resetPasswordRequest

### Community 195 - "mysql/001_core.sql"
Cohesion: 0.33
Nodes (5): permissions, role_permissions, roles, user_roles, users

### Community 196 - "postgres/001_core.sql"
Cohesion: 0.33
Nodes (5): permissions, role_permissions, roles, user_roles, users

### Community 197 - "bug_report.md"
Cohesion: 0.29
Nodes (6): 复现步骤, 实际行为, 描述, 期望行为, 环境, 附加信息

### Community 198 - "ClamAVScanner"
Cohesion: 0.31
Nodes (7): ClamAVConfig, ClamAVScanner, Scanner, Context, Duration, Reader, NewClamAVScanner()

### Community 199 - "events.go"
Cohesion: 0.40
Nodes (4): UserCreatedEvent, UserDeletedEvent, UserLoggedInEvent, UserUpdatedEvent

### Community 200 - "快速开始"
Cohesion: 0.29
Nodes (7): 前置条件, 可观测性（可选）, 安装, 快速开始, 方式一：本地运行, 方式二：Docker Compose 一键启动, 配置

### Community 201 - "分支策略"
Cohesion: 0.33
Nodes (6): Tag 与发布, 分支模型, 分支策略, 合并与 PR, 命名约定, 回滚

### Community 202 - "NewLocalStorage"
Cohesion: 0.15
Nodes (16): TestLocalStorageDeleteMissingIsNoop(), TestLocalStorageDownloadNotFound(), TestLocalStoragePathTraversal(), TestLocalStoragePresignedUploadUnsupported(), TestLocalStoragePresignedURL(), TestLocalStorageSizeMismatch(), TestLocalStorageSizeMissing(), TestNewDefaultsLocal() (+8 more)

### Community 203 - "newRateLimitHandler"
Cohesion: 0.33
Nodes (8): AdminRateLimitHandler, Client, NewAdminRateLimitHandler(), Client, newRateLimitHandler(), TestAdminRateLimitAuthPeek_ExistingCount(), TestAdminRateLimitAuthPeek_KeyAbsent(), TestAdminRateLimitAuthPeek_MissingParam()

### Community 204 - "CaptchaHandler"
Cohesion: 0.83
Nodes (3): CaptchaHandler, Service, NewCaptchaHandler()

### Community 206 - "PermissionMiddleware"
Cohesion: 0.50
Nodes (3): Enforcer, HandlerFunc, PermissionMiddleware()

### Community 207 - "PULL_REQUEST_TEMPLATE.md"
Cohesion: 0.33
Nodes (5): 变更类型, 变更说明, 检查清单, 相关 Issue, 风险与注意事项

### Community 225 - "As"
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

### Community 235 - "配置说明"
Cohesion: 0.40
Nodes (5): 多环境配置, 环境变量, 配置说明, 配置项, 静态加密（Data at Rest）

### Community 236 - "Router"
Cohesion: 0.20
Nodes (5): Router, RouterGroup, RegisterAuditRoutes(), RouterGroup, RegisterPermissionRoutes()

### Community 239 - "mysqlPermissionRepository"
Cohesion: 0.31
Nodes (4): mysqlPermissionRepository, Context, DB, NewMysqlPermissionRepository()

### Community 241 - "newTestGRPCService"
Cohesion: 0.48
Nodes (6): ClientConnInterface, newTestGRPCService(), TestUserInfoService_GetUser(), TestUserInfoService_ListUsers(), NewUserInfoServiceClient(), UserInfoServiceClient

### Community 242 - "newWSHandler"
Cohesion: 0.73
Nodes (5): NewAdminWSHandler(), newWSHandler(), TestAdminWSHandlerOnlineUsers(), TestAdminWSHandlerPresence(), TestAdminWSHandlerPush()

### Community 243 - "NewPathEnforcer"
Cohesion: 0.38
Nodes (5): TestProtectedMiddlewareRequiresAccessToken(), DB, Enforcer, NewEnforcer(), NewPathEnforcer()

### Community 244 - "newHistoryTestDB"
Cohesion: 0.40
Nodes (4): TestMysqlDeadLetterRepositoryCRUD(), DB, newHistoryTestDB(), TestMysqlJobHistoryRepositoryCreateAndList()

### Community 245 - "securityRouter"
Cohesion: 0.53
Nodes (5): Engine, securityRouter(), TestSecurityHandlesAllowedPreflight(), TestSecurityRejectsOversizedBody(), TestSecurityUsesOriginAllowList()

### Community 252 - "JobDef"
Cohesion: 0.18
Nodes (8): Context, RWMutex, Context, Time, failStore, JobDef, MemoryStore, Store

### Community 265 - "fakePermissionRepository"
Cohesion: 0.29
Nodes (5): errDeletePermissionRepository, fakePermissionRepository, Context, TestPermissionCreateReturnsCreated(), TestPermissionUpdateReturnsOK()

### Community 266 - "AuditService"
Cohesion: 0.23
Nodes (8): AuditLogResponse, AuditService, Change, Time, ToAuditLogResponse(), ToAuditLogResponses(), Context, serializeChanges()

### Community 268 - "v0.1.0"
Cohesion: 0.25
Nodes (7): v0.1.0, 亮点, 修复, 变更, 新增, 说明, 验证

### Community 272 - "newRedisTestQueue"
Cohesion: 0.39
Nodes (7): Client, newRedisTestQueue(), TestQueueContract_SubmitConsume(), TestRedisQueueAckRemovesFromProcessing(), TestRedisQueueImplementsInterfaces(), TestRedisQueueNackRequeues(), TestRedisQueueRequeueExpired()

### Community 275 - "newMockS3Storage"
Cohesion: 0.70
Nodes (4): newMockS3Storage(), TestS3DeleteMissingIsNoop(), TestS3ExistsAndSizeMissing(), TestS3UploadDownloadRoundTrip()

## Knowledge Gaps
- **269 isolated node(s):** `[Unreleased]`, `新增`, `变更`, `修复`, `亮点` (+264 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **48 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `T()` connect `T` to `RoleService`, `api_contract_test.go`, `New`, `NewSnowflake`, `New`, `User`, `ws_integration_test.go`, `New`, `oauth/application/service_test.go`, `gorm_logger_test.go`, `NewAdminConfigService`, `AdminMonitoringService`, `Manager`, `DefaultCSRFConfig`, `middleware/middleware_test.go`, `migrate.go`, `NewCleanupService`, `Context`, `bindImportFile`, `NewUserRateLimiter`, `JWT`, `generator/module.go`, `signature_test.go`, `Email`, `SMS`, `New`, `New`, `NewWebhook`, `Page`, `NewWithStore`, `NewAdminAPIKeyService`, `Role`, `mockTokenServer`, `responseBodyWriter`, `New`, `oauth/interfaces/handler_test.go`, `NewAuthService`, `rabbitmq_queue_test.go`, `newMockGormDB`, `Now`, `dispatcher`, `AuthorizationMiddleware`, `RegisterAuthRoutes`, `Limiter`, `DBConfig`, `NewCSVExporter`, `New`, `SetupRouter`, `NewChannelManager`, `NewService`, `Worker`, `RedisStore`, `newMockGormDB`, `gzipRouter`, `Event`, `newMonitoringHandler`, `MySQLStore`, `New`, `TestDB`, `Application`, `fakeAuthzModule`, `NewMysqlRepository`, `NewDBAPIKeyStore`, `IdempotencyMiddleware`, `New`, `NewManagementServer`, `NewServer`, `Permission`, `newEncryptionTestDB`, `i18n.go`, `ValidateJSON`, `newLock`, `ImportService`, `Logger`, `New`, `adminAuthRouter`, `newResetService`, `SecurityHeadersFromConfig`, `NewCSVImporter`, `.Validate`, `AuditMiddleware`, `NewDBCollector`, `TestWebSocketNotification`, `fakeContainer`, `Metrics`, `Timeout`, `NewPermissionService`, `EventBus`, `NewMQPublisher`, `Registry`, `AuditRepository`, `startTestGRPCServer`, `newTestScheduler`, `setupTestCache`, `Server`, `NewAdminUserService`, `Security`, `newRepoTestDB`, `TestReadinessBoundsCheckerDuration`, `NewLocalStorage`, `newRateLimitHandler`, `newTestGRPCService`, `newWSHandler`, `NewPathEnforcer`, `newHistoryTestDB`, `securityRouter`, `fakePermissionRepository`, `newRedisTestQueue`, `newMockS3Storage`?**
  _High betweenness centrality (0.649) - this node is a cross-community bridge._
- **Why does `New()` connect `New` to `upload_handler_test.go`, `ImportService`, `run`, `AdminTaskService`, `NewAdminConfigService`, `Manager`, `bindImportFile`, `OK`, `AuditRepository`, `Module`, `newTestScheduler`, `NewAdminAPIKeyService`, `Fail`, `NewAdminUserService`, `newRateLimitHandler`, `NewService`, `AdminUserService`, `AdminAPIKeyHandler`, `newMonitoringHandler`, `AdminJobHandler`, `newWSHandler`, `AdminWSHandler`?**
  _High betweenness centrality (0.093) - this node is a cross-community bridge._
- **Why does `Now()` connect `Now` to `T`, `New`, `NewSnowflake`, `ImportService`, `ws_integration_test.go`, `gorm_logger_test.go`, `newRedisTestQueue`, `.Validate`, `AdminMonitoringService`, `AuditMiddleware`, `DefaultCSRFConfig`, `middleware/middleware_test.go`, `TestWebSocketNotification`, `Metrics`, `NewCleanupService`, `Context`, `NewUserRateLimiter`, `Wrap`, `JWT`, `circuit`, `signature_test.go`, `MySQLStore`, `OK`, `auth/apikey.go`, `UserHandler`, `NewWebhook`, `RedisCache`, `CronScheduler`, `NewWithStore`, `NewAdminAPIKeyService`, `mysqlAPIKeyRepository`, `NewAdminUserService`, `TestReadinessBoundsCheckerDuration`, `AuthorizationMiddleware`, `NewService`, `DeadLetter`, `ImportResult`, `newMockGormDB`, `Event`, `NewMysqlRepository`, `NewDBAPIKeyStore`, `Client`, `Lock`, `RedisQueue`, `PresenceManager`, `ClientHub`, `JobDef`?**
  _High betweenness centrality (0.084) - this node is a cross-community bridge._
- **Are the 4 inferred relationships involving `T()` (e.g. with `translateValidationDetails()` and `ValidateJSON()`) actually correct?**
  _`T()` has 4 INFERRED edges - model-reasoned connections that need verification._
- **Are the 93 inferred relationships involving `New()` (e.g. with `.CreateKey()` and `.ToggleTask()`) actually correct?**
  _`New()` has 93 INFERRED edges - model-reasoned connections that need verification._
- **Are the 80 inferred relationships involving `Now()` (e.g. with `.CreateKey()` and `.generateResetCode()`) actually correct?**
  _`Now()` has 80 INFERRED edges - model-reasoned connections that need verification._
- **Are the 78 inferred relationships involving `Fail()` (e.g. with `.Generate()` and `.Create()`) actually correct?**
  _`Fail()` has 78 INFERRED edges - model-reasoned connections that need verification._