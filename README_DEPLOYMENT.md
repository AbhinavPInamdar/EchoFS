# EchoFS Deployment Guide

## Current Status
- **Frontend**: ✅ Successfully deployed on Vercel
- **Backend**: 🔄 Ready for native Go deployment on Render

## Backend Deployment on Render (Native Go)

### Manual Deployment Steps:

1. Go to https://render.com
2. Delete your current service (if it exists)
3. Click 'New +' → 'Web Service'
4. Connect your GitHub repository
5. Use these settings:
   - **Name**: echofs-backend
   - **Environment**: Go (NOT Docker!)
   - **Root Directory**: Backend
   - **Build Command**: `go mod tidy && go build -o server ./cmd/master/server/`
   - **Start Command**: `./server`
   - **Port**: 10000

6. Add these environment variables:
   ```
   PORT=10000
   MASTER_PORT=10000
   AWS_ACCESS_KEY_ID=[YOUR_AWS_ACCESS_KEY]
   AWS_SECRET_ACCESS_KEY=[YOUR_AWS_SECRET_KEY]
   AWS_REGION=us-east-1
   S3_BUCKET_NAME=echofs-storage-1761369400
   DYNAMODB_FILES_TABLE=echofs-files
   DYNAMODB_CHUNKS_TABLE=echofs-chunks
   DYNAMODB_SESSIONS_TABLE=echofs-sessions
   JWT_SECRET=[YOUR_JWT_SECRET]
   LOG_LEVEL=info
   ```

7. Click 'Create Web Service'

## Key Changes Made:
- Removed Docker completely
- Using native Go environment on Render
- Simplified build process
- Removed Redis dependency (DynamoDB only)
- Made DATABASE_URL optional

## Architecture:
```
Internet → Vercel (Frontend) → Render (Backend) → AWS (S3 + DynamoDB)
```

Expected deployment time: 2-3 minutes