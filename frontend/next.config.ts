import type { NextConfig } from 'next'

const backendUrl = process.env.BACKEND_URL || 'http://127.0.0.1:7330'

// rewrites require a Node server, which static export does not produce. The
// presence of the `rewrites` key alone triggers a build warning under
// `output: 'export'`, so it is attached only for the dev server. Production
// relies on Caddy reverse-proxying /api/v1 and /swagger to the backend.
const nextConfig: NextConfig =
  process.env.NODE_ENV === 'production'
    ? { output: 'export' }
    : {
        output: 'export',
        async rewrites() {
          return [
            {
              source: '/api/v1/:path*',
              destination: `${backendUrl}/api/v1/:path*`,
            },
            {
              source: '/swagger/:path*',
              destination: `${backendUrl}/swagger/:path*`,
            },
          ]
        },
      }

export default nextConfig
