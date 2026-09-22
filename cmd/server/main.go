package main

import (
	"fmt"
	"os"

	"jimu/internal/assembly"
	"jimu/internal/profiles/full"
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

// main 是薄包装：能力清单归 internal/profiles/full，装配与生命周期归 internal/assembly。
// 构建版本仍由本包经 ldflags 注入后覆盖清单默认值（Makefile 的 `-X main.version` 不变）。
func main() {
	a := full.Assembly()
	a.Version = version
	if err := assembly.Run(a); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
