# Graph Report - .  (2026-08-17)

## Corpus Check
- 24 files · ~140,478 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3700 nodes · 8057 edges · 282 communities (218 shown, 64 thin omitted)
- Extraction: 82% EXTRACTED · 18% INFERRED · 0% AMBIGUOUS · INFERRED: 1427 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- Permission Module
- Snowflake ID & Encryption
- API Contract & OpenAPI Tests
- File Storage (Local/S3/OSS/MinIO)
- HTTP Upload & Fake Storage
- Redis Pipeline (Fake)
- Database Migration Engine
- Role Module
- WebSocket Core
- Outbox User Repository
- User MySQL Integration Tests
- Specify Bash Scripts
- OAuth Provider Tests
- Helm Chart Templates
- Gorm Logger
- Outbox MQ Publisher
- Queue Worker Pool
- OpenTelemetry Tracing
- E2E Test Infrastructure
- API Key Repo (Infra)
- Admin API Key Service
- Admin Config Service
- Auth Module Bootstrap
- Data Import Service
- Admin Monitoring & Status
- Role Application Service
- API Key Repo (App)
- Admin HTTP Handlers
- Configuration Types
- User Domain & Repository
- Admin Job Handlers
- Feature Flag Manager
- Admin Import Handler
- CSRF Middleware
- HTTP Middleware (CORS/Logger/RequestID)
- Specification Tasks (v001)
- Docs & CI Concepts
- OAuth Binding Repository
- Redis Queue Implementation
- DB Cleanup Service
- Code Generator (Module Scaffold)
- HTTP Server Lifecycle
- User Rate Limit Middleware
- Auth Application Service
- API Key Auth Verifier
- JWT Token Service
- gRPC Server
- HTTP Client (Retry/CircuitBreaker)
- OAuth HTTP Handler
- MySQL Connection & Tests
- API Signature Middleware
- Cron Scheduler
- Global Rate Limit (Token Bucket)
- Webhook Notification
- WebSocket Hub & Handler
- User Application Service
- Redis Cache (Cache-Aside)
- Session Store & Notifier
- Role HTTP Handler
- WebSocket Notification Channel
- WebSocket Presence
- Admin User Service
- Audit Repository (Fake)
- Import Job Repository
- Auth HTTP Handler (Login/Register)
- Redis Distributed Lock
- User HTTP Handler
- Health Check (livez/readyz)
- Scheduler Store & Observability
- Project Constitution
- OAuth Application Service
- User Fake Repository
- Rate Limiter (Auth)
- OAuth Module Config
- AES-256-GCM Encryption
- Auth Benchmark Tests
- User Handler Tests
- RabbitMQ Queue
- Email Notification
- Notification Dispatcher
- OAuth Login Flow
- Audit Worker
- Casbin RBAC Authorization
- Database Config & Connection
- Audit HTTP Handler
- Email SMTP Service
- Scheduler Memory Store
- WebSocket Channels
- Admin Application Service
- Zap Logger
- Group 90
- Group 91
- Group 92
- Group 93
- Group 94
- Group 95
- Group 96
- Group 97
- Group 98
- Group 99
- Group 100
- Group 101
- Group 102
- Group 103
- Group 104
- Group 105
- Group 106
- Group 107
- Group 108
- Group 109
- Group 110
- Group 111
- Group 112
- Group 113
- Group 114
- Group 115
- Group 116
- Group 117
- Group 118
- Group 119
- Group 120
- Group 121
- Group 122
- Group 123
- Group 124
- Group 125
- Group 126
- Group 127
- Group 128
- Group 129
- Group 130
- Group 131
- Group 132
- Group 133
- Group 134
- Group 135
- Group 136
- Group 137
- Group 138
- Group 139
- Group 140
- Group 141
- Group 142
- Group 143
- Group 144
- Group 145
- Group 146
- Group 147
- Group 148
- Group 149
- Group 150
- Group 151
- Group 152
- Group 153
- Group 154
- Group 155
- Group 156
- Group 157
- Group 158
- Group 159
- Group 160
- Group 161
- Group 162
- Group 163
- Group 164
- Group 165
- Group 166
- Group 167
- Group 168
- Group 169
- Group 170
- Group 171
- Group 172
- Group 173
- Group 174
- Group 175
- Group 176
- Group 177
- Group 178
- Group 179
- Group 180
- Group 181
- Group 182
- Group 183
- Group 184
- Group 185
- Group 186
- Group 187
- Group 188
- Group 189
- Group 190
- Group 191
- Group 192
- Group 193
- Group 194
- Group 195
- Group 196
- Group 197
- Group 198
- Group 199
- Group 200
- Group 201
- Group 202
- Group 203
- Group 204
- Group 205
- Group 206
- Group 207
- Group 208
- Group 209
- Group 210
- Group 211
- Group 212
- Group 213
- Group 214
- Group 215
- Group 216
- Group 217
- Group 218
- Group 219
- Group 220
- Group 221
- Group 223
- Group 224
- Group 225
- Group 226
- Group 227
- Group 228
- Group 229
- Group 230
- Group 231
- Group 232
- Group 233
- Group 235
- Group 236
- Group 237
- Group 238
- Group 239
- Group 240
- Group 241
- Group 242
- Group 243
- Group 244
- Group 245
- Group 246
- Group 247
- Group 248
- Group 249
- Group 250
- Group 251
- Group 252
- Group 253
- Group 254
- Group 255
- Group 256
- Group 257
- Group 258
- Group 259
- Group 260
- Group 261
- Group 263
- Group 264
- Group 265
- Group 266
- Group 267
- Group 268
- Group 269
- Group 270
- Group 271
- Group 272
- Group 273
- Group 274
- Group 275
- Group 276
- Group 277
- Group 278
- Group 279
- Group 280

## God Nodes (most connected - your core abstractions)
1. `T()` - 674 edges
2. `New()` - 85 edges
3. `Fail()` - 82 edges
4. `Now()` - 82 edges
5. `User` - 71 edges
6. `OK()` - 55 edges
7. `New()` - 52 edges
8. `Wrap()` - 49 edges
9. `NewServer()` - 36 edges
10. `Message` - 36 edges

## Surprising Connections (you probably didn't know these)
- `Principle III: Composable Modules` --semantically_similar_to--> `Module Layering & contract.Module`  [INFERRED] [semantically similar]
  .specify/memory/constitution.md → CLAUDE.md
- `GitHub Flow Branch Strategy` --semantically_similar_to--> `Branch Strategy`  [INFERRED] [semantically similar]
  CLAUDE.md → CONTRIBUTING.md
- `Principle I: Business-Agnostic` --semantically_similar_to--> `Non-Goal: Tenant Isolation`  [INFERRED] [semantically similar]
  .specify/memory/constitution.md → CLAUDE.md
- `Conventional Commits` --semantically_similar_to--> `Commit Style`  [INFERRED] [semantically similar]
  AGENTS.md → CONTRIBUTING.md
- `Clean Architecture Layering` --semantically_similar_to--> `Modular Architecture`  [INFERRED] [semantically similar]
  AGENTS.md → README.md

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **CI Pipeline Stages (lint → test → bench + build → docker → security → smoke)** — github_workflows_ci_lint_job, github_workflows_ci_test_job, github_workflows_ci_bench_job, github_workflows_ci_security_job, github_workflows_ci_smoke_job [EXTRACTED 1.00]
- **Jimu Core Capabilities** — readmemd_modular_architecture, readmemd_jwt_casbin_auth, readmemd_rate_limiting, readmemd_event_bus, readmemd_outbox_pattern, readmemd_audit_logging, readmemd_feature_flag, readmemd_opentelemetry, readmemd_prometheus_metrics, readmemd_snowflake_id, readmemd_cron_scheduler, readmemd_grpc_server, readmemd_postgresql_support [EXTRACTED 1.00]
- **Observability Stack** — deploy_prometheus_prometheus_config, deploy_alertmanager_alertmanager_config, deploy_alert_rules_alert_rules, deploy_promtail_promtail_config, deploy_grafana_provisioning_datasources_prometheus_grafana_ds [EXTRACTED 0.95]
- **RBAC Entity Model** — specs_001_jimu_framework_spec_data_model_user, specs_001_jimu_framework_spec_data_model_role, specs_001_jimu_framework_spec_data_model_permission [EXTRACTED 0.95]
- **Constitution Core Principles** — _specify_memory_constitution_business_agnostic, _specify_memory_constitution_api_stability, _specify_memory_constitution_composable_modules, _specify_memory_constitution_simplicity, _specify_memory_constitution_verification, _specify_memory_constitution_documentation [EXTRACTED 1.00]
- **Jimu Helm Chart Resource Set** — deploy_helm_chart, deploy_helm_values, deploy_helm_templates_configmap, deploy_helm_templates_deployment, deploy_helm_templates_hpa, deploy_helm_templates_ingress, deploy_helm_templates_networkpolicy, deploy_helm_templates_pdb, deploy_helm_templates_prometheusrule, deploy_helm_templates_service [EXTRACTED 1.00]
- **jimu-server Kubernetes Resource Set** — deploy_k8s_configmap, deploy_k8s_config_files, deploy_k8s_deployment, deploy_k8s_hpa, deploy_k8s_ingress, deploy_k8s_networkpolicy, deploy_k8s_pdb, deploy_k8s_prometheusrule, deploy_k8s_service [EXTRACTED 1.00]
- **Jimu Prometheus Alert Rules** — deploy_k8s_prometheusrule_jimuhigherrorrate, deploy_k8s_prometheusrule_jimuhighlatency, deploy_k8s_prometheusrule_jimupodcrashlooping, deploy_k8s_prometheusrule_jimupoddown, deploy_k8s_prometheusrule_jimudbpoolhigh, deploy_k8s_prometheusrule_jimureadinessfailing [EXTRACTED 1.00]

## Communities (282 total, 64 thin omitted)

### Community 0 - "Permission Module"
Cohesion: 0.06
Nodes (35): CreatePermissionRequest, fakePermissionRepository, PermissionService, UpdatePermissionRequest, Permission, PermissionRepository, mysqlPermissionRepository, fakePermissionRepository (+27 more)

### Community 1 - "Snowflake ID & Encryption"
Cohesion: 0.06
Nodes (47): snowflakeModel, stringKeyModel, Generator, snowflake, uuidGenerator, applyBlindIndexFields(), applyEncryptedFields(), DB (+39 more)

### Community 2 - "API Contract & OpenAPI Tests"
Cohesion: 0.06
Nodes (39): assertQueryParams(), readOpenAPI(), TestOpenAPIIncludesCRUDContract(), TestAPIKeyTableName(), TestHashKey(), TestImportJobTableNameAndStatus(), TestOAuthBindingTableName(), TestEntityValues() (+31 more)

### Community 3 - "File Storage (Local/S3/OSS/MinIO)"
Cohesion: 0.06
Nodes (31): New(), newOSSStorage(), Context, Duration, ReadCloser, Reader, NewLocalStorage(), TestLocalStorageFilePersisted() (+23 more)

### Community 4 - "HTTP Upload & Fake Storage"
Cohesion: 0.09
Nodes (38): FileHeader, fakeStorage, UploadConfig, UploadHandler, UploadResponse, Context, HandlerFunc, Reader (+30 more)

### Community 5 - "Redis Pipeline (Fake)"
Cohesion: 0.08
Nodes (25): fakePipeline, fakeRedis, redisClientAdapter, RedisSessionStore, sessionPipeline, sessionRedis, SessionStore, BoolCmd (+17 more)

### Community 6 - "Database Migration Engine"
Cohesion: 0.09
Nodes (42): AutoMigrate(), findUp(), DB, DBConfig, isDir(), Migrate(), MigrateWithRetry(), MigrationDir() (+34 more)

### Community 7 - "Role Module"
Cohesion: 0.09
Nodes (19): fakeRoleRepository, Role, RoleRepository, rolePermission, fakeRoleRepository, NewRoleService(), Context, roleAppCode() (+11 more)

### Community 8 - "WebSocket Core"
Cohesion: 0.16
Nodes (35): NewMessage(), connID(), Conn, Duration, Mutex, Server, Time, mustDecodeTitle() (+27 more)

### Community 9 - "Outbox User Repository"
Cohesion: 0.10
Nodes (15): fakeOutboxUserRepo, recordingOutboxStore, NewUserService(), appCode(), createOutboxUserService(), fakeUserRepository, Context, TestCreateWritesOutbox() (+7 more)

### Community 10 - "User MySQL Integration Tests"
Cohesion: 0.12
Nodes (28): TestMysqlRepositoryMySQLIntegration(), NewMysqlRepository(), DB, newTestUser(), newUserTestDB(), TestMysqlRepositoryCreateAndFindByID(), TestMysqlRepositoryFindByEmailHash(), TestMysqlRepositoryFindByPhoneHash() (+20 more)

### Community 11 - "Specify Bash Scripts"
Cohesion: 0.08
Nodes (17): check-prerequisites.sh script, check_dir(), check_file(), get_feature_paths(), get_repo_root(), has_jq(), _persist_feature_json(), resolve_specify_init_dir() (+9 more)

### Community 12 - "OAuth Provider Tests"
Cohesion: 0.24
Nodes (32): oauthStateKey(), dupSubjectProviders(), githubProviders(), Client, DB, Miniredis, newRedisClient(), newRepoResult() (+24 more)

### Community 13 - "Helm Chart Templates"
Cohesion: 0.12
Nodes (34): Jimu Helm Chart, Helm ConfigMap Template, Helm Deployment Template, Helm HorizontalPodAutoscaler Template, Helm Ingress Template, Helm NetworkPolicy Template, Helm PodDisruptionBudget Template, Helm PrometheusRule Template (+26 more)

### Community 14 - "Gorm Logger"
Cohesion: 0.11
Nodes (26): gormLogger, Interface, Context, Duration, Time, isSensitiveField(), NewGormLogger(), sanitizeArgs() (+18 more)

### Community 15 - "Outbox MQ Publisher"
Cohesion: 0.11
Nodes (19): Context, Queue, RawMessage, NewMQPublisher(), TestMQPublisher_Publish(), TestMQPublisher_PublishError(), traceFromMetadata(), Context (+11 more)

### Community 16 - "Queue Worker Pool"
Cohesion: 0.14
Nodes (17): NewWorkerPool(), RegisterWorker(), fakeStore(), Context, Duration, MySQLStore, newFakeJobRepo(), TestWorkerPoolConsumeJob() (+9 more)

### Community 17 - "OpenTelemetry Tracing"
Cohesion: 0.11
Nodes (19): CancelFunc, ContextWithTrace(), DefaultTracingConfig(), Context, TracerProvider, InitTracing(), ShutdownTracing(), TraceFromContext() (+11 more)

### Community 18 - "E2E Test Infrastructure"
Cohesion: 0.18
Nodes (24): Context, apiResp, testAppDB, Engine, Event, doJSON(), DB, T (+16 more)

### Community 19 - "API Key Repo (Infra)"
Cohesion: 0.11
Nodes (9): fakeAPIKeyRepo, fakeDeadLetterRepo, fakeEventBus, fakeJobRepo, testRole, testUserRole, APIKey, Context (+1 more)

### Community 20 - "Admin API Key Service"
Cohesion: 0.12
Nodes (18): AdminAPIKeyService, CreateKeyInput, APIKey, APIKeyRepository, APIKey, Context, NewAdminAPIKeyService(), TestAdminAPIKeyServiceCreateKey() (+10 more)

### Community 21 - "Admin Config Service"
Cohesion: 0.14
Nodes (20): AdminConfigService, Client, Context, EventBus, NewAdminConfigService(), Miniredis, newConfigTestService(), TestAdminConfigServiceConfigKey() (+12 more)

### Community 22 - "Auth Module Bootstrap"
Cohesion: 0.17
Nodes (22): Module, AuthConfig, RouterGroup, Service, RegisterAuthRoutes(), RegisterCaptchaRoute(), Engine, newRouterLimiter() (+14 more)

### Community 23 - "Data Import Service"
Cohesion: 0.14
Nodes (19): ImportService, ImportJobRepository, UserRepository, DB, newSqliteDB(), Context, DB, Format (+11 more)

### Community 24 - "Admin Monitoring & Status"
Cohesion: 0.13
Nodes (17): AdminMonitoringService, HealthStatus, MemoryStats, SystemStatus, AdminMonitoringHandler, Client, Context, Time (+9 more)

### Community 25 - "Role Application Service"
Cohesion: 0.14
Nodes (15): AssignPermissionsRequest, CreateRoleRequest, RoleResponse, RoleService, UpdateRoleRequest, PermissionResponse, Time, TestRoleResponseUsesDTO() (+7 more)

### Community 26 - "API Key Repo (App)"
Cohesion: 0.14
Nodes (6): fakeAPIKeyRepo, fakeEventBus, APIKey, fakeUserRepository, Context, Mutex

### Community 27 - "Admin HTTP Handlers"
Cohesion: 0.13
Nodes (10): AdminConfigHandler, AdminTaskHandler, AdminWSHandler, Context, Context, Context, NewAdminTaskHandler(), Context (+2 more)

### Community 28 - "Configuration Types"
Cohesion: 0.14
Nodes (25): AuditConfig, CacheConfig, CaptchaResult, EmailConfig, GRPCConfig, HTTPClientConfig, HTTPConfig, IDConfig (+17 more)

### Community 29 - "User Domain & Repository"
Cohesion: 0.14
Nodes (7): User, fakeUserRepository, DeletedAt, Time, Context, DB, mysqlRepository

### Community 30 - "Admin Job Handlers"
Cohesion: 0.17
Nodes (22): fakeUserRepository, NewAdminJobHandler(), TestAdminJobHandlerGet(), TestAdminJobHandlerList(), TestAdminJobHandlerListDeadLetters(), TestAdminJobHandlerResolveDeadLetter(), TestAdminJobHandlerRetry(), TestAdminJobHandlerSubmit() (+14 more)

### Community 31 - "Feature Flag Manager"
Cohesion: 0.14
Nodes (16): contextKey, Flag, Manager, AdminFeatureHandler, NewAdminFeatureHandler(), TestAdminFeatureHandlerList(), TestAdminFeatureHandlerUpdate(), FromContext() (+8 more)

### Community 32 - "Admin Import Handler"
Cohesion: 0.15
Nodes (18): AdminImportHandler, TestAdminAuditHandlerList(), DB, newSqliteDB(), bindImportFile(), Buffer, Context, Format (+10 more)

### Community 33 - "CSRF Middleware"
Cohesion: 0.18
Nodes (23): CSRF(), DefaultCSRFConfig(), generateToken(), Context, HandlerFunc, isSafeMethod(), setCSRFCookie(), csrfRouter() (+15 more)

### Community 34 - "HTTP Middleware (CORS/Logger/RequestID)"
Cohesion: 0.15
Nodes (23): CORSMiddleware(), DefaultLogConfig(), HandlerFunc, isAllowedOrigin(), Logger(), Recovery(), RequestID(), shouldLogBody() (+15 more)

### Community 35 - "Specification Tasks (v001)"
Cohesion: 0.08
Nodes (24): Dependencies & Execution Order, Format: `[ID] [P?] [Story] Description`, Implementation for User Story 1, Implementation for User Story 2, Implementation for User Story 3, Implementation for User Story 4, Implementation Strategy, Incremental Delivery (+16 more)

### Community 36 - "Docs & CI Concepts"
Cohesion: 0.09
Nodes (24): Agent Collaboration Rules, graphify Knowledge Graph, Simple-First Principle, Single-Tenant Design Boundary, CI Performance Regression Gate, Coverage Threshold (70%), CI Lint Job (gofmt + vet + golangci-lint), CI Pipeline (GitHub Actions) (+16 more)

### Community 37 - "OAuth Binding Repository"
Cohesion: 0.12
Nodes (10): fakeBindingRepo, OAuthBinding, fakeBindingRepo, fakeProvider, fakeSessionStore, Time, Context, Context (+2 more)

### Community 38 - "Redis Queue Implementation"
Cohesion: 0.17
Nodes (10): Context, Duration, Client, Context, Duration, NewRedisQueue(), errorQueue, fakeQueue (+2 more)

### Community 39 - "DB Cleanup Service"
Cohesion: 0.15
Nodes (18): CleanupConfig, cleanupModel, CleanupResult, CleanupService, CleanupTable, noTableNameModel, DefaultCleanupConfig(), Context (+10 more)

### Community 40 - "Code Generator (Module Scaffold)"
Cohesion: 0.21
Nodes (21): targetFile, templateData, camel(), GenerateModule(), GenerateModuleAt(), lowerCamel(), mkdirAllTracked(), nextMigrationNumber() (+13 more)

### Community 41 - "HTTP Server Lifecycle"
Cohesion: 0.13
Nodes (20): HandlerFunc, Metrics(), ConfigureTrustedProxies(), formatAddr(), Engine, SetupRouter(), freeAddr(), TestConfigureTrustedProxies() (+12 more)

### Community 42 - "User Rate Limit Middleware"
Cohesion: 0.19
Nodes (19): defaultKeyFunc(), Client, Context, Duration, HandlerFunc, Time, NewUserRateLimiter(), Engine (+11 more)

### Community 43 - "Auth Application Service"
Cohesion: 0.20
Nodes (10): AuthService, AuthServiceInterface, TokenPair, ErrAccountLocked(), Context, Duration, invalidCredentials(), normalizeUsername() (+2 more)

### Community 44 - "API Key Auth Verifier"
Cohesion: 0.16
Nodes (14): APIKey, APIKeyContextKey, APIKeyStore, APIKeyVerifier, dbAPIKeyStore, APIKeyFromContext(), ContextWithAPIKey(), Context (+6 more)

### Community 45 - "JWT Token Service"
Cohesion: 0.16
Nodes (12): Claims, JWT, FuzzJWTParse(), F, Duration, New(), NewWithRotation(), TestJWTPopulatesTypedClaims() (+4 more)

### Community 46 - "gRPC Server"
Cohesion: 0.16
Nodes (15): ClientConn, Config, Server, Context, Listener, New(), dial(), Server (+7 more)

### Community 47 - "HTTP Client (Retry/CircuitBreaker)"
Cohesion: 0.16
Nodes (15): circuit, circuitState, Client, Config, hostRateLimiter, Context, Duration, Limit (+7 more)

### Community 48 - "OAuth HTTP Handler"
Cohesion: 0.23
Nodes (17): OAuthHandler, NewOAuthHandler(), assertCode(), doRequest(), githubProvider(), Engine, ResponseRecorder, newTestHandler() (+9 more)

### Community 49 - "MySQL Connection & Tests"
Cohesion: 0.17
Nodes (19): TestCleanupService_RunError(), New(), DB, Sqlmock, newMockGormDB(), TestConfigurePool(), TestConnectWithRetry_ExhaustsRetries(), TestConnectWithRetry_WithLogger() (+11 more)

### Community 50 - "API Signature Middleware"
Cohesion: 0.19
Nodes (20): buildSignString(), DefaultSignatureConfig(), Context, Duration, HandlerFunc, hmacSign(), Signature(), SignRequest() (+12 more)

### Community 51 - "Cron Scheduler"
Cohesion: 0.17
Nodes (8): Cron, EntryID, Context, RWMutex, Time, CronScheduler, JobInfo, Store

### Community 52 - "Global Rate Limit (Token Bucket)"
Cohesion: 0.20
Nodes (16): GlobalRateLimit(), HandlerFunc, Limit, RWMutex, NewRateLimiter(), Engine, Request, ratelimitRequest() (+8 more)

### Community 53 - "Webhook Notification"
Cohesion: 0.16
Nodes (15): BenchmarkWebhookSend(), B, Channel, Client, Context, NewWebhook(), signPayload(), TestWebhookNoSignatureWhenNoSecret() (+7 more)

### Community 54 - "WebSocket Hub & Handler"
Cohesion: 0.18
Nodes (6): mustEncode(), RawMessage, Time, broadcastMsg, ClientHub, WSMessage

### Community 55 - "User Application Service"
Cohesion: 0.21
Nodes (11): BatchDeleteRequest, BatchResult, CreateUserRequest, UpdateUserRequest, UserResponse, UserService, Time, ToUserResponse() (+3 more)

### Community 56 - "Redis Cache (Cache-Aside)"
Cohesion: 0.23
Nodes (7): Cache, RedisCache, Client, Context, Duration, NewRedisCache(), randomToken()

### Community 57 - "Session Store & Notifier"
Cohesion: 0.17
Nodes (5): handlerNotifier, handlerSessionStore, handlerUserRepo, Context, Duration

### Community 58 - "Role HTTP Handler"
Cohesion: 0.17
Nodes (9): RoleHandler, Context, NewRoleHandler(), TestRoleCreateReturnsCreated(), TestRoleDeleteReturnsNoContent(), DB, EventBus, New() (+1 more)

### Community 59 - "WebSocket Notification Channel"
Cohesion: 0.17
Nodes (9): Channel, Context, RWMutex, NewWebSocket(), Connection, Hub, Registration, WebSocket (+1 more)

### Community 60 - "WebSocket Presence"
Cohesion: 0.16
Nodes (16): NewPresenceManager(), TestPresenceIsOnline(), TestPresenceIsTyping(), TestPresenceManagerAllPresences(), TestPresenceManagerHeartbeat(), TestPresenceManagerHeartbeatRestoresOnlineStatus(), TestPresenceManagerOfflineMissing(), TestPresenceManagerOnlineCountFiltersOffline() (+8 more)

### Community 61 - "Admin User Service"
Cohesion: 0.18
Nodes (10): AdminCreateUserRequest, AdminUpdateUserRequest, AdminUser, AdminUserService, ListUserFilter, userRole, Context, DB (+2 more)

### Community 62 - "Audit Repository (Fake)"
Cohesion: 0.19
Nodes (8): fakeAuditRepository, AuditLog, fakeAuditRepository, fakeQueue, Context, Time, Context, TestAuditMiddlewareAllowsAnonymousRequest()

### Community 63 - "Import Job Repository"
Cohesion: 0.17
Nodes (8): fakeImportJobRepo, ImportJob, mysqlImportJobRepository, fakeImportJobRepo, Time, Context, DB, NewMysqlImportJobRepository()

### Community 64 - "Auth HTTP Handler (Login/Register)"
Cohesion: 0.27
Nodes (9): CaptchaConfig, AuthHandler, authContext(), Context, Duration, Service, NewAuthHandler(), normalizeUsername() (+1 more)

### Community 65 - "Redis Distributed Lock"
Cohesion: 0.18
Nodes (16): RedisConfig, Client, newLock(), TestLock_AcquireRelease(), TestLock_ConcurrentAcquire(), TestLock_Extend(), TestLock_ReleaseOnlyOwnToken(), TestLock_WithLock() (+8 more)

### Community 66 - "User HTTP Handler"
Cohesion: 0.18
Nodes (8): UserHandler, Context, Client, Config, DB, EventBus, New(), Module

### Community 67 - "Health Check (livez/readyz)"
Cohesion: 0.19
Nodes (15): Client, Context, DB, Duration, ResponseWriter, NewReadiness(), NewRedisChecker(), NewSQLChecker() (+7 more)

### Community 68 - "Scheduler Store & Observability"
Cohesion: 0.23
Nodes (14): NewMemoryStore(), TestObserveCountsPanicAsFailure(), TestObserveEmitsSuccessMetrics(), TestSetEnabledNotFound(), TestSetEnabledToggle(), TestTriggerJobNotFound(), TestTriggerJobRunsCommand(), NewWithStore() (+6 more)

### Community 69 - "Project Constitution"
Cohesion: 0.11
Nodes (18): Principle II: API Stability & SemVer, Principle I: Business-Agnostic, Principle III: Composable Modules, Jimu Constitution, Principle VI: Documentation & Examples, Governance & Revision Process, Principle IV: Simplicity & Minimal Dependencies, Principle V: Verification & Quality (+10 more)

### Community 70 - "OAuth Application Service"
Cohesion: 0.15
Nodes (12): fakeProvider, Duration, refreshTTL(), fakeSessionStore, sessionRecord, Context, Duration, UserInfo (+4 more)

### Community 71 - "User Fake Repository"
Cohesion: 0.20
Nodes (5): fakeUserRepo, fakeSessionStore, sessionRecord, Context, Duration

### Community 72 - "Rate Limiter (Auth)"
Cohesion: 0.18
Nodes (13): Limiter, Context, Duration, Scripter, limitKey(), NewLimiter(), newTestLimiter(), TestLimiterAllowsOnlyConfiguredWindow() (+5 more)

### Community 73 - "OAuth Module Config"
Cohesion: 0.17
Nodes (13): OAuthConfig, OAuthProviderConfig, buildProviders(), Client, DB, EventBus, New(), newTestModule() (+5 more)

### Community 74 - "AES-256-GCM Encryption"
Cohesion: 0.20
Nodes (13): Cipher, New(), TestBlindIndexDeterministicAndDistinct(), TestBlindIndexEmpty(), TestBlindIndexWithoutKey(), TestDecryptTamperedFails(), TestDecryptWrongKeyFails(), TestEmptyValuePassthrough() (+5 more)

### Community 75 - "Auth Benchmark Tests"
Cohesion: 0.31
Nodes (16): BenchmarkLogin(), benchUser(), B, TestForgotPasswordNotConfigured(), NewAuthService(), appCode(), newFakeSessionStore(), newTestService() (+8 more)

### Community 76 - "User Handler Tests"
Cohesion: 0.20
Nodes (7): NewUserHandler(), Context, fakeUserRepository, TestUserCreateReturnsCreatedDTO(), TestUserDeleteReturnsNoContent(), TestUserListRejectsInvalidPaginationContract(), TestUserListUsesDefaultPaginationBeforeService()

### Community 77 - "RabbitMQ Queue"
Cohesion: 0.20
Nodes (12): Context, Delivery, newTestRabbitQueue(), TestRabbitMQQueue_ConsumeTimeout(), TestRabbitMQQueue_ConsumeUnmarshalError(), TestRabbitMQQueue_NackRequeues(), TestRabbitMQQueue_SubmitConsumeAck(), TestRabbitMQQueueImplementsInterfaces() (+4 more)

### Community 78 - "Email Notification"
Cohesion: 0.18
Nodes (9): Context, buildEmailHeaders(), Context, Dispatcher, Context, Channel, Message, fakeKafkaReader (+1 more)

### Community 79 - "Notification Dispatcher"
Cohesion: 0.15
Nodes (10): Channel, Channel, Channel, Context, dispatcher, RWMutex, NewDispatcher(), TestDispatcherDispatch() (+2 more)

### Community 80 - "OAuth Login Flow"
Cohesion: 0.21
Nodes (9): OAuthService, BindingRepository, Client, Context, DB, UserInfo, NewOAuthService(), Provider (+1 more)

### Community 81 - "Audit Worker"
Cohesion: 0.20
Nodes (11): Worker, AuditRepository, Context, RWMutex, NewWorker(), Duration, testWorker(), TestWorkerFlushesConfiguredBatch() (+3 more)

### Community 82 - "Casbin RBAC Authorization"
Cohesion: 0.18
Nodes (12): AuthorizationStore, DBAuthorizationStore, Enforcer, HandlerFunc, ProtectedMiddleware(), HandlerFunc, AuthorizationMiddleware(), Context (+4 more)

### Community 83 - "Database Config & Connection"
Cohesion: 0.24
Nodes (13): DBConfig, configurePool(), ConnectWithRetry(), dsn(), Context, DB, openByDriver(), openMySQL() (+5 more)

### Community 84 - "Audit HTTP Handler"
Cohesion: 0.21
Nodes (12): AuditHandler, NewAuditService(), auditAppCode(), TestAuditServiceGetMapsNotFound(), TestAuditServiceListReturnsDTOAndPassesPagination(), Context, NewAuditHandler(), TestAuditGetInvalidIDReturnsStableBadRequest() (+4 more)

### Community 85 - "Email SMTP Service"
Cohesion: 0.18
Nodes (12): Channel, NewEmail(), Conn, Listener, newFakeSMTPServer(), TestBuildEmailHeaders(), TestEmailSendEmptyRecipient(), TestEmailSendMissingConfig() (+4 more)

### Community 86 - "Scheduler Memory Store"
Cohesion: 0.17
Nodes (8): Context, RWMutex, Context, Time, failStore, JobDef, MemoryStore, Store

### Community 87 - "WebSocket Channels"
Cohesion: 0.16
Nodes (6): RWMutex, NewChannel(), TestChannelSubscribeUnsubscribe(), TestNewChannel(), Channel, ChannelManager

### Community 88 - "Admin Application Service"
Cohesion: 0.16
Nodes (10): fakeRouter, Service, Client, Time, NewService(), TestService(), Engine, RouterGroup (+2 more)

### Community 89 - "Zap Logger"
Cohesion: 0.17
Nodes (11): AtomicLevel, Context, LogConfig, New(), TestFileOutput(), TestNewConsoleStdout(), TestNewJSONLevels(), TestSetLevel() (+3 more)

### Community 90 - "Group 90"
Cohesion: 0.22
Nodes (9): RedisStore, Service, Client, Context, Duration, NewRedisStore(), NewService(), TestGenerate() (+1 more)

### Community 91 - "Group 91"
Cohesion: 0.17
Nodes (7): DeadLetter, mysqlDeadLetterRepository, Context, DB, NewMysqlDeadLetterRepository(), Time, fakeDeadRepo

### Community 92 - "Group 92"
Cohesion: 0.16
Nodes (11): CSVExporter, ExcelExporter, Context, Writer, NewCSVExporter(), Context, Writer, NewExcelExporter() (+3 more)

### Community 93 - "Group 93"
Cohesion: 0.19
Nodes (8): ExcelImporter, ImportError, ImportResult, Context, Reader, NewExcelImporter(), Time, NewImportResult()

### Community 94 - "Group 94"
Cohesion: 0.24
Nodes (14): MySQLBindingRepository, DB, NewMySQLBindingRepository(), bindingRows(), DB, Sqlmock, newMockGormDB(), TestCreateError() (+6 more)

### Community 95 - "Group 95"
Cohesion: 0.17
Nodes (10): Buffer, ResponseWriter, newResponseBodyWriter(), sanitizeBody(), sanitizeJSONField(), TestResponseBodyWriterCapturesAndTruncates(), TestResponseBodyWriterWriteHeaderPassthrough(), TestSanitizeBody() (+2 more)

### Community 96 - "Group 96"
Cohesion: 0.17
Nodes (12): HandlerFunc, ResponseWriter, Writer, GzipCompression(), isAlreadyCompressed(), Engine, gzipRouter(), TestGzipCompressionEncodesBody() (+4 more)

### Community 97 - "Group 97"
Cohesion: 0.22
Nodes (8): Channel, Context, NewSMS(), TestSMSAliyunReject(), TestSMSAliyunSend(), TestSMSUnknownProvider(), SMS, SMSConfig

### Community 98 - "Group 98"
Cohesion: 0.14
Nodes (14): BuildRoomChannel(), BuildUserChannel(), TestBuildRoomChannel(), TestBuildUserChannel(), TestNewMessage(), TestNewMessageMarshalError(), TestWSMessageDecodePayload(), TestWSMessageDecodePayloadError() (+6 more)

### Community 99 - "Group 99"
Cohesion: 0.19
Nodes (4): RWMutex, Time, Presence, PresenceManager

### Community 100 - "Group 100"
Cohesion: 0.19
Nodes (12): New(), TestAppError(), TestAppErrorWithCause(), TestFailDoesNotLeakInfrastructureDetails(), TestOKIncludesStableEnvelopeAndRequestID(), TestPageIncludesStableEnvelopeAndPagination(), TestCreatedUsesStandardEnvelope(), TestFailHidesInternalCauseAndIncludesRequestID() (+4 more)

### Community 101 - "Group 101"
Cohesion: 0.14
Nodes (5): MessageState, SizeCache, UnknownFields, GetUserRequest, ListUsersRequest

### Community 102 - "Group 102"
Cohesion: 0.21
Nodes (9): Application, Component, forwardError(), Context, Duration, NewApplication(), TestApplicationReturnsComponentRuntimeError(), TestApplicationRollsBackStartedComponents() (+1 more)

### Community 103 - "Group 103"
Cohesion: 0.23
Nodes (8): AuditLogResponse, AuditService, Change, Time, ToAuditLogResponse(), ToAuditLogResponses(), Context, serializeChanges()

### Community 104 - "Group 104"
Cohesion: 0.23
Nodes (13): fakeDispatcher, fakeSessionStore, Client, Miniredis, Mutex, newResetRedis(), newResetService(), TestForgotPasswordHidesMissingUser() (+5 more)

### Community 105 - "Group 105"
Cohesion: 0.18
Nodes (8): JobHistory, mysqlJobHistoryRepository, Context, DB, NewMysqlJobHistoryRepository(), Time, Mutex, fakeHistoryRepo

### Community 106 - "Group 106"
Cohesion: 0.19
Nodes (9): EventBus, Handler, RWMutex, New(), TestEventBus_Clear(), TestEventBus_HandlerPanicRecovered(), TestEventBus_MultipleHandlers(), TestEventBus_PublishAsync() (+1 more)

### Community 107 - "Group 107"
Cohesion: 0.27
Nodes (6): AdminUserHandler, Context, Context, paginationFromQuery(), Context, Fail()

### Community 108 - "Group 108"
Cohesion: 0.22
Nodes (12): RouterGroup, RegisterRoleRoutes(), Context, HandlerFunc, localeOf(), TestValidateJSONTranslatesFieldErrors(), translateValidationDetails(), translateValidationMessage() (+4 more)

### Community 109 - "Group 109"
Cohesion: 0.17
Nodes (10): Context, DB, DeletedAt, Time, NewMySQLStore(), DB, testDB(), TestMySQLStore() (+2 more)

### Community 110 - "Group 110"
Cohesion: 0.24
Nodes (7): Conn, Context, HandlerFunc, RWMutex, Time, WSHandler(), Client

### Community 111 - "Group 111"
Cohesion: 0.16
Nodes (5): fakeAuthzModule, fakeBusinessModule, EventBus, HandlerFunc, TestBusinessRoutesRequireProtectedMiddleware()

### Community 112 - "Group 112"
Cohesion: 0.23
Nodes (7): Job, JobRepository, mysqlJobRepository, Context, DB, NewMysqlJobRepository(), Time

### Community 113 - "Group 113"
Cohesion: 0.20
Nodes (9): userInfoService, DB, DB, NewUserInfoGRPCService(), RegisterUserInfoServiceServer(), ServiceRegistrar, UnimplementedUserInfoServiceServer, UnsafeUserInfoServiceServer (+1 more)

### Community 114 - "Group 114"
Cohesion: 0.23
Nodes (8): mysqlAuditRepository, AdminAuditHandler, DB, NewAdminAuditHandler(), deserializeChanges(), Context, DB, NewMysqlAuditRepository()

### Community 115 - "Group 115"
Cohesion: 0.23
Nodes (6): PermissionHandler, Context, DB, EventBus, New(), Module

### Community 116 - "Group 116"
Cohesion: 0.22
Nodes (13): Config, TestDBPasswordOverride(), TestJWTSecretOverride(), TestLoad(), TestStorageConfigFieldMapping(), TestValidateHTTPMode(), TestValidateLogFormat(), TestValidateLogLevel() (+5 more)

### Community 117 - "Group 117"
Cohesion: 0.33
Nodes (12): createKey(), APIKey, DB, newTestDB(), TestDBAPIKeyStore_GetByKeyHash(), TestDBAPIKeyStore_GetByKeyHashNotFound(), TestDBAPIKeyStore_UpdateLastUsed(), TestVerifyWithDBStore() (+4 more)

### Community 118 - "Group 118"
Cohesion: 0.26
Nodes (7): Context, Delivery, Duration, Mutex, NewRabbitMQQueue(), RabbitMQChannel, RabbitMQQueue

### Community 119 - "Group 119"
Cohesion: 0.33
Nodes (8): generateToken(), Client, Context, Duration, Time, NewLock(), AcquireResult, Lock

### Community 120 - "Group 120"
Cohesion: 0.22
Nodes (5): fakeStorage, Context, Duration, ReadCloser, Reader

### Community 121 - "Group 121"
Cohesion: 0.15
Nodes (7): ComponentProvider, ErrorSource, EventBus, HTTPMiddlewareProvider, JobRegistry, Module, ProtectedHTTPMiddlewareProvider

### Community 122 - "Group 122"
Cohesion: 0.22
Nodes (10): Container, Client, Config, Context, DB, EventBus, Server, Service (+2 more)

### Community 123 - "Group 123"
Cohesion: 0.29
Nodes (5): mysqlAPIKeyRepository, APIKey, Context, DB, NewMysqlAPIKeyRepository()

### Community 124 - "Group 124"
Cohesion: 0.38
Nodes (12): NewServer(), New(), TestCircuitHalfOpenOnlyAllowsProbe(), TestCircuitOpensAfterConsecutiveFailures(), TestCircuitRecoversAfterCooldown(), TestDoCancelsDuringRetryWait(), TestDoInjectsTraceParent(), TestDoNoRetryOn4xx() (+4 more)

### Community 125 - "Group 125"
Cohesion: 0.19
Nodes (9): Client, Queue, New(), TestNew_InvalidType(), TestNew_Redis(), Config, KafkaConfig, RabbitMQConfig (+1 more)

### Community 126 - "Group 126"
Cohesion: 0.29
Nodes (6): Context, Duration, NewKafkaQueue(), KafkaMessageReader, KafkaMessageWriter, KafkaQueue

### Community 127 - "Group 127"
Cohesion: 0.26
Nodes (12): NewChannelManager(), TestChannelManagerGetChannelMissing(), TestChannelManagerGetSubscribers(), TestChannelManagerPreCreatedBroadcast(), TestChannelManagerSubscribeCreatesWithType(), TestChannelManagerSubscribeExisting(), TestChannelManagerUnsubscribe(), TestChannelManagerUnsubscribeAll() (+4 more)

### Community 128 - "Group 128"
Cohesion: 0.15
Nodes (12): Quickstart: 框架能力端到端验证, V1 认证闭环（FR-008/009/010/011）, V2 运维与观测（FR-005/006/029/031）, V3 模块扩展（FR-032/033/契约）, V4 可选能力（FR-012/013/022/024/025/026/027）, V5 安全与限流（FR-014/015/016/017）, 前置条件, 环境搭建 (+4 more)

### Community 129 - "Group 129"
Cohesion: 0.23
Nodes (6): Module, Client, DB, EventBus, HandlerFunc, Service

### Community 130 - "Group 130"
Cohesion: 0.26
Nodes (8): ResetStore, Client, Context, Duration, NewResetStore(), newHandlerService(), TestForgotPasswordHandler(), TestResetPasswordHandlerInvalidCode()

### Community 131 - "Group 131"
Cohesion: 0.17
Nodes (5): Module, RouterGroup, RegisterAuditRoutes(), EventBus, HandlerFunc

### Community 132 - "Group 132"
Cohesion: 0.33
Nodes (8): LockoutConfig, LoginFailureTracker, DefaultLockoutConfig(), Client, Context, Duration, lockKey(), NewLoginFailureTracker()

### Community 133 - "Group 133"
Cohesion: 0.24
Nodes (9): contact, contactPtr, DB, newEncryptionTestDB(), TestEncryptionHookBatchCreate(), TestEncryptionHookEmptySourceStoresNullAndCoexists(), TestEncryptionHookEncryptsOnWriteDecryptsOnRead(), TestEncryptionHookNoKeyStoresPlaintextButHashes() (+1 more)

### Community 134 - "Group 134"
Cohesion: 0.18
Nodes (7): HandlerFunc, Locale(), ParseAcceptLanguage(), TestParseAcceptLanguage(), TestTDefaultsToChinese(), TestTfFormatsArgs(), Tf()

### Community 135 - "Group 135"
Cohesion: 0.26
Nodes (6): Context, DB, NewMySQLStore(), TestMySQLStore_AddWithinTx(), TestMySQLStore_AddWithNilTx(), MySQLStore

### Community 136 - "Group 136"
Cohesion: 0.18
Nodes (5): Router, RouterGroup, RegisterPermissionRoutes(), RouterGroup, RegisterUserRoutes()

### Community 137 - "Group 137"
Cohesion: 0.29
Nodes (9): testRole, NewAdminUserService(), TestAdminUserServiceAssignRoles(), TestAdminUserServiceAssignRolesRoleQueryError(), TestAdminUserServiceCreateUser(), TestAdminUserServiceDisableUser(), TestAdminUserServiceGetUser(), TestAdminUserServiceListUsers() (+1 more)

### Community 138 - "Group 138"
Cohesion: 0.24
Nodes (6): fakeAuthorizationStore, Policy, fakeAuthzStore, Context, TestProtectedMiddlewareRequiresAccessToken(), Context

### Community 139 - "Group 139"
Cohesion: 0.35
Nodes (5): DeadLetterRepository, JobHistoryRepository, Context, NewMySQLStore(), MySQLStore

### Community 140 - "Group 140"
Cohesion: 0.24
Nodes (8): FieldLevel, FuzzValidateRules(), F, Validate(), validateIDCard(), validateMobile(), validatePassword(), validateUsername()

### Community 141 - "Group 141"
Cohesion: 0.20
Nodes (8): Handler, passingChecker, Server, HealthRouter(), NewManagementServer(), Context, TestManagementRouterExposure(), TestNewManagementServer()

### Community 142 - "Group 142"
Cohesion: 0.22
Nodes (7): CSVImporter, TestCSVExporterRoundTrip(), Context, Reader, NewCSVImporter(), FuzzCSVImporterParse(), F

### Community 143 - "Group 143"
Cohesion: 0.36
Nodes (10): Int32, Client, Engine, idempotencyRouter(), newRedisForTest(), TestIdempotencyCachedResponseSerializes(), TestIdempotencyCachesAndReplays(), TestIdempotencyDoesNotCacheFailure() (+2 more)

### Community 144 - "Group 144"
Cohesion: 0.33
Nodes (5): routerLimiterRedis, BoolSliceCmd, Cmd, Context, StringCmd

### Community 145 - "Group 145"
Cohesion: 0.27
Nodes (9): DB, Enforcer, NewEnforcer(), NewPathEnforcer(), Enforcer, TestAuthorizationMiddlewareAllowsRolePolicy(), TestAuthorizationMiddlewareHidesStoreErrors(), TestAuthorizationMiddlewareRejectsMissingRole() (+1 more)

### Community 146 - "Group 146"
Cohesion: 0.27
Nodes (9): AdminAuth(), HandlerFunc, adminAuthRouter(), Engine, TestAdminAuthAllowsAdminRole(), TestAdminAuthAllowsEnglishAdminRole(), TestAdminAuthRejectsNonAdminRole(), TestAdminAuthRequiresAuthentication() (+1 more)

### Community 147 - "Group 147"
Cohesion: 0.29
Nodes (6): Channel, Context, NewLogChannel(), TestLogChannelSendBatchNoError(), TestLogChannelSendNoError(), LogChannel

### Community 148 - "Group 148"
Cohesion: 0.40
Nodes (10): Client, Server, mockClient(), mockTokenServer(), TestGitHubProviderExchange(), TestGoogleProviderExchange(), TestGoogleProviderExchangeBadUserInfo(), TestGoogleProviderExchangeTokenError() (+2 more)

### Community 149 - "Group 149"
Cohesion: 0.24
Nodes (7): Client, Config, Context, UserInfo, NewGitHubProvider(), GitHubConfig, GitHubProvider

### Community 150 - "Group 150"
Cohesion: 0.24
Nodes (7): Client, Config, Context, UserInfo, NewGoogleProvider(), GoogleConfig, GoogleProvider

### Community 151 - "Group 151"
Cohesion: 0.24
Nodes (7): Client, Config, Context, UserInfo, NewWeChatProvider(), WeChatConfig, WeChatProvider

### Community 152 - "Group 152"
Cohesion: 0.36
Nodes (10): Created(), FailWithDetails(), Context, localeFrom(), NoContent(), Page(), requestID(), StatusForCode() (+2 more)

### Community 153 - "Group 153"
Cohesion: 0.29
Nodes (5): AdminTaskService, TaskExecution, TaskInfo, Context, Time

### Community 154 - "Group 154"
Cohesion: 0.27
Nodes (9): SecurityConfig, DefaultSecurityConfig(), TestSecurityHeadersMiddleware(), TestSecurityHeadersNilValuesSkipped(), TestSecurityHeadersRespectsCustomValues(), Context, HandlerFunc, SecurityHeadersFromConfig() (+1 more)

### Community 155 - "Group 155"
Cohesion: 0.33
Nodes (10): Jimu API Swagger Definition, Audit Log Endpoints, Auth Flow Endpoints (login/refresh/logout/register), BearerAuth Security (API Key JWT), Captcha Endpoint, contract.ErrorResponse, OAuth Endpoints, contract.PageResponse (+2 more)

### Community 156 - "Group 156"
Cohesion: 0.33
Nodes (8): FieldRule, FieldType, ValidationRules, Validator, TestValidateUnique(), checkType(), Context, NewValidator()

### Community 158 - "Group 158"
Cohesion: 0.44
Nodes (8): bridgeFn(), fakeContainer(), newTestLogger(), TestBridgeWorkerConversionFailureErrors(), TestBridgeWorkerPublishesStrongTypeToBareTopic(), TestBridgeWorkerUnknownTypeErrors(), TestEventBusBridgePublishesToBareTopic(), TestRegisterOutboxWorkersRegistersAll()

### Community 159 - "Group 159"
Cohesion: 0.33
Nodes (5): Context, UserInfo, _UserInfoService_GetUser_Handler(), _UserInfoService_ListUsers_Handler(), UnaryServerInterceptor

### Community 160 - "Group 160"
Cohesion: 0.22
Nodes (6): Duration, HandlerFunc, TestTimeoutOnlyPropagatesContextDeadline(), Timeout(), RouterGroup, RegisterSwagger()

### Community 161 - "Group 161"
Cohesion: 0.33
Nodes (5): NewHub(), TestHubLifecycle(), TestWebSocketNotification(), waitHub(), mockConn

### Community 162 - "Group 162"
Cohesion: 0.22
Nodes (8): Complexity Tracking, Constitution Check, Documentation (this feature), Implementation Plan: Jimu 后端框架能力规格, Project Structure, Source Code (repository root), Summary, Technical Context

### Community 163 - "Group 163"
Cohesion: 0.39
Nodes (7): main(), run(), buildViper(), Load(), unmarshalConfig(), Watch(), Viper

### Community 164 - "Group 164"
Cohesion: 0.36
Nodes (5): AppError, ErrorInfo, AllErrorCodes(), As(), HTTPStatus()

### Community 165 - "Group 165"
Cohesion: 0.29
Nodes (6): PingServer, pingService, Context, Server, RegisterPingServer(), StringValue

### Community 166 - "Group 166"
Cohesion: 0.54
Nodes (5): Format, Importer, Registry, NewRegistry(), TestRegistryGetUnsupported()

### Community 168 - "Group 168"
Cohesion: 0.32
Nodes (5): Handler, Context, Service, NewHandler(), TestHandlerGetErrorCodes()

### Community 169 - "Group 169"
Cohesion: 0.43
Nodes (7): Queue, AuditMiddleware(), Context, HandlerFunc, isManagementPath(), optionalString(), optionalUint64()

### Community 170 - "Group 170"
Cohesion: 0.29
Nodes (3): file_proto_jimu_v1_userinfo_proto_init(), file_proto_jimu_v1_userinfo_proto_rawDescGZIP(), init()

### Community 171 - "Group 171"
Cohesion: 0.32
Nodes (6): Context, EventBus, RawMessage, NewEventBusPublisher(), EventBusPublisher, EventPayload

### Community 172 - "Group 172"
Cohesion: 0.39
Nodes (7): Client, newRedisTestQueue(), TestQueueContract_SubmitConsume(), TestRedisQueueAckRemovesFromProcessing(), TestRedisQueueImplementsInterfaces(), TestRedisQueueNackRequeues(), TestRedisQueueRequeueExpired()

### Community 173 - "Group 173"
Cohesion: 0.25
Nodes (7): api_keys, dead_letters, import_jobs, job_history, jobs, scheduled_jobs, user_oauth_bindings

### Community 174 - "Group 174"
Cohesion: 0.25
Nodes (7): api_keys, dead_letters, import_jobs, job_history, jobs, scheduled_jobs, user_oauth_bindings

### Community 176 - "Group 176"
Cohesion: 0.29
Nodes (4): fakeEventBus, TestModuleInitWSIdempotent(), TestModuleNameAndNew(), TestModuleWSHandler()

### Community 177 - "Group 177"
Cohesion: 0.52
Nodes (6): moduleLogger, registerRouter, Bootstrap(), registerEventBusBridge(), registerHTTP(), registerOutboxWorkers()

### Community 178 - "Group 178"
Cohesion: 0.43
Nodes (3): fakeBatchRepository, Context, Mutex

### Community 179 - "Group 179"
Cohesion: 0.48
Nodes (6): ClientConnInterface, newTestGRPCService(), TestUserInfoService_GetUser(), TestUserInfoService_ListUsers(), NewUserInfoServiceClient(), UserInfoServiceClient

### Community 180 - "Group 180"
Cohesion: 0.43
Nodes (7): Alert Rules (Jimu HTTP/Infra/Queue/Process), AlertManager Routing Config, Grafana Datasources (Prometheus+Loki), Prometheus Scrape Config, Promtail Log Collection Config, Docker Compose Observability Stack, Observability OTel+Prometheus (FR-029)

### Community 181 - "Group 181"
Cohesion: 0.62
Nodes (4): Exporter, Format, Registry, NewRegistry()

### Community 183 - "Group 183"
Cohesion: 0.57
Nodes (6): T, startTestGRPCServer(), TestGRPCHealthCheck(), TestGRPCPingEcho(), TestGRPCReflection(), Server

### Community 184 - "Group 184"
Cohesion: 0.62
Nodes (6): NewAdminTaskService(), newTestScheduler(), TestAdminTaskServiceGetHistory(), TestAdminTaskServiceListTasks(), TestAdminTaskServiceToggleTask(), TestAdminTaskServiceTriggerTask()

### Community 185 - "Group 185"
Cohesion: 0.43
Nodes (4): Duration, ExponentialBackoff, FixedRetry, RetryStrategy

### Community 186 - "Group 186"
Cohesion: 0.29
Nodes (6): Admin 模块, Auth 模块, HTTP API 契约, 中间件, 健康与观测（非业务）, 完整接口文档

### Community 187 - "Group 187"
Cohesion: 0.40
Nodes (4): TestMysqlDeadLetterRepositoryCRUD(), DB, newHistoryTestDB(), TestMysqlJobHistoryRepositoryCreateAndList()

### Community 188 - "Group 188"
Cohesion: 0.53
Nodes (5): DB, newRepoTestDB(), TestMysqlAPIKeyRepository(), TestMysqlImportJobRepository(), TestMysqlJobRepository()

### Community 189 - "Group 189"
Cohesion: 0.73
Nodes (5): NewAdminWSHandler(), newWSHandler(), TestAdminWSHandlerOnlineUsers(), TestAdminWSHandlerPresence(), TestAdminWSHandlerPush()

### Community 190 - "Group 190"
Cohesion: 0.33
Nodes (5): Client, Duration, HandlerFunc, IdempotencyMiddleware(), cachedResponse

### Community 191 - "Group 191"
Cohesion: 0.60
Nodes (5): gatherHTTPMetric(), Engine, setupMetricsEngine(), TestMetricsLabelsUseRouteTemplate(), TestMetricsRecordsUnmatchedPath()

### Community 192 - "Group 192"
Cohesion: 0.40
Nodes (5): TestSecurityAddVary(), addVary(), Context, HandlerFunc, Security()

### Community 193 - "Group 193"
Cohesion: 0.53
Nodes (5): Engine, securityRouter(), TestSecurityHandlesAllowedPreflight(), TestSecurityRejectsOversizedBody(), TestSecurityUsesOriginAllowList()

### Community 194 - "Group 194"
Cohesion: 0.47
Nodes (3): Channel, Context, mockNotification

### Community 195 - "Group 195"
Cohesion: 0.47
Nodes (4): Context, TestReadinessBoundsCheckerDuration(), TestReadinessStatus(), checkerFunc

### Community 196 - "Group 196"
Cohesion: 0.47
Nodes (4): CollectRuntime(), DB, NewDBCollector(), DBCollector

### Community 198 - "Group 198"
Cohesion: 0.33
Nodes (5): permissions, role_permissions, roles, user_roles, users

### Community 199 - "Group 199"
Cohesion: 0.33
Nodes (5): permissions, role_permissions, roles, user_roles, users

### Community 200 - "Group 200"
Cohesion: 0.33
Nodes (5): Content Quality, Feature Readiness, Notes, Requirement Completeness, Specification Quality Checklist: Jimu 后端框架能力规格

### Community 201 - "Group 201"
Cohesion: 0.73
Nodes (5): copyGoSum(), copyRootFile(), TestGeneratedModuleCompiles(), writeFileForTest(), writeStubPackages()

### Community 202 - "Group 202"
Cohesion: 0.40
Nodes (5): Release Workflow, Conventional Commits, Release Note Structure, Commit Style, Commit Messages in English

### Community 204 - "Group 204"
Cohesion: 0.50
Nodes (5): Development Config (app.yaml), Production Config (app.prod.yaml), Config Contract, Config Enum Validation (FR-002), File-Based Secret Injection (FR-003)

### Community 205 - "Group 205"
Cohesion: 0.40
Nodes (4): UserCreatedEvent, UserDeletedEvent, UserLoggedInEvent, UserUpdatedEvent

### Community 207 - "Group 207"
Cohesion: 0.40
Nodes (4): forgotPasswordRequest, loginRequest, refreshRequest, resetPasswordRequest

### Community 208 - "Group 208"
Cohesion: 0.40
Nodes (3): AdminAuthMiddleware(), HandlerFunc, TestAdminAuthMiddleware()

### Community 209 - "Group 209"
Cohesion: 0.60
Nodes (4): csvReader(), Reader, TestCSVParseAndValidate(), TestCSVParseEmptyFile()

### Community 210 - "Group 210"
Cohesion: 0.40
Nodes (5): Cache-Aside Pattern, Captcha Verification, JWT + Casbin RBAC Authentication, OAuth Third-Party Login, Rate Limiting (Token Bucket + Redis Window)

### Community 211 - "Group 211"
Cohesion: 0.40
Nodes (4): CLI 契约, 命令表, 种子数据, 迁移命名

### Community 212 - "Group 212"
Cohesion: 0.50
Nodes (4): No .env & Secrets Hook, Password Strength Rule, Security Policy, Vulnerability Reporting Process

### Community 214 - "Group 214"
Cohesion: 0.83
Nodes (3): CaptchaHandler, Service, NewCaptchaHandler()

### Community 217 - "Group 217"
Cohesion: 0.50
Nodes (3): Enforcer, HandlerFunc, PermissionMiddleware()

### Community 218 - "Group 218"
Cohesion: 0.50
Nodes (4): Audit Logging (Bounded Queue), Event Bus, Multi-Queue Support (Redis/Kafka/RabbitMQ), Outbox Pattern

### Community 219 - "Group 219"
Cohesion: 0.67
Nodes (3): GolangCI Lint Configuration, Lint Rule Set, Pre-commit Hooks Configuration

### Community 220 - "Group 220"
Cohesion: 0.67
Nodes (3): Clean Architecture Layering, contract.Module Interface, Modular Architecture

### Community 221 - "Group 221"
Cohesion: 0.67
Nodes (3): Unreleased Capabilities, Aliyun SMS Implementation (D2), Notification Abstraction (FR-028)

### Community 226 - "Group 226"
Cohesion: 0.67
Nodes (3): Cron Scheduler (robfig/cron), Redis Distributed Lock, Snowflake ID Generator

### Community 230 - "Group 230"
Cohesion: 0.67
Nodes (3): Permission Entity, Role Entity, RBAC (FR-010)

## Knowledge Gaps
- **216 isolated node(s):** `common.sh script`, `CaptchaResult`, `LogConfig`, `UserCreatedEvent`, `UserUpdatedEvent` (+211 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **64 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `T()` connect `API Contract & OpenAPI Tests` to `Permission Module`, `Snowflake ID & Encryption`, `File Storage (Local/S3/OSS/MinIO)`, `HTTP Upload & Fake Storage`, `Role Module`, `WebSocket Core`, `Outbox User Repository`, `User MySQL Integration Tests`, `OAuth Provider Tests`, `Gorm Logger`, `Outbox MQ Publisher`, `Queue Worker Pool`, `Admin API Key Service`, `Admin Config Service`, `Auth Module Bootstrap`, `Data Import Service`, `Admin Monitoring & Status`, `Role Application Service`, `Admin Job Handlers`, `Feature Flag Manager`, `Admin Import Handler`, `CSRF Middleware`, `HTTP Middleware (CORS/Logger/RequestID)`, `DB Cleanup Service`, `HTTP Server Lifecycle`, `User Rate Limit Middleware`, `JWT Token Service`, `gRPC Server`, `OAuth HTTP Handler`, `MySQL Connection & Tests`, `API Signature Middleware`, `Global Rate Limit (Token Bucket)`, `Webhook Notification`, `User Application Service`, `Role HTTP Handler`, `WebSocket Presence`, `Audit Repository (Fake)`, `Redis Distributed Lock`, `Scheduler Store & Observability`, `OAuth Application Service`, `Rate Limiter (Auth)`, `OAuth Module Config`, `AES-256-GCM Encryption`, `Auth Benchmark Tests`, `User Handler Tests`, `RabbitMQ Queue`, `Notification Dispatcher`, `Audit Worker`, `Database Config & Connection`, `Audit HTTP Handler`, `Email SMTP Service`, `WebSocket Channels`, `Admin Application Service`, `Zap Logger`, `Group 90`, `Group 92`, `Group 94`, `Group 95`, `Group 96`, `Group 97`, `Group 98`, `Group 100`, `Group 102`, `Group 104`, `Group 106`, `Group 108`, `Group 109`, `Group 111`, `Group 116`, `Group 117`, `Group 124`, `Group 125`, `Group 127`, `Group 130`, `Group 133`, `Group 134`, `Group 135`, `Group 137`, `Group 138`, `Group 141`, `Group 142`, `Group 143`, `Group 145`, `Group 146`, `Group 147`, `Group 148`, `Group 152`, `Group 154`, `Group 156`, `Group 158`, `Group 160`, `Group 161`, `Group 166`, `Group 168`, `Group 172`, `Group 176`, `Group 179`, `Group 184`, `Group 187`, `Group 188`, `Group 189`, `Group 191`, `Group 192`, `Group 193`, `Group 195`, `Group 201`, `Group 208`, `Group 209`?**
  _High betweenness centrality (0.548) - this node is a cross-community bridge._
- **Why does `Now()` connect `WebSocket Presence` to `Snowflake ID & Encryption`, `API Contract & OpenAPI Tests`, `HTTP Upload & Fake Storage`, `Group 135`, `WebSocket Core`, `Group 137`, `User MySQL Integration Tests`, `Group 139`, `Gorm Logger`, `OpenTelemetry Tracing`, `Admin API Key Service`, `Data Import Service`, `Admin Monitoring & Status`, `Group 156`, `CSRF Middleware`, `HTTP Middleware (CORS/Logger/RequestID)`, `Group 161`, `Redis Queue Implementation`, `DB Cleanup Service`, `Group 169`, `HTTP Server Lifecycle`, `Auth Application Service`, `API Key Auth Verifier`, `JWT Token Service`, `Group 172`, `HTTP Client (Retry/CircuitBreaker)`, `User Rate Limit Middleware`, `API Signature Middleware`, `Cron Scheduler`, `Webhook Notification`, `WebSocket Hub & Handler`, `Redis Cache (Cache-Aside)`, `User HTTP Handler`, `Group 195`, `Scheduler Store & Observability`, `OAuth Application Service`, `Casbin RBAC Authorization`, `Scheduler Memory Store`, `Admin Application Service`, `Group 91`, `Group 93`, `Group 94`, `Group 99`, `Group 110`, `Group 117`, `Group 119`, `Group 123`?**
  _High betweenness centrality (0.117) - this node is a cross-community bridge._
- **Why does `TestMysqlRepositoryMySQLIntegration()` connect `User MySQL Integration Tests` to `API Contract & OpenAPI Tests`?**
  _High betweenness centrality (0.042) - this node is a cross-community bridge._
- **Are the 4 inferred relationships involving `T()` (e.g. with `translateValidationDetails()` and `ValidateJSON()`) actually correct?**
  _`T()` has 4 INFERRED edges - model-reasoned connections that need verification._
- **Are the 81 inferred relationships involving `New()` (e.g. with `.CreateKey()` and `.ToggleTask()`) actually correct?**
  _`New()` has 81 INFERRED edges - model-reasoned connections that need verification._
- **Are the 80 inferred relationships involving `Fail()` (e.g. with `.Generate()` and `.HandleDelete()`) actually correct?**
  _`Fail()` has 80 INFERRED edges - model-reasoned connections that need verification._
- **Are the 80 inferred relationships involving `Now()` (e.g. with `.CreateKey()` and `.generateResetCode()`) actually correct?**
  _`Now()` has 80 INFERRED edges - model-reasoned connections that need verification._