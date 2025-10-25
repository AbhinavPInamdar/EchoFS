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
    const tokenResponse = await fetch('https://github.com/login/oauth/access_token', {
      method: 'POST',
      headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        client_id: process.env.NEXT_PUBLIC_GITHUB_CLIENT_ID,
        client_secret: process.env.GITHUB_CLIENT_SECRET,
        code: code,
      }),
    })

    const tokenData = await tokenResponse.json()

    if (tokenData.error) {
      return NextResponse.redirect(`http://localhost:3001?error=${encodeURIComponent(tokenData.error)}`)
    }

    // Get user info
    const userResponse = await fetch('https://api.github.com/user', {
      headers: {
        'Authorization': `Bearer ${tokenData.access_token}`,
        'Accept': 'application/json',
      },
    })

    const userData = await userResponse.json()

    // Get user email (GitHub might not return email in user endpoint)
    const emailResponse = await fetch('https://api.github.com/user/emails', {
      headers: {
        'Authorization': `Bearer ${tokenData.access_token}`,
        'Accept': 'application/json',
      },
    })

    const emailData = await emailResponse.json()
    const primaryEmail = emailData.find((email: any) => email.primary)?.email || userData.email

    // Create user info object
    const userInfo = {
      id: userData.id,
      username: userData.login,
      email: primaryEmail,
      name: userData.name,
      provider: 'github'
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
          id: userData.id.toString(),
          email: primaryEmail,
          name: userData.name,
          provider: 'github',
          avatar_url: userData.avatar_url
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
    console.error('GitHub OAuth error:', error)
    return NextResponse.redirect(`http://localhost:3001?error=oauth_failed`)
  }
}