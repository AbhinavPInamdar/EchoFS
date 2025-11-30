# EchoFS Authentication System

## Overview

EchoFS now includes a complete authentication system with user registration, login, and user-specific file access control. Each user can only see and access their own files.

## Features

- **User Registration**: Create new user accounts with username, email, and password
- **User Login**: JWT-based authentication with 24-hour token expiration
- **Protected Routes**: All file operations require authentication
- **User Isolation**: Users can only access their own files
- **Secure Password Storage**: Passwords are hashed using bcrypt
- **Access Control**: File ownership verification on download and delete operations

## Database Setup

### PostgreSQL Required

The authentication system requires PostgreSQL. Set up your database:

```bash
# Install PostgreSQL (macOS)
brew install postgresql
brew services start postgresql

# Create database
createdb echofs

# Or connect to existing PostgreSQL server
psql -U your_username -h localhost
CREATE DATABASE echofs;
```

### Environment Variables

Set the following environment variables:

```bash
export DATABASE_URL="postgres://username:password@localhost:5432/echofs?sslmode=disable"
export JWT_SECRET="your-secret-key-change-in-production"
```

Or add to your `.env` file:

```
DATABASE_URL=postgres://username:password@localhost:5432/echofs?sslmode=disable
JWT_SECRET=your-secret-key-change-in-production
```

## API Endpoints

### Authentication Endpoints (Public)

#### Register New User
```bash
POST /api/v1/auth/register
Content-Type: application/json

{
  "username": "john_doe",
  "email": "john@example.com",
  "password": "securepassword123"
}

Response:
{
  "success": true,
  "message": "User registered successfully",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "uuid",
    "username": "john_doe",
    "email": "john@example.com",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### Login
```bash
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "john@example.com",
  "password": "securepassword123"
}

Response:
{
  "success": true,
  "message": "Login successful",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "uuid",
    "username": "john_doe",
    "email": "john@example.com",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

### Protected Endpoints (Require Authentication)

All protected endpoints require the `Authorization` header:

```
Authorization: Bearer <your_jwt_token>
```

#### Get User Profile
```bash
GET /api/v1/auth/profile
Authorization: Bearer <token>

Response:
{
  "success": true,
  "user": {
    "id": "uuid",
    "username": "john_doe",
    "email": "john@example.com",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### Upload File
```bash
POST /api/v1/files/upload
Authorization: Bearer <token>
Content-Type: multipart/form-data

file: <binary file data>

Response:
{
  "success": true,
  "message": "File uploaded, compressed, and chunked successfully",
  "data": {
    "file_id": "uuid",
    "session_id": "uuid",
    "chunks": 5,
    "compressed": true,
    "file_size": 1024000,
    "owner_id": "user_uuid"
  }
}
```

#### List User's Files
```bash
GET /api/v1/files
Authorization: Bearer <token>

Response:
{
  "success": true,
  "message": "Files listed successfully",
  "data": [
    {
      "file_id": "uuid",
      "name": "document.pdf",
      "size": 1024000,
      "uploaded": "2024-01-01T00:00:00Z",
      "status": "completed",
      "chunks": 5
    }
  ]
}
```

#### Download File
```bash
GET /api/v1/files/{fileId}/download
Authorization: Bearer <token>

Response: Binary file data
```

#### Delete File
```bash
DELETE /api/v1/files/{fileId}
Authorization: Bearer <token>

Response:
{
  "success": true,
  "message": "File deleted successfully"
}
```

## Usage Examples

### 1. Register and Login

```bash
# Register a new user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "password": "password123"
  }'

# Save the token from the response
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# Or login with existing user
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "password123"
  }'
```

### 2. Upload a File

```bash
curl -X POST http://localhost:8080/api/v1/files/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@/path/to/your/file.pdf"
```

### 3. List Your Files

```bash
curl -X GET http://localhost:8080/api/v1/files \
  -H "Authorization: Bearer $TOKEN"
```

### 4. Download a File

```bash
FILE_ID="your-file-id-here"
curl -X GET http://localhost:8080/api/v1/files/$FILE_ID/download \
  -H "Authorization: Bearer $TOKEN" \
  -o downloaded_file.pdf
```

### 5. Delete a File

```bash
curl -X DELETE http://localhost:8080/api/v1/files/$FILE_ID \
  -H "Authorization: Bearer $TOKEN"
```

## Security Features

### Password Security
- Passwords are hashed using bcrypt with default cost (10)
- Plain text passwords are never stored
- Password minimum length: 6 characters

### JWT Tokens
- Tokens expire after 24 hours
- Tokens include user ID, email, and username
- Signed using HMAC-SHA256
- Secret key should be changed in production

### Access Control
- All file operations require authentication
- Users can only access their own files
- File ownership is verified on download and delete
- Database foreign key constraints ensure data integrity

## Database Schema

### Users Table
```sql
CREATE TABLE users (
    id VARCHAR(255) PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### Files Table
```sql
CREATE TABLE files (
    file_id VARCHAR(255) PRIMARY KEY,
    owner_id VARCHAR(255) NOT NULL,
    original_name VARCHAR(500) NOT NULL,
    size BIGINT NOT NULL,
    chunk_size INTEGER NOT NULL,
    total_chunks INTEGER NOT NULL,
    md5_hash VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
);
```

## Error Handling

### Common Error Responses

#### 401 Unauthorized
```json
{
  "success": false,
  "message": "Authorization header required"
}
```

#### 403 Forbidden
```json
{
  "success": false,
  "message": "Access denied: You don't own this file"
}
```

#### 409 Conflict
```json
{
  "success": false,
  "message": "User with this email or username already exists"
}
```

## Testing

### Run the System

```bash
# Terminal 1: Start PostgreSQL (if not running)
brew services start postgresql

# Terminal 2: Start Worker
cd Backend
source ./aws_test_config.sh
WORKER_ID=worker1 WORKER_PORT=8091 go run cmd/worker1/main.go cmd/worker1/server.go

# Terminal 3: Start Master with Database
cd Backend
source ./aws_test_config.sh
export DATABASE_URL="postgres://user:password@localhost:5432/echofs?sslmode=disable"
export JWT_SECRET="test-secret-key"
go run cmd/master/server/main.go cmd/master/server/server.go
```

### Test Authentication Flow

```bash
# 1. Register user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"test123"}'

# 2. Login (save token)
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"test123"}' | jq -r '.token')

# 3. Upload file
curl -X POST http://localhost:8080/api/v1/files/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@test.txt"

# 4. List files
curl -X GET http://localhost:8080/api/v1/files \
  -H "Authorization: Bearer $TOKEN"
```

## Migration from Old System

If you have existing files without user ownership:

1. All new uploads will be associated with authenticated users
2. Old files in the filesystem won't appear in the database
3. Consider running a migration script to associate old files with users

## Production Considerations

1. **Change JWT Secret**: Use a strong, random secret key
2. **Use HTTPS**: Always use TLS in production
3. **Database Backups**: Regular backups of PostgreSQL
4. **Token Refresh**: Consider implementing refresh tokens for better UX
5. **Rate Limiting**: Add rate limiting to prevent abuse
6. **Password Policy**: Enforce stronger password requirements
7. **Email Verification**: Add email verification for new accounts
8. **2FA**: Consider adding two-factor authentication

## Troubleshooting

### Database Connection Issues
```bash
# Check PostgreSQL is running
brew services list | grep postgresql

# Test connection
psql -U your_username -d echofs -c "SELECT 1;"
```

### Token Issues
- Ensure JWT_SECRET is set consistently
- Check token hasn't expired (24 hour default)
- Verify Authorization header format: "Bearer <token>"

### File Access Issues
- Verify user is authenticated
- Check file ownership in database
- Ensure file exists in both database and filesystem
