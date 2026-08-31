package domain

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID          uint64         `gorm:"primaryKey" json:"id"`
	Username    string         `gorm:"size:64;uniqueIndex;not null" json:"username"`
	Password    string         `gorm:"size:255;not null" json:"-"`
	Email       string         `gorm:"type:text" encryption:"true" json:"email"`   // AES-GCM 密文（encryption_key 配置时）
	EmailHash   *string        `gorm:"size:64;uniqueIndex" blind:"email" json:"-"` // 空邮箱时 NULL，多个空值不冲突
	Phone       string         `gorm:"type:text" encryption:"true" json:"phone"`
	PhoneHash   *string        `gorm:"size:64;uniqueIndex" blind:"phone" json:"-"` // 空手机号时 NULL，多个空值不冲突
	TOTPSecret  string         `gorm:"type:text" encryption:"true" json:"-"`       // TOTP 密钥（base32，AES-GCM 密文）
	TOTPEnabled bool           `gorm:"default:false" json:"totp_enabled"`          // 是否启用 TOTP 二次验证
	Status      int8           `gorm:"default:1" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}
