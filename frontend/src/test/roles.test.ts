import { describe, expect, it } from 'vitest'
import { translate, type MessageKey } from '@/i18n/dictionary'
import { formatRoleName } from '@/lib/roles'

const id = (key: MessageKey) => translate('id', key)
const en = (key: MessageKey) => translate('en', key)

describe('formatRoleName', () => {
  it('memetakan nama peran backend ke label Indonesia', () => {
    expect(formatRoleName('admin', id)).toBe('Administrator')
    expect(formatRoleName('operator', id)).toBe('Operator')
    expect(formatRoleName('atasan', id)).toBe('Atasan')
    expect(formatRoleName('pegawai', id)).toBe('Pegawai')
  })

  it('memetakan nama peran backend ke label Inggris', () => {
    expect(formatRoleName('atasan', en)).toBe('Supervisor')
    expect(formatRoleName('pegawai', en)).toBe('Employee')
  })

  it('menampilkan nama peran apa adanya bila tidak dikenal', () => {
    expect(formatRoleName('peran-baru', id)).toBe('peran-baru')
  })

  it('menyatakan peran belum dimuat bila kosong', () => {
    expect(formatRoleName(undefined, id)).toBe('Peran belum dimuat')
  })
})
