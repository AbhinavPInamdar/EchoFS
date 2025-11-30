# 🚀 EchoFS Production Deployment Checklist

## Pre-Deployment

### Code Preparation
- [x] Authentication system implemented
- [x] Database schema created
- [x] API endpoints tested locally
- [x] Frontend API client created
- [ ] Remove unnecessary documentation files
- [ ] Update .gitignore for sensitive files

### Environment Setup
- [ ] Render account created
- [ ] Vercel account created
- [ ] AWS account configured (optional)
- [ ] GitHub repository up to date

## Backend Deployment (Render)

### 1. PostgreSQL Database
- [ ] Create PostgreSQL database on Render
  - Name: `echofs-db`
  - Plan: Free (or Starter for production)
  - Region: Oregon (or preferred)
- [ ] Save Internal Database URL
- [ ] Verify database is running

### 2. Backend Service
- [ ] Push code to GitHub
- [ ] Create Web Service on Render
  - Connect GitHub repository
  - Root Directory: `Backend`
  - Build Command: `go mod tidy && go build -o server ./cmd/master/server/`
  - Start Command: `./server`
- [ ] Configure Environment Variables:
  ```
  PORT=10000
  MASTER_PORT=10000
  DATABASE_URL=<from-database>
  JWT_SECRET=<generate-random-32-chars>
  AWS_REGION=us-east-1
  AWS_ACCESS_KEY_ID=<your-key>
  AWS_SECRET_ACCESS_KEY=<your-secret>
  S3_BUCKET_NAME=<your-bucket>
  LOG_LEVEL=info
  ```
- [ ] Deploy and wait for build to complete
- [ ] Check logs for errors
- [ ] Verify database tables were created

### 3. Test Backend
```bash
# Health check
curl https://your-backend.onrender.com/api/v1/health

# Register test user
curl -X POST https://your-backend.onrender.com/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"test","email":"test@example.com","password":"test123"}'

# Login
curl -X POST https://your-backend.onrender.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"test123"}'
```

## Frontend Deployment (Vercel)

### 1. Update Frontend
- [ ] Copy `frontend/lib/api.ts` (already created)
- [ ] Update `.env.local`:
  ```
  NEXT_PUBLIC_API_URL=https://your-backend.onrender.com
  ```
- [ ] Test locally with production backend
- [ ] Commit and push changes

### 2. Deploy to Vercel
- [ ] Go to Vercel Dashboard
- [ ] Import GitHub repository
- [ ] Configure:
  - Framework: Next.js
  - Root Directory: `frontend`
- [ ] Add Environment Variable:
  - `NEXT_PUBLIC_API_URL=https://your-backend.onrender.com`
- [ ] Deploy
- [ ] Wait for deployment to complete

### 3. Test Frontend
- [ ] Visit Vercel URL
- [ ] Register new account
- [ ] Login
- [ ] Upload file
- [ ] List files
- [ ] Download file
- [ ] Delete file
- [ ] Logout and verify can't access files

## Post-Deployment

### Verification
- [ ] Backend health endpoint responds
- [ ] Database has users and files tables
- [ ] User registration works
- [ ] User login works
- [ ] File upload works
- [ ] File listing shows only user's files
- [ ] File download works
- [ ] File deletion works
- [ ] File isolation works (users can't access others' files)

### Security
- [ ] JWT_SECRET is strong and random
- [ ] Database uses SSL connection
- [ ] AWS credentials are environment variables
- [ ] No sensitive data in git repository
- [ ] CORS is properly configured
- [ ] HTTPS is enabled (automatic on Render/Vercel)

### Monitoring
- [ ] Set up Render alerts for errors
- [ ] Monitor database usage
- [ ] Check backend logs regularly
- [ ] Monitor Vercel deployment logs
- [ ] Set up uptime monitoring (optional)

### Documentation
- [ ] Update README with production URLs
- [ ] Document environment variables
- [ ] Create user guide
- [ ] Document API endpoints

## Cleanup

### Remove Unnecessary Files
```bash
# These were created for documentation but not needed in production
rm Backend/MIGRATION_GUIDE.md
rm Backend/AUTHENTICATION_COMPLETE.md
rm Backend/AUTH_IMPLEMENTATION_SUMMARY.md
rm Backend/docs/AUTH_FLOW.md

# Keep these:
# - Backend/AUTH_README.md (API documentation)
# - Backend/QUICKSTART_AUTH.md (setup guide)
# - Backend/DEPLOY_PRODUCTION.md (this guide)
```

## Troubleshooting

### Backend won't start
1. Check Render logs
2. Verify DATABASE_URL is set
3. Ensure JWT_SECRET is set
4. Check Go version compatibility

### Database connection fails
1. Use Internal Database URL (not External)
2. Verify database is running
3. Check connection string format
4. Ensure SSL mode is correct

### Frontend can't connect to backend
1. Verify NEXT_PUBLIC_API_URL is correct
2. Check CORS settings on backend
3. Ensure backend is deployed and running
4. Check browser console for errors

### Authentication not working
1. Verify JWT_SECRET is set on backend
2. Check token is being saved in localStorage
3. Verify Authorization header format
4. Check token hasn't expired (24h default)

## URLs to Save

- **Backend**: https://_____________________.onrender.com
- **Frontend**: https://_____________________.vercel.app
- **Database**: echofs-db (Internal URL from Render)
- **GitHub**: https://github.com/_____/_____

## Next Steps

1. [ ] Set up custom domain (optional)
2. [ ] Configure database backups
3. [ ] Add monitoring and alerts
4. [ ] Implement email verification
5. [ ] Add password reset functionality
6. [ ] Set up CI/CD pipeline
7. [ ] Add integration tests
8. [ ] Scale to paid plans if needed

## Support

- Render Docs: https://render.com/docs
- Vercel Docs: https://vercel.com/docs
- EchoFS Docs: See Backend/AUTH_README.md

---

**Deployment Date**: _____________________
**Deployed By**: _____________________
**Status**: ⬜ Not Started | ⬜ In Progress | ⬜ Complete
