import { createFileRoute } from '@tanstack/react-router'
import LegalPage from '@/components/LegalPage'

export const Route = createFileRoute('/privacy')({
  component: PrivacyPage,
})

const SECTIONS = [
  { title: 'privacy.s1t', body: 'privacy.s1b' },
  { title: 'privacy.s2t', body: 'privacy.s2b' },
  { title: 'privacy.s3t', body: 'privacy.s3b' },
  { title: 'privacy.s4t', body: 'privacy.s4b' },
] as const

function PrivacyPage() {
  return <LegalPage title="privacy.title" sections={SECTIONS} />
}
