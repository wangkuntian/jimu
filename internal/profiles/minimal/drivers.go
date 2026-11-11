package minimal

// auth 存储驱动：minimal 形态同时编入 Redis 与内存实现，
// 无 Redis 客户端时由 auth.Wire 告警后降级到内存。
import (
	_ "jimu/internal/capabilities/auth/memory"
	_ "jimu/internal/capabilities/auth/redis"
)
