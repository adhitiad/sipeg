import { RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'

type RetryButtonProps = {
  label: string
  onClick: () => void
}

/** Tombol coba lagi dengan ikon muat ulang yang konsisten. */
export default function RetryButton({ label, onClick }: RetryButtonProps) {
  return (
    <Button type="button" onClick={onClick} className="gap-2">
      <RefreshCw className="h-4 w-4" aria-hidden="true" />
      {label}
    </Button>
  )
}
