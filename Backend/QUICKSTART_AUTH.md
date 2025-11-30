# Quick Start Guide - EchoFS with Authentication

## Prerequisites

1. **PostgreSQL** - Required for user and file metadata storage
2. **Go 1.24+** - For running the backend
3. **AWS Credentials** (optional) - For S3 storage

## Step 1: Install PostgreSQL

### macOS
```bash
brew install postgresql@14
brew services start postgresql@14
```

### Linux (Ubuntu/Debian)
```bash
sudo apt-get update
sudo apt-get install postgresql postgresql-contrib
sudo systemctl start postgresql
```

### Create Database
```bash
# Connect to PostgreSQL
psql postgres

# Create database and user
CREATE DATABASE echofs;
CREATE USER echofs_user WITH PASSWORD 'echofs_password';
GRANT ALL PRIVILEGES ON DATABASE echofs TO echofs_user;
\q
```

## Step 2: Configure Environment

```bash
cd Backend

# Copy example environment file
cp .env.example .env

# Edit .env with your settings
nano .env
```

Update these values in `.env`:
```bash
DATABASE_URL=postgres://echofs_user:echofs_password@localhost:5432/echofs?sslmode=disable
JWT_SECRET=change-this-to-a-random-secret-key
AWS_ACCESS_KEY_ID=your-aws-key  # Optional
AWS_SECRET_ACCESS_KEY=your-aws-secret  # Optional
```

## Step 3: Install Dependencies

```bash
cd Backend
go mod download
```

## Step 4: Start the System

### Terminal 1 - Start Worker
```bash
cd Backend
source .env
WORKER_ID=worker1 WORKER_PORT=8091 go run cmd/worker1/main.go cmd/worker1/server.go
```

### Terminal 2 - Start Master Server
```bash
cd Backend
source .env
go run cmd/master/server/main.go cmd/master/server/server.go
```

You should see:
```
[MASTER] Starting EchoFS Master Node...
[MASTER] Starting server on port 8080
```

## Step 5: Test the System

### Option A: Use the Test Script
```bash
cd Backend
./test/auth_test.sh
```

### Option B: Manual Testing

#### 1. Register a User
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "password": "password123"
  }'
```

Save the token from the response:
```bash
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

#### 2. Upload a File
```bash
echo "Hello EchoFS!" > test.txt

curl -X POST http://localhost:8080/api/v1/files/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@test.txt"
```

#### 3. List Your Files
```bash
curl -X GET http://localhost:8080/api/v1/files \
  -H "Authorization: Bearer $TOKEN"
```

#### 4. Download a File
```bash
# Use the file_id from the list response
FILE_ID="your-file-id-here"

curl -X GET http://localhost:8080/api/v1/files/$FILE_ID/download \
  -H "Authorization: Bearer $TOKEN" \
  -o downloaded.txt
```

## Step 6: Verify Database

```bash
# Connect to database
psql -U echofs_user -d echofs

# Check users
SELECT id, username, email, created_at FROM users;

# Check files
SELECT file_id, owner_id, original_name, size, created_at FROM files;

# Exit
\q
```

## Common Issues

### Issue: "Failed to connect to PostgreSQL"
**Solution**: Ensure PostgreSQL is running
```bash
# macOS
brew services list | grep postgresql

# Linux
sudo systemctl status postgresql
```

### Issue: "Authorization header required"
**Solution**: Make sure you're including the Bearer token
```bash
curl -H "Authorization: Bearer YOUR_TOKEN_HERE" ...
```

### Issue: "User already exists"
**Solution**: Use a different email or login with existing credentials
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"password123"}'
```

### Issue: "Access denied: You don't own this file"
**Solution**: You're trying to access another user's file. Each user can only access their own files.

## Next Steps

1. **Frontend Integration**: Update your frontend to handle authentication
2. **Production Deployment**: See `AUTH_README.md` for production considerations
3. **Advanced Features**: Explore file sharing, permissions, etc.

## API Documentation

For complete API documentation, see `AUTH_README.md`

## Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ JWT Token
       ▼
┌─────────────┐     ┌──────────────┐
│   Master    │────▶│  PostgreSQL  │
│   Server    │     │  (Users +    │
└──────┬──────┘     │   Files)     │
       │            └──────────────┘
       │
       ▼
┌─────────────┐     ┌──────────────┐
│   Workers   │────▶│  S3 Storage  │
│  (Chunks)   │     │  (Optional)  │
└─────────────┘     └──────────────┘
```

## Security Notes

- **Change JWT_SECRET**: Use a strong random key in production
- **Use HTTPS**: Always use TLS/SSL in production
- **Strong Passwords**: Enforce password policies
- **Regular Backups**: Backup PostgreSQL database regularly
- **Token Expiration**: Tokens expire after 24 hours by default

## Support

For issues or questions:
1. Check `AUTH_README.md` for detailed documentation
2. Review error messages in server logs
3. Verify database connectivity
4. Ensure all environment variables are set correctly
