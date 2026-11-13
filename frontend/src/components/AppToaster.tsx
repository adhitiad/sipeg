import { Toaster } from '@/components/ui/sonner'

/**
 * Notifikasi aplikasi memakai komponen shadcn/ui (`components/ui/sonner`).
 * Tema mengikuti tema aplikasi (terang bawaan) supaya toast tidak tampil
 * sebagai satu-satunya elemen gelap di halaman terang.
 */
export default function AppToaster() {
  return (
    <Toaster
      position="bottom-right"
      closeButton
      richColors={false}
      toastOptions={{ duration: 4200 }}
    />
  )
}
