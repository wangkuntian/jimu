# 新增形态（profile）/ 给能力加驱动

**何时读**：要新增形态、给现有能力接入第三方驱动，或调整某形态编进二进制的驱动集合时。

**权威口径**：README「[形态（profile）](../../../README.md#形态profile)」、「[驱动](../../../README.md#驱动)」、「[非代码资产](../../../README.md#非代码资产)」；稳定边界见[形态与项目生成](../../../docs/design/profiles-and-project-generation.md)。

## 一、新增形态（profile）

形态是**编译期**概念：它决定哪些包与符号编进二进制，**不减小 `go.mod`**（真正减小依赖的是层① `jimu new`）。

1. **建形态包** `internal/profiles/<name>/`：`assembly.go` 声明能力清单（`[]assembly.Capability`，顺序即装配顺序，端口提供者必须排在消费者之前），有驱动时加 `drivers.go` 做 blank import
2. **在 `internal/profiles/registry` 登记**：`names`（固定顺序，报告行序与门禁遍历顺序都依赖它）与 `All`。`tools/profileoverlay -list`、`checkcapabilities`、`composereport`、`scripts/check_profiles.sh` 全部从它派生 —— 漏登记则门禁与报告看不见该形态
3. **同步 golden 闭包清单**：`scripts/check_profiles.sh` 的 `EXPECTED_<name>` / `FORBIDDEN_<name>`（能力根包逐值锁定）。不更新即 `make profiles-check` 失败 —— 这是刻意的「新增能力必须显式声明」刹车
4. **守住两条不变式**：
   - 选点包 `internal/profiles/active` **恰好** import 一个形态
   - `internal/profiles/registry` **不得**被 `cmd/server` 或选点包 import（它 import 全部形态包，被引用会把所有形态拉回二进制、层②裁剪失效）。两条都由 `make check-capabilities` ④ 号断言检查
5. **非 catalog 条目按 `Ungated` 声明**：非 catalog 能力在形态清单中标记 `Ungated`，不受 `capabilities.enabled` 门控

**验收**：

```bash
make check-capabilities                       # 7 条 ✅（④ 号管入口与选点包，⑦ 号管生成器边界）
make profiles-check                           # 新形态 overlay 构建 + golden 闭包通过
PROFILE=<name> make build-server              # 产物 bin/jimu-server-<name>
JIMU_PROFILES_SMOKE=1 make profiles-check     # 启动并轮询管理端 /readyz（需 DB+Redis）
```

## 二、给能力新增第三方驱动（设计 §3.7 五步）

1. **驱动独立成包** `internal/capabilities/<cap>/<driver>/`，包内 `init()` 调用能力核心的 `Register`。能力核心只留接口 + 注册表（`New`/`Get` 查表，**未注册即 fail-closed**，不静默回退），**不得** import 任何驱动包或第三方重型依赖
2. **声明可用集**：能力 `Descriptor.Drivers` 加驱动的**包名**（如 `storage` = `[local s3]`、`queue` = `[redis kafka rabbitmq]`、`dataops` = `[csv excel]`）。一个驱动包覆盖多个配置取值时仍是同一个包名（`s3` 包同时注册 `s3`/`oss`/`minio`）
3. **声明选中子集**：在需要该驱动的形态 `internal/profiles/<name>/assembly.go` 的 `assembly.Capability.Drivers` 里列出选中项（装配期强制 `⊆` 可用集）
4. **落实 import**：`internal/profiles/<name>/drivers.go` blank import `_ "jimu/internal/capabilities/<cap>/<driver>"`，并与第 3 步**逐值一致**
5. **复核门禁与报告**：`make check-capabilities` 的驱动段（可用集 ↔ 目录存在 / 核心包生产闭包零驱动零重型依赖 / 形态选中集 == 形态生产 import 闭包（集合比较）/ 形态生产代码只 import 已声明驱动）+ `make compose-report` 的「重型依赖」列确实随形态消失

**验收**：

```bash
make check-capabilities        # ②③ 号断言管驱动
make compose-report            # 看「重型依赖」列（aws-sdk-go-v2 / kafka-go / amqp091-go / excelize）
make compose-report-check      # 报告入库后：实测与入库一致
```

## 常见红与定位

| 症状 | 定位 |
|---|---|
| `profiles-check` 报某形态闭包集合不匹配 | 更新 `scripts/check_profiles.sh` 的 `EXPECTED_<name>`（新增能力）或检查是否漏了 `FORBIDDEN_<name>` |
| `check-capabilities` ④ 号红 | 选点包被改成了 import 多个形态，或 `registry` 被入口/选点包引用 |
| 新增驱动目录 + 形态 blank import 了，但门禁没报错 | **预期行为**：集合比较看不见未声明项（被 `available` 过滤）。唯一捕获点是「驱动包只被 `internal/profiles/*` import」与「形态生产代码只 import 已声明驱动」两条断言 —— 所以驱动**必须**先在 `Descriptor.Drivers` 声明 |
| 某形态下配置写了某驱动却启动报错 | 该驱动没编进这个形态（fail-closed 的刻意行为，错误里会列出已编译的驱动集合） |
