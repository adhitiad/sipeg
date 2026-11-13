import { createFileRoute } from '@tanstack/react-router'
import LegalPage from '@/components/LegalPage'

export const Route = createFileRoute('/terms-of-reference')({
  component: TermsOfReferencePage,
})

const SECTIONS = [
  { title: 'reference.s1t', body: 'reference.s1b' },
  { title: 'reference.s2t', body: 'reference.s2b' },
  { title: 'reference.s3t', body: 'reference.s3b' },
  { title: 'reference.s4t', body: 'reference.s4b' },
] as const

function TermsOfReferencePage() {
  return <LegalPage title="reference.title" lead="reference.lead" sections={SECTIONS} />
}
