---

description: "Task list for jimu-framework-spec gap closure"
---

# Tasks: Jimu 后端框架能力规格

**Input**: Design documents from `/specs/001-jimu-framework-spec/`

**Prerequisites**: plan.md（必需）、spec.md（用户故事）、research.md、data-model.md、contracts/

**Tests**: 仅含宪法 VI 要求的 SMS 契约测试（见 research D2）；其余验证走 quickstart。

**Organization**: 本特性是"现状能力契约 + 缺口收敛"，非绿地开发。US1/US3 已实现，任务为验证；真实代码工作集中在 US2（发布门禁）与 US4（阿里云 SMS）。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行（不同文件、无依赖）
- **[Story]**: US1/US2/US3/US4（映射 spec 用户故事）
- 含精确文件路径

## Path Conventions

- Go 框架仓库，路径基于 `internal/`、`configs/`、`.github/workflows/`、`Makefile`、`README.md`

## Phase 1: Setup (Baseline)

**Purpose**: 建立可验证基线，后续改动可对照

- [X] T001 跑 `make release-check` 确认 master 基线通过，记录结果到 plan.md Notes（基线证据）
- [X] T002 [P] 刷新知识图谱基线：`graphify update .`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 消除两处实现前的工具/版本风险

**⚠️ CRITICAL**: US2 与 US4 均依赖此处确认的工具可用性

- [X] T003 确认 govulncheck 已安装可用（`govulncheck version`），记录版本；若缺失，给出安装命令（`go run golang.org/x/vuln/cmd/govulncheck@latest`）
- [X] T004 [P] 确认阿里云 SDK 最新版本：`go list -m -versions github.com/alibabacloud-go/dysmsapi-20170525`，选定 v5.x 最高版并记录（依赖 T010 精确版本）

**Checkpoint**: 工具与版本风险清零，US2/US4 可开工

---

## Phase 3: User Story 1 - 认证服务搭建（P1）🎯 已实现·验证

**Goal**: 确认认证闭环（注册/登录/刷新/登出/RBAC/统一响应）真实可用

**Independent Test**: quickstart V1 —— 登录拿 token、RBAC 访问、未认证拒绝、刷新/登出

### Implementation for User Story 1

- [X] T005 [US1] 按 quickstart.md V1 跑认证验证（docker compose 起依赖 → migrate up → seed → login/refresh/logout/RBAC），记录结果到 quickstart 备注

**Checkpoint**: US1 认证闭环验证通过

---

## Phase 4: User Story 2 - 部署与观测（P2）🎯 发布门禁修复

**Goal**: 修复宪法 V 违规——发布路径缺失 govulncheck/Trivy

**Independent Test**: `make release-check` 输出含 govulncheck 步骤；`.github/workflows/release.yml` 含漏洞扫描

### Implementation for User Story 2

- [X] T006 [P] [US2] 在 Makefile 新增 govulncheck 目标并并入 `release-check`（Makefile:227 `fmt-check vet test` 后追加 `govulncheck ./...`）
- [X] T007 [US2] 对齐 .github/workflows/release.yml：tag 发布前运行完整门禁（含 govulncheck）；确认 ci.yml 的 govulncheck/Trivy/覆盖率在推送路径保留
- [X] T008 [US2] 跑 `make release-check` 验证 govulncheck 生效；更新 README「安全扫描」/「质量门禁」描述

**Checkpoint**: 发布门禁合规，宪法 V 违规关闭

---

## Phase 5: User Story 3 - 模块扩展（P3）🎯 已实现·验证

**Goal**: 确认脚手架与模块契约可独立扩展

**Independent Test**: quickstart V3 —— `jimu module create` 生成骨架、实现 contract.Module、路由注册在 /api/v1、返回统一格式

### Implementation for User Story 3

- [X] T009 [US3] 按 quickstart.md V3 跑模块扩展验证（module create + 注册 + 启动），记录结果

**Checkpoint**: 模块扩展能力验证通过

---

## Phase 6: User Story 4 - 高级能力集成（P4）🎯 阿里云 SMS 实现

**Goal**: 实现 FR-028 阿里云短信渠道（用户指定 SDK 方案，research D2），消除桩代码

**Independent Test**: `go test ./internal/platform/notification/ -run SMS` 通过；dispatcher 注册 SMS 渠道

### Implementation for User Story 4

- [X] T010 [P] [US4] 加阿里云 SDK 依赖：`go get github.com/alibabacloud-go/dysmsapi-20170525/v5@<T004确认版本>`（含 Tea 系传递依赖）
- [X] T011 [US4] 实现 `sendAliyun`（internal/platform/notification/sms.go）：dysmsapi client 调 SendSms，模板码=msg.TemplateID、模板变量=msg.Data、手机号=msg.To、签名=config.SignName；错误包装统一错误信息（依赖 T010）
- [X] T012 [US4] 配置接线：config.go 增 `Notification.SMS SMSConfig` 结构 + 校验；configs/app.yaml、.env.example 增 `notification.sms`（provider/api_key/api_secret/sign_name）段（依赖 T011 字段）
- [X] T013 [US4] 装配：internal/app/container.go（参考 :124-134 email 模式）注册 SMS 渠道——未启用时 LogChannel 兜底、启用时 NewSMS（依赖 T012）
- [X] T014 [P] [US4] 契约测试 internal/platform/notification/sms_test.go：client Endpoint 指向 httptest mock server，断言请求体（手机号/签名/模板/变量）与错误路径（依赖 T011）
- [X] T015 [US4] 降级 tencent 桩：`sendTencent` 改为明确报错"provider not configured"；确认 SMS 未配置时调度器正常回退

**Checkpoint**: 阿里云 SMS 可调用，桩代码消除，宪法 VI 违规关闭

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: 文档对齐与全量验证

- [X] T016 [P] 修 FR-014 措辞：spec.md 与 README 限流描述改为「全局=令牌桶、登录=固定窗口、用户维度=滑动窗口」（对齐 research D3）
- [X] T017 更新 README：SMS 能力表述（阿里云 SDK 已实现、tencent 未配置）+ 发布门禁（release-check 含 govulncheck）
- [X] T018 更新 spec.md FR-028/FR-034 状态为 implemented；同步 checklists/requirements.md
- [X] T019 全量验证：`make release-check`、`golangci-lint run`、quickstart V1–V5 全跑，结果记录到 quickstart 备注
- [X] T020 [P] 代码改动后刷新知识图谱：`graphify update .`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup（Phase 1）**: 无依赖，立即开始
- **Foundational（Phase 2）**: 依赖 Setup；阻塞 US2/US4 的工具与版本确认
- **US1/US3（Phase 3/5）**: 纯验证，可与 Phase 2 并行
- **US2（Phase 4）**: 依赖 T003
- **US4（Phase 6）**: 依赖 T004；内部顺序 T010 → T011 → T012 → T013，T014 并行于 T012/T013
- **Polish（Phase 7）**: 依赖 US2/US4 完成

### User Story Dependencies

- **US1（P1）**: 无代码依赖，仅验证现有实现
- **US2（P2）**: 独立于其他 story
- **US3（P3）**: 独立
- **US4（P4）**: 独立；内部按依赖顺序推进

### 各 Story 内顺序

- US4：依赖（T010）→ 实现（T011）→ 配置（T012）→ 装配（T013）；契约测试 T014 在 T011 后可并行
- 验证任务（T005/T008/T009/T019）必须输出可复现结果到对应文档

### Parallel Opportunities

- T002/T003/T004 可并行（工具检查）
- T005（US1）与 T009（US3）与 US2/US4 实现可并行
- T006 与 T010 完全独立，可并行（不同文件）
- T014 与 T012/T013 不同文件，可并行
- T016 与 T017 均文档，可并行

---

## Parallel Example: US4（SMS）

```bash
# 依赖与契约测试可并行（T010 先于 T011，T014 待 T011）：
Task: "T010 加阿里云 SDK 依赖（go.mod）"
# T011 完成后并行：
Task: "T011 实现 sendAliyun（sms.go）"
Task: "T014 契约测试（sms_test.go，mock server）"
```

---

## Implementation Strategy

### MVP First（宪法强制的最小闭环）

1. Phase 1 Setup（基线）
2. Phase 2 Foundational（T003 govulncheck 确认）
3. **US2 发布门禁（T006-T008）**——宪法 V 违规修复，最小、独立、合规优先
4. **STOP & VALIDATE**: `make release-check` 含 govulncheck
5. 再入 US4（SMS），随后 Polish

### Incremental Delivery

1. 基线通过 → 门禁合规（US2）→ 验证 US1/US3 → SMS 实现（US4）→ 文档对齐
2. 每个 story 独立验证、独立提交，不互相阻塞

---

## Notes

- SMS 需阿里云 AccessKey 才能做真实发送 smoke test；无凭据时以契约测试（mock）为准，真实发送列为待验证项（quickstart V4 备注）
- [P] 任务 = 不同文件、无依赖
- [Story] 标签映射 spec 用户故事
- 每完成逻辑组提交一次（Conventional Commits），勿混入无关改动
- 检查点处停下独立验证 story
