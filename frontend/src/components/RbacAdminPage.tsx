import { useMemo, useState } from 'react'
import { ShieldAlert, Trash2 } from 'lucide-react'
import { z } from 'zod'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { toast } from 'sonner'
import { useMessages } from '@/i18n/locale-context'
import { useSession } from '@/lib/session'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { ApiError } from '@/lib/api'
import {
  assignPermissionToRole,
  createRbacDelegasi,
  createRbacPermission,
  createRbacRole,
  createRbacUserScope,
  deleteRbacPermission,
  deleteRbacRole,
  deleteRbacUserScope,
  fetchRbacDelegasi,
  fetchRbacPermissions,
  fetchRbacRoles,
  fetchRbacUserScopes,
  revokeRbacDelegasi,
  revokePermissionFromRole,
  updateRbacRole,
} from '@/lib/rbac'
import type { RbacDelegasi, RbacRole, RbacUserRoleScope } from '@/validations/schemas'
import LoadingRadar from '@/components/LoadingRadar'
import RetryButton from '@/components/RetryButton'

const roleFormSchema = z.object({
  code: z.string().trim().min(1),
  name: z.string().trim().min(1),
  tierLevel: z.number().int().min(1).max(10),
})
const permFormSchema = z.object({
  module: z.string().trim().min(1),
  action: z.string().trim().min(1),
})
const userIdSchema = z.string().trim().min(1)
const scopeFormSchema = z.object({
  userId: z.string().trim().min(1),
  roleId: z.number().int().positive(),
})
const delegasiFormSchema = z.object({
  userId: z.string().trim().min(1),
  cakupan: z.enum(['CUTI', 'LEMBUR', 'SEMUA']),
  mulai: z.string().min(1),
  selesai: z.string().min(1),
})

export default function RbacAdminPage() {
  const t = useMessages()
  const session = useSession()
  const queryClient = useQueryClient()
  const permissions = session.data?.permissions ?? []
  const canRead = permissions.includes('rbac:read') || permissions.includes('rbac:manage')
  const canManage = permissions.includes('rbac:manage')

  const rolesQuery = useQuery({
    queryKey: ['rbac', 'roles'] as const,
    queryFn: fetchRbacRoles,
    enabled: canRead,
    staleTime: 15_000,
    retry: false,
  })
  const permsQuery = useQuery({
    queryKey: ['rbac', 'permissions'] as const,
    queryFn: fetchRbacPermissions,
    enabled: canRead,
    staleTime: 15_000,
    retry: false,
  })

  const [roleCode, setRoleCode] = useState('')
  const [roleName, setRoleName] = useState('')
  const [roleTier, setRoleTier] = useState<number | ''>(1)
  const [editRoleId, setEditRoleId] = useState<number | null>(null)

  const [permModule, setPermModule] = useState('')
  const [permAction, setPermAction] = useState('')

  const [assignRoleId, setAssignRoleId] = useState<number | ''>('')
  const [assignPermId, setAssignPermId] = useState<number | ''>('')

  const [userId, setUserId] = useState('')
  const [scopes, setScopes] = useState<RbacUserRoleScope[]>([])
  const [delegasi, setDelegasi] = useState<RbacDelegasi[]>([])
  const [scopeRoleId, setScopeRoleId] = useState<number | ''>('')
  const [scopeUnitId, setScopeUnitId] = useState('')
  const [scopeFakultasId, setScopeFakultasId] = useState('')
  const [scopeJurusanId, setScopeJurusanId] = useState('')
  const [scopeBiroCode, setScopeBiroCode] = useState('')
  const [delCakupan, setDelCakupan] = useState('CUTI')
  const [delMulai, setDelMulai] = useState('')
  const [delSelesai, setDelSelesai] = useState('')
  const [delPemberi, setDelPemberi] = useState('')
  const [loadingUser, setLoadingUser] = useState(false)

  const roleValid = roleFormSchema.safeParse({
    code: roleCode,
    name: roleName,
    tierLevel: Number(roleTier),
  }).success
  const permValid = permFormSchema.safeParse({
    module: permModule,
    action: permAction,
  }).success
  const userValid = userIdSchema.safeParse(userId).success
  const assignValid = useMemo(
    () =>
      typeof assignRoleId === 'number' &&
      typeof assignPermId === 'number' &&
      assignRoleId > 0 &&
      assignPermId > 0,
    [assignRoleId, assignPermId],
  )
  const scopeValid = scopeFormSchema.safeParse({
    userId,
    roleId: typeof scopeRoleId === 'number' ? scopeRoleId : NaN,
  }).success
  const delegasiValid =
    delegasiFormSchema.safeParse({
      userId,
      cakupan: delCakupan,
      mulai: delMulai,
      selesai: delSelesai,
    }).success &&
    (!delMulai || !delSelesai || delMulai <= delSelesai)

  if (session.data && !canRead) {
    return (
      <div className="page-wrap px-4 py-10 sm:py-14">
        <section className="island-shell mx-auto max-w-md rounded-2xl p-6 text-center">
          <ShieldAlert
            className="mx-auto mb-3 h-8 w-8 text-[var(--sea-ink-soft)]"
            aria-hidden="true"
          />
          <h1 className="display-title mb-2 text-2xl font-bold text-[var(--sea-ink)]">
            {t('state.noAccess.title')}
          </h1>
          <p className="m-0 text-sm text-[var(--sea-ink-soft)]">
            {t('state.noAccess.description')}
          </p>
        </section>
      </div>
    )
  }

  const roles = rolesQuery.data ?? []
  const perms = permsQuery.data ?? []

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ['rbac', 'roles'] })
    queryClient.invalidateQueries({ queryKey: ['rbac', 'permissions'] })
  }

  const errorMessage = (err: unknown) => {
    if (err instanceof ApiError) return err.message || t('rbac.failed')
    return t('rbac.failed')
  }

  const handleSaveRole = async () => {
    const parsed = roleFormSchema.safeParse({
      code: roleCode,
      name: roleName,
      tierLevel: Number(roleTier),
    })
    if (!parsed.success) {
      toast.error(t('rbac.invalidRole'))
      return
    }
    try {
      if (editRoleId) {
        await updateRbacRole(editRoleId, parsed.data)
      } else {
        await createRbacRole(parsed.data)
      }
      toast.success(t('rbac.saved'))
      setRoleCode('')
      setRoleName('')
      setRoleTier(1)
      setEditRoleId(null)
      invalidate()
    } catch (err) {
      toast.error(errorMessage(err))
    }
  }

  const handleEditRole = (role: RbacRole) => {
    setEditRoleId(role.id)
    setRoleCode(role.code)
    setRoleName(role.name)
    setRoleTier(role.tierLevel)
  }

  const handleDeleteRole = async (id: number) => {
    try {
      await deleteRbacRole(id)
      toast.success(t('rbac.deleted'))
      invalidate()
    } catch (err) {
      toast.error(errorMessage(err))
    }
  }

  const handleAddPermission = async () => {
    const parsed = permFormSchema.safeParse({ module: permModule, action: permAction })
    if (!parsed.success) {
      toast.error(t('rbac.invalidPermission'))
      return
    }
    try {
      await createRbacPermission(parsed.data)
      toast.success(t('rbac.saved'))
      setPermModule('')
      setPermAction('')
      invalidate()
    } catch (err) {
      toast.error(errorMessage(err))
    }
  }

  const handleDeletePermission = async (id: number) => {
    try {
      await deleteRbacPermission(id)
      toast.success(t('rbac.deleted'))
      invalidate()
    } catch (err) {
      toast.error(errorMessage(err))
    }
  }

  const handleAssign = async () => {
    if (!assignValid) return
    try {
      await assignPermissionToRole(Number(assignRoleId), Number(assignPermId))
      toast.success(t('rbac.saved'))
    } catch (err) {
      toast.error(errorMessage(err))
    }
  }

  const handleRevoke = async () => {
    if (!assignValid) return
    try {
      await revokePermissionFromRole(Number(assignRoleId), Number(assignPermId))
      toast.success(t('rbac.deleted'))
    } catch (err) {
      toast.error(errorMessage(err))
    }
  }

  const loadUserData = async () => {
    if (!userIdSchema.safeParse(userId).success) {
      toast.error(t('rbac.invalidUser'))
      return
    }
    setLoadingUser(true)
    try {
      const [s, d] = await Promise.all([
        fetchRbacUserScopes(userId.trim()),
        fetchRbacDelegasi(userId.trim()),
      ])
      setScopes(s)
      setDelegasi(d)
    } catch (err) {
      toast.error(errorMessage(err))
    } finally {
      setLoadingUser(false)
    }
  }

  const handleAddScope = async () => {
    const parsed = scopeFormSchema.safeParse({
      userId,
      roleId: typeof scopeRoleId === 'number' ? scopeRoleId : NaN,
    })
    if (!parsed.success) {
      toast.error(t('rbac.invalidScope'))
      return
    }
    try {
      await createRbacUserScope(userId.trim(), {
        roleId: Number(scopeRoleId),
        unitId: scopeUnitId || undefined,
        fakultasId: scopeFakultasId || undefined,
        jurusanId: scopeJurusanId || undefined,
        biroCode: scopeBiroCode || undefined,
      })
      toast.success(t('rbac.saved'))
      await loadUserData()
    } catch (err) {
      toast.error(errorMessage(err))
    }
  }

  const handleDeleteScope = async (id: number) => {
    try {
      await deleteRbacUserScope(id)
      toast.success(t('rbac.deleted'))
      await loadUserData()
    } catch (err) {
      toast.error(errorMessage(err))
    }
  }

  const handleAddDelegasi = async () => {
    const parsed = delegasiFormSchema.safeParse({
      userId,
      cakupan: delCakupan,
      mulai: delMulai,
      selesai: delSelesai,
    })
    if (!parsed.success || delMulai > delSelesai) {
      toast.error(t('rbac.invalidDelegasi'))
      return
    }
    try {
      await createRbacDelegasi(userId.trim(), {
        pemberiUserId: delPemberi || undefined,
        cakupan: parsed.data.cakupan,
        mulai: new Date(delMulai).toISOString(),
        selesai: new Date(delSelesai).toISOString(),
      })
      toast.success(t('rbac.saved'))
      await loadUserData()
    } catch (err) {
      toast.error(errorMessage(err))
    }
  }

  const handleRevokeDelegasi = async (id?: number) => {
    if (!id) return
    try {
      await revokeRbacDelegasi(id)
      toast.success(t('rbac.deleted'))
      await loadUserData()
    } catch (err) {
      toast.error(errorMessage(err))
    }
  }

  if (rolesQuery.isPending || permsQuery.isPending) {
    return (
      <div className="page-wrap flex flex-col items-center gap-4 px-4 py-10 sm:py-14">
        <LoadingRadar size="md" decorative />
        <p className="m-0 text-sm text-[var(--sea-ink-soft)]">{t('rbac.loading')}</p>
      </div>
    )
  }

  if (rolesQuery.isError || permsQuery.isError) {
    return (
      <div className="page-wrap px-4 py-10 sm:py-14">
        <div className="island-shell max-w-md rounded-2xl p-6">
          <h2 className="mb-2 text-lg font-semibold text-[var(--sea-ink)]">
            {t('rbac.errorTitle')}
          </h2>
          <p className="mb-4 text-sm text-[var(--sea-ink-soft)]">
            {t('rbac.errorDescription')}
          </p>
          <RetryButton
            label={t('common.retry')}
            onClick={() => {
              void rolesQuery.refetch()
              void permsQuery.refetch()
            }}
          />
        </div>
      </div>
    )
  }

  return (
    <div className="page-wrap px-4 py-10 sm:py-14">
      <p className="island-kicker mb-2">{t('rbac.kicker')}</p>
      <h1 className="display-title mb-2 text-2xl font-semibold tracking-tight text-[var(--sea-ink)] sm:text-4xl">
        {t('rbac.title')}
      </h1>
      <p className="mb-6 max-w-3xl text-sm leading-relaxed text-[var(--sea-ink-soft)]">
        {t('rbac.lead')}
      </p>

      <div className="grid gap-6 lg:grid-cols-2">
        <section className="island-shell rounded-2xl p-5">
          <h2 className="mb-3 text-lg font-semibold text-[var(--sea-ink)]">
            {t('rbac.roles')}
          </h2>
          {canManage && (
            <div className="mb-4 grid gap-2 sm:grid-cols-4">
              <div>
                <Label htmlFor="rbac-role-code">{t('rbac.roleCode')}</Label>
                <Input
                  id="rbac-role-code"
                  value={roleCode}
                  onChange={(e) => setRoleCode(e.target.value)}
                />
              </div>
              <div>
                <Label htmlFor="rbac-role-name">{t('rbac.roleName')}</Label>
                <Input
                  id="rbac-role-name"
                  value={roleName}
                  onChange={(e) => setRoleName(e.target.value)}
                />
              </div>
              <div>
                <Label htmlFor="rbac-role-tier">{t('rbac.tierLevel')}</Label>
                <Input
                  id="rbac-role-tier"
                  type="number"
                  value={roleTier}
                  onChange={(e) =>
                    setRoleTier(e.target.value === '' ? '' : Number(e.target.value))
                  }
                />
              </div>
              <div className="flex items-end">
                <Button
                  type="button"
                  onClick={() => void handleSaveRole()}
                  disabled={!roleValid}
                >
                  {editRoleId ? t('rbac.editRole') : t('rbac.addRole')}
                </Button>
              </div>
            </div>
          )}
          <div className="space-y-2">
            {roles.length === 0 ? (
              <p className="text-sm text-[var(--sea-ink-soft)]">{t('rbac.empty')}</p>
            ) : (
              roles.map((r) => (
                <div
                  key={r.id}
                  className="flex items-center justify-between gap-2 rounded-lg border border-[var(--line)] p-2 text-sm"
                >
                  <div>
                    <span className="font-medium">{r.name}</span>{' '}
                    <span className="text-[var(--sea-ink-soft)]">({r.code})</span>{' '}
                    <Badge variant="outline">tier {r.tierLevel}</Badge>
                  </div>
                  {canManage && (
                    <div className="flex gap-2">
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => handleEditRole(r)}
                      >
                        {t('common.edit')}
                      </Button>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => void handleDeleteRole(r.id)}
                        aria-label={t('common.delete')}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  )}
                </div>
              ))
            )}
          </div>
        </section>

        <section className="island-shell rounded-2xl p-5">
          <h2 className="mb-3 text-lg font-semibold text-[var(--sea-ink)]">
            {t('rbac.permissions')}
          </h2>
          {canManage && (
            <div className="mb-4 grid gap-2 sm:grid-cols-3">
              <div>
                <Label htmlFor="rbac-perm-module">{t('rbac.module')}</Label>
                <Input
                  id="rbac-perm-module"
                  value={permModule}
                  onChange={(e) => setPermModule(e.target.value)}
                />
              </div>
              <div>
                <Label htmlFor="rbac-perm-action">{t('rbac.action')}</Label>
                <Input
                  id="rbac-perm-action"
                  value={permAction}
                  onChange={(e) => setPermAction(e.target.value)}
                />
              </div>
              <div className="flex items-end">
                <Button
                  type="button"
                  onClick={() => void handleAddPermission()}
                  disabled={!permValid}
                >
                  {t('rbac.addPermission')}
                </Button>
              </div>
            </div>
          )}
          <div className="max-h-72 space-y-2 overflow-auto pr-1">
            {perms.length === 0 ? (
              <p className="text-sm text-[var(--sea-ink-soft)]">{t('rbac.empty')}</p>
            ) : (
              perms.map((p) => (
                <div
                  key={p.id}
                  className="flex items-center justify-between gap-2 rounded-lg border border-[var(--line)] p-2 text-sm"
                >
                  <span>
                    <span className="font-medium">{p.module}</span>:
                    <span>{p.action}</span>{' '}
                    <span className="text-[var(--sea-ink-soft)]">#{p.id}</span>
                  </span>
                  {canManage && (
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => void handleDeletePermission(p.id)}
                      aria-label={t('common.delete')}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              ))
            )}
          </div>
          {canManage && (
            <div className="mt-4 grid gap-2 sm:grid-cols-4">
              <div>
                <Label htmlFor="rbac-assign-role">{t('rbac.roleId')}</Label>
                <Input
                  id="rbac-assign-role"
                  type="number"
                  value={assignRoleId}
                  onChange={(e) =>
                    setAssignRoleId(e.target.value === '' ? '' : Number(e.target.value))
                  }
                />
              </div>
              <div>
                <Label htmlFor="rbac-assign-perm">{t('rbac.permissionId')}</Label>
                <Input
                  id="rbac-assign-perm"
                  type="number"
                  value={assignPermId}
                  onChange={(e) =>
                    setAssignPermId(e.target.value === '' ? '' : Number(e.target.value))
                  }
                />
              </div>
              <div className="flex items-end">
                <Button
                  type="button"
                  onClick={() => void handleAssign()}
                  disabled={!assignValid}
                >
                  {t('rbac.assign')}
                </Button>
              </div>
              <div className="flex items-end">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => void handleRevoke()}
                  disabled={!assignValid}
                >
                  {t('rbac.revoke')}
                </Button>
              </div>
            </div>
          )}
        </section>
      </div>

      <section className="island-shell mt-6 rounded-2xl p-5">
        <h2 className="mb-3 text-lg font-semibold text-[var(--sea-ink)]">
          {t('rbac.scopes')} / {t('rbac.delegasi')}
        </h2>
        <div className="mb-4 flex flex-wrap items-end gap-2">
          <div className="min-w-[280px] flex-1">
            <Label htmlFor="rbac-user-id">{t('rbac.userId')}</Label>
            <Input
              id="rbac-user-id"
              value={userId}
              onChange={(e) => setUserId(e.target.value)}
              placeholder={t('rbac.loadUserHint')}
            />
          </div>
          <Button
            type="button"
            onClick={() => void loadUserData()}
            disabled={loadingUser || !userValid}
          >
            {t('rbac.loadUser')}
          </Button>
        </div>

        <div className="grid gap-6 lg:grid-cols-2">
          <div>
            <h3 className="mb-2 font-medium text-[var(--sea-ink)]">{t('rbac.scopes')}</h3>
            {canManage && (
              <div className="mb-3 grid gap-2 sm:grid-cols-3">
                <div>
                  <Label htmlFor="rbac-scope-role">{t('rbac.roleId')}</Label>
                  <Input
                    id="rbac-scope-role"
                    type="number"
                    value={scopeRoleId}
                    onChange={(e) =>
                      setScopeRoleId(e.target.value === '' ? '' : Number(e.target.value))
                    }
                  />
                </div>
                <div>
                  <Label htmlFor="rbac-scope-unit">{t('rbac.unitId')}</Label>
                  <Input
                    id="rbac-scope-unit"
                    value={scopeUnitId}
                    onChange={(e) => setScopeUnitId(e.target.value)}
                  />
                </div>
                <div>
                  <Label htmlFor="rbac-scope-fak">{t('rbac.fakultasId')}</Label>
                  <Input
                    id="rbac-scope-fak"
                    value={scopeFakultasId}
                    onChange={(e) => setScopeFakultasId(e.target.value)}
                  />
                </div>
                <div>
                  <Label htmlFor="rbac-scope-jur">{t('rbac.jurusanId')}</Label>
                  <Input
                    id="rbac-scope-jur"
                    value={scopeJurusanId}
                    onChange={(e) => setScopeJurusanId(e.target.value)}
                  />
                </div>
                <div>
                  <Label htmlFor="rbac-scope-biro">{t('rbac.biroCode')}</Label>
                  <Input
                    id="rbac-scope-biro"
                    value={scopeBiroCode}
                    onChange={(e) => setScopeBiroCode(e.target.value)}
                  />
                </div>
                <div className="flex items-end">
                  <Button
                    type="button"
                    onClick={() => void handleAddScope()}
                    disabled={!scopeValid}
                  >
                    {t('rbac.addScope')}
                  </Button>
                </div>
              </div>
            )}
            <div className="space-y-2">
              {scopes.length === 0 ? (
                <p className="text-sm text-[var(--sea-ink-soft)]">{t('rbac.empty')}</p>
              ) : (
                scopes.map((s) => (
                  <div
                    key={s.id}
                    className="flex items-center justify-between gap-2 rounded-lg border border-[var(--line)] p-2 text-sm"
                  >
                    <span>
                      #{s.id} role {s.roleId} {s.unitId ? `unit:${s.unitId}` : ''}{' '}
                      {s.fakultasId ? `fak:${s.fakultasId}` : ''}{' '}
                      {s.jurusanId ? `jur:${s.jurusanId}` : ''}{' '}
                      {s.biroCode ? `biro:${s.biroCode}` : ''}
                    </span>
                    {canManage && (
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => void handleDeleteScope(s.id)}
                        aria-label={t('common.delete')}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    )}
                  </div>
                ))
              )}
            </div>
          </div>

          <div>
            <h3 className="mb-2 font-medium text-[var(--sea-ink)]">
              {t('rbac.delegasi')}
            </h3>
            {canManage && (
              <div className="mb-3 grid gap-2 sm:grid-cols-2">
                <div>
                  <Label htmlFor="rbac-del-cakupan">{t('rbac.cakupan')}</Label>
                  <Input
                    id="rbac-del-cakupan"
                    value={delCakupan}
                    onChange={(e) => setDelCakupan(e.target.value)}
                    placeholder="CUTI/LEMBUR/SEMUA"
                  />
                </div>
                <div>
                  <Label htmlFor="rbac-del-pemberi">{t('rbac.pemberiUserId')}</Label>
                  <Input
                    id="rbac-del-pemberi"
                    value={delPemberi}
                    onChange={(e) => setDelPemberi(e.target.value)}
                  />
                </div>
                <div>
                  <Label htmlFor="rbac-del-mulai">{t('rbac.mulai')}</Label>
                  <Input
                    id="rbac-del-mulai"
                    type="date"
                    value={delMulai}
                    onChange={(e) => setDelMulai(e.target.value)}
                  />
                </div>
                <div>
                  <Label htmlFor="rbac-del-selesai">{t('rbac.selesai')}</Label>
                  <Input
                    id="rbac-del-selesai"
                    type="date"
                    value={delSelesai}
                    onChange={(e) => setDelSelesai(e.target.value)}
                  />
                </div>
                <div className="flex items-end sm:col-span-2">
                  <Button
                    type="button"
                    onClick={() => void handleAddDelegasi()}
                    disabled={!delegasiValid}
                  >
                    {t('rbac.addDelegasi')}
                  </Button>
                </div>
              </div>
            )}
            <div className="space-y-2">
              {delegasi.length === 0 ? (
                <p className="text-sm text-[var(--sea-ink-soft)]">{t('rbac.empty')}</p>
              ) : (
                delegasi.map((d) => (
                  <div
                    key={d.id ?? `${d.cakupan}-${d.mulai}`}
                    className="flex items-center justify-between gap-2 rounded-lg border border-[var(--line)] p-2 text-sm"
                  >
                    <span>
                      #{d.id} {d.cakupan} {String(d.mulai ?? '').slice(0, 10)}→
                      {String(d.selesai ?? '').slice(0, 10)}{' '}
                      {d.aktif === false ? '(nonaktif)' : ''}
                    </span>
                    {canManage && (
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => void handleRevokeDelegasi(d.id)}
                        aria-label={t('common.delete')}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    )}
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      </section>
    </div>
  )
}
