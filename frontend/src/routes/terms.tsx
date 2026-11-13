import { createFileRoute } from '@tanstack/react-router'
import LegalPage from '@/components/LegalPage'

export const Route = createFileRoute('/terms')({
  component: TermsPage,
})

const SECTIONS = [
  { title: 'terms.s1t', body: 'terms.s1b' },
  { title: 'terms.s2t', body: 'terms.s2b' },
  { title: 'terms.s3t', body: 'terms.s3b' },
  { title: 'terms.s4t', body: 'terms.s4b' },
] as const

function TermsPage() {
  return <LegalPage title="terms.title" sections={SECTIONS} />
}
