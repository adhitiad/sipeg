import { apiRequest } from '@/lib/api'
import {
  rbacDelegasiListSchema,
  rbacDelegasiSchema,
  rbacPermissionListSchema,
  rbacPermissionSchema,
  rbacRoleListSchema,
  rbacRoleSchema,
  rbacUserRoleScopeListSchema,
  rbacUserRoleScopeSchema,
  type RbacDelegasi,
  type RbacPermission,
  type RbacRole,
  type RbacUserRoleScope,
} from '@/validations/schemas'

export async function fetchRbacRoles(): Promise<RbacRole[]> {
  const payload = await apiRequest<unknown>('/admin/rbac/roles')
  return rbacRoleListSchema.parse(payload).items
}

export async function createRbacRole(data: {
  code: string
  name: string
  tierLevel: number
}): Promise<RbacRole> {
  const payload = await apiRequest<unknown>('/admin/rbac/roles', {
    method: 'POST',
    body: data,
  })
  return rbacRoleSchema.parse(payload)
}

export async function updateRbacRole(
  id: number,
  data: { code: string; name: string; tierLevel: number },
): Promise<RbacRole> {
  const payload = await apiRequest<unknown>(`/admin/rbac/roles/${id}`, {
    method: 'PATCH',
    body: data,
  })
  return rbacRoleSchema.parse(payload)
}

export async function deleteRbacRole(id: number): Promise<void> {
  await apiRequest<unknown>(`/admin/rbac/roles/${id}`, { method: 'DELETE' })
}

export async function fetchRbacPermissions(): Promise<RbacPermission[]> {
  const payload = await apiRequest<unknown>('/admin/rbac/permissions')
  return rbacPermissionListSchema.parse(payload).items
}

export async function createRbacPermission(data: {
  module: string
  action: string
}): Promise<RbacPermission> {
  const payload = await apiRequest<unknown>('/admin/rbac/permissions', {
    method: 'POST',
    body: data,
  })
  return rbacPermissionSchema.parse(payload)
}

export async function deleteRbacPermission(id: number): Promise<void> {
  await apiRequest<unknown>(`/admin/rbac/permissions/${id}`, { method: 'DELETE' })
}

export async function assignPermissionToRole(
  roleId: number,
  permissionId: number,
): Promise<void> {
  await apiRequest<unknown>(`/admin/rbac/roles/${roleId}/permissions`, {
    method: 'POST',
    body: { permissionId },
  })
}

export async function revokePermissionFromRole(
  roleId: number,
  permissionId: number,
): Promise<void> {
  await apiRequest<unknown>(`/admin/rbac/roles/${roleId}/permissions/${permissionId}`, {
    method: 'DELETE',
  })
}

export async function fetchRbacUserScopes(userId: string): Promise<RbacUserRoleScope[]> {
  const payload = await apiRequest<unknown>(`/admin/rbac/users/${userId}/scopes`)
  return rbacUserRoleScopeListSchema.parse(payload).items
}

export async function createRbacUserScope(
  userId: string,
  data: {
    roleId: number
    biroCode?: string
    fakultasId?: string
    jurusanId?: string
    unitId?: string
    deniedPermissions?: string[]
  },
): Promise<RbacUserRoleScope> {
  const payload = await apiRequest<unknown>(`/admin/rbac/users/${userId}/scopes`, {
    method: 'POST',
    body: data,
  })
  return rbacUserRoleScopeSchema.parse(payload)
}

export async function deleteRbacUserScope(id: number): Promise<void> {
  await apiRequest<unknown>(`/admin/rbac/scopes/${id}`, { method: 'DELETE' })
}

export async function fetchRbacDelegasi(userId: string): Promise<RbacDelegasi[]> {
  const payload = await apiRequest<unknown>(`/admin/rbac/users/${userId}/delegasi`)
  return rbacDelegasiListSchema.parse(payload).items
}

export async function createRbacDelegasi(
  userId: string,
  data: { pemberiUserId?: string; cakupan: string; mulai: string; selesai: string },
): Promise<RbacDelegasi> {
  const payload = await apiRequest<unknown>(`/admin/rbac/users/${userId}/delegasi`, {
    method: 'POST',
    body: data,
  })
  return rbacDelegasiSchema.parse(payload)
}

export async function revokeRbacDelegasi(id: number): Promise<void> {
  await apiRequest<unknown>(`/admin/rbac/delegasi/${id}`, { method: 'DELETE' })
}
