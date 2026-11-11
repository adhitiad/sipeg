package models

import (
	"time"
)

type Pengumuman struct {
	BaseModel
	Judul           string
	Isi             string `gorm:"type:text"`
	Disematkan      bool
	DiterbitkanOleh *string `gorm:"type:uuid"`
}

type Notifikasi struct {
	IDModel
	UserId    string `gorm:"type:uuid;index:notifikasi_user_idx"`
	Judul     string
	Isi       string
	Tautan    *string
	Dibaca    bool      `gorm:"index:notifikasi_user_idx"`
	CreatedAt time.Time `gorm:"column:createdAt;type:timestamp(3)"`
}

type Lembur struct {
	BaseModel
	UserId      string    `gorm:"type:uuid;uniqueIndex:lembur_user_tanggal_key"`
	Tanggal     time.Time `gorm:"type:date;uniqueIndex:lembur_user_tanggal_key"`
	Menit       int
	Keperluan   *string
	Status      StatusPersetujuan `gorm:"index:lembur_status_idx"`
	DisetujuiBy *string           `gorm:"type:uuid"`
	Catatan     *string
}

type TugasBelajar struct {
	BaseModel
	UserId         string `gorm:"type:uuid;index:tugas_belajar_user_idx"`
	Jenjang        string
	Institusi      string
	Jurusan        *string
	TanggalMulai   time.Time `gorm:"type:date"`
	TanggalSelesai time.Time `gorm:"type:date"`
	IkatanTahun    int
	NomorSk        *string
	Status         StatusPersetujuan `gorm:"index:tugas_belajar_status_idx"`
	DisetujuiBy    *string           `gorm:"type:uuid"`
	Catatan        *string
}

type Survei struct {
	BaseModel
	Judul      string
	Deskripsi  *string
	Mulai      time.Time
	Selesai    time.Time
	Aktif      bool
	DibuatOleh *string `gorm:"type:uuid"`
}

type PertanyaanSurvei struct {
	// Skema acuan tidak menyimpan createdAt/updatedAt untuk model ini,
	// sehingga hanya IDModel yang di-embed.
	IDModel
	SurveiId string `gorm:"type:uuid;index:pertanyaan_survei_idx"`
	Teks     string
	Urutan   int
}

type JawabanSurvei struct {
	IDModel
	SurveiId     string `gorm:"type:uuid;index:jawaban_survei_idx"`
	PertanyaanId string `gorm:"type:uuid"`
	Nilai        int
	CreatedAt    time.Time `gorm:"column:createdAt;type:timestamp(3)"`
}

type PartisipasiSurvei struct {
	IDModel
	SurveiId  string    `gorm:"type:uuid;uniqueIndex:partisipasi_survei_key"`
	UserHash  string    `gorm:"uniqueIndex:partisipasi_survei_key"`
	CreatedAt time.Time `gorm:"column:createdAt;type:timestamp(3)"`
}

type Presensi struct {
	BaseModel
	UserId      string    `gorm:"type:uuid;uniqueIndex:presensi_user_tanggal_key"`
	Fingerprint *string   `gorm:"uniqueIndex"`
	Tanggal     time.Time `gorm:"uniqueIndex:presensi_user_tanggal_key"`
	Status      StatusPresensi
	Keterangan  *string
	Lokasi      *string
	Sumber      *string
	Latitude    *float64
	Longitude   *float64
	FotoUrl     *string
	CreatedBy   *string `gorm:"type:uuid"`
	UpdatedBy   *string `gorm:"type:uuid"`
}

type MesinAbsensi struct {
	BaseModel
	Nama            string
	Merek           string
	Host            *string
	Port            *int
	ComKey          *int
	Aktif           bool
	TerakhirSinkron *time.Time `gorm:"type:timestamp(3)"`
}

type AttendanceEvent struct {
	IDModel
	MachineId      string    `gorm:"column:machineId;type:uuid;not null;uniqueIndex:attendance_event_machine_idempotency_key,priority:1;index:attendance_event_machine_time_idx,priority:1"`
	MachinePin     string    `gorm:"column:machinePin;size:100;not null;index:attendance_event_pin_time_idx,priority:1"`
	OccurredAt     time.Time `gorm:"column:occurredAt;type:timestamptz(3);not null;index:attendance_event_machine_time_idx,priority:2;index:attendance_event_pin_time_idx,priority:2"`
	IdempotencyKey string    `gorm:"column:idempotencyKey;size:200;not null;uniqueIndex:attendance_event_machine_idempotency_key,priority:2"`
	CreatedAt      time.Time `gorm:"column:createdAt;type:timestamptz(3);not null"`
}

// MachineCredential menyimpan kredensial autentikasi perangkat mesin absensi.
// Nilai kredensial disimpan sebagai hash (credentialHash), bukan plaintext.
type MachineCredential struct {
	BaseModel
	MachineId      string     `gorm:"column:machineId;type:uuid;not null;index:machine_credential_machine_idx"`
	CredentialType string     `gorm:"column:credentialType;size:50;not null"`
	CredentialHash string     `gorm:"column:credentialHash;type:text;not null"`
	Active         bool       `gorm:"column:active;default:true"`
	LastUsedAt     *time.Time `gorm:"column:lastUsedAt;type:timestamp(3)"`
}

type TemplateSidikJari struct {
	BaseModel
	UserId         string `gorm:"type:uuid;uniqueIndex:template_user_jari_key;index:template_user_idx"`
	JariKe         int    `gorm:"uniqueIndex:template_user_jari_key"`
	Template       string `gorm:"type:text"`
	MesinSumber    *string
	DiperbaruiOleh *string `gorm:"type:uuid"`
}
