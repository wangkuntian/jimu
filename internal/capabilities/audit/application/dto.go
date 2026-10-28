package application

import (
	"strconv"
	"time"

	"jimu/internal/capabilities/audit/domain"
)

type AuditLogResponse struct {
	ID        uint64    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	UserID    uint64    `json:"user_id"`
	Username  string    `json:"username"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Detail    string    `json:"detail"`
	IP        string    `json:"ip"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type AuditLogPageResponse struct {
	Code      int                `json:"code"`
	Message   string             `json:"message"`
	Data      []AuditLogResponse `json:"data"`
	Total     int64              `json:"total"`
	Page      int                `json:"page"`
	PageSize  int                `json:"page_size"`
	RequestID string             `json:"request_id,omitempty"`
}

type AuditLogBodyResponse struct {
	Code      int              `json:"code"`
	Message   string           `json:"message"`
	Data      AuditLogResponse `json:"data"`
	RequestID string           `json:"request_id,omitempty"`
}

func ToAuditLogResponse(log domain.AuditLog) AuditLogResponse {
	return AuditLogResponse{
		ID:        log.ID,
		TenantID:  strconv.FormatUint(log.TenantID, 10),
		UserID:    log.UserID,
		Username:  log.Username,
		Action:    log.Action,
		Resource:  log.Resource,
		Detail:    log.Detail,
		IP:        log.IP,
		Method:    log.Method,
		Path:      log.Path,
		Status:    log.Status,
		CreatedAt: log.CreatedAt,
	}
}

func ToAuditLogResponses(logs []domain.AuditLog) []AuditLogResponse {
	out := make([]AuditLogResponse, 0, len(logs))
	for _, log := range logs {
		out = append(out, ToAuditLogResponse(log))
	}
	return out
}
