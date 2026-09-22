// Command logcheck 检查日志调用的结构化规范（与 AGENTS.md「日志调用规范」同步）：
//
//	R1 [错误] 禁止 Debug/Info/Warn/Error 传多个参数（sugared logger 会把
//	        所有参数 fmt.Sprint 拼进消息，导致 k/v 粘连、字段丢失）
//	R2 [错误] 字段 key 必须是字符串字面量（禁止动态 key，避免字段爆炸）
//	R3 [警告] 字段 key 建议在标准词汇表内（AGENTS.md 日志调用规范，未登记提示）
//	R4 [错误] 值禁止直接传 struct/map/slice/interface 对象
//	        （OTLP attribute 不支持嵌套对象，会被字符串化；error/time.Time/
//	         []byte 例外）
//
// 用法：go run ./tools/logcheck ./internal ./cmd ./tools
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"
)

// vocabulary 标准字段词汇表：与 AGENTS.md「日志调用规范」同步维护。
// 新字段 key 需登记，避免同一概念出现多个 key（如 order_id / orderId / oid）。
var vocabulary = map[string]string{
	"trace_id":           "追踪 ID（WithContext 自动注入，业务不手动写）",
	"span_id":            "Span ID（WithContext 自动注入，业务不手动写）",
	"caller":             "调用位置（链路自动注入）",
	"module":             "模块名（auth/user/admin/scheduler/audit…）",
	"error":              "错误对象（写 \"error\", err，不要 err.Error()）",
	"error_code":         "业务错误码（与统一响应 body.code 一致）",
	"id":                 "实体 ID",
	"name":               "任务/模块/事件名称",
	"names":              "名称清单（聚合名，如当前启用的能力清单）",
	"missing":            "缺失项清单（如缺失的可选依赖名）",
	"mount":              "HTTP 挂载点（能力 Descriptor 的 Mount）",
	"type":               "类型/事件类型",
	"event_type":         "业务事件类型",
	"user_id":            "用户 ID",
	"request_id":         "请求 ID",
	"job_id":             "任务 ID",
	"order_id":           "订单 ID",
	"key":                "配置键",
	"value":              "配置值/新值",
	"level":              "日志/配置级别",
	"spec":               "cron 表达式",
	"entry_id":           "cron 条目 ID",
	"jobs":               "任务数量",
	"attempt":            "重试次数（第几次）",
	"max_retries":        "最大重试次数",
	"interval_sec":       "重试间隔（秒）",
	"direction":          "迁移方向（up/down）",
	"table":              "表名",
	"deleted":            "删除行数",
	"count":              "数量",
	"addr":               "监听/连接地址",
	"mode":               "运行模式",
	"channel":            "通知渠道",
	"to":                 "收件人",
	"subject":            "主题",
	"body":               "内容",
	"template_id":        "模板 ID",
	"duration":           "耗时（纳秒数值，可聚合）",
	"panic":              "panic 值",
	"elapsed":            "耗时字符串（gorm 慢查询）",
	"rows":               "影响行数（gorm）",
	"sql":                "脱敏后的 SQL（gorm）",
	"latency":            "请求耗时字符串（HTTP 访问日志）",
	"client_ip":          "客户端 IP（HTTP 访问日志）",
	"user_agent":         "用户代理（HTTP 访问日志）",
	"request_body":       "请求体（HTTP 访问日志，脱敏截断）",
	"response_body":      "响应体（HTTP 访问日志，脱敏截断）",
	"response_truncated": "响应体是否截断（布尔）",
	"data":               "模板变量/扩展数据（JSON 字符串）",
}

// zapLogMethods zap sugared/structured Logger 上可识别的日志方法。
var zapLogMethods = map[string]bool{
	"Debug": true, "Info": true, "Warn": true, "Error": true,
	"Debugw": true, "Infow": true, "Warnw": true, "Errorw": true,
	"Debugf": true, "Infof": true, "Warnf": true, "Errorf": true,
	"Debugln": true, "Infoln": true, "Warnln": true, "Errorln": true,
	"DPanicw": true, "Panicw": true, "Fatalw": true,
}

type issue struct {
	pos     token.Position
	kind    string // ERROR / WARN
	message string
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法: logcheck <packages...>\n%s", "检查日志调用的结构化规范（R1 防粘连 / R2 禁动态 key / R3 词汇表 / R4 禁嵌套对象）\n")
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
	}
	pkgs, err := packages.Load(cfg, flag.Args()...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load packages: %v\n", err)
		os.Exit(2)
	}
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(2)
	}

	issues := scanPkgs(pkgs)
	for _, it := range issues {
		if it.kind == "ERROR" {
			fmt.Printf("%s: [ERROR] %s\n", it.pos, it.message)
		} else {
			fmt.Printf("%s: [WARN ] %s\n", it.pos, it.message)
		}
	}
	if n := countErrors(issues); n > 0 {
		fmt.Printf("❌ logcheck: %d error(s) found\n", n)
		os.Exit(1)
	}
	fmt.Println("✅ logcheck: 日志调用符合结构化规范")
}

func scanPkgs(pkgs []*packages.Package) []issue {
	var issues []issue
	for _, p := range pkgs {
		if p.PkgPath == "jimu/tools/logcheck" { // 跳过检查器自身
			continue
		}
		for _, file := range p.Syntax {
			if strings.HasSuffix(p.Fset.Position(file.Pos()).Filename, "_test.go") {
				continue
			}
			issues = append(issues, checkFile(p, file)...)
		}
	}
	return issues
}

func countErrors(issues []issue) int {
	n := 0
	for _, it := range issues {
		if it.kind == "ERROR" {
			n++
		}
	}
	return n
}

func checkFile(pkg *packages.Package, file *ast.File) []issue {
	var issues []issue
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if !isZapLogMethod(pkg.TypesInfo, sel) {
			return true
		}
		method := sel.Sel.Name
		pos := pkg.Fset.Position(sel.Pos())

		// R1: 非 *w/*f/*ln 方法与 ≥2 个参数 → 粘连风险
		if isPlainMethod(method) && len(call.Args) >= 2 {
			issues = append(issues, issue{pos: pos, kind: "ERROR",
				message: fmt.Sprintf("R1 禁止 %s(...) 传多个参数（参数会被 fmt.Sprint 拼进消息导致粘连），请改用 %sw(...)",
					method, method)})
			return true
		}
		if !isStructuredMethod(method) || len(call.Args) < 2 {
			return true
		}

		// *w 系列：args[0] 是 msg，args[1:] 是 keysAndValues
		for i := 1; i < len(call.Args); i += 2 {
			// ellipsis 展开（如 append(fields, "error", err)...）由代码内字面量 k/v
			// 构建，属受控模式，跳过静态检查。
			if call.Ellipsis.IsValid() && i == len(call.Args)-1 {
				break
			}
			keyExpr := call.Args[i]
			lit, isLit := keyExpr.(*ast.BasicLit)
			// R2: key 必须是字符串字面量
			if !isLit || lit.Kind != token.STRING {
				issues = append(issues, issue{pos: pos, kind: "ERROR",
					message: "R2 字段 key 必须是字符串字面量，禁止动态 key（变量/表达式当 key 会导致字段爆炸）"})
				continue
			}
			key := strings.Trim(lit.Value, `"`)

			// R3: key 在词汇表内（未登记仅告警）
			if _, ok := vocabulary[key]; !ok && !strings.HasSuffix(key, "_id") && !strings.HasSuffix(key, "_ms") &&
				!strings.HasSuffix(key, "_sec") && !strings.HasSuffix(key, "_bytes") && !strings.HasSuffix(key, "_ns") {
				issues = append(issues, issue{pos: pos, kind: "WARN",
					message: fmt.Sprintf("R3 字段 %q 不在标准词汇表，请登记到 AGENTS.md 日志调用规范（或用 *_id/_ms/_sec/_bytes/_ns 带单位后缀）", key)})
			}

			// R4: value 不得是嵌套对象
			if i+1 < len(call.Args) {
				if reason := nestedObjectReason(pkg.TypesInfo, call.Args[i+1]); reason != "" {
					issues = append(issues, issue{pos: pos, kind: "ERROR",
						message: fmt.Sprintf("R4 字段 %q 的值是 %s，OTLP attribute 不支持嵌套对象（会被字符串化），请拆成标量字段或序列化", key, reason)})
				}
			}
		}
		return true
	})
	return issues
}

func isZapLogMethod(info *types.Info, sel *ast.SelectorExpr) bool {
	selInfo, ok := info.Selections[sel]
	if !ok {
		return false
	}
	fn, ok := selInfo.Obj().(*types.Func)
	if !ok || fn.Pkg() == nil {
		return false
	}
	if fn.Pkg().Path() != "go.uber.org/zap" {
		return false
	}
	return zapLogMethods[fn.Name()]
}

func isPlainMethod(name string) bool {
	return name == "Debug" || name == "Info" || name == "Warn" || name == "Error"
}

func isStructuredMethod(name string) bool {
	return strings.HasSuffix(name, "w") && !strings.HasSuffix(name, "nw")
}

// nestedObjectReason 返回值类型不适合作为日志字段的原因；空串表示允许。
func nestedObjectReason(info *types.Info, expr ast.Expr) string {
	tv, ok := info.Types[expr]
	if !ok {
		return ""
	}
	t := tv.Type
	if t == nil {
		return ""
	}
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	// 例外：time.Time、url.URL（值语义类型）
	if named, ok := t.(*types.Named); ok && named.Obj() != nil && named.Obj().Pkg() != nil {
		pkgPath := named.Obj().Pkg().Path()
		if (pkgPath == "time" && named.Obj().Name() == "Time") ||
			(pkgPath == "net/url" && (named.Obj().Name() == "URL" || named.Obj().Name() == "Values")) {
			return ""
		}
	}
	// error 接口（内置）允许：Errorw(msg, "error", err)
	if t.String() == "error" {
		return ""
	}
	switch u := t.Underlying().(type) {
	case *types.Struct:
		return "struct 对象"
	case *types.Map:
		return "map 对象"
	case *types.Slice:
		if e, ok := u.Elem().Underlying().(*types.Basic); ok && e.Kind() == types.Byte {
			return "" // []byte 允许（zap ByteString）
		}
		return "slice 对象（[]byte 除外）"
	case *types.Interface:
		return "interface 动态类型（请显式转换为标量再记录）"
	}
	return ""
}
