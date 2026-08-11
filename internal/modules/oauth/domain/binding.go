// internal/modules/oauth/domain/binding.go
package domain

import "time"

// OAuthBinding 第三方登录绑定记录
type OAuthBinding struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	UserID    uint64    `gorm:"not null" json:"user_id"`
	Provider  string    `gorm:"size:32;not null" json:"provider"`
	Subject   string    `gorm:"size:128;not null" json:"subject"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 表名
func (OAuthBinding) TableName() string { return "user_oauth_bindings" }
