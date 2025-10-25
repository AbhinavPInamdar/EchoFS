import { NextRequest, NextResponse } from 'next/server'
import { auth } from '@/lib/auth'

export async function POST(request: NextRequest) {
  try {
    const session = await auth()
    
    if (!session?.user) {
      return NextResponse.json({ error: 'No session found' }, { status: 401 })
    }

    // Send user data to backend for processing
    const backendUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
    
    const oauthUser = {
      id: session.user.id || session.user.email,
      email: session.user.email,
      name: session.user.name,
      provider: (session as any).provider || 'unknown',
      avatar_url: session.user.image
    }

    const response = await fetch(`${backendUrl}/api/auth/oauth/callback`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(oauthUser),
    })

    if (!response.ok) {
      throw new Error('Backend OAuth processing failed')
    }

    const result = await response.json()
    
    return NextResponse.json({
      success: true,
      token: result.data.token,
      user: result.data.user
    })

  } catch (error) {
    console.error('OAuth callback error:', error)
    return NextResponse.json(
      { error: 'OAuth processing failed' },
      { status: 500 }
    )
  }
}