package auth

import (
	"os"
	"path/filepath"

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

func NewPathEnforcer() (*casbin.Enforcer, error) {
	for _, path := range []string{
		filepath.Join("conf", "rbac_model.conf"),
		filepath.Join("..", "..", "..", "conf", "rbac_model.conf"),
		filepath.Join("..", "..", "..", "..", "conf", "rbac_model.conf"),
	} {
		if _, err := os.Stat(path); err == nil {
			return casbin.NewEnforcer(path)
		}
	}
	return casbin.NewEnforcer(filepath.Join("conf", "rbac_model.conf"))
}
