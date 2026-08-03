package application

import (
	"time"

	"jimu/internal/modules/audit/domain"
)

type AuditLogResponse struct {
	ID        uint64    `json:"id"`
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

func ToAuditLogResponse(log domain.AuditLog) AuditLogResponse {
	return AuditLogResponse{
		ID:        log.ID,
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
