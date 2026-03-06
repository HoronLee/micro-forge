CREATE TABLE IF NOT EXISTS users (
    "id" BIGSERIAL PRIMARY KEY, -- 用户ID，PostgreSQL 自增主键
    "name" VARCHAR(64) NOT NULL UNIQUE, -- 用户名
    "email" VARCHAR(128) NOT NULL UNIQUE, -- 用户邮箱，唯一
    "password" VARCHAR(255) NOT NULL, -- 用户密码（加密存储）
    "role" VARCHAR(32) NOT NULL DEFAULT 'user', -- 用户权限角色
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, -- 创建时间（带时区）
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP -- 更新时间（带时区）
);

-- 创建触发器，在更新用户记录时自动更新 updated_at 字段
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- 创建触发器函数来模拟 MySQL 的 ON UPDATE CURRENT_TIMESTAMP
CREATE TRIGGER trigger_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
