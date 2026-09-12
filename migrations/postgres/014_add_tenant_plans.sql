-- +goose Up
CREATE TABLE IF NOT EXISTS tenant_plans (
    id BIGINT NOT NULL,
    code VARCHAR(32) NOT NULL,
    name VARCHAR(64) NOT NULL,
    max_users INT NOT NULL DEFAULT 0,
    max_roles INT NOT NULL DEFAULT 0,
    max_api_keys INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_tenant_plans_code ON tenant_plans (code);

COMMENT ON TABLE tenant_plans IS '租户套餐表（配额上限，0 表示不限）';
COMMENT ON COLUMN tenant_plans.code IS '套餐编码（全局唯一，小写）';
COMMENT ON COLUMN tenant_plans.name IS '套餐名称';
COMMENT ON COLUMN tenant_plans.max_users IS '用户数上限（0=不限）';
COMMENT ON COLUMN tenant_plans.max_roles IS '角色数上限（0=不限）';
COMMENT ON COLUMN tenant_plans.max_api_keys IS 'API Key 数量上限（0=不限）';
COMMENT ON COLUMN tenant_plans.created_at IS '创建时间';
COMMENT ON COLUMN tenant_plans.updated_at IS '更新时间';

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS plan_id BIGINT NOT NULL DEFAULT 0;
COMMENT ON COLUMN tenants.plan_id IS '套餐ID（0=未分配，不受配额限制）';

-- +goose Down
ALTER TABLE tenants DROP COLUMN IF EXISTS plan_id;
DROP TABLE IF EXISTS tenant_plans;
