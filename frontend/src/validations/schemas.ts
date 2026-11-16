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

/** Jawaban verifikasi publik slip gaji: nama, periode, status — tanpa nominal. */
export const publicSlipVerificationSchema = z
  .object({
    nama: z.string(),
    periode: z.string(),
    periodeLabel: z.string().optional(),
    status: z.string(),
    statusLabel: z.string().optional(),
  })
  .loose()

export type PublicSlipVerification = z.infer<typeof publicSlipVerificationSchema>

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
    status: z.enum(['MENUNGGU', 'DISETUJUI', 'DITOLAK', 'BATAL']),
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
    subJenisBersalin: z.enum(['MELAHIRKAN', 'KEGUGURAN']).nullable().optional(),
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
  subJenisBersalin: z.enum(['MELAHIRKAN', 'KEGUGURAN']).nullable().optional(),
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
    subJenisBersalin: z.enum(['MELAHIRKAN', 'KEGUGURAN']).nullable().optional(),
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

/** Taxonomy kategori status pegawai untuk matriks penggajian. */
export const kategoriStatusPegawaiSchema = z
  .object({
    id: z.uuid(),
    kode: z.string(),
    nama: z.string(),
    deskripsi: z.string().nullable().optional(),
    gajiDariNegara: z.boolean(),
    butuhKontrak: z.boolean(),
    wajibBpjs: z.boolean(),
    bupSesuaiYayasan: z.boolean(),
    aktif: z.boolean(),
  })
  .loose()

export const kategoriStatusPegawaiListSchema = z.array(kategoriStatusPegawaiSchema)
export type KategoriStatusPegawai = z.infer<typeof kategoriStatusPegawaiSchema>

/** Baris nominal skala gaji per tingkat × kategori. */
export const payrollScaleDetailSchema = z
  .object({
    id: z.uuid(),
    skalaGajiId: z.uuid(),
    kategoriStatusId: z.uuid(),
    kategoriKode: z.string(),
    kategoriNama: z.string(),
    tingkat: z.number().int().min(0).max(99),
    nominal: z.number().min(0),
  })
  .loose()

export type PayrollScaleDetail = z.infer<typeof payrollScaleDetailSchema>

/** Master skala gaji pokok yayasan. */
export const payrollScaleSchema = z
  .object({
    id: z.uuid(),
    kode: z.string(),
    nama: z.string(),
    berlakuMulai: z.string(),
    berlakuSelesai: z.string().nullable().optional(),
    aktif: z.boolean(),
    createdAt: z.string(),
    updatedAt: z.string(),
  })
  .loose()

export type PayrollScale = z.infer<typeof payrollScaleSchema>

export const payrollScaleWithDetailsSchema = payrollScaleSchema.extend({
  details: z.array(payrollScaleDetailSchema),
})

export type PayrollScaleWithDetails = z.infer<typeof payrollScaleWithDetailsSchema>

export const payrollScalePageSchema = z.object({
  items: z.array(payrollScaleSchema),
  page: z.number(),
  pageSize: z.number(),
  totalItems: z.number(),
  totalPages: z.number(),
})

export type PayrollScalePage = z.infer<typeof payrollScalePageSchema>

export const payrollScaleCreateSchema = z.object({
  kode: z.string().min(1, 'Kode skala wajib diisi').max(50),
  nama: z.string().min(1, 'Nama skala wajib diisi').max(200),
  berlakuMulai: z.string().min(1, 'Tanggal berlaku wajib diisi'),
  berlakuSelesai: z.string().nullable().optional(),
})

export type PayrollScaleCreate = z.infer<typeof payrollScaleCreateSchema>

export const payrollScaleUpdateSchema = payrollScaleCreateSchema
  .extend({
    kode: z.string().min(1).max(50).optional(),
    nama: z.string().min(1).max(200).optional(),
    berlakuMulai: z.string().optional(),
    aktif: z.boolean().optional(),
  })
  .refine((data) => Object.keys(data).length > 0, {
    message: 'Minimal satu field harus diisi',
  })

export type PayrollScaleUpdate = z.infer<typeof payrollScaleUpdateSchema>

export const payrollScaleDetailCreateSchema = z.object({
  kategoriStatusId: z.uuid('Kategori wajib dipilih'),
  tingkat: z.number().int().min(0).max(99),
  nominal: z.number().min(0),
})

export type PayrollScaleDetailCreate = z.infer<typeof payrollScaleDetailCreateSchema>

export const payrollScaleDetailUpdateSchema = payrollScaleDetailCreateSchema
  .partial()
  .refine((data) => Object.keys(data).length > 0, {
    message: 'Minimal satu field harus diisi',
  })

export type PayrollScaleDetailUpdate = z.infer<typeof payrollScaleDetailUpdateSchema>

/** Tarif honor per SKS untuk dosen LB. */
export const honorSksRateSchema = z
  .object({
    id: z.uuid(),
    berlakuMulai: z.string(),
    perSks: z.number().min(0),
    aktif: z.boolean(),
    createdAt: z.string(),
    updatedAt: z.string(),
  })
  .loose()

export type HonorSksRate = z.infer<typeof honorSksRateSchema>
export const honorSksRateListSchema = z.array(honorSksRateSchema)
export const honorSksRateCreateSchema = z.object({
  berlakuMulai: z.string().min(1, 'Tanggal berlaku wajib diisi'),
  perSks: z.number().min(0),
})
export type HonorSksRateCreate = z.infer<typeof honorSksRateCreateSchema>
export const honorSksRateUpdateSchema = honorSksRateCreateSchema
  .extend({
    berlakuMulai: z.string().optional(),
    perSks: z.number().min(0).optional(),
    aktif: z.boolean().optional(),
  })
  .refine((data) => Object.keys(data).length > 0, {
    message: 'Minimal satu field harus diisi',
  })
export type HonorSksRateUpdate = z.infer<typeof honorSksRateUpdateSchema>

/** Beban mengajar dosen LB per periode akademik. */
export const honorSksPeriodeSchema = z
  .object({
    id: z.uuid(),
    userId: z.uuid(),
    periodeAkademikId: z.uuid(),
    sks: z.number().min(0).max(200),
    ratePerSks: z.number().min(0),
    totalHonor: z.number().min(0),
    skId: z.string().nullable().optional(),
    createdAt: z.string(),
    updatedAt: z.string(),
  })
  .loose()

export type HonorSksPeriode = z.infer<typeof honorSksPeriodeSchema>
export const honorSksPeriodePageSchema = z.object({
  items: z.array(honorSksPeriodeSchema),
  page: z.number(),
  pageSize: z.number(),
  totalItems: z.number(),
  totalPages: z.number(),
})
export type HonorSksPeriodePage = z.infer<typeof honorSksPeriodePageSchema>
export const honorSksPeriodeCreateSchema = z.object({
  userId: z.uuid(),
  periodeAkademikId: z.uuid(),
  sks: z.number().min(0).max(200),
  ratePerSks: z.number().min(0),
})
export type HonorSksPeriodeCreate = z.infer<typeof honorSksPeriodeCreateSchema>
export const honorSksPeriodeUpdateSchema = z
  .object({
    sks: z.number().min(0).max(200).optional(),
    ratePerSks: z.number().min(0).optional(),
  })
  .refine((data) => Object.keys(data).length > 0, {
    message: 'Minimal satu field harus diisi',
  })
export type HonorSksPeriodeUpdate = z.infer<typeof honorSksPeriodeUpdateSchema>

/** Konfigurasi tarif iuran BPJS per jenis dan periode. */
export const bpjsKonfigurasiSchema = z
  .object({
    id: z.uuid(),
    jenis: z.enum(['KESEHATAN', 'KETENAGAKERJAAN']),
    iuranPegawaiPct: z.number().min(0).max(100),
    iuranPemberiKerjaPct: z.number().min(0).max(100),
    dasar: z.enum(['GAJI_POKOK', 'TOTAL_GAJI']),
    batasAtas: z.number().nullable().optional(),
    berlakuMulai: z.string(),
    aktif: z.boolean(),
    createdAt: z.string(),
    updatedAt: z.string(),
  })
  .loose()

export type BpjsKonfigurasi = z.infer<typeof bpjsKonfigurasiSchema>
/** Pratinjau kompensasi pengajuan pemberhentian (sebelum approve). */
export const kompensasiPreviewSchema = z
  .object({
    honoured: z.boolean(),
    tersedia: z.boolean(),
    masaKerjaTahun: z.number().int().min(0),
    pesangonBulan: z.number().min(0),
    tambahanBulan: z.number().min(0),
    totalBulan: z.number().min(0),
  })
  .loose()
export type KompensasiPreview = z.infer<typeof kompensasiPreviewSchema>
/** Satu pengajuan/keputusan pemberhentian. */
export const pemberhentianSchema = z
  .object({
    id: z.string(),
    userId: z.string(),
    jenis: z.enum(['BUP', 'ATAS_PERMINTAAN_SENDIRI', 'HORMAT', 'TIDAK_HORMAT']),
    alasan: z.string(),
    tanggal: z.string(),
    status: z.enum(['MENUNGGU', 'DISETUJUI', 'DITOLAK']),
    nomorSk: z.string(),
    hakTerakhir: z.string().nullable().optional(),
    keterangan: z.string().nullable().optional(),
    kompensasiBulan: z.number(),
    createdAt: z.string(),
  })
  .loose()
export type Pemberhentian = z.infer<typeof pemberhentianSchema>
export const pemberhentianListSchema = z.object({
  items: z.array(pemberhentianSchema),
  total: z.number(),
})
export type PemberhentianList = z.infer<typeof pemberhentianListSchema>
/** Payload pengajuan pemberhentian via POST /api/v1/pemberhentian. */
export const pemberhentianProposeSchema = z.object({
  userId: z.uuid(),
  alasan: z.enum([
    'MENINGGAL_DUNIA',
    'SAKIT_BERKEPANJANGAN',
    'BUP',
    'MENGUNDUR_DIRI',
    'PENYEDERHANAAN_ORGANISASI',
    'KESALAHAN_BERAT',
  ]),
  tanggal: z.string().min(1, 'Tanggal wajib diisi'),
  keterangan: z.string().max(1000).nullable().optional(),
})
export type PemberhentianPropose = z.infer<typeof pemberhentianProposeSchema>
/** Satu kontrak kerja. */
export const kontrakSchema = z
  .object({
    id: z.string(),
    userId: z.string(),
    jenis: z.enum(['PKWT', 'PKWTT']),
    nomorSk: z.string(),
    mulai: z.string(),
    selesai: z.string().nullable().optional(),
    status: z.string(),
    skId: z.string().nullable().optional(),
    createdAt: z.string(),
  })
  .loose()
export type Kontrak = z.infer<typeof kontrakSchema>
export const kontrakListSchema = z.object({
  items: z.array(kontrakSchema),
  total: z.number(),
})
/** Payload perpanjangan perjanjian kerja tahunan pasca-BUP. */
export const perpanjanganSchema = z.object({
  userId: z.uuid(),
  mulai: z.string().nullable().optional(),
  durasiBulan: z.number().int().min(1).max(60).nullable().optional(),
  allowOverlap: z.boolean().nullable().optional(),
})
export type Perpanjangan = z.infer<typeof perpanjanganSchema>
/** Satu baris tabel aturan pesangon (admin input). */
export const pesangonAturanSchema = z
  .object({
    id: z.string(),
    kategoriStatusId: z.string(),
    masaKerjaTahun: z.number().int().min(0),
    pesangonBulan: z.number().min(0),
    penghargaanBulan: z.number().min(0),
    gantiKerugianBulan: z.number().min(0),
    sumberRegulasi: z.string(),
  })
  .loose()
export type PesangonAturan = z.infer<typeof pesangonAturanSchema>
export const pesangonAturanListSchema = z.object({
  items: z.array(pesangonAturanSchema),
})
/** Payload membuat/mengganti satu baris aturan pesangon. */
export const pesangonAturanSaveSchema = z.object({
  kategoriStatusId: z.uuid(),
  masaKerjaTahun: z.number().int().min(0),
  pesangonBulan: z.number().min(0),
  penghargaanBulan: z.number().min(0),
  gantiKerugianBulan: z.number().min(0),
  sumberRegulasi: z.string().nullable().optional(),
})
/** Satu baris aturan pesangon (admin input). */
/** Satu konfigurasi THR per tahun. */
export const thrKonfigurasiSchema = z
  .object({
    tahun: z.number().int().min(2026),
    besaranTipe: z.enum(['SATU_GAJI_POKOK', 'GAJI_KOTOR', 'PERSEN', 'PERSEN_TAKE_HOME']),
    persen: z.number().min(0).max(400).nullable().optional(),
    syaratMasaKerjaBulan: z.number().int().min(0).max(60),
    berlakuTanggal: z.string().min(1, 'Tanggal berlaku wajib diisi'),
    updatedAt: z.string(),
  })
  .loose()
export type ThrConfig = z.infer<typeof thrKonfigurasiSchema>
export const thrConfigListSchema = z
  .object({ items: z.array(thrKonfigurasiSchema) })
  .loose()
  .transform((value) => value.items)
/** Payload membuat/mengganti konfigurasi THR. */
export const thrKonfigurasiUpsertSchema = z.object({
  tahun: z.number().int().min(2026),
  besaranTipe: z.enum(['SATU_GAJI_POKOK', 'GAJI_KOTOR', 'PERSEN', 'PERSEN_TAKE_HOME']),
  persen: z.number().min(0).max(400).nullable().optional(),
  syaratMasaKerjaBulan: z.number().int().min(0).max(60),
  berlakuTanggal: z.string().min(1, 'Tanggal berlaku wajib diisi'),
})
export type ThrKonfigurasiUpsert = z.infer<typeof thrKonfigurasiUpsertSchema>
/** Laporan hasil generate THR tahun tertentu. */
export const thrGenerateReportSchema = z.object({
  tahun: z.number().int(),
  besaranTipe: z.string(),
  berlakuTanggal: z.string(),
  generated: z.number().int().min(0),
  created: z
    .array(
      z
        .object({
          userId: z.string(),
          nama: z.string(),
          nominal: z.number().min(0),
          sumber: z.string(),
        })
        .loose(),
    )
    .optional(),
  skipped: z.array(
    z
      .object({
        userId: z.string(),
        nama: z.string().optional(),
        reason: z.string(),
      })
      .loose(),
  ),
})
export type ThrGenerateReport = z.infer<typeof thrGenerateReportSchema>
/** Rencana THR: kandidat dengan nominal otomatis + yang dilewati. */
export const thrPlanSchema = z.object({
  tahun: z.number().int(),
  besaranTipe: z.string(),
  berlakuTanggal: z.string(),
  items: z.array(
    z
      .object({
        userId: z.string(),
        nama: z.string(),
        masaKerjaTahun: z.number().int().min(0),
        nominalOtomatis: z.number().min(0),
        sumber: z.string(),
      })
      .loose(),
  ),
  lewat: z.array(
    z
      .object({
        userId: z.string(),
        nama: z.string().optional(),
        reason: z.string(),
      })
      .loose(),
  ),
})
export type ThrPlan = z.infer<typeof thrPlanSchema>
export const bpjsKonfigurasiListSchema = z.array(bpjsKonfigurasiSchema)
export const bpjsKonfigurasiCreateSchema = z.object({
  jenis: z.enum(['KESEHATAN', 'KETENAGAKERJAAN']),
  iuranPegawaiPct: z.number().min(0).max(100),
  iuranPemberiKerjaPct: z.number().min(0).max(100),
  dasar: z.enum(['GAJI_POKOK', 'TOTAL_GAJI']),
  batasAtas: z.number().nullable().optional(),
  berlakuMulai: z.string().min(1, 'Tanggal berlaku wajib diisi'),
})
export type BpjsKonfigurasiCreate = z.infer<typeof bpjsKonfigurasiCreateSchema>
export const bpjsKonfigurasiUpdateSchema = bpjsKonfigurasiCreateSchema
  .extend({
    jenis: z.enum(['KESEHATAN', 'KETENAGAKERJAAN']).optional(),
    iuranPegawaiPct: z.number().min(0).max(100).optional(),
    iuranPemberiKerjaPct: z.number().min(0).max(100).optional(),
    dasar: z.enum(['GAJI_POKOK', 'TOTAL_GAJI']).optional(),
    berlakuMulai: z.string().optional(),
    aktif: z.boolean().optional(),
  })
  .refine((data) => Object.keys(data).length > 0, {
    message: 'Minimal satu field harus diisi',
  })
export type BpjsKonfigurasiUpdate = z.infer<typeof bpjsKonfigurasiUpdateSchema>

/** Kepesertaan BPJS per pegawai. */
export const bpjsPesertaSchema = z
  .object({
    id: z.uuid(),
    userId: z.uuid(),
    jenis: z.enum(['KESEHATAN', 'KETENAGAKERJAAN']),
    nomorKepesertaan: z.string().nullable().optional(),
    mulai: z.string(),
    selesai: z.string().nullable().optional(),
    aktif: z.boolean(),
    createdAt: z.string(),
    updatedAt: z.string(),
  })
  .loose()

export type BpjsPeserta = z.infer<typeof bpjsPesertaSchema>
export const bpjsPesertaPageSchema = z.object({
  items: z.array(bpjsPesertaSchema),
  page: z.number(),
  pageSize: z.number(),
  totalItems: z.number(),
  totalPages: z.number(),
})
export type BpjsPesertaPage = z.infer<typeof bpjsPesertaPageSchema>
export const bpjsPesertaCreateSchema = z.object({
  userId: z.uuid(),
  jenis: z.enum(['KESEHATAN', 'KETENAGAKERJAAN']),
  mulai: z.string().min(1, 'Tanggal mulai wajib diisi'),
  selesai: z.string().nullable().optional(),
  nomorKepesertaan: z.string().nullable().optional(),
})
export type BpjsPesertaCreate = z.infer<typeof bpjsPesertaCreateSchema>
export const bpjsPesertaUpdateSchema = z
  .object({
    nomorKepesertaan: z.string().nullable().optional(),
    mulai: z.string().optional(),
    selesai: z.string().nullable().optional(),
    aktif: z.boolean().optional(),
  })
  .refine((data) => Object.keys(data).length > 0, {
    message: 'Minimal satu field harus diisi',
  })
export type BpjsPesertaUpdate = z.infer<typeof bpjsPesertaUpdateSchema>

/** Satu baris tarif efektif PPh 21 TER (PMK 168/2023). */
export const pph21TerTarifSchema = z
  .object({
    id: z.uuid(),
    kategori: z.enum(['A', 'B', 'C']),
    lapisan: z.number().int().min(1),
    batasBawah: z.number().min(0),
    batasAtas: z.number().nullable().optional(),
    tarifPct: z.number().min(0).max(100),
    berlakuMulai: z.string(),
  })
  .loose()

export type Pph21TerTarif = z.infer<typeof pph21TerTarifSchema>
export const pph21TerTarifListSchema = z.array(pph21TerTarifSchema)
export const pph21TerTarifCreateSchema = z.object({
  kategori: z.enum(['A', 'B', 'C']),
  lapisan: z.number().int().min(1),
  batasBawah: z.number().min(0),
  batasAtas: z.number().nullable().optional(),
  tarifPct: z.number().min(0).max(100),
  berlakuMulai: z.string().min(1, 'Tanggal berlaku wajib diisi'),
})
export type Pph21TerTarifCreate = z.infer<typeof pph21TerTarifCreateSchema>

/** Satu pemetaan machinePin mesin absensi ke pegawai. */
export const mesinAbsensiPinSchema = z
  .object({
    id: z.uuid(),
    machinePin: z.string().min(1),
    userId: z.uuid(),
    nama: z.string(),
    aktif: z.boolean(),
    catatan: z.string().nullable().optional(),
    createdAt: z.string(),
  })
  .loose()

export type MesinAbsensiPin = z.infer<typeof mesinAbsensiPinSchema>
export const mesinAbsensiPinListSchema = z.object({
  items: z.array(mesinAbsensiPinSchema),
})
export type MesinAbsensiPinList = z.infer<typeof mesinAbsensiPinListSchema>
export const mesinAbsensiPinCreateSchema = z.object({
  machinePin: z.string().min(1, 'PIN mesin wajib diisi').max(100),
  userId: z.uuid('UUID pegawai'),
  catatan: z.string().max(200).nullable().optional(),
})
export type MesinAbsensiPinCreate = z.infer<typeof mesinAbsensiPinCreateSchema>
export const pinImportIssueSchema = z
  .object({
    row: z.number().int(),
    field: z.string(),
    code: z.string(),
    message: z.string(),
  })
  .loose()

export type PinImportIssue = z.infer<typeof pinImportIssueSchema>
export const pinImportResultSchema = z.object({
  imported: z.number().int(),
  issues: z.array(pinImportIssueSchema),
})
export type PinImportResult = z.infer<typeof pinImportResultSchema>
export const derivePresensiRequestSchema = z.object({
  from: z.string().min(1, 'Tanggal awal wajib diisi'),
  to: z.string().min(1, 'Tanggal akhir wajib diisi'),
})
export type DerivePresensiRequest = z.infer<typeof derivePresensiRequestSchema>
export const derivePresensiResponseSchema = z
  .object({
    from: z.string(),
    to: z.string(),
    created: z.number().int(),
    updated: z.number().int(),
    skippedManual: z.number().int(),
    byStatus: z.record(z.string(), z.number().int()),
    auditFailed: z.boolean().optional(),
  })
  .loose()

export type DerivePresensiResponse = z.infer<typeof derivePresensiResponseSchema>
export const alpaSuspectSchema = z
  .object({
    userId: z.uuid(),
    nama: z.string(),
    homebaseId: z.string().nullable().optional(),
    alpa: z.record(z.string(), z.number().int()),
    totalAlpa: z.number().int(),
  })
  .loose()

export type AlpaSuspect = z.infer<typeof alpaSuspectSchema>
export const alpaSuspectsResponseSchema = z.object({
  months: z.number().int(),
  items: z.array(alpaSuspectSchema),
})
export type AlpaSuspectsResponse = z.infer<typeof alpaSuspectsResponseSchema>
export const presensiUpsertRequestSchema = z.object({
  userId: z.uuid('UUID pegawai'),
  tanggal: z.string().min(1, 'Tanggal wajib diisi'),
  status: z.enum(['H', 'I', 'S', 'A', 'C', 'LD'], {
    message: 'Status harus H/I/S/A/C/LD',
  }),
  keterangan: z.string().max(500).nullable().optional(),
})
export type PresensiUpsertRequest = z.infer<typeof presensiUpsertRequestSchema>
export const presensiUpsertResponseSchema = z.object({
  userId: z.uuid(),
  tanggal: z.string(),
  status: z.enum(['H', 'I', 'S', 'A', 'C', 'LD']),
  keterangan: z.string().nullable().optional(),
  created: z.boolean(),
})
export type PresensiUpsertResponse = z.infer<typeof presensiUpsertResponseSchema>

export const rbacRoleSchema = z
  .object({
    id: z.number().int(),
    code: z.string(),
    name: z.string(),
    tierLevel: z.number().int(),
  })
  .loose()
export type RbacRole = z.infer<typeof rbacRoleSchema>
export const rbacRoleListSchema = z.object({ items: z.array(rbacRoleSchema) })

export const rbacPermissionSchema = z
  .object({
    id: z.number().int(),
    module: z.string(),
    action: z.string(),
  })
  .loose()
export type RbacPermission = z.infer<typeof rbacPermissionSchema>
export const rbacPermissionListSchema = z.object({
  items: z.array(rbacPermissionSchema),
})

export const rbacUserRoleScopeSchema = z
  .object({
    id: z.number().int(),
    userId: z.string(),
    roleId: z.number().int(),
    biroCode: z.string().nullable().optional(),
    fakultasId: z.string().nullable().optional(),
    jurusanId: z.string().nullable().optional(),
    unitId: z.string().nullable().optional(),
    deniedPermissions: z.unknown().optional(),
  })
  .loose()
export type RbacUserRoleScope = z.infer<typeof rbacUserRoleScopeSchema>
export const rbacUserRoleScopeListSchema = z.object({
  items: z.array(rbacUserRoleScopeSchema),
})

export const rbacDelegasiSchema = z
  .object({
    id: z.number().int().optional(),
    pemberiUserId: z.string().optional(),
    penerimaUserId: z.string().optional(),
    cakupan: z.string().optional(),
    mulai: z.string().optional(),
    selesai: z.string().optional(),
    aktif: z.boolean().optional(),
  })
  .loose()
export type RbacDelegasi = z.infer<typeof rbacDelegasiSchema>
export const rbacDelegasiListSchema = z.object({
  items: z.array(rbacDelegasiSchema),
})

export const pensiunKonfigurasiSchema = z
  .object({
    id: z.uuid(),
    kategoriStatusId: z.uuid(),
    bupUmur: z.number().int(),
    dplkWajib: z.boolean(),
    iuranPegawaiPct: z.number(),
    iuranPemberiKerjaPct: z.number(),
    penyediaDplk: z.string().nullable().optional(),
    berlakuMulai: z.string(),
    aktif: z.boolean(),
    createdAt: z.string().optional(),
  })
  .loose()
export type PensiunKonfigurasi = z.infer<typeof pensiunKonfigurasiSchema>
export const pensiunKonfigurasiListSchema = z.object({
  items: z.array(pensiunKonfigurasiSchema),
})

export const pensiunKepesertaanSchema = z
  .object({
    id: z.uuid(),
    userId: z.uuid(),
    nama: z.string().optional(),
    penyediaDplk: z.string(),
    nomorKepesertaan: z.string().nullable().optional(),
    tanggalMulaiPensiun: z.string().nullable().optional(),
    mulai: z.string(),
    aktif: z.boolean(),
    createdAt: z.string().optional(),
  })
  .loose()
export type PensiunKepesertaan = z.infer<typeof pensiunKepesertaanSchema>
export const pensiunKepesertaanListSchema = z.object({
  items: z.array(pensiunKepesertaanSchema),
})

export const skSchema = z
  .object({
    id: z.uuid(),
    nomorSk: z.string(),
    jenis: z.string(),
    penerbit: z.string(),
    userId: z.string().optional(),
    nama: z.string().optional(),
    homebaseId: z.string().nullable().optional(),
    tanggalSk: z.string(),
    mulaiBerlaku: z.string().nullable().optional(),
    selesaiBerlaku: z.string().nullable().optional(),
    fileUrl: z.string().nullable().optional(),
    metadata: z.string().optional(),
    createdAt: z.string().optional(),
  })
  .loose()
export type Sk = z.infer<typeof skSchema>
export const skListSchema = z.object({
  items: z.array(skSchema),
  page: z.number().optional(),
  pageSize: z.number().optional(),
  totalItems: z.number().optional(),
  totalPages: z.number().optional(),
})
export type SkList = z.infer<typeof skListSchema>
