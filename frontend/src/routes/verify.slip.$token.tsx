import { createFileRoute } from '@tanstack/react-router'
import SlipVerifyPage from '@/components/SlipVerifyPage'

export const Route = createFileRoute('/verify/slip/$token')({
  component: SlipVerifyPage,
})
