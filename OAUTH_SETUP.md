# OAuth Setup Guide for EchoFS

EchoFS has OAuth authentication infrastructure ready! The UI includes GitHub and Google OAuth buttons, but they need to be configured to work.

## 🚀 Current Status

- ✅ OAuth UI components are ready (GitHub & Google buttons)
- ✅ Backend OAuth handling infrastructure is implemented
- ⚠️ OAuth providers need to be configured (optional)
- ✅ Traditional email/password authentication works fully

## 🔧 OAuth Setup (Optional)

### 1. GitHub OAuth Setup

1. Go to [GitHub Developer Settings](https://github.com/settings/developers)
2. Click "New OAuth App"
3. Fill in the details:
   - **Application name**: EchoFS Local
   - **Homepage URL**: `http://localhost:3001`
   - **Authorization callback URL**: `http://localhost:3001/api/auth/callback/github`
4. Click "Register application"
5. Copy the **Client ID** and **Client Secret**

### 2. Google OAuth Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing one
3. Enable the Google+ API
4. Go to "Credentials" → "Create Credentials" → "OAuth 2.0 Client IDs"
5. Configure OAuth consent screen if prompted
6. Set application type to "Web application"
7. Add authorized redirect URIs:
   - `http://localhost:3001/api/auth/callback/google`
8. Copy the **Client ID** and **Client Secret**

### 3. Update Environment Variables

Update your `frontend/.env.local` file:

```bash
# NextAuth Configuration
NEXTAUTH_URL=http://localhost:3001
NEXTAUTH_SECRET=your-nextauth-secret-key-here

# GitHub OAuth
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret

# Google OAuth
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
```

### 4. Generate NextAuth Secret

Run this command to generate a secure secret:

```bash
openssl rand -base64 32
```

Copy the output and use it as your `NEXTAUTH_SECRET`.

## 🚀 Testing the System

1. **Frontend is running**: Go to `http://localhost:3001`
2. **Click "Sign In"** to see the authentication modal
3. **OAuth buttons are visible** but will show a "not configured" message until you set up OAuth credentials
4. **Email/password authentication works fully** - you can create accounts and sign in

### With OAuth Configured
Once you set up OAuth credentials following the steps above:
1. The OAuth buttons will work for GitHub/Google sign-in
2. Users will be redirected to the OAuth provider and back to EchoFS
3. OAuth users get the same file access as email/password users

## 🔒 Security Notes

- Never commit your OAuth secrets to version control
- Use different OAuth apps for development and production
- For production, update the callback URLs to your production domain

## 🎯 Features

- **Seamless Integration**: OAuth users are automatically created in the EchoFS system
- **Unified Experience**: OAuth and email/password users have the same file access
- **Provider Display**: The UI shows which OAuth provider was used
- **Secure Tokens**: Backend generates JWT tokens for OAuth users

## 🛠 Development Mode

For testing without setting up OAuth providers, you can still use the email/password authentication. The system supports both methods simultaneously.

## 📝 Production Deployment

For production deployment:

1. Update `NEXTAUTH_URL` to your production domain
2. Update OAuth app callback URLs to production URLs
3. Use secure, unique secrets for production
4. Consider using environment-specific OAuth apps

That's it! EchoFS now supports modern OAuth authentication alongside traditional email/password login.