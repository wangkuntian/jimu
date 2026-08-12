# 统一响应契约

来源：`internal/shared/response/response.go`

## 规则

- **HTTP 状态码恒为 200**；业务结果通过 `body.code` 表达
- `code=0` 表示成功；非 0 为业务错误码

## 成功响应

```json
{"code": 0, "message": "ok", "data": {}}
```

## 错误响应

```json
{"code": 1001, "message": "参数错误", "data": null}
```

## 错误码区间

定义于 `internal/shared/errors/errors.go`：

| 区间 | 归属 |
|------|------|
| 1xxx | 通用错误（参数、鉴权、系统） |
| 2xxx | 用户/认证模块 |
| 3xxx+ | 后续模块按序分配 |

## 分页

列表接口 data 内含分页元数据（`internal/shared/pagination`）：`list` + `total` + `page` + `page_size`。

## 国际化

`message` 按请求 `Accept-Language` 返回中/英文（`internal/shared/i18n`）。
