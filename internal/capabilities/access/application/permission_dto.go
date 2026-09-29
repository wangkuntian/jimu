package application

type CreatePermissionRequest struct {
	Name     string `json:"name" binding:"required"`
	Resource string `json:"resource" binding:"required"`
	Action   string `json:"action" binding:"required"`
}

type UpdatePermissionRequest struct {
	Name     string `json:"name" binding:"required"`
	Resource string `json:"resource" binding:"required"`
	Action   string `json:"action" binding:"required"`
}
