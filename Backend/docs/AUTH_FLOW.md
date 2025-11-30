# EchoFS Authentication Flow

## System Architecture with Authentication

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client/Frontend                          │
│  (Browser, Mobile App, CLI)                                      │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         │ HTTP/HTTPS
                         │
┌────────────────────────▼────────────────────────────────────────┐
│                      Master Server                               │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              Authentication Middleware                    │  │
│  │  - Validates JWT tokens                                   │  │
│  │  - Extracts user context                                  │  │
│  │  - Protects routes                                        │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐ │
│  │ Auth Handler │  │ File Handler │  │  Worker Registry     │ │
│  │ - Register   │  │ - Upload     │  │  - Health checks     │ │
│  │ - Login      │  │ - Download   │  │  - Load balancing    │ │
│  │ - Profile    │  │ - List       │  │                      │ │
│  └──────┬───────┘  └──────┬───────┘  └──────────┬───────────┘ │
│         │                  │                      │              │
└─────────┼──────────────────┼──────────────────────┼──────────────┘
          │                  │                      │
          │                  │                      │ gRPC
          ▼                  ▼                      ▼
┌─────────────────┐  ┌──────────────┐  ┌──────────────────────┐
│   PostgreSQL    │  │  File System │  │   Worker Nodes       │
│                 │  │              │  │                      │
│  ┌───────────┐  │  │  storage/    │  │  ┌────────────────┐ │
│  │   users   │  │  │  uploads/    │  │  │  Chunk Storage │ │
│  │           │  │  │              │  │  │  (S3/Local)    │ │
│  │  - id     │  │  │  ├─file1/   │  │  └────────────────┘ │
│  │  - email  │  │  │  ├─file2/   │  │                      │
│  │  - pass   │  │  │  └─file3/   │  │  Worker 1, 2, 3...   │
│  └───────────┘  │  │              │  │                      │
│                 │  └──────────────┘  └──────────────────────┘
│  ┌───────────┐  │
│  │   files   │  │
│  │           │  │
│  │  - id     │  │
│  │  - owner  │  │
│  │  - name   │  │
│  └───────────┘  │
└─────────────────┘
```

## Authentication Flow

### 1. User Registration

```
┌────────┐                ┌────────┐                ┌──────────┐
│ Client │                │ Master │                │ Database │
└───┬────┘                └───┬────┘                └────┬─────┘
    │                         │                          │
    │ POST /auth/register     │                          │
    │ {username, email, pass} │                          │
    ├────────────────────────>│                          │
    │                         │                          │
    │                         │ Hash password (bcrypt)   │
    │                         │                          │
    │                         │ INSERT INTO users        │
    │                         ├─────────────────────────>│
    │                         │                          │
    │                         │ User record created      │
    │                         │<─────────────────────────┤
    │                         │                          │
    │                         │ Generate JWT token       │
    │                         │                          │
    │ {success, token, user}  │                          │
    │<────────────────────────┤                          │
    │                         │                          │
    │ Store token locally     │                          │
    │                         │                          │
```

### 2. User Login

```
┌────────┐                ┌────────┐                ┌──────────┐
│ Client │                │ Master │                │ Database │
└───┬────┘                └───┬────┘                └────┬─────┘
    │                         │                          │
    │ POST /auth/login        │                          │
    │ {email, password}       │                          │
    ├────────────────────────>│                          │
    │                         │                          │
    │                         │ SELECT * FROM users      │
    │                         │ WHERE email = ?          │
    │                         ├─────────────────────────>│
    │                         │                          │
    │                         │ User record              │
    │                         │<─────────────────────────┤
    │                         │                          │
    │                         │ Verify password (bcrypt) │
    │                         │                          │
    │                         │ Generate JWT token       │
    │                         │                          │
    │ {success, token, user}  │                          │
    │<────────────────────────┤                          │
    │                         │                          │
    │ Store token locally     │                          │
    │                         │                          │
```

### 3. File Upload (Authenticated)

```
┌────────┐          ┌────────┐          ┌──────────┐          ┌────────┐
│ Client │          │ Master │          │ Database │          │ Worker │
└───┬────┘          └───┬────┘          └────┬─────┘          └───┬────┘
    │                   │                    │                    │
    │ POST /files/upload│                    │                    │
    │ Authorization:    │                    │                    │
    │ Bearer <token>    │                    │                    │
    ├──────────────────>│                    │                    │
    │                   │                    │                    │
    │                   │ Validate JWT       │                    │
    │                   │ Extract user_id    │                    │
    │                   │                    │                    │
    │                   │ Chunk & compress   │                    │
    │                   │                    │                    │
    │                   │ Store chunks       │                    │
    │                   ├───────────────────────────────────────>│
    │                   │                    │                    │
    │                   │                    │ Chunks stored      │
    │                   │<───────────────────────────────────────┤
    │                   │                    │                    │
    │                   │ INSERT INTO files  │                    │
    │                   │ (owner_id=user_id) │                    │
    │                   ├───────────────────>│                    │
    │                   │                    │                    │
    │                   │ File record created│                    │
    │                   │<───────────────────┤                    │
    │                   │                    │                    │
    │ {success, file_id}│                    │                    │
    │<──────────────────┤                    │                    │
    │                   │                    │                    │
```

### 4. File Download (Authenticated)

```
┌────────┐          ┌────────┐          ┌──────────┐          ┌────────┐
│ Client │          │ Master │          │ Database │          │ Worker │
└───┬────┘          └───┬────┘          └────┬─────┘          └───┬────┘
    │                   │                    │                    │
    │ GET /files/{id}/  │                    │                    │
    │ download          │                    │                    │
    │ Authorization:    │                    │                    │
    │ Bearer <token>    │                    │                    │
    ├──────────────────>│                    │                    │
    │                   │                    │                    │
    │                   │ Validate JWT       │                    │
    │                   │ Extract user_id    │                    │
    │                   │                    │                    │
    │                   │ Check ownership    │                    │
    │                   │ WHERE file_id=?    │                    │
    │                   │ AND owner_id=?     │                    │
    │                   ├───────────────────>│                    │
    │                   │                    │                    │
    │                   │ Ownership verified │                    │
    │                   │<───────────────────┤                    │
    │                   │                    │                    │
    │                   │ Retrieve chunks    │                    │
    │                   ├───────────────────────────────────────>│
    │                   │                    │                    │
    │                   │                    │ Chunks data        │
    │                   │<───────────────────────────────────────┤
    │                   │                    │                    │
    │                   │ Decompress & merge │                    │
    │                   │                    │                    │
    │ File data         │                    │                    │
    │<──────────────────┤                    │                    │
    │                   │                    │                    │
```

### 5. File Listing (User-Specific)

```
┌────────┐                ┌────────┐                ┌──────────┐
│ Client │                │ Master │                │ Database │
└───┬────┘                └───┬────┘                └────┬─────┘
    │                         │                          │
    │ GET /files              │                          │
    │ Authorization:          │                          │
    │ Bearer <token>          │                          │
    ├────────────────────────>│                          │
    │                         │                          │
    │                         │ Validate JWT             │
    │                         │ Extract user_id          │
    │                         │                          │
    │                         │ SELECT * FROM files      │
    │                         │ WHERE owner_id = ?       │
    │                         ├─────────────────────────>│
    │                         │                          │
    │                         │ User's files only        │
    │                         │<─────────────────────────┤
    │                         │                          │
    │ {success, data: [...]}  │                          │
    │<────────────────────────┤                          │
    │                         │                          │
```

## JWT Token Structure

```
Header:
{
  "alg": "HS256",
  "typ": "JWT"
}

Payload:
{
  "user_id": "uuid-here",
  "email": "user@example.com",
  "username": "johndoe",
  "exp": 1234567890,  // Expiration (24h from issue)
  "iat": 1234567890,  // Issued at
  "nbf": 1234567890   // Not before
}

Signature:
HMACSHA256(
  base64UrlEncode(header) + "." +
  base64UrlEncode(payload),
  JWT_SECRET
)
```

## Security Layers

```
┌─────────────────────────────────────────────────────────┐
│                    Request Flow                          │
└─────────────────────────────────────────────────────────┘

1. Client Request
   ↓
   [HTTPS/TLS Layer]
   ↓
2. CORS Middleware
   ↓
   [Origin Validation]
   ↓
3. Auth Middleware
   ↓
   [JWT Validation]
   ├─ Token present?
   ├─ Token valid?
   ├─ Token expired?
   └─ Extract user context
   ↓
4. Route Handler
   ↓
   [Business Logic]
   ├─ Check ownership
   ├─ Validate input
   └─ Process request
   ↓
5. Database Layer
   ↓
   [Data Access]
   ├─ Parameterized queries
   ├─ Foreign key constraints
   └─ Row-level security
   ↓
6. Response
```

## Access Control Matrix

```
┌──────────────────┬──────────┬──────────┬──────────┐
│ Endpoint         │ Public   │ User     │ Admin    │
├──────────────────┼──────────┼──────────┼──────────┤
│ /auth/register   │    ✓     │    ✓     │    ✓     │
│ /auth/login      │    ✓     │    ✓     │    ✓     │
│ /health          │    ✓     │    ✓     │    ✓     │
├──────────────────┼──────────┼──────────┼──────────┤
│ /auth/profile    │    ✗     │    ✓     │    ✓     │
│ /files/upload    │    ✗     │    ✓     │    ✓     │
│ /files           │    ✗     │  Own     │   All    │
│ /files/{id}      │    ✗     │  Own     │   All    │
│ /files/{id} DEL  │    ✗     │  Own     │   All    │
└──────────────────┴──────────┴──────────┴──────────┘

Legend:
✓ = Allowed
✗ = Denied
Own = Only user's own resources
All = All resources
```

## Error Handling Flow

```
Request
  │
  ├─ No Auth Header ──────────> 401 Unauthorized
  │
  ├─ Invalid Token ───────────> 401 Unauthorized
  │
  ├─ Expired Token ───────────> 401 Unauthorized
  │
  ├─ Valid Token
  │   │
  │   ├─ Resource Not Found ──> 404 Not Found
  │   │
  │   ├─ Not Owner ───────────> 403 Forbidden
  │   │
  │   ├─ Invalid Input ───────> 400 Bad Request
  │   │
  │   └─ Success ─────────────> 200 OK / 201 Created
  │
  └─ Server Error ────────────> 500 Internal Server Error
```

## Token Lifecycle

```
┌──────────────┐
│ User Registers│
│  or Logs In  │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Token Created│
│  (24h TTL)   │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Token Stored │
│  (Client)    │
└──────┬───────┘
       │
       ▼
┌──────────────────────────────┐
│ Token Used for Requests      │
│ (Included in Auth Header)    │
└──────┬───────────────────────┘
       │
       ├─────────────────┐
       │                 │
       ▼                 ▼
┌──────────────┐  ┌──────────────┐
│ Token Valid  │  │ Token Expired│
│ (< 24h old)  │  │ (> 24h old)  │
└──────┬───────┘  └──────┬───────┘
       │                 │
       ▼                 ▼
┌──────────────┐  ┌──────────────┐
│ Request OK   │  │ 401 Error    │
└──────────────┘  │ Re-login     │
                  └──────────────┘
```

## Database Relationships

```
┌─────────────────────┐
│       users         │
│─────────────────────│
│ id (PK)             │◄──────┐
│ username (UNIQUE)   │       │
│ email (UNIQUE)      │       │
│ password_hash       │       │
│ created_at          │       │
│ updated_at          │       │
└─────────────────────┘       │
                              │
                              │ Foreign Key
                              │ ON DELETE CASCADE
                              │
┌─────────────────────┐       │
│       files         │       │
│─────────────────────│       │
│ file_id (PK)        │       │
│ owner_id (FK) ──────┼───────┘
│ original_name       │
│ size                │
│ chunk_size          │
│ total_chunks        │
│ md5_hash            │
│ status              │
│ created_at          │
│ updated_at          │
└─────────────────────┘

Indexes:
- users.email (UNIQUE)
- users.username (UNIQUE)
- files.owner_id (for fast user file lookups)
- files.created_at (for sorting)
```

## Conclusion

This authentication system provides:
- ✅ Secure user authentication
- ✅ JWT-based stateless sessions
- ✅ User-specific file isolation
- ✅ Ownership verification
- ✅ Database-backed persistence
- ✅ Scalable architecture

For implementation details, see:
- [AUTH_README.md](../AUTH_README.md) - Complete API documentation
- [QUICKSTART_AUTH.md](../QUICKSTART_AUTH.md) - Setup guide
- [AUTH_IMPLEMENTATION_SUMMARY.md](../AUTH_IMPLEMENTATION_SUMMARY.md) - Technical details
