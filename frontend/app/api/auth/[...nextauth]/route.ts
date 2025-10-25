import { NextRequest, NextResponse } from 'next/server'

// Simple OAuth redirect handler
export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url)
  const provider = searchParams.get('provider')
  
  if (provider === 'github') {
    const githubAuthUrl = `https://github.com/login/oauth/authorize?client_id=${process.env.NEXT_PUBLIC_GITHUB_CLIENT_ID}&redirect_uri=${encodeURIComponent('http://localhost:3001/api/auth/callback/github')}&scope=user:email`
    return NextResponse.redirect(githubAuthUrl)
  }
  
  if (provider === 'google') {
    const googleAuthUrl = `https://accounts.google.com/oauth/authorize?client_id=${process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID}&redirect_uri=${encodeURIComponent('http://localhost:3001/api/auth/callback/google')}&scope=email profile&response_type=code`
    return NextResponse.redirect(googleAuthUrl)
  }
  
  return NextResponse.json({ error: 'Invalid provider' }, { status: 400 })
}

export async function POST(request: NextRequest) {
  return NextResponse.json({ error: 'Method not allowed' }, { status: 405 })
}