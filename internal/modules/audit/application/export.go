package application

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"jimu/internal/modules/audit/domain"
	"jimu/internal/platform/tenant"
	"jimu/internal/shared/errors"
)

// 导出约束：格式取值、单次条目上限、分批大小与最大时间跨度
const (
	ExportFormatCSV  = "csv"
	ExportFormatJSON = "json"
	// exportMaxRows 单次导出条目上限，超过需缩小时间范围（避免长事务与大内存占用）
	exportMaxRows = 50000
	// exportBatchSize 分批读取大小
	exportBatchSize = 500
	// exportMaxRangeDays 单次导出的最大时间跨度（天）
	exportMaxRangeDays = 90
)

// ExportOptions 审计日志导出选项
type ExportOptions struct {
	Format string    // csv（默认）或 json（NDJSON）
	Start  time.Time // 起始时间（含）；零值表示最近 7 天
	End    time.Time // 结束时间（不含）；零值表示当前时间
}

// ExportSummary 导出结果摘要
type ExportSummary struct {
	Format string    `json:"format"`
	Rows   int64     `json:"rows"`
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
}

// exportColumns CSV 表头，顺序与行输出一致
var exportColumns = []string{
	"id", "created_at", "tenant_id", "user_id", "username", "action", "resource",
	"method", "path", "status", "ip", "detail", "prev_hash", "entry_hash",
}

// Export 按时间范围导出审计日志（当前租户；平台级视角导全部租户）。
// 先统计条目数并校验上限，再分批流式写出，避免一次性把结果读入内存。
func (s *AuditService) Export(ctx context.Context, opts ExportOptions, w io.Writer) (*ExportSummary, error) {
	if w == nil {
		return nil, errors.New(errors.CodeInternalError, "export writer is not configured")
	}
	format := opts.Format
	if format == "" {
		format = ExportFormatCSV
	}
	if format != ExportFormatCSV && format != ExportFormatJSON {
		return nil, errors.New(errors.CodeInvalidParam, "format must be csv or json")
	}

	start, end := opts.Start, opts.End
	if end.IsZero() {
		end = time.Now()
	}
	if start.IsZero() {
		start = end.AddDate(0, 0, -7)
	}
	if !start.Before(end) {
		return nil, errors.New(errors.CodeInvalidParam, "start must be earlier than end")
	}
	if end.Sub(start) > exportMaxRangeDays*24*time.Hour {
		return nil, errors.New(errors.CodeInvalidParam,
			fmt.Sprintf("time range must not exceed %d days", exportMaxRangeDays))
	}

	tenantID := tenant.FromContext(ctx)
	total, err := s.repo.CountRange(ctx, tenantID, start, end)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to count audit logs for export", err)
	}
	if total > exportMaxRows {
		return nil, errors.New(errors.CodeInvalidParam,
			fmt.Sprintf("too many rows to export (%d > %d), narrow the time range", total, exportMaxRows))
	}

	var rows int64
	if format == ExportFormatJSON {
		rows, err = s.exportJSON(ctx, tenantID, start, end, w)
	} else {
		rows, err = s.exportCSV(ctx, tenantID, start, end, w)
	}
	if err != nil {
		return nil, err
	}
	return &ExportSummary{Format: format, Rows: rows, Start: start, End: end}, nil
}

// exportCSV 以 CSV 流式写出
func (s *AuditService) exportCSV(ctx context.Context, tenantID uint64, start, end time.Time, w io.Writer) (int64, error) {
	var written int64
	writer := csv.NewWriter(w)
	if err := writer.Write(exportColumns); err != nil {
		return 0, errors.Wrap(errors.CodeInternalError, "failed to write csv header", err)
	}
	err := s.forEachBatch(ctx, tenantID, start, end, func(logs []domain.AuditLog) error {
		for _, log := range logs {
			if err := writer.Write(csvRow(log)); err != nil {
				return err
			}
			written++
		}
		return nil
	})
	if err != nil {
		return written, err
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return written, errors.Wrap(errors.CodeInternalError, "failed to flush csv export", err)
	}
	return written, nil
}

// exportJSON 以 NDJSON（每行一个 JSON 对象）流式写出
func (s *AuditService) exportJSON(ctx context.Context, tenantID uint64, start, end time.Time, w io.Writer) (int64, error) {
	var written int64
	encoder := json.NewEncoder(w)
	err := s.forEachBatch(ctx, tenantID, start, end, func(logs []domain.AuditLog) error {
		for _, log := range logs {
			if err := encoder.Encode(ToAuditLogResponse(log)); err != nil {
				return err
			}
			written++
		}
		return nil
	})
	if err != nil {
		return written, err
	}
	return written, nil
}

// forEachBatch 分批读取并回调，直到读满一批不足 batchSize 为止
func (s *AuditService) forEachBatch(ctx context.Context, tenantID uint64, start, end time.Time, fn func([]domain.AuditLog) error) error {
	for offset := 0; ; offset += exportBatchSize {
		logs, err := s.repo.ListRange(ctx, tenantID, start, end, offset, exportBatchSize)
		if err != nil {
			return errors.Wrap(errors.CodeInternalError, "failed to read audit logs for export", err)
		}
		if len(logs) == 0 {
			return nil
		}
		if err := fn(logs); err != nil {
			return errors.Wrap(errors.CodeInternalError, "failed to write export", err)
		}
		if len(logs) < exportBatchSize {
			return nil
		}
	}
}

// csvRow 按 exportColumns 顺序输出一行
func csvRow(log domain.AuditLog) []string {
	return []string{
		strconv.FormatUint(log.ID, 10),
		log.CreatedAt.UTC().Format(time.RFC3339),
		strconv.FormatUint(log.TenantID, 10),
		strconv.FormatUint(log.UserID, 10),
		log.Username,
		log.Action,
		log.Resource,
		log.Method,
		log.Path,
		strconv.Itoa(log.Status),
		log.IP,
		log.Detail,
		log.PrevHash,
		log.EntryHash,
	}
}
