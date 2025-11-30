# Production Deployment Guide - EchoFS with Authentication

## Prerequisites

- Render account (for backend and PostgreSQL)
- Vercel account (for frontend)
- AWS account (for S3 storage - optional)

## Step 1: Deploy PostgreSQL Database on Render

1. Go to [Render Dashboard](https://dashboard.render.com/)
2. Click "New +" → "PostgreSQL"
3. Configure:
   - **Name**: `echofs-db`
   - **Database**: `echofs`
   - **User**: `echofs`
   - **Region**: Oregon (or your preferred region)
   - **Plan**: Free
4. Click "Create Database"
5. **Save the Internal Database URL** - you'll need this

## Step 2: Deploy Backend to Render

### Option A: Using render.yaml (Recommended)

1. Push your code to GitHub
2. Go to Render Dashboard → "New +" → "Blueprint"
3. Connect your GitHub repository
4. Select `Backend/render.yaml`
5. Render will automatically:
   - Create the PostgreSQL database
   - Deploy the backend service
   - Set up environment variables

### Option B: Manual Deployment

1. Go to Render Dashboard → "New +" → "Web Service"
2. Connect your GitHub repository
3. Configure:
   - **Name**: `echofs-backend`
   - **Region**: Oregon
   - **Branch**: main
   - **Root Directory**: `Backend`
   - **Runtime**: Go
   - **Build Command**: `go mod tidy && go build -o server ./cmd/master/server/`
   - **Start Command**: `./server`

4. **Add Environment Variables**:
   ```
   PORT=10000
   MASTER_PORT=10000
   DATABASE_URL=<your-postgres-internal-url>
   JWT_SECRET=<generate-random-secret>
   AWS_REGION=us-east-1
   AWS_ACCESS_KEY_ID=<your-aws-key>
   AWS_SECRET_ACCESS_KEY=<your-aws-secret>
   S3_BUCKET_NAME=<your-s3-bucket>
   WORKER1_URL=echofs-worker1.onrender.com:443
   LOG_LEVEL=info
   ```

5. Click "Create Web Service"

### Generate JWT Secret

```bash
# On macOS/Linux
openssl rand -hex 32

# Or use this
node -e "console.log(require('crypto').randomBytes(32).toString('hex'))"
```

## Step 3: Deploy Workers (Optional - if using multiple workers)

Repeat for each worker:

1. Go to Render Dashboard → "New +" → "Web Service"
2. Configure:
   - **Name**: `echofs-worker1`
   - **Root Directory**: `Backend`
   - **Build Command**: `go mod tidy && go build -o worker1 ./cmd/worker1/`
   - **Start Command**: `./worker1`
   - **Environment Variables**:
     ```
     PORT=9081
     WORKER_ID=worker1
     AWS_REGION=us-east-1
     AWS_ACCESS_KEY_ID=<your-aws-key>
     AWS_SECRET_ACCESS_KEY=<your-aws-secret>
     S3_BUCKET_NAME=<your-s3-bucket>
     ```

## Step 4: Update Frontend for Authentication

### Update Frontend API Client

Create `frontend/lib/api.ts`:

```typescript
const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export interface AuthResponse {
  success: boolean;
  message: string;
  token?: string;
  user?: {
    id: string;
    username: string;
    email: string;
  };
}

export interface FileResponse {
  success: boolean;
  message: string;
  data?: any;
}

class APIClient {
  private getAuthHeader(): HeadersInit {
    const token = localStorage.getItem('authToken');
    return token ? { 'Authorization': `Bearer ${token}` } : {};
  }

  async register(username: string, email: string, password: string): Promise<AuthResponse> {
    const response = await fetch(`${API_URL}/api/v1/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, email, password }),
    });
    return response.json();
  }

  async login(email: string, password: string): Promise<AuthResponse> {
    const response = await fetch(`${API_URL}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    });
    return response.json();
  }

  async getProfile(): Promise<AuthResponse> {
    const response = await fetch(`${API_URL}/api/v1/auth/profile`, {
      headers: this.getAuthHeader(),
    });
    return response.json();
  }

  async uploadFile(file: File): Promise<FileResponse> {
    const formData = new FormData();
    formData.append('file', file);

    const response = await fetch(`${API_URL}/api/v1/files/upload`, {
      method: 'POST',
      headers: this.getAuthHeader(),
      body: formData,
    });
    return response.json();
  }

  async listFiles(): Promise<FileResponse> {
    const response = await fetch(`${API_URL}/api/v1/files`, {
      headers: this.getAuthHeader(),
    });
    return response.json();
  }

  async downloadFile(fileId: string): Promise<Blob> {
    const response = await fetch(`${API_URL}/api/v1/files/${fileId}/download`, {
      headers: this.getAuthHeader(),
    });
    return response.blob();
  }

  async deleteFile(fileId: string): Promise<FileResponse> {
    const response = await fetch(`${API_URL}/api/v1/files/${fileId}`, {
      method: 'DELETE',
      headers: this.getAuthHeader(),
    });
    return response.json();
  }
}

export const api = new APIClient();
```

### Update Frontend Environment Variables

Update `frontend/.env.local`:
```
NEXT_PUBLIC_API_URL=https://echofs-backend.onrender.com
```

For production on Vercel, add this in Vercel dashboard.

## Step 5: Deploy Frontend to Vercel

1. Go to [Vercel Dashboard](https://vercel.com/dashboard)
2. Click "Add New..." → "Project"
3. Import your GitHub repository
4. Configure:
   - **Framework Preset**: Next.js
   - **Root Directory**: `frontend`
   - **Build Command**: `npm run build`
   - **Output Directory**: `.next`

5. **Add Environment Variable**:
   - Key: `NEXT_PUBLIC_API_URL`
   - Value: `https://echofs-backend.onrender.com`

6. Click "Deploy"

## Step 6: Verify Deployment

### Test Backend

```bash
# Health check
curl https://echofs-backend.onrender.com/api/v1/health

# Register user
curl -X POST https://echofs-backend.onrender.com/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"password123"}'

# Login
curl -X POST https://echofs-backend.onrender.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'
```

### Test Frontend

1. Visit your Vercel URL (e.g., `https://your-app.vercel.app`)
2. Register a new account
3. Upload a file
4. Verify file appears in your list
5. Download the file
6. Delete the file

## Step 7: Database Verification

Connect to your Render PostgreSQL database:

```bash
# Get connection string from Render dashboard
psql <your-external-database-url>

# Check tables
\dt

# Check users
SELECT id, username, email, created_at FROM users;

# Check files
SELECT file_id, owner_id, original_name, size FROM files;
```

## Troubleshooting

### Backend Issues

**"Failed to connect to PostgreSQL"**
- Verify DATABASE_URL is set correctly in Render
- Check database is running in Render dashboard
- Ensure using Internal Database URL (not External)

**"No workers available"**
- Workers are optional for basic deployment
- Backend can run without workers for small files
- Deploy workers separately if needed

**"JWT token invalid"**
- Ensure JWT_SECRET is set and consistent
- Don't change JWT_SECRET after users register
- Users will need to re-login if secret changes

### Frontend Issues

**"Network Error"**
- Verify NEXT_PUBLIC_API_URL is correct
- Check CORS is enabled on backend (already configured)
- Ensure backend is deployed and running

**"Unauthorized"**
- Check token is being stored in localStorage
- Verify Authorization header format: `Bearer <token>`
- Token expires after 24 hours - users need to re-login

### Database Issues

**"Table does not exist"**
- Tables are auto-created on first backend startup
- Check backend logs in Render for errors
- Manually create tables if needed (see schema below)

## Manual Database Schema (if needed)

```sql
-- Connect to database
psql <your-database-url>

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(255) PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- Create files table
CREATE TABLE IF NOT EXISTS files (
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

CREATE INDEX IF NOT EXISTS idx_files_owner_id ON files(owner_id);
CREATE INDEX IF NOT EXISTS idx_files_created_at ON files(created_at DESC);
```

## Security Checklist

- [ ] JWT_SECRET is a strong random value (32+ characters)
- [ ] DATABASE_URL uses SSL in production
- [ ] AWS credentials are set as environment variables (not in code)
- [ ] CORS is properly configured
- [ ] Frontend uses HTTPS (Vercel provides this automatically)
- [ ] Backend uses HTTPS (Render provides this automatically)
- [ ] Database backups are enabled in Render
- [ ] Environment variables are not committed to git

## Monitoring

### Render Dashboard
- Monitor backend logs
- Check database connection pool
- View request metrics
- Set up alerts for errors

### Vercel Dashboard
- Monitor frontend deployments
- Check build logs
- View analytics
- Monitor API calls

## Scaling Considerations

### Free Tier Limitations
- Render Free: Spins down after 15 minutes of inactivity
- PostgreSQL Free: 256MB storage, 97 hours/month
- First request after spin-down takes 30-60 seconds

### Upgrade Path
1. **Starter Plan ($7/month)**: No spin-down, better performance
2. **PostgreSQL Paid**: More storage, always-on
3. **Multiple Workers**: Deploy additional workers for load balancing
4. **CDN**: Use Vercel Edge Network for faster frontend delivery

## Backup Strategy

### Database Backups
- Render automatically backs up paid PostgreSQL databases
- For free tier, manually export data:
  ```bash
  pg_dump <database-url> > backup.sql
  ```

### File Storage
- S3 provides 99.999999999% durability
- Enable S3 versioning for file history
- Set up S3 lifecycle policies for old files

## Cost Estimate

### Minimal Setup (Free)
- Render Backend: Free
- Render PostgreSQL: Free (256MB)
- Vercel Frontend: Free
- AWS S3: ~$0.023/GB/month
- **Total**: ~$0-5/month

### Production Setup
- Render Backend Starter: $7/month
- Render PostgreSQL: $7/month
- Vercel Pro: $20/month
- AWS S3: Variable
- **Total**: ~$35-50/month

## Support

- Backend logs: Render Dashboard → echofs-backend → Logs
- Database: Render Dashboard → echofs-db → Connect
- Frontend logs: Vercel Dashboard → Project → Deployments
- Issues: Check GitHub repository issues

## Next Steps

1. Set up monitoring and alerts
2. Configure database backups
3. Add email verification (future enhancement)
4. Implement password reset (future enhancement)
5. Add file sharing between users (future enhancement)
6. Set up CI/CD pipeline
7. Add integration tests
8. Configure custom domain

---

**Deployment Date**: _____________________
**Backend URL**: https://echofs-backend.onrender.com
**Frontend URL**: https://your-app.vercel.app
**Database**: echofs-db on Render
