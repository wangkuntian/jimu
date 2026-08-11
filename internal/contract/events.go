package contract

// 领域事件名称
const (
	EventUserCreated  = "user.created"
	EventUserUpdated  = "user.updated"
	EventUserDeleted  = "user.deleted"
	EventUserLoggedIn = "user.logged_in"
)

// 通知事件名称
const (
	UserCreatedEmailNotification = "notification.user.created"
	UserDeletedEventLog          = "event.user.deleted"
)

// UserCreatedEvent 用户创建事件
type UserCreatedEvent struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
}

// UserUpdatedEvent 用户更新事件
type UserUpdatedEvent struct {
	UserID  uint64   `json:"user_id"`
	Changes []string `json:"changes"` // 变更字段列表
}

// UserDeletedEvent 用户删除事件
type UserDeletedEvent struct {
	UserID uint64 `json:"user_id"`
}

// UserLoggedInEvent 用户登录成功事件
type UserLoggedInEvent struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
}
