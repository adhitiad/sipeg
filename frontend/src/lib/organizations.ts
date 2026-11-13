import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { apiRequest } from '@/lib/api'
import {
  organizationPageSchema,
  organizationSchema,
  type Organization,
  type OrganizationCreate,
  type OrganizationPage,
  type OrganizationUpdate,
} from '@/validations/schemas'

export const organizationsQueryKey = ['organizations'] as const

export const ORGANIZATION_KINDS = [
  'UNIT',
  'FAKULTAS',
  'JURUSAN',
  'BIRO',
  'BAGIAN',
] as const

/** Kolom urut yang diizinkan backend untuk daftar unit. */
export const ORGANIZATION_SORT_FIELDS = [
  'name',
  'jenis',
  'createdAt',
  'updatedAt',
] as const
export type OrganizationSortField = (typeof ORGANIZATION_SORT_FIELDS)[number]

export type OrganizationSortOrder = 'asc' | 'desc'

export type OrganizationListParams = {
  page: number
  pageSize: number
  search: string
  sort: OrganizationSortField
  order: OrganizationSortOrder
  jenis: string | null
}

/**
 * Membaca unit organisasi dalam cakupan pemohon. Backend menegakkan cakupan
 * `organization:read` / `organization:read:homebase`; hasil dipakai untuk
 * menampilkan nama unit, bukan untuk menambah akses.
 */
export async function fetchOrganizations(): Promise<Organization[]> {
  const payload = await apiRequest<unknown>('/organizations?page=1&pageSize=100')
  return organizationPageSchema.parse(payload).items
}

/** Satu halaman daftar unit dari GET /api/v1/organizations. */
export async function fetchOrganizationsPage(
  params: OrganizationListParams,
): Promise<OrganizationPage> {
  const query = new URLSearchParams({
    page: String(params.page),
    pageSize: String(params.pageSize),
    sort: params.sort,
    order: params.order,
  })
  if (params.search) {
    query.set('search', params.search)
  }
  if (params.jenis) {
    query.set('jenis', params.jenis)
  }
  const payload = await apiRequest<unknown>(`/organizations?${query.toString()}`)
  return organizationPageSchema.parse(payload)
}

/** Kueri daftar unit berpaginasi. `keepPreviousData` menahan data lama saat filter berubah. */
export function useOrganizationsPage(params: OrganizationListParams) {
  return useQuery({
    queryKey: ['organizations', 'page', params] as const,
    queryFn: () => fetchOrganizationsPage(params),
    placeholderData: keepPreviousData,
    staleTime: 15_000,
    retry: false,
  })
}

/** Satu organisasi dari GET /api/v1/organizations/{organizationID}. */
export async function fetchOrganization(id: string): Promise<Organization> {
  const payload = await apiRequest<unknown>(`/organizations/${id}`)
  return organizationSchema.parse(payload)
}

/** Buat organisasi baru via POST /api/v1/organizations. Butuh organization:manage. */
export async function createOrganization(
  data: OrganizationCreate,
): Promise<Organization> {
  const payload = await apiRequest<unknown>('/organizations', {
    method: 'POST',
    body: data,
  })
  return organizationSchema.parse(payload)
}

/** Perbarui organisasi via PATCH /api/v1/organizations/{organizationID}. Butuh organization:manage. */
export async function updateOrganization(
  id: string,
  data: OrganizationUpdate,
): Promise<Organization> {
  const payload = await apiRequest<unknown>(`/organizations/${id}`, {
    method: 'PATCH',
    body: data,
  })
  return organizationSchema.parse(payload)
}

/** Arsipkan organisasi via DELETE /api/v1/organizations/{organizationID}. Butuh organization:manage. */
export async function archiveOrganization(id: string): Promise<void> {
  await apiRequest<unknown>(`/organizations/${id}`, { method: 'DELETE' })
}

export function useOrganizations(enabled = true) {
  return useQuery({
    queryKey: organizationsQueryKey,
    queryFn: fetchOrganizations,
    enabled,
    staleTime: 60_000,
    retry: false,
  })
}
