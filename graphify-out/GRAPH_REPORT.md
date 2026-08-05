# Graph Report - .  (2026-08-05)

## Corpus Check
- 149 files · ~54,260 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1072 nodes · 2018 edges · 71 communities (43 shown, 28 thin omitted)
- Extraction: 82% EXTRACTED · 18% INFERRED · 0% AMBIGUOUS · INFERRED: 356 edges (avg confidence: 0.81)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- Auth Handlers
- Auth Service & RBAC
- Casbin Authorization
- App Bootstrap
- Permission Repository
- Session Store
- Auth Module & Limiter
- Application Lifecycle
- Config Management
- User Service & DTO
- Role Repository
- Architecture Docs
- Redis Auth Pipeline
- HTTP Middleware
- Module Generator
- Audit Module
- Cache Layer
- Error Handling
- Audit Middleware
- Audit Repository
- Community 20
- Community 21
- Community 22
- Community 23
- Community 24
- Community 25
- Community 26
- Community 27
- Community 28
- Community 29
- Community 30
- Community 31
- Community 32
- Community 33
- Community 34
- Community 35
- Community 36
- Community 37
- Community 38
- Community 39
- Community 41
- Community 42
- Community 43
- Community 44
- Community 46
- Community 47
- Community 48
- Community 49
- Community 50
- Community 51
- Community 52
- Community 53
- Community 54
- Community 55
- Community 56
- Community 57
- Community 58
- Community 59
- Community 60
- Community 62
- Community 63
- Community 64
- Community 65
- Community 66
- Community 67
- Community 68
- Community 69

## God Nodes (most connected - your core abstractions)
1. `Fail()` - 36 edges
2. `New()` - 33 edges
3. `User` - 31 edges
4. `Wrap()` - 30 edges
5. `AuditLog` - 28 edges
6. `Permission` - 24 edges
7. `Role` - 19 edges
8. `OK()` - 17 edges
9. `AuthService` - 16 edges
10. `New()` - 16 edges

## Surprising Connections (you probably didn't know these)
- `Clean Architecture` --semantically_similar_to--> `Clean Architecture Layers`  [INFERRED] [semantically similar]
  README.md → docs/superpowers/plans/2026-07-28-jimu-backend-framework.md
- `Typed JWT Tokens` --semantically_similar_to--> `JWT Authentication`  [INFERRED] [semantically similar]
  docs/superpowers/plans/2026-07-29-jimu-runtime-security-baseline.md → README.md
- `Module Generator` --semantically_similar_to--> `Generator Templates`  [INFERRED] [semantically similar]
  README.md → docs/superpowers/plans/2026-07-31-jimu-unified-crud-generator.md
- `Rate Limiting` --semantically_similar_to--> `Redis Rate Limiter`  [INFERRED] [semantically similar]
  README.md → docs/superpowers/plans/2026-07-29-jimu-runtime-security-baseline.md
- `Module Structure Convention` --semantically_similar_to--> `Clean Architecture Layers`  [INFERRED] [semantically similar]
  CLAUDE.md → docs/superpowers/plans/2026-07-28-jimu-backend-framework.md

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Clean Architecture Layers Group** — arch_domain_layer, arch_application_layer, arch_infrastructure_layer, arch_interfaces_layer [EXTRACTED 1.00]
- **Authz Middleware Chain** — authz_protected_middleware, authz_role_loader, authz_casbin_enforcement, authz_middleware_provider [EXTRACTED 1.00]
- **CI/CD Pipeline Stages** — github_workflows_ci_lint_stage, github_workflows_ci_test_stage, github_workflows_ci_build_stage, github_workflows_ci_docker_stage [EXTRACTED 1.00]

## Communities (71 total, 28 thin omitted)

### Community 0 - "Auth Handlers"
Cohesion: 0.07
Nodes (35): AuthHandler, loginRequest, PermissionHandler, refreshRequest, RoleHandler, UserHandler, authContext(), Context (+27 more)

### Community 1 - "Auth Service & RBAC"
Cohesion: 0.06
Nodes (36): AssignPermissionsRequest, AuthService, CreatePermissionRequest, CreateRoleRequest, RoleResponse, RoleService, UpdatePermissionRequest, UpdateRoleRequest (+28 more)

### Community 2 - "Casbin Authorization"
Cohesion: 0.06
Nodes (42): AuthorizationStore, Claims, DBAuthorizationStore, fakeAuthorizationStore, JWT, Policy, fakeAuthzStore, Enforcer (+34 more)

### Community 3 - "App Bootstrap"
Cohesion: 0.05
Nodes (44): Container, moduleLogger, registerRouter, main(), run(), passingChecker, Server, Bootstrap() (+36 more)

### Community 4 - "Permission Repository"
Cohesion: 0.06
Nodes (32): fakePermissionRepository, PermissionService, Permission, PermissionRepository, mysqlPermissionRepository, fakePermissionRepository, NewPermissionService(), Context (+24 more)

### Community 5 - "Session Store"
Cohesion: 0.08
Nodes (26): fakeSessionStore, fakeUserRepo, fakeUserRepository, sessionRecord, User, fakeUserRepository, NewAuthService(), appCode() (+18 more)

### Community 6 - "Auth Module & Limiter"
Cohesion: 0.08
Nodes (43): Limiter, Module, AuthConfig, Router, NewAuthHandler(), RouterGroup, RegisterAuthRoutes(), Engine (+35 more)

### Community 7 - "Application Lifecycle"
Cohesion: 0.05
Nodes (25): Application, fakeAuthzModule, fakeBusinessModule, fakeComponent, Component, ComponentProvider, ErrorSource, EventBus (+17 more)

### Community 8 - "Config Management"
Cohesion: 0.07
Nodes (36): CacheConfig, DBConfig, ManagementConfig, RedisConfig, ServerConfig, applyEnvOverrides(), contains(), Config (+28 more)

### Community 9 - "User Service & DTO"
Cohesion: 0.08
Nodes (29): CreateUserRequest, UpdateUserRequest, UserResponse, UserService, UserRepository, Time, ToUserResponse(), ToUserResponses() (+21 more)

### Community 10 - "Role Repository"
Cohesion: 0.09
Nodes (20): fakeRoleRepository, Role, RoleRepository, rolePermission, fakeRoleRepository, NewRoleService(), Context, T (+12 more)

### Community 11 - "Architecture Docs"
Cohesion: 0.05
Nodes (46): Error Message Hygiene, Request ID Tracking, Application Layer, Clean Architecture Layers, Domain Layer, Infrastructure Layer, Interfaces Layer, Casbin Path Enforcement (+38 more)

### Community 12 - "Redis Auth Pipeline"
Cohesion: 0.12
Nodes (13): fakePipeline, fakeRedis, BoolCmd, Cmder, IntCmd, BoolSliceCmd, Cmd, Context (+5 more)

### Community 13 - "HTTP Middleware"
Cohesion: 0.09
Nodes (24): HTTPConfig, CORS(), HandlerFunc, Logger(), Recovery(), RequestID(), addVary(), Context (+16 more)

### Community 14 - "Module Generator"
Cohesion: 0.16
Nodes (27): targetFile, templateData, copyGoSum(), copyRootFile(), T, TestGeneratedModuleCompiles(), writeFileForTest(), writeStubPackages() (+19 more)

### Community 15 - "Audit Module"
Cohesion: 0.14
Nodes (13): Module, AuditHandler, Context, NewAuditHandler(), T, TestAuditGetInvalidIDReturnsStableBadRequest(), TestAuditGetReturnsLogDTO(), TestAuditListInvalidQueryReturnsStableBadRequest() (+5 more)

### Community 16 - "Cache Layer"
Cohesion: 0.19
Nodes (12): Cache, RedisCache, Client, Context, Duration, NewRedisCache(), T, setupTestCache() (+4 more)

### Community 17 - "Error Handling"
Cohesion: 0.15
Nodes (15): AppError, New(), T, TestAppError(), TestAppErrorWithCause(), T, TestFailDoesNotLeakInfrastructureDetails(), TestOKIncludesStableEnvelopeAndRequestID() (+7 more)

### Community 18 - "Audit Middleware"
Cohesion: 0.12
Nodes (16): Queue, AuditMiddleware(), Context, HandlerFunc, isManagementPath(), optionalString(), optionalUint64(), T (+8 more)

### Community 19 - "Audit Repository"
Cohesion: 0.19
Nodes (8): fakeAuditRepository, fakeBatchRepository, AuditLog, fakeQueue, Context, Context, Mutex, Time

### Community 20 - "Community 20"
Cohesion: 0.22
Nodes (12): redisClientAdapter, RedisSessionStore, sessionPipeline, sessionRedis, SessionStore, Client, Context, Duration (+4 more)

### Community 21 - "Community 21"
Cohesion: 0.18
Nodes (10): Worker, AuditConfig, LogConfig, Context, RWMutex, NewWorker(), New(), Logger (+2 more)

### Community 22 - "Community 22"
Cohesion: 0.22
Nodes (9): EventBus, Handler, RWMutex, New(), T, TestEventBus_Clear(), TestEventBus_MultipleHandlers(), TestEventBus_PublishAsync() (+1 more)

### Community 23 - "Community 23"
Cohesion: 0.29
Nodes (6): routerLimiterRedis, BoolSliceCmd, Cmd, Context, StringCmd, routerLimiterInt()

### Community 24 - "Community 24"
Cohesion: 0.27
Nodes (7): AuditService, AuditRepository, NewAuditService(), auditAppCode(), T, TestAuditServiceGetMapsNotFound(), TestAuditServiceListReturnsDTOAndPassesPagination()

### Community 25 - "Community 25"
Cohesion: 0.18
Nodes (11): Auth Configuration, Database Configuration, Development Profile, Environment Variable Override, HTTP Configuration, Logging Configuration, Management Configuration, Production Profile (+3 more)

### Community 26 - "Community 26"
Cohesion: 0.36
Nodes (4): mysqlAuditRepository, Context, DB, NewMysqlAuditRepository()

### Community 27 - "Community 27"
Cohesion: 0.46
Nodes (5): AuditLogResponse, Time, ToAuditLogResponse(), ToAuditLogResponses(), Context

### Community 28 - "Community 28"
Cohesion: 0.39
Nodes (6): FieldLevel, Validate(), validateIDCard(), validateMobile(), validatePassword(), validateUsername()

### Community 29 - "Community 29"
Cohesion: 0.52
Nodes (6): Duration, T, testWorker(), TestWorkerFlushesConfiguredBatch(), TestWorkerRejectsWhenQueueFull(), TestWorkerStopDrainsAcceptedRecords()

### Community 30 - "Community 30"
Cohesion: 0.33
Nodes (6): Release Workflow, Build Stage, Docker Build Stage, Lint Stage, OpenAPI Diff Check, Test Stage

### Community 32 - "Community 32"
Cohesion: 0.40
Nodes (5): Adminer Service, Health Check Configuration, MariaDB Service, Redis Service, Server Service

### Community 33 - "Community 33"
Cohesion: 0.40
Nodes (5): OpenAPI Schema, Bearer Auth Security, OpenAPI Definition, Pagination Query Parameters, Swagger Documentation

### Community 34 - "Community 34"
Cohesion: 0.80
Nodes (4): assertQueryParams(), T, readOpenAPI(), TestOpenAPIIncludesCRUDContract()

### Community 35 - "Community 35"
Cohesion: 0.60
Nodes (4): T, TestValidateMobile(), TestValidatePassword(), TestValidateUsername()

### Community 36 - "Community 36"
Cohesion: 0.67
Nodes (3): DB, Transaction(), WithTx()

### Community 37 - "Community 37"
Cohesion: 0.67
Nodes (3): T, TestNormalizeDefaultsAndCapsPageSize(), TestNormalizeRejectsInvalidSortAndOrder()

### Community 38 - "Community 38"
Cohesion: 0.67
Nodes (3): Bootstrap Assembly, Dependency Inversion, Module Interface

### Community 39 - "Community 39"
Cohesion: 0.67
Nodes (3): Branch Strategy, Commit Message Convention, Release Note Convention

## Knowledge Gaps
- **84 isolated node(s):** `jimu`, `ErrorSource`, `ComponentProvider`, `HTTPMiddlewareProvider`, `ProtectedHTTPMiddlewareProvider` (+79 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **28 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Wrap()` connect `Auth Service & RBAC` to `User Service & DTO`, `Casbin Authorization`, `Community 27`, `Error Handling`?**
  _High betweenness centrality (0.208) - this node is a cross-community bridge._
- **Why does `Fail()` connect `Auth Handlers` to `Casbin Authorization`, `Auth Module & Limiter`, `HTTP Middleware`, `Audit Module`, `Error Handling`?**
  _High betweenness centrality (0.158) - this node is a cross-community bridge._
- **Why does `New()` connect `Auth Module & Limiter` to `Auth Handlers`, `Auth Service & RBAC`, `Casbin Authorization`, `Session Store`, `Community 20`?**
  _High betweenness centrality (0.124) - this node is a cross-community bridge._
- **Are the 32 inferred relationships involving `Fail()` (e.g. with `.Get()` and `.List()`) actually correct?**
  _`Fail()` has 32 INFERRED edges - model-reasoned connections that need verification._
- **Are the 28 inferred relationships involving `New()` (e.g. with `.Logout()` and `.Refresh()`) actually correct?**
  _`New()` has 28 INFERRED edges - model-reasoned connections that need verification._
- **Are the 28 inferred relationships involving `Wrap()` (e.g. with `.Get()` and `.List()`) actually correct?**
  _`Wrap()` has 28 INFERRED edges - model-reasoned connections that need verification._
- **What connects `jimu`, `ErrorSource`, `ComponentProvider` to the rest of the system?**
  _84 weakly-connected nodes found - possible documentation gaps or missing edges._