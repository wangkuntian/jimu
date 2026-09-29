package main

import (
	"fmt"
	"os"

	"jimu/internal/assembly"
	"jimu/internal/profiles/active"
)

// @title           Jimu API
// @version         1.0
// @description     Jimu 后端框架 API - 提供用户认证、权限管理、角色管理、系统监控等功能
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization

// version 版本号，通过 ldflags 注入：-ldflags "-X main.version=v0.1.0"
var version = "dev"

// main 是唯一入口：当前形态由 internal/profiles/active 决定（提交态默认 full），
// 构建期用 `PROFILE=<name> make build-server` 切形态；装配与生命周期归 internal/assembly。
func main() {
	a := active.Assembly()
	a.Version = version
	if err := assembly.Run(a); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
