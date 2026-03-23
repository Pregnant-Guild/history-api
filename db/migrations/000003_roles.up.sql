CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name TEXT UNIQUE NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_user_roles_user_id 
ON user_roles (user_id);

CREATE INDEX idx_user_roles_role_id 
ON user_roles (role_id);

CREATE INDEX idx_roles_active
ON roles (name)
WHERE is_deleted = false;

INSERT INTO roles (name) VALUES 
    ('USER'),
    ('ADMIN'),
    ('MOD'),
    ('HISTORIAN'),
    ('BANNED')
ON CONFLICT (name) DO NOTHING;

CREATE TRIGGER trigger_roles_updated_at
BEFORE UPDATE ON roles
FOR EACH ROW
EXECUTE FUNCTION update_updated_at();