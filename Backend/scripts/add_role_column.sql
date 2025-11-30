-- Add role column to existing users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(50) NOT NULL DEFAULT 'user';

-- Update existing admin user if exists
UPDATE users SET role = 'admin' WHERE username = 'admin';

-- Create index on role for faster queries
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
