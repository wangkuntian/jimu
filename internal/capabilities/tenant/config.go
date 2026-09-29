package tenant

import "jimu/internal/capabilities/tenant/application"

// 开通式注册配置类型对外公开（实际定义在 application 包，供 provisioner 消费）。
//
// `auth.provisioning` 段由 auth 能力拥有（设计 §8 ¶2 不拆段）；tenant 被 auth 依赖，
// 不得 import auth，故组合根从 auth 段取值构造本类型后传入 tenant.New。
type (
	ProvisioningConfig    = application.ProvisioningConfig
	ProvisionRoleTemplate = application.ProvisionRoleTemplate
	ProvisionPermission   = application.ProvisionPermission
)
