// Package heavydeps 登记「重型第三方依赖」的模块前缀（设计 §3.7 的三类驱动）。
// 门禁（tools/checkcapabilities）与报告（tools/composereport）共用这一份，
// 避免「门禁说没有、报告说有时」的两处漂移。
package heavydeps

import (
	"sort"
	"strings"
)

// deps 模块前缀 → 报告与错误文案里的展示名。
var deps = map[string]string{
	"github.com/aws/aws-sdk-go-v2":   "aws-sdk-go-v2",
	"github.com/segmentio/kafka-go":  "kafka-go",
	"github.com/rabbitmq/amqp091-go": "amqp091-go",
	"github.com/xuri/excelize":       "excelize",
}

// Of 返回包路径所属的重型依赖展示名；不属于时返回空串。
func Of(pkgPath string) string {
	for prefix, name := range deps {
		if pkgPath == prefix || strings.HasPrefix(pkgPath, prefix+"/") {
			return name
		}
	}
	return ""
}

// Names 返回全部展示名（升序）。
func Names() []string {
	out := make([]string, 0, len(deps))
	for _, name := range deps {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
