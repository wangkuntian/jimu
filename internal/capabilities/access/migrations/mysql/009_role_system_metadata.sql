-- +goose Up
ALTER TABLE roles
    ADD COLUMN code VARCHAR(64) NOT NULL DEFAULT '' COMMENT '系统角色稳定编码；自定义角色为空' AFTER description,
    ADD COLUMN scope VARCHAR(16) NOT NULL DEFAULT 'tenant' COMMENT '角色范围：platform/tenant' AFTER code,
    ADD COLUMN is_system TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否系统内置角色' AFTER scope;

CREATE INDEX idx_roles_code ON roles (code);
CREATE INDEX idx_roles_scope ON roles (scope);

-- +goose Down
DROP INDEX idx_roles_scope ON roles;
DROP INDEX idx_roles_code ON roles;

ALTER TABLE roles
    DROP COLUMN is_system,
    DROP COLUMN scope,
    DROP COLUMN code;
