package httpserver

import (
	"net/http"
	"strings"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/timeutil"
)

// CutiDTO adalah bentuk publik pengajuan cuti pegawai.
type CutiDTO struct {
	ID                 string                  `json:"id"`
	UserID             string                  `json:"userId"`
	Jenis              JenisCutiDTO            `json:"jenis"`
	TanggalMulai       time.Time               `json:"tanggalMulai"`
	TanggalSelesai     time.Time               `json:"tanggalSelesai"`
	AlamatCuti         *string                 `json:"alamatCuti"`
	Alasan             *string                 `json:"alasan"`
	Status             string                  `json:"status"`
	DisetujuiBy        *string                 `json:"disetujuiBy"`
	Catatan            *string                 `json:"catatan"`
	SubJenis           *SubJenisIzinPentingDTO `json:"subJenis"`
	NomorSurat         *string                 `json:"nomorSurat"`
	TanggalSuratDokter *time.Time              `json:"tanggalSuratDokter"`
	JenisEviden        *JenisEvidenDTO         `json:"jenisEviden"`
	FileEvidenUrl      *string                 `json:"fileEvidenUrl"`
	WaktuEviden        *time.Time              `json:"waktuEviden"`
	CatatanEviden      *string                 `json:"catatanEviden"`
	PersenGaji         float64                 `json:"persenGaji"`
	UntukHaji          bool                    `json:"untukHaji"`
	CreatedAt          time.Time               `json:"createdAt"`
	UpdatedAt          time.Time               `json:"updatedAt"`
}

// JenisCutiDTO adalah representasi string jenis cuti.
type JenisCutiDTO string

const (
	JenisCutiTahunanDTO             JenisCutiDTO = "TAHUNAN"
	JenisCutiPanjangDTO             JenisCutiDTO = "PANJANG"
	JenisCutiBersalinDTO            JenisCutiDTO = "BERSALIN"
	JenisCutiSakitDTO               JenisCutiDTO = "SAKIT"
	JenisCutiSakitBerkepanjanganDTO JenisCutiDTO = "SAKIT_BERKEPANJANGAN"
	JenisCutiIzinPentingDTO         JenisCutiDTO = "IZIN_PENTING"
	JenisCutiPolitikDTO             JenisCutiDTO = "POLITIK"
)

// SubJenisIzinPentingDTO adalah representasi string sub jenis izin penting.
type SubJenisIzinPentingDTO string

const (
	SubJenisMenikahDTO         SubJenisIzinPentingDTO = "MENIKAH"
	SubJenisMelahirkanIstriDTO SubJenisIzinPentingDTO = "MELAHIRKAN_ISTRI"
	SubJenisMenikahkanAnakDTO  SubJenisIzinPentingDTO = "MENIKAHKAN_ANAK"
	SubJenisKeluargaIntiDTO    SubJenisIzinPentingDTO = "KELUARGA_INTI"
	SubJenisKeluargaLainDTO    SubJenisIzinPentingDTO = "KELUARGA_LAIN"
)

// JenisEvidenDTO adalah representasi string jenis bukti pengajuan cuti.
type JenisEvidenDTO string

const (
	JenisEvidenSuratDokterDTO JenisEvidenDTO = "SURAT_DOKTER"
	JenisEvidenFotoDTO        JenisEvidenDTO = "FOTO"
	JenisEvidenLainnyaDTO     JenisEvidenDTO = "LAINNYA"
)

func cutiDTO(item models.Cuti) CutiDTO {
	return CutiDTO{
		ID:                 item.ID,
		UserID:             item.UserId,
		Jenis:              JenisCutiDTO(item.Jenis),
		TanggalMulai:       timeutil.InLocation(item.TanggalMulai),
		TanggalSelesai:     timeutil.InLocation(item.TanggalSelesai),
		AlamatCuti:         item.AlamatCuti,
		Alasan:             item.Alasan,
		Status:             string(item.Status),
		DisetujuiBy:        item.DisetujuiBy,
		Catatan:            item.Catatan,
		SubJenis:           (*SubJenisIzinPentingDTO)(item.SubJenis),
		NomorSurat:         item.NomorSurat,
		TanggalSuratDokter: item.TanggalSuratDokter,
		JenisEviden:        (*JenisEvidenDTO)(item.JenisEviden),
		FileEvidenUrl:      item.FileEvidenUrl,
		WaktuEviden:        item.WaktuEviden,
		CatatanEviden:      item.CatatanEviden,
		PersenGaji:         item.PersenGaji,
		UntukHaji:          item.UntukHaji,
		CreatedAt:          timeutil.InLocation(item.CreatedAt),
		UpdatedAt:          timeutil.InLocation(item.UpdatedAt),
	}
}

type cutiCreateRequest struct {
	UserID             string   `json:"userId"`
	Jenis              string   `json:"jenis"`
	TanggalMulai       string   `json:"tanggalMulai"`
	TanggalSelesai     string   `json:"tanggalSelesai"`
	AlamatCuti         *string  `json:"alamatCuti"`
	Alasan             *string  `json:"alasan"`
	SubJenis           *string  `json:"subJenis"`
	NomorSurat         *string  `json:"nomorSurat"`
	TanggalSuratDokter *string  `json:"tanggalSuratDokter"`
	JenisEviden        *string  `json:"jenisEviden"`
	FileEvidenUrl      *string  `json:"fileEvidenUrl"`
	WaktuEviden        *string  `json:"waktuEviden"`
	CatatanEviden      *string  `json:"catatanEviden"`
	PersenGaji         *float64 `json:"persenGaji"`
	UntukHaji          *bool    `json:"untukHaji"`
}

type cutiActionRequest struct {
	Catatan *string `json:"catatan"`
}

func (api directoryAPI) listCuti(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, domain.CutiSortFields(), "tanggalMulai")
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	userID := r.URL.Query().Get("userId")
	status := r.URL.Query().Get("status")
	if err != nil || textLength(search) > listing.MaxSearchLength || (userID != "" && !validUUID(userID)) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{
		Search: search,
		UserID: userID,
		Status: status,
	})
	page, err := api.leave.List(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}
	items := make([]CutiDTO, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, cutiDTO(item))
	}
	writeJSON(w, http.StatusOK, PageDTO[CutiDTO]{Items: items, Page: page.Page, PageSize: page.PageSize, TotalItems: page.TotalItems, TotalPages: page.TotalPages})
}

func (api directoryAPI) getCuti(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("cutiId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.leave.Get(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cutiDTO(item))
}

func (api directoryAPI) createCuti(w http.ResponseWriter, r *http.Request) {
	var request cutiCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	tanggalMulai, err := time.Parse("2006-01-02", request.TanggalMulai)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	tanggalSelesai, err := time.Parse("2006-01-02", request.TanggalSelesai)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	var subJenis *models.SubJenisIzinPenting
	if request.SubJenis != nil {
		v := models.SubJenisIzinPenting(*request.SubJenis)
		subJenis = &v
	}
	var tanggalSuratDokter *time.Time
	if request.TanggalSuratDokter != nil {
		t, err := time.Parse("2006-01-02", *request.TanggalSuratDokter)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
			return
		}
		tanggalSuratDokter = &t
	}
	var jenisEviden *models.JenisEviden
	if request.JenisEviden != nil {
		v := models.JenisEviden(*request.JenisEviden)
		jenisEviden = &v
	}
	var waktuEviden *time.Time
	if request.WaktuEviden != nil {
		t, err := time.Parse(time.RFC3339, *request.WaktuEviden)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
			return
		}
		waktuEviden = &t
	}
	var persenGaji float64
	if request.PersenGaji != nil {
		persenGaji = *request.PersenGaji
	}
	var untukHaji bool
	if request.UntukHaji != nil {
		untukHaji = *request.UntukHaji
	}
	input := domain.CreateCuti{
		UserID:             request.UserID,
		Jenis:              models.JenisCuti(request.Jenis),
		TanggalMulai:       tanggalMulai,
		TanggalSelesai:     tanggalSelesai,
		AlamatCuti:         request.AlamatCuti,
		Alasan:             request.Alasan,
		SubJenis:           subJenis,
		NomorSurat:         request.NomorSurat,
		TanggalSuratDokter: tanggalSuratDokter,
		JenisEviden:        jenisEviden,
		FileEvidenUrl:      request.FileEvidenUrl,
		WaktuEviden:        waktuEviden,
		CatatanEviden:      request.CatatanEviden,
		PersenGaji:         persenGaji,
		UntukHaji:          untukHaji,
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	created, err := api.leave.Create(r.Context(), principal, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, cutiDTO(created))
}

func (api directoryAPI) updateCuti(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("cutiId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	fields, err := decodePatchFields(w, r, domain.CutiPatchFields)
	if err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	updated, err := api.leave.Update(r.Context(), principal, id, domain.NewCutiPatch(fields))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cutiDTO(updated))
}

func (api directoryAPI) approveCuti(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("cutiId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	var request cutiActionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	updated, err := api.leave.Approve(r.Context(), principal, id, request.Catatan)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cutiDTO(updated))
}

func (api directoryAPI) rejectCuti(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("cutiId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	var request cutiActionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	updated, err := api.leave.Reject(r.Context(), principal, id, request.Catatan)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cutiDTO(updated))
}
