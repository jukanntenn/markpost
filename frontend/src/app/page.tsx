import { buildPageMetadata } from '@/lib/metadata'
import { LandingPage } from '@/components/landing/LandingPage'

export const generateMetadata = buildPageMetadata('landing')

export default function Home() {
  return <LandingPage />
}
