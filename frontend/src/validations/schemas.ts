import { z } from 'zod'

/**
 * Bentuk error seragam backend (lihat backend/internal/apierror):
 * {"error":{"code","message","requestId","fields","rows"}}.
 */
export const apiErrorEnvelopeSchema = z.object({
  error: z.object({
    code: z.string(),
    message: z.string().optional(),
    requestId: z.string().optional(),
    fields: z.record(z.string(), z.string()).optional(),
    rows: z.unknown().optional(),
  }),
})

/** Identitas principal dari GET /api/v1/me beserta permission role-nya. */
export const currentAccountSchema = z.object({
  userId: z.uuid(),
  email: z.email(),
  roleName: z.string().optional(),
  homebaseId: z.uuid().optional(),
  permissions: z.array(z.string()).optional(),
  nama: z.string().optional(),
  noHp: z.string().optional(),
})

/** Pola nomor HP Indonesia yang diterima backend (awalan 0 atau 62). */
export const phonePattern = /^\+?(62|0)[0-9]{8,15}$/

/** Membuang pemisah yang lazim ditulis pada nomor HP (sama seperti backend). */
export function compactPhoneInput(value: string): string {
  return value.replace(/[\s.\-()]/g, '')
}

/** Mengira spasi berlebih menjadi satu spasi dan memangkas ujung (backend sama). */
export function compactNamaInput(value: string): string {
  return value
    .split(/\s+/)
    .map((part) => part.trim())
    .filter(Boolean)
    .join(' ')
}

/** Formulir profil mandiri (PATCH /api/v1/me). */
export const updateProfileSchema = z.object({
  nama: z
    .string()
    .transform((value) => compactNamaInput(value))
    .refine((value) => value.length >= 1, 'settings.profileNameRequired')
    .refine((value) => value.length <= 200, 'settings.profileNameTooLong'),
  noHp: z
    .string()
    .trim()
    .transform((value) => compactPhoneInput(value))
    .refine((value) => value === '' || phonePattern.test(value), 'settings.noHpInvalid'),
})

export type UpdateProfileInput = z.infer<typeof updateProfileSchema>

/** Formulir ubah kata sandi (POST /api/auth/change-password). */
export const changePasswordSchema = z
  .object({
    currentPassword: z.string().min(1, 'settings.currentPasswordRequired'),
    newPassword: z
      .string()
      .min(8, 'settings.newPasswordTooShort')
      .max(128, 'settings.newPasswordTooLong'),
    confirmPassword: z.string().min(1, 'settings.confirmPasswordRequired'),
  })
  .refine((value) => value.newPassword === value.confirmPassword, {
    message: 'settings.passwordMismatch',
    path: ['confirmPassword'],
  })

export type ChangePasswordInput = z.infer<typeof changePasswordSchema>

export type CurrentAccount = z.infer<typeof currentAccountSchema>

/** Hasil dari POST /api/auth/sign-in/email: sesi atau tantangan 2FA. */
export const twoFactorChallengeSchema = z.object({
  twoFactorRedirect: z.literal(true),
  twoFactorMethods: z.array(z.string()).optional(),
})

/** Respons POST /api/auth/two-factor/enable (TOTP). */
export const enableTwoFactorResponseSchema = z.object({
  method: z.literal('totp'),
  totpURI: z.string(),
  backupCodes: z.array(z.string()).optional(),
})

export type EnableTwoFactorResponse = z.infer<typeof enableTwoFactorResponseSchema>

/** Respons POST /api/auth/two-factor/verify-totp dan verify-backup-code. */
export const twoFactorVerificationSchema = z.object({
  token: z.string().min(1),
  user: z
    .object({
      id: z.string(),
      email: z.email(),
      name: z.string().optional(),
    })
    .loose(),
})

export type TwoFactorVerificationResponse = z.infer<typeof twoFactorVerificationSchema>

/** Respons pencatatan masuk dari runtime Better Auth. */
export const signInResponseSchema = z.object({
  token: z.string().min(1),
  user: z
    .object({
      id: z.string().min(1),
      email: z.email(),
      name: z.string().optional(),
    })
    .loose(),
})

export type SignInResponse = z.infer<typeof signInResponseSchema>

/** Bentuk error runtime autentikasi, dipakai bila Better Auth menolak kredensial. */
export const authErrorSchema = z.object({
  code: z.string().optional(),
  message: z.string().optional(),
})

/** Ringkasan metrik satu rute pada GET /api/v1/admin/slo/metrics (angka latensi dalam milidetik). */
export const sloRouteSchema = z.object({
  route: z.string(),
  method: z.string(),
  requests: z.number(),
  errors: z.number(),
  errorRate: z.number(),
  avgMs: z.number(),
  p95Ms: z.number(),
  p99Ms: z.number(),
  maxMs: z.number(),
})

/** Ringkasan metrik SLO pada rentang waktu. `routes` kosong bila pengoleki belum pernah menulis metrik. */
export const sloMetricsSchema = z.object({
  windowStart: z.string(),
  windowEnd: z.string(),
  requests: z.number(),
  errors: z.number(),
  errorRate: z.number(),
  avgMs: z.number(),
  p95Ms: z.number(),
  p99Ms: z.number(),
  maxMs: z.number(),
  routes: z.array(sloRouteSchema),
})

export type SloMetrics = z.infer<typeof sloMetricsSchema>

/** Profil pegawai pada halaman GET /api/v1/employees. */
export const employeeSchema = z
  .object({
    id: z.uuid(),
    nama: z.string(),
    email: z.email(),
    nip: z.string().nullable(),
    homebaseId: z.uuid().nullable(),
    jabatanStruktural: z.string().nullable(),
    jabatanFungsional: z.string().nullable(),
    aktif: z.boolean(),
    roleName: z.string().nullable().optional(),
    createdAt: z.string(),
    updatedAt: z.string(),
  })
  .loose()

export type Employee = z.infer<typeof employeeSchema>

/** Satu halaman daftar pegawai beserta penomoran halaman. */
export const employeePageSchema = z.object({
  items: z.array(employeeSchema),
  page: z.number(),
  pageSize: z.number(),
  totalItems: z.number(),
  totalPages: z.number(),
})

export type EmployeePage = z.infer<typeof employeePageSchema>

/** Ringkasan jumlah pegawai dari GET /api/v1/employees/summary. */
export const employeeSummarySchema = z.object({
  total: z.number(),
  active: z.number(),
  inactive: z.number(),
  byUnit: z.array(
    z.object({
      unitId: z.uuid().nullable(),
      count: z.number(),
    }),
  ),
})

export type EmployeeSummaryResponse = z.infer<typeof employeeSummarySchema>

/** Field yang boleh diubah melalui PATCH /api/v1/employees/{employeeId}. */
export const employeeUpdateSchema = z
  .object({
    nama: z.string().min(1).max(200).optional(),
    nip: z.string().max(50).nullable().optional(),
    kategoriPegawaiId: z.uuid().nullable().optional(),
    homebaseId: z.uuid().nullable().optional(),
    jabatanStruktural: z.string().max(200).nullable().optional(),
    jabatanFungsional: z.string().max(200).nullable().optional(),
  })
  .refine((data) => Object.keys(data).length > 0, {
    message: 'Minimal satu field harus diisi',
  })

export type EmployeeUpdate = z.infer<typeof employeeUpdateSchema>

/** Unit organisasi pada GET /api/v1/organizations. */
export const organizationSchema = z
  .object({
    id: z.uuid(),
    name: z.string().nullable(),
    jenis: z.enum(['UNIT', 'FAKULTAS', 'JURUSAN', 'BIRO', 'BAGIAN']),
    parentId: z.uuid().nullable(),
    aktif: z.boolean(),
    createdAt: z.string(),
    updatedAt: z.string(),
  })
  .loose()

export type Organization = z.infer<typeof organizationSchema>

export const organizationPageSchema = z.object({
  items: z.array(organizationSchema),
  page: z.number(),
  pageSize: z.number(),
  totalItems: z.number(),
  totalPages: z.number(),
})

export type OrganizationPage = z.infer<typeof organizationPageSchema>

/** Satu master jabatan pada GET /api/v1/jabatan. */
export const jabatanSchema = z
  .object({
    id: z.uuid(),
    nama: z.string(),
    jenis: z.enum(['STRUKTURAL', 'FUNGSIONAL']),
    aktif: z.boolean(),
    createdAt: z.string(),
    updatedAt: z.string(),
  })
  .loose()

export type Jabatan = z.infer<typeof jabatanSchema>

export const jabatanPageSchema = z.object({
  items: z.array(jabatanSchema),
  page: z.number(),
  pageSize: z.number(),
  totalItems: z.number(),
  totalPages: z.number(),
})

export type JabatanPage = z.infer<typeof jabatanPageSchema>

export const jabatanCreateSchema = z.object({
  nama: z.string().min(1, 'Nama jabatan wajib diisi').max(200),
  jenis: z.enum(['STRUKTURAL', 'FUNGSIONAL']).optional(),
})

export type JabatanCreate = z.infer<typeof jabatanCreateSchema>

export const jabatanUpdateSchema = jabatanCreateSchema
  .extend({ nama: z.string().min(1).max(200).optional() })
  .refine((data) => Object.keys(data).length > 0, {
    message: 'Minimal satu field harus diisi',
  })

export type JabatanUpdate = z.infer<typeof jabatanUpdateSchema>

/** Payload membuat organisasi baru via POST /api/v1/organizations. */
export const organizationCreateSchema = z.object({
  name: z.string().min(1).max(200),
  description: z.string().max(2000).nullable().optional(),
  jenis: z.enum(['UNIT', 'FAKULTAS', 'JURUSAN', 'BIRO', 'BAGIAN']).optional(),
  parentId: z.uuid().nullable().optional(),
})

export type OrganizationCreate = z.infer<typeof organizationCreateSchema>

/** Satu entri jejak audit pada GET /api/v1/audit-logs. */
export const auditLogSchema = z
  .object({
    id: z.string(),
    aktorId: z.string().nullable(),
    aksi: z.string(),
    sasaranId: z.string().nullable(),
    keterangan: z.string().nullable(),
    ip: z.string().nullable(),
    createdAt: z.string(),
  })
  .loose()

export type AuditLog = z.infer<typeof auditLogSchema>

export const auditLogPageSchema = z.object({
  items: z.array(auditLogSchema),
  page: z.number(),
  pageSize: z.number(),
  totalItems: z.number(),
  totalPages: z.number(),
})

export type AuditLogPage = z.infer<typeof auditLogPageSchema>

/** Kategori pegawai pada GET /api/v1/categories. */
export const categorySchema = z
  .object({
    id: z.uuid(),
    kode: z.string(),
    nama: z.string(),
    usiaPensiun: z.number(),
    aktif: z.boolean(),
    createdAt: z.string(),
    updatedAt: z.string(),
  })
  .loose()

export type Category = z.infer<typeof categorySchema>

export const categoryPageSchema = z.object({
  items: z.array(categorySchema),
  page: z.number(),
  pageSize: z.number(),
  totalItems: z.number(),
  totalPages: z.number(),
})

export type CategoryPage = z.infer<typeof categoryPageSchema>

/** Hasil impor pegawai dari POST /api/v1/employees/import. */
export const employeeImportResultSchema = z.object({
  imported: z.number(),
})

export type EmployeeImportResult = z.infer<typeof employeeImportResultSchema>

/** Slip gaji (Payroll) dari API penggajian. Nominal sudah diformat Rupiah. */
export const payrollTunjanganItemSchema = z.object({
  jenis: z.string(),
  nilai: z.string(),
})

export const payrollPotonganItemSchema = z.object({
  jenis: z.string(),
  nominal: z.string(),
})

export const payrollSlipSchema = z
  .object({
    id: z.uuid(),
    nomorSlip: z.string(),
    userId: z.uuid().optional(),
    nama: z.string().optional(),
    periode: z.string(),
    periodeLabel: z.string().optional(),
    status: z.string(),
    statusLabel: z.string().optional(),
    tanggalGaji: z.string(),
    gajiPokok: z.string(),
    pendapatan: z.array(payrollTunjanganItemSchema).optional(),
    potongan: z.array(payrollPotonganItemSchema).optional(),
    totalGaji: z.string(),
    tanggalDibayar: z.string().nullable().optional(),
  })
  .loose()

export type PayrollSlip = z.infer<typeof payrollSlipSchema>

export const payrollPageSchema = z.object({
  items: z.array(payrollSlipSchema),
  page: z.number(),
  pageSize: z.number(),
  totalItems: z.number(),
  totalPages: z.number(),
})

export type PayrollPage = z.infer<typeof payrollPageSchema>

/** Jadwal tanggal gajian per unit. */
export const payrollScheduleSchema = z
  .object({
    unitId: z.uuid(),
    unitNama: z.string().optional(),
    tanggalGajian: z.number(),
    updatedAt: z.string(),
  })
  .loose()

export type PayrollSchedule = z.infer<typeof payrollScheduleSchema>

export const payrollScheduleListSchema = z.object({
  items: z.array(payrollScheduleSchema),
})

export type PayrollScheduleList = z.infer<typeof payrollScheduleListSchema>

/** Payload memperbarui organisasi via PATCH /api/v1/organizations/{organizationID}. */
export const organizationUpdateSchema = z
  .object({
    name: z.string().min(1).max(200).optional(),
    description: z.string().max(2000).nullable().optional(),
    jenis: z.enum(['UNIT', 'FAKULTAS', 'JURUSAN', 'BIRO', 'BAGIAN']).optional(),
    parentId: z.uuid().nullable().optional(),
  })
  .refine((data) => Object.keys(data).length > 0, {
    message: 'Minimal satu field harus diisi',
  })

export type OrganizationUpdate = z.infer<typeof organizationUpdateSchema>

/**
 * Satu baris catatan kehadiran dari GET /api/v1/attendance/my-records.
 */
export const attendanceRecordSchema = z.object({
  id: z.string(),
  date: z.string(),
  status: z.string(),
  note: z.string().nullable(),
  source: z.string().nullable(),
})

export type AttendanceRecord = z.infer<typeof attendanceRecordSchema>

/** Satu halaman daftar kehadiran. */
export const attendancePageSchema = z.object({
  items: z.array(attendanceRecordSchema),
  page: z.number(),
  pageSize: z.number(),
  totalItems: z.number(),
  totalPages: z.number(),
})

export type AttendancePage = z.infer<typeof attendancePageSchema>

/**
 * Satu hari agregat kehadiran unit dari GET /api/v1/attendance/unit-summary.
 * `counts` selalu memuat keenam kode status; hari tanpa pegawai tercatat
 * tidak muncul pada respons.
 */
export const attendanceSummaryDaySchema = z.object({
  date: z.string(),
  counts: z.object({
    H: z.number(),
    I: z.number(),
    S: z.number(),
    A: z.number(),
    C: z.number(),
    LD: z.number(),
  }),
})

export type AttendanceSummaryDay = z.infer<typeof attendanceSummaryDaySchema>

export const attendanceSummarySchema = z.array(attendanceSummaryDaySchema)

export type AttendanceSummary = z.infer<typeof attendanceSummarySchema>

/**
 * Satu pegawai aktif beserta catatan presensinya pada satu tanggal (GET
 * /api/v1/attendance/unit-records). `status` null berarti belum tercatat;
 * bukan alpa. `unitId`/`unitName` null untuk pegawai tanpa penempatan unit.
 */
export const unitDayRecordSchema = z.object({
  id: z.string(),
  nama: z.string(),
  nip: z.string().nullable(),
  unitId: z.string().nullable(),
  unitName: z.string().nullable(),
  status: z.enum(['H', 'I', 'S', 'A', 'C', 'LD']).nullable(),
  note: z.string().nullable(),
})

export type UnitDayRecord = z.infer<typeof unitDayRecordSchema>

export const unitDayRecordListSchema = z.array(unitDayRecordSchema)

export type UnitDayRecordList = z.infer<typeof unitDayRecordListSchema>

/** Satu pengajuan cuti dari GET /api/v1/cuti/{cutiId}. */
export const cutiSchema = z
  .object({
    id: z.uuid(),
    userId: z.uuid(),
    jenis: z.enum([
      'TAHUNAN',
      'PANJANG',
      'BERSALIN',
      'SAKIT',
      'SAKIT_BERKEPANJANGAN',
      'IZIN_PENTING',
      'POLITIK',
    ]),
    tanggalMulai: z.string(),
    tanggalSelesai: z.string(),
    alamatCuti: z.string().nullable().optional(),
    alasan: z.string().nullable().optional(),
    status: z.enum(['MENUNGGU', 'DISETUJUI', 'DITOLAK']),
    disetujuiBy: z.string().nullable().optional(),
    catatan: z.string().nullable().optional(),
    subJenis: z
      .enum([
        'MENIKAH',
        'MELAHIRKAN_ISTRI',
        'MENIKAHKAN_ANAK',
        'KELUARGA_INTI',
        'KELUARGA_LAIN',
      ])
      .nullable()
      .optional(),
    nomorSurat: z.string().nullable().optional(),
    tanggalSuratDokter: z.string().nullable().optional(),
    jenisEviden: z.enum(['SURAT_DOKTER', 'FOTO', 'LAINNYA']).nullable().optional(),
    fileEvidenUrl: z.string().nullable().optional(),
    waktuEviden: z.string().nullable().optional(),
    catatanEviden: z.string().nullable().optional(),
    persenGaji: z.number().nullable().optional(),
    untukHaji: z.boolean().nullable().optional(),
    createdAt: z.string(),
    updatedAt: z.string(),
  })
  .loose()

export type Cuti = z.infer<typeof cutiSchema>

export const cutiPageSchema = z.object({
  items: z.array(cutiSchema),
  page: z.number(),
  pageSize: z.number(),
  totalItems: z.number(),
  totalPages: z.number(),
})

export type CutiPage = z.infer<typeof cutiPageSchema>

/** Payload pengajuan cuti baru via POST /api/v1/cuti. */
export const cutiCreateSchema = z.object({
  userId: z.uuid(),
  jenis: z.enum([
    'TAHUNAN',
    'PANJANG',
    'BERSALIN',
    'SAKIT',
    'SAKIT_BERKEPANJANGAN',
    'IZIN_PENTING',
    'POLITIK',
  ]),
  tanggalMulai: z.string().min(1, 'Tanggal mulai wajib diisi'),
  tanggalSelesai: z.string().min(1, 'Tanggal selesai wajib diisi'),
  alamatCuti: z.string().max(500).nullable().optional(),
  alasan: z.string().min(1, 'Alasan wajib diisi').max(1000),
  subJenis: z
    .enum([
      'MENIKAH',
      'MELAHIRKAN_ISTRI',
      'MENIKAHKAN_ANAK',
      'KELUARGA_INTI',
      'KELUARGA_LAIN',
    ])
    .nullable()
    .optional(),
  nomorSurat: z.string().max(100).nullable().optional(),
  tanggalSuratDokter: z.string().nullable().optional(),
  jenisEviden: z.enum(['SURAT_DOKTER', 'FOTO', 'LAINNYA']).nullable().optional(),
  fileEvidenUrl: z.string().max(500).nullable().optional(),
  waktuEviden: z.string().nullable().optional(),
  catatanEviden: z.string().max(1000).nullable().optional(),
  persenGaji: z.number().min(0).max(100).nullable().optional(),
  untukHaji: z.boolean().nullable().optional(),
})

export type CutiCreate = z.infer<typeof cutiCreateSchema>

/** Payload perubahan pengajuan cuti via PATCH /api/v1/cuti/{cutiId}. */
export const cutiUpdateSchema = z
  .object({
    jenis: z
      .enum([
        'TAHUNAN',
        'PANJANG',
        'BERSALIN',
        'SAKIT',
        'SAKIT_BERKEPANJANGAN',
        'IZIN_PENTING',
        'POLITIK',
      ])
      .optional(),
    tanggalMulai: z.string().min(1).optional(),
    tanggalSelesai: z.string().min(1).optional(),
    alamatCuti: z.string().max(500).nullable().optional(),
    alasan: z.string().min(1).max(1000).optional(),
    subJenis: z
      .enum([
        'MENIKAH',
        'MELAHIRKAN_ISTRI',
        'MENIKAHKAN_ANAK',
        'KELUARGA_INTI',
        'KELUARGA_LAIN',
      ])
      .nullable()
      .optional(),
    nomorSurat: z.string().max(100).nullable().optional(),
    tanggalSuratDokter: z.string().nullable().optional(),
    jenisEviden: z.enum(['SURAT_DOKTER', 'FOTO', 'LAINNYA']).nullable().optional(),
    fileEvidenUrl: z.string().max(500).nullable().optional(),
    waktuEviden: z.string().nullable().optional(),
    catatanEviden: z.string().max(1000).nullable().optional(),
    persenGaji: z.number().min(0).max(100).nullable().optional(),
    untukHaji: z.boolean().nullable().optional(),
  })
  .refine((data) => Object.keys(data).length > 0, {
    message: 'Minimal satu field harus diisi',
  })

export type CutiUpdate = z.infer<typeof cutiUpdateSchema>

/** Payload setuju/tolak cuti via POST /api/v1/cuti/{cutiId}|approve|reject. */
export const cutiActionSchema = z.object({
  catatan: z.string().max(1000).nullable().optional(),
})

export type CutiAction = z.infer<typeof cutiActionSchema>
