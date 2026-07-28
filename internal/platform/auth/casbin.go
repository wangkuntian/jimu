package auth

import (
	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

func NewEnforcer(db *gorm.DB) (*casbin.Enforcer, error) {
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, err
	}
	enforcer, err := casbin.NewEnforcer("conf/rbac_model.conf", adapter)
	if err != nil {
		return nil, err
	}
	return enforcer, nil
}
