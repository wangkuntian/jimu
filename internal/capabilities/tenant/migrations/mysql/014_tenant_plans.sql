-- +goose Up
CREATE TABLE IF NOT EXISTS tenant_plans (
    id BIGINT UNSIGNED NOT NULL COMMENT '套餐ID',
    code VARCHAR(32) NOT NULL COMMENT '套餐编码（全局唯一，小写）',
    name VARCHAR(64) NOT NULL COMMENT '套餐名称',
    max_users INT NOT NULL DEFAULT 0 COMMENT '用户数上限（0=不限）',
    max_roles INT NOT NULL DEFAULT 0 COMMENT '角色数上限（0=不限）',
    max_api_keys INT NOT NULL DEFAULT 0 COMMENT 'API Key 数量上限（0=不限）',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (id),
    UNIQUE INDEX uk_tenant_plans_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='租户套餐表（配额上限，0 表示不限）';

ALTER TABLE tenants
    ADD COLUMN plan_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '套餐ID（0=未分配，不受配额限制）';

-- +goose Down
ALTER TABLE tenants DROP COLUMN plan_id;
DROP TABLE IF EXISTS tenant_plans;
