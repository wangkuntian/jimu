-- +goose Up
ALTER TABLE roles
    ADD COLUMN code VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN scope VARCHAR(16) NOT NULL DEFAULT 'tenant',
    ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN roles.code IS '系统角色稳定编码；自定义角色为空';
COMMENT ON COLUMN roles.scope IS '角色范围：platform/tenant';
COMMENT ON COLUMN roles.is_system IS '是否系统内置角色';

CREATE INDEX idx_roles_code ON roles (code);
CREATE INDEX idx_roles_scope ON roles (scope);

-- +goose Down
DROP INDEX idx_roles_scope;
DROP INDEX idx_roles_code;

ALTER TABLE roles
    DROP COLUMN is_system,
    DROP COLUMN scope,
    DROP COLUMN code;
