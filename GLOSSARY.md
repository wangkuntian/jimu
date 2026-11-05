# Jimu

Jimu 是一个生成独立 Go 后端项目的工程基线；这个术语表只描述产品与生成流程中的领域概念。

## 语言

**Template bundle**：
CLI 用于生成项目的版本化输入文件集合。中文首次出现可写“template bundle（模板包）”，之后统一使用英文术语。
_Avoid_：模板载荷、payload、模板源、源码包、template source

**Embedded bundle**：
编译进 `jimu` 二进制的 template bundle。
_Avoid_：模板载荷、embedded payload

**External bundle**：
从开发者或测试指定的源根加载的 template bundle；不是用户安装模式。
_Avoid_：外部模板、用户模板源

**Template bundle manifest**：
记录 template bundle 的版本、commit、契约版本和文件摘要的 `template-bundle.json`。
_Avoid_：版本文件、bundle 配置

**Framework checkout**：
Jimu 仓库的开发者工作树，包含 `.git` 和仅开发时需要的文件。
_Avoid_：源码根、模板目录

**Generated project**：
由 `jimu new` 产出、拥有独立 module 的 Go 项目；生成后归使用者所有。
_Avoid_：脚手架目录、示例项目

**Upgrade validation project**：
由 Release CLI 基于 official template 生成、已被真实团队用于实际服务，并包含团队修改的项目；它用于验证跨版本升级。中文首次出现可写“升级验证项目”。
_Avoid_：真实项目、示例项目、测试 fixture

**Upgrade evidence record**：
一次 upgrade validation project 从指定起始版本升级到目标版本的可审计记录，包含命令输出、冲突路径与原因、耗时和回滚次数。
_Avoid_：升级日志、测试报告

**Project lifecycle check**：
对 generated project 的组合、受管文件状态、可用升级和下一步操作进行的只读检查；深检可以再执行配置、build 和 test。中文首次出现可写“项目生命周期检查”。
_Avoid_：doctor、status、健康检查

**Upgrade conflict diagnosis**：
对一次只读 upgrade plan 中需要处理的条目展示分类、三方哈希、local↔target 内容差异和下一步动作的稳定诊断。中文首次出现可写“升级冲突诊断”。
_Avoid_：自动合并、升级修复

**Design partner**：
愿意在真实服务中采用一个 official template 并提供反馈的团队；其项目采用记录和案例授权是两种独立证据。
_Avoid_：用户、客户、尝鲜者

**Local metric collection**：
只在用户或维护者控制的文件与流程中记录和汇总产品指标的方式；数据默认不上传。
_Avoid_：产品遥测、埋点

**Official template**：
`basic-api` 和 `auth-api` 两个 certified 模板，受发布阻断门禁保护。
_Avoid_：预设项目、样例

**Profile**：
Jimu 服务二进制的编译期组成形态；不是 `jimu new` 模板，也不是对外产品契约。
_Avoid_：模板、模板源、运行形态

**Frozen profile**：
保留构建和门禁、但不再新增能力或进入产品路线的内部 profile。
_Avoid_：废弃 profile、实验 profile

**Retired profile**：
已从 registry、构建脚本、发布矩阵和文档中移除的内部 profile。
_Avoid_：隐藏 profile、禁用 profile

**Full-stack demo**：
用 `full` profile 运行 Jimu server 并连接 `jimu-web` 的可演示全栈体验；它不证明 `jimu new` 生成全栈项目。
_Avoid_：全栈模板、generated full-stack

**Full-stack preview**：
以 experimental 支持级别发布的 full-stack demo 版本；不是 certified 模板或生产承诺。
_Avoid_：全栈正式版、稳定版

**Experimental composition**：
通过 `--with` 或 `capability add` 形成的能力组合，不对应官方模板，支持级别为 experimental。
_Avoid_：自定义模板、扩展包

**Clean-room environment**：
没有 Jimu checkout、没有预置生成项目、没有模板缓存的验证环境；允许使用 Go、Docker 和网络。
_Avoid_：本机、临时目录

**Clean-room gate**：
在 clean-room environment 中验证安装版 CLI 的发布门禁。
_Avoid_：冒烟测试、集成测试

**Release CLI**：
Release 中发布的、只负责项目生成与升级的 `jimu` 二进制。
_Avoid_：框架 CLI、server CLI、全功能 CLI

**CLI version**：
已安装 `jimu` 二进制的发布标识，由 Release tag 派生。
_Avoid_：工具版本、程序版本

**Template bundle version**：
Template bundle 的发布标识，与 CLI 使用同一个 Release tag 和 commit。
_Avoid_：模板载荷版本、模板版本、模板源版本

**Template contract version**：
官方模板的兼容契约版本，当前为 `v0.1`；它不是 CLI 或 template bundle 的发布版本。
_Avoid_：接口版本、模板版本

**Upgrade support window**：
Jimu 承诺可以显式升级到当前 certified 模板的版本范围；当前从 `v0.5.0` 起的 manifest schema v2/v3 项目，不包含 `v0.4.x`、schema v1 或 experimental 组合。
_Avoid_：永久兼容、任意旧版本升级

**Production readiness checklist**：
针对某个 official template 的上线前检查集合，分为必须配置、可选配置和上线前验证，并映射到生成项目的配置键、命令或文档。
_Avoid_：通用安全清单、部署教程

**Security remediation drill**：
在受控旧项目上验证“安全修复进入模板 → 发布候选 CLI → 执行升级 → 验证修复生效”的可重复演练；它不等于真实漏洞公告。
_Avoid_：CVE 公告、渗透测试
