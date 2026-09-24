package generator

import "embed"

// 模块骨架模板已外置到 templates/（P2.7 Task 1）：模板改动不再需要改 Go 代码，也不再与真实能力
// 结构无门禁地漂移。模板名相对本目录，如 "module.go.tmpl"、"migrations/mysql/000_create.sql.tmpl"。
//
//go:embed templates
var templateFS embed.FS

// templateRoot 是 embed 的根目录，模板名相对它解析。
const templateRoot = "templates"
