import { createFileRoute } from '@tanstack/react-router'
import { AuthNotConfiguredError, getAuth } from '@/lib/server/better-auth'

/**
 * Seluruh permintaan /api/auth/* diteruskan ke runtime Better Auth yang
 * tertanam di proses yang sama. Handler mengembalikan 500 JSON bila
 * DATABASE_URL atau BETTER_AUTH_SECRET belum diisi, bukan menyalurkan
 * stack trace ke klien.
 */
async function handleAuthRequest(request: Request): Promise<Response> {
  try {
    return await getAuth().handler(request)
  } catch (error) {
    if (error instanceof AuthNotConfiguredError) {
      return new Response(
        JSON.stringify({
          error: { code: 'auth_not_configured', message: error.message },
        }),
        { status: 500, headers: { 'content-type': 'application/json' } },
      )
    }

    console.error('auth handler failed', error)
    return new Response(
      JSON.stringify({
        error: {
          code: 'auth_handler_failed',
          message: 'Runtime autentikasi gagal memproses permintaan.',
        },
      }),
      { status: 500, headers: { 'content-type': 'application/json' } },
    )
  }
}

export const Route = createFileRoute('/api/auth/$')({
  server: {
    handlers: {
      ANY: ({ request }) => handleAuthRequest(request),
    },
  },
})
