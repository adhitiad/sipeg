import type { MessageKey } from '@/i18n/dictionary'
import type { Translate } from '@/lib/types/i18n'

const ROLE_KEYS: Record<string, MessageKey> = {
  admin: 'role.admin',
  operator: 'role.operator',
  atasan: 'role.atasan',
  pegawai: 'role.pegawai',
}

/**
 * Label tampilan untuk nama peran. Pemetaan ini hanya untuk teks antarmuka;
 * hak akses tetap ditentukan backend dari roleName yang sama.
 */
export function formatRoleName(roleName: string | undefined, t: Translate): string {
  if (!roleName) {
    return t('role.missing')
  }

  const key = ROLE_KEYS[roleName]
  return key ? t(key) : roleName
}
