package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IDModel struct {
	ID string `gorm:"column:id;type:uuid;primaryKey"`
}

func (m *IDModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}

type BaseModel struct {
	IDModel
	CreatedAt time.Time `gorm:"column:createdAt;type:timestamp(3)"`
	UpdatedAt time.Time `gorm:"column:updatedAt;type:timestamp(3)"`
}

type User struct {
	BaseModel
	Email             string `gorm:"uniqueIndex"`
	EmailVerified     bool
	Image             *string
	Nama              string
	Nik               *string       `gorm:"uniqueIndex"`
	Nip               *string       `gorm:"uniqueIndex"`
	Nidn              *string       `gorm:"uniqueIndex"`
	Nidk              *string       `gorm:"uniqueIndex"`
	JenisKelamin      *JenisKelamin `gorm:"column:jenis_kelamin"`
	TanggalLahir      *time.Time    `gorm:"column:tanggal_lahir;type:timestamp(3)"`
	NoHp              *string       `gorm:"column:no_hp"`
	NoWa              *string       `gorm:"column:no_wa"`
	Alamat            *string
	Desa              *string
	Kecamatan         *string
	Kota              *string
	Provinsi          *string
	JabatanStruktural *string `gorm:"column:jabatan_struktural"`
	JabatanFungsional *string `gorm:"column:jabatan_fungsional"`
	KategoriPegawaiId *string `gorm:"type:uuid"`
	BankNama          *string
	BankRekening      *string
	BankAtasNama      *string
	TanggalMasuk      *time.Time `gorm:"type:timestamp(3)"`
	GajiPokok         *float64
	Npwp              *string
	StatusPtkp        *string
	NoKk              *string `gorm:"column:no_kk"`
	NoKkEnc           bool    `gorm:"column:no_kk_encrypted"`
	RoleId            *string `gorm:"type:uuid"`
	HomebaseId        *string `gorm:"type:uuid"`
	TwoFactorEnabled  *bool
	Aktif             bool `gorm:"default:true"`
}

type Homebase struct {
	BaseModel
	Name        string `gorm:"uniqueIndex"`
	Description *string
	Jenis       JenisOrganisasi `gorm:"type:JenisOrganisasi;default:UNIT"`
	ParentId    *string         `gorm:"type:uuid;index:homebase_parent_idx"`
	Aktif       bool            `gorm:"default:true"`
	Latitude    *float64
	Longitude   *float64
	RadiusM     *int
}

type Kategori struct {
	BaseModel
	Kode        string `gorm:"uniqueIndex"`
	Nama        string
	UsiaPensiun int
	Aktif       bool
}

type Delegasi struct {
	BaseModel
	PemberiUserId  string `gorm:"type:uuid"`
	PenerimaUserId string `gorm:"type:uuid;index:delegasi_penerima_idx"`
	Cakupan        CakupanDelegasi
	Mulai          time.Time `gorm:"type:date"`
	Selesai        time.Time `gorm:"type:date"`
	Aktif          bool      `gorm:"index:delegasi_penerima_idx"`
}

type RiwayatLogin struct {
	IDModel
	UserId    string `gorm:"type:uuid;index:riwayat_login_user_idx"`
	Ip        *string
	UserAgent *string
	CreatedAt time.Time `gorm:"column:createdAt;type:timestamp(3)"`
}

type Role struct {
	BaseModel
	Name        string `gorm:"uniqueIndex"`
	Description *string
}

type RiwayatJabatan struct {
	BaseModel
	UserId         string `gorm:"type:uuid;index:riwayat_jabatan_user_idx"`
	Jenis          JenisJabatan
	Jabatan        string
	TanggalMulai   time.Time  `gorm:"type:timestamp(3);index:riwayat_jabatan_user_idx"`
	TanggalSelesai *time.Time `gorm:"type:timestamp(3)"`
	NomorSk        *string
	Keterangan     *string
}

type RiwayatPendidikan struct {
	BaseModel
	UserId     string `gorm:"type:uuid;index:riwayat_pendidikan_user_idx"`
	Jenjang    string
	Institusi  string
	Jurusan    *string
	TahunLulus *int
	Keterangan *string
}

type RiwayatGaji struct {
	IDModel
	UserId      string `gorm:"type:uuid;index:riwayat_gaji_user_idx"`
	GajiLama    *float64
	GajiBaru    float64
	Sumber      string
	ReferensiId *string   `gorm:"type:uuid"`
	DiubahOleh  *string   `gorm:"type:uuid"`
	CreatedAt   time.Time `gorm:"column:createdAt;type:timestamp(3)"`
}

type Session struct {
	BaseModel
	UserId    string    `gorm:"type:uuid;index:session_user_idx"`
	ExpiresAt time.Time `gorm:"type:timestamp(3)"`
	Token     string    `gorm:"uniqueIndex"`
	IpAddress *string
	UserAgent *string
}

type Account struct {
	BaseModel
	UserId                string `gorm:"type:uuid;index:account_user_idx"`
	ProviderId            string
	AccountId             string
	AccessToken           *string
	RefreshToken          *string
	IdToken               *string
	AccessTokenExpiresAt  *time.Time `gorm:"type:timestamp(3)"`
	RefreshTokenExpiresAt *time.Time `gorm:"type:timestamp(3)"`
	Scope                 *string
	Password              *string
}

type Verification struct {
	BaseModel
	Identifier string `gorm:"index:verification_identifier_idx"`
	Value      string
	ExpiresAt  time.Time `gorm:"type:timestamp(3)"`
}

type TwoFactor struct {
	IDModel
	Secret                  string `gorm:"index"`
	BackupCodes             string
	UserId                  string `gorm:"type:uuid;uniqueIndex:twoFactor_user_id;index:twoFactor_user_idx"`
	Verified                *bool
	FailedVerificationCount *int
	LockedUntil             *time.Time `gorm:"type:timestamp(3)"`
}
