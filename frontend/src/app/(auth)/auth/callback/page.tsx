import { Metadata } from 'next'
import { Suspense } from 'react'
import LoginCallbackPage from '@/components/login/LoginCallbackPage'
import { Spinner } from '@/components/ui/spinner'

export const metadata: Metadata = {
  title: 'OAuth Callback - Markpost',
}

export default function AuthCallback() {
  return (
    <Suspense
      fallback={
        <div className="flex justify-center pt-10">
          <div className="flex flex-col items-center gap-2 text-center text-sm text-muted-foreground">
            <Spinner className="size-5" />
            <div>Loading...</div>
          </div>
        </div>
      }
    >
      <LoginCallbackPage />
    </Suspense>
  )
}
