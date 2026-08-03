package contract

import (
	"context"

	"github.com/gin-gonic/gin"
)

type Component interface {
	Start(context.Context) error
	Stop(context.Context) error
}

type ErrorSource interface {
	Errors() <-chan error
}

type ComponentProvider interface {
	Components() []Component
}

type HTTPMiddlewareProvider interface {
	HTTPMiddleware() []gin.HandlerFunc
}

type ProtectedHTTPMiddlewareProvider interface {
	ProtectedHTTPMiddleware() ([]gin.HandlerFunc, error)
}

// Router 抽象路由注册器
type Router interface {
	GET(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	POST(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	PUT(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	DELETE(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	Group(relativePath string, handlers ...gin.HandlerFunc) *gin.RouterGroup
}

// JobRegistry 定时任务注册器
type JobRegistry interface {
	AddFunc(spec string, cmd func()) error
}

// EventBus 事件总线
type EventBus interface {
	Subscribe(event string, handler func(payload interface{}))
	Publish(event string, payload interface{})
}

// Module 统一模块接口
type Module interface {
	Name() string
	RegisterHTTP(r Router)
	RegisterJobs(j JobRegistry)
	RegisterEvents(e EventBus)
}
