# Migration Guide: Adding Authentication to Existing EchoFS Installation

## Overview

This guide helps you migrate an existing EchoFS installation to the new authentication system.

## What Changed

### Before (No Auth)
- Anyone could upload/download files
- No user accounts
- Files stored without ownership
- No access control

### After (With Auth)
- Users must register/login
- JWT-based authentication
- Files are owned by users
- Users can only access their own files
- PostgreSQL database required

## Migration Steps

### Step 1: Backup Existing Data

```bash
# Backup existing file storage
cd Backend
tar -czf echofs_backup_$(date +%Y%m%d).tar.gz storage/

# If you have any existing database
pg_dump echofs > echofs_backup_$(date +%Y%m%d).sql
```

### Step 2: Install PostgreSQL

#### macOS
```bash
brew install postgresql@14
brew services start postgresql@14
```

#### Linux
```bash
sudo apt-get update
sudo apt-get install postgresql postgresql-contrib
sudo systemctl start postgresql
```

### Step 3: Create Database

```bash
# Connect to PostgreSQL
psql postgres

# Create database and user
CREATE DATABASE echofs;
CREATE USER echofs_user WITH PASSWORD 'echofs_password';
GRANT ALL PRIVILEGES ON DATABASE echofs TO echofs_user;
\q
```

### Step 4: Update Dependencies

```bash
cd Backend
go mod tidy
```

### Step 5: Update Configuration

```bash
# Update .env file
cat >> .env << EOF
DATABASE_URL=postgres://echofs_user:echofs_password@localhost:5432/echofs?sslmode=disable
JWT_SECRET=$(openssl rand -hex 32)
EOF
```

### Step 6: Start the System

The new system will automatically create the required database tables on first start.

```bash
# Terminal 1: Start Worker
source .env
WORKER_ID=worker1 WORKER_PORT=8091 go run cmd/worker1/main.go cmd/worker1/server.go

# Terminal 2: Start Master
source .env
go run cmd/master/server/main.go cmd/master/server/server.go
```

### Step 7: Verify Database Schema

```bash
psql -U echofs_user -d echofs

# Check tables were created
\dt

# Should show:
#  public | files | table | echofs_user
#  public | users | table | echofs_user

\q
```

## Handling Existing Files

### Option 1: Fresh Start (Recommended)

Start fresh with the new authentication system:

1. Users register new accounts
2. Users re-upload their files
3. Old files remain in `storage/` but aren't accessible via API
4. Keep old files as backup

### Option 2: Assign Ownership to Existing Files

If you need to preserve existing files, create a migration script:

```bash
# Create migration script
cat > migrate_files.sh << 'EOF'
#!/bin/bash

# This script assigns ownership of existing files to a user
# Usage: ./migrate_files.sh <user_email>

USER_EMAIL=$1
if [ -z "$USER_EMAIL" ]; then
    echo "Usage: $0 <user_email>"
    exit 1
fi

# Get user ID from database
USER_ID=$(psql -U echofs_user -d echofs -t -c "SELECT id FROM users WHERE email='$USER_EMAIL'")

if [ -z "$USER_ID" ]; then
    echo "User not found: $USER_EMAIL"
    exit 1
fi

# Scan existing files and add to database
for dir in storage/uploads/*/; do
    FILE_ID=$(basename "$dir")
    
    # Get file info
    for file in "$dir"*; do
        if [ -f "$file" ] && [[ ! "$file" =~ \.gz$ ]]; then
            FILENAME=$(basename "$file")
            FILESIZE=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file")
            
            # Insert into database
            psql -U echofs_user -d echofs << SQL
INSERT INTO files (file_id, owner_id, original_name, size, chunk_size, total_chunks, status, created_at, updated_at)
VALUES ('$FILE_ID', '$USER_ID', '$FILENAME', $FILESIZE, 1048576, 1, 'completed', NOW(), NOW())
ON CONFLICT (file_id) DO NOTHING;
SQL
            
            echo "Migrated: $FILENAME (ID: $FILE_ID)"
            break
        fi
    done
done

echo "Migration complete!"
EOF

chmod +x migrate_files.sh
```

Usage:
```bash
# First, create a user account via API or directly in database
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","email":"admin@example.com","password":"password123"}'

# Then run migration
./migrate_files.sh admin@example.com
```

### Option 3: Create Admin User with All Files

```sql
-- Connect to database
psql -U echofs_user -d echofs

-- Create admin user (password: admin123)
INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
VALUES (
    'admin-user-id',
    'admin',
    'admin@example.com',
    '$2a$10$YourHashedPasswordHere',  -- Use bcrypt to hash 'admin123'
    NOW(),
    NOW()
);

-- Assign all existing files to admin
-- (You'll need to scan filesystem and insert records)
```

## API Changes

### Before
```bash
# Upload (no auth)
curl -X POST http://localhost:8080/api/v1/files/upload \
  -F "file=@test.txt"

# Download (no auth)
curl http://localhost:8080/api/v1/files/FILE_ID/download
```

### After
```bash
# Register/Login first
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password"}' \
  | jq -r '.token')

# Upload (with auth)
curl -X POST http://localhost:8080/api/v1/files/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@test.txt"

# Download (with auth)
curl http://localhost:8080/api/v1/files/FILE_ID/download \
  -H "Authorization: Bearer $TOKEN"
```

## Frontend Changes

### Before
```javascript
// Upload without auth
fetch('http://localhost:8080/api/v1/files/upload', {
    method: 'POST',
    body: formData
});
```

### After
```javascript
// Login first
const loginResponse = await fetch('http://localhost:8080/api/v1/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password })
});
const { token } = await loginResponse.json();

// Store token
localStorage.setItem('authToken', token);

// Upload with auth
fetch('http://localhost:8080/api/v1/files/upload', {
    method: 'POST',
    headers: { 'Authorization': `Bearer ${token}` },
    body: formData
});
```

See `examples/frontend_auth_example.html` for complete example.

## Testing the Migration

### 1. Test User Registration
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"test123"}'
```

### 2. Test Login
```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"test123"}' \
  | jq -r '.token')

echo "Token: $TOKEN"
```

### 3. Test File Upload
```bash
echo "Test content" > test.txt
curl -X POST http://localhost:8080/api/v1/files/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@test.txt"
```

### 4. Test File Listing
```bash
curl http://localhost:8080/api/v1/files \
  -H "Authorization: Bearer $TOKEN"
```

### 5. Run Full Test Suite
```bash
./test/auth_test.sh
```

## Rollback Plan

If you need to rollback to the old system:

### 1. Stop the New System
```bash
# Kill running processes
pkill -f "go run cmd/master"
pkill -f "go run cmd/worker"
```

### 2. Restore Old Code
```bash
git checkout <previous-commit>
go mod tidy
```

### 3. Restore Files
```bash
# If you backed up
tar -xzf echofs_backup_YYYYMMDD.tar.gz
```

### 4. Start Old System
```bash
# Start without DATABASE_URL and JWT_SECRET
go run cmd/master/server/main.go cmd/master/server/server.go
```

## Troubleshooting

### Issue: "Failed to connect to PostgreSQL"
**Solution**: Ensure PostgreSQL is running and DATABASE_URL is correct
```bash
brew services list | grep postgresql
psql -U echofs_user -d echofs -c "SELECT 1"
```

### Issue: "Table already exists"
**Solution**: The system auto-creates tables. If you see this error, tables already exist.
```bash
psql -U echofs_user -d echofs -c "\dt"
```

### Issue: "Old files not showing up"
**Solution**: Old files don't have owners. Either:
1. Re-upload files with new accounts
2. Run migration script to assign ownership
3. Manually insert records into `files` table

### Issue: "Token invalid"
**Solution**: Ensure JWT_SECRET is consistent across restarts
```bash
# Add to .env permanently
echo "JWT_SECRET=$(openssl rand -hex 32)" >> .env
```

## Performance Impact

### Database Queries
- Additional query on file upload (insert metadata)
- Additional query on file list (fetch from database)
- Additional query on download (verify ownership)
- Typical overhead: 1-5ms per operation

### Memory Usage
- PostgreSQL: ~50-100MB base
- Connection pool: ~25 connections
- Minimal impact on Go application

### Storage
- Database size: ~1KB per user, ~500 bytes per file
- No change to file storage size

## Security Considerations

### Before Migration
- [ ] Change JWT_SECRET to a strong random value
- [ ] Use strong database password
- [ ] Enable SSL for PostgreSQL in production
- [ ] Review firewall rules
- [ ] Set up database backups

### After Migration
- [ ] Test authentication flow
- [ ] Verify file isolation
- [ ] Check unauthorized access is blocked
- [ ] Review logs for errors
- [ ] Monitor database performance

## Support

For issues during migration:
1. Check logs: `tail -f /var/log/echofs.log`
2. Check database: `psql -U echofs_user -d echofs`
3. Review [AUTH_README.md](AUTH_README.md)
4. Run test suite: `./test/auth_test.sh`

## Next Steps

After successful migration:
1. Update frontend to use authentication
2. Train users on new login process
3. Set up monitoring for auth failures
4. Plan for token refresh implementation
5. Consider adding email verification
6. Implement password reset flow

## Conclusion

The migration adds significant security and multi-user capabilities to EchoFS. While it requires PostgreSQL and changes to the API, the benefits of user isolation and access control are substantial.

For production deployments, carefully review the security considerations and test thoroughly before going live.
