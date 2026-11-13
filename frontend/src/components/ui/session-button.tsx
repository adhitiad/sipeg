import type * as React from 'react'
import { Button } from '@/components/ui/button'

type SessionButtonProps = Omit<React.ComponentProps<typeof Button>, 'children'> & {
  /** Isi utama tombol. */
  label?: React.ReactNode
  /** Ikon penanda aksi, diletakkan setelah label. */
  icon?: React.ReactNode
}

/**
 * Tombol aksi sesi (masuk dan keluar). Memakai tombol utama shadcn agar
 * warna, radius, dan cincin fokus mengikuti token produk. Tidak ada gradien,
 * warna literal, atau outline fokus yang dimatikan di sini.
 */
export function SessionButton({
  label,
  icon,
  className,
  variant = 'default',
  size = 'default',
  ...props
}: SessionButtonProps) {
  return (
    <Button variant={variant} size={size} className={className} {...props}>
      {label}
      {icon ? <span className="-mr-1">{icon}</span> : null}
    </Button>
  )
}
