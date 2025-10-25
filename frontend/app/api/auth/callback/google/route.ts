import { NextRequest, NextResponse } from 'next/server'

export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url)
  const code = searchParams.get('code')
  const error = searchParams.get('error')

  if (error) {
    return NextResponse.redirect(`http://localhost:3001?error=${encodeURIComponent(error)}`)
  }

  if (!code) {
    return NextResponse.redirect(`http://localhost:3001?error=no_code`)
  }

  try {
    // Exchange code for access token
    const tokenResponse = await fetch('https://oauth2.googleapis.com/token', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
      body: new URLSearchParams({
        client_id: process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID!,
        client_secret: process.env.GOOGLE_CLIENT_SECRET!,
        code: code,
        grant_type: 'authorization_code',
        redirect_uri: 'http://localhost:3001/api/auth/callback/google',
      }),
    })

    const tokenData = await tokenResponse.json()

    if (tokenData.error) {
      return NextResponse.redirect(`http://localhost:3001?error=${encodeURIComponent(tokenData.error)}`)
    }

    // Get user info
    const userResponse = await fetch('https://www.googleapis.com/oauth2/v2/userinfo', {
      headers: {
        'Authorization': `Bearer ${tokenData.access_token}`,
      },
    })

    const userData = await userResponse.json()

    // Create user info object
    const userInfo = {
      id: userData.id,
      username: userData.name || userData.email.split('@')[0],
      email: userData.email,
      name: userData.name,
      provider: 'google'
    }

    // Send to backend to create/get user and generate JWT token
    try {
      const backendUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      const backendResponse = await fetch(`${backendUrl}/api/v1/auth/oauth/simple`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          id: userData.id,
          email: userData.email,
          name: userData.name,
          provider: 'google',
          avatar_url: userData.picture
        }),
      })

      if (backendResponse.ok) {
        const backendData = await backendResponse.json()
        const token = backendData.data.token
        const user = backendData.data.user
        
        // Redirect with both user info and token
        const userParam = encodeURIComponent(JSON.stringify(user))
        const tokenParam = encodeURIComponent(token)
        return NextResponse.redirect(`http://localhost:3001?oauth_success=true&user=${userParam}&token=${tokenParam}`)
      }
    } catch (backendError) {
      console.error('Backend OAuth processing failed:', backendError)
    }

    // Fallback: redirect with user info only
    const userParam = encodeURIComponent(JSON.stringify(userInfo))
    return NextResponse.redirect(`http://localhost:3001?oauth_success=true&user=${userParam}`)

  } catch (error) {
    console.error('Google OAuth error:', error)
    return NextResponse.redirect(`http://localhost:3001?error=oauth_failed`)
  }
}