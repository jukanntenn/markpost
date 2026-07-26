import type { NextConfig } from 'next'

const backendUrl = process.env.BACKEND_URL || 'http://127.0.0.1:7330'

const nextConfig: NextConfig = {
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
