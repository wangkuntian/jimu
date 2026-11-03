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

**Official template**：
`basic-api` 和 `auth-api` 两个 certified 模板，受发布阻断门禁保护。
_Avoid_：预设项目、样例

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
