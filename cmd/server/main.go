package main

import (
	"jimu/internal/app"
	"jimu/internal/config"
	auditmodule "jimu/internal/modules/audit"
	authmodule "jimu/internal/modules/auth"
	"jimu/internal/modules/permission"
	"jimu/internal/modules/role"
	"jimu/internal/modules/user"
	"jimu/internal/platform/db"
)

// @title           Jimu API
// @version         1.0
// @description     Jimu Backend Framework API
// @host            localhost:8080
// @BasePath        /api/v1
func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	dbConn, err := db.New(cfg.DB)
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}

	userModule := user.New(dbConn)
	authModule := authmodule.New(dbConn, cfg.Auth)
	roleModule := role.New(dbConn)
	permModule := permission.New(dbConn)
	auditModule := auditmodule.New(dbConn)

	server := app.Bootstrap(userModule, authModule, roleModule, permModule, auditModule)
	server.Run()
}
