package dataops

import (
	"strings"

	"jimu/internal/assembly"
	"jimu/internal/capabilities/dataops/exporter"
	"jimu/internal/capabilities/dataops/importer"
	"jimu/internal/contract"
)

// Wire 装配数据导入导出能力：返回用户导入管理端模块（/api/v1/admin/users/import*）。
func Wire(ctx *assembly.Context) (contract.Module, error) {
	// 驱动级可插拔（设计 §3.7）：打印本构建编入的导入/导出格式，便于诊断「格式未编译」类错误。
	// 值渲染为标量字符串（logcheck R4 禁 slice）；key 取标准词汇表（names）以避免 R3 告警。
	ctx.Logger().Infow("dataops import formats compiled", "names", formatNames(importer.RegisteredFormats()))
	ctx.Logger().Infow("dataops export formats compiled", "names", formatNames(exporter.RegisteredFormats()))
	return New(ctx.DB()), nil
}

// formatNames 把格式清单渲染为逗号分隔字符串；日志字段必须是标量（日志调用规范禁 slice）。
func formatNames[T ~string](formats []T) string {
	names := make([]string, len(formats))
	for i, f := range formats {
		names[i] = string(f)
	}
	return strings.Join(names, ",")
}
