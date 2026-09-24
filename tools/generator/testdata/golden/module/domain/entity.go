package domain

import (
	"time"

	"gorm.io/gorm"
)

type Product struct {
	ID          uint64         `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:128;uniqueIndex;not null" json:"name"`
	Description string         `gorm:"size:255;default:''" json:"description"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Product) TableName() string {
	return "products"
}
