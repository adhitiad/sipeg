package httpserver

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/timeutil"
)

// AuditLogDTO adalah bentuk publik jejak audit. Isi profil pegawai tidak pernah
// disalin ke audit; hanya aktor, aksi, sasaran, keterangan minimal, dan waktu.
type AuditLogDTO struct {
	ID         string    `json:"id"`
	AktorID    *string   `json:"aktorId"`
	Aksi       string    `json:"aksi"`
	SasaranID  *string   `json:"sasaranId"`
	Keterangan *string   `json:"keterangan"`
	IP         *string   `json:"ip"`
	CreatedAt  time.Time `json:"createdAt"`
}

// CategoryDTO adalah bentuk publik kategori pegawai.
type CategoryDTO struct {
	ID          string    `json:"id"`
	Kode        string    `json:"kode"`
	Nama        string    `json:"nama"`
	UsiaPensiun int       `json:"usiaPensiun"`
	Aktif       bool      `json:"aktif"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func auditLogDTO(entry models.AuditLog) AuditLogDTO {
	return AuditLogDTO{
		ID: entry.ID, AktorID: entry.AktorId, Aksi: entry.Aksi,
		SasaranID: entry.Sasaran, Keterangan: entry.Keterangan,
		IP: entry.Ip, CreatedAt: timeutil.InLocation(entry.CreatedAt),
	}
}

func categoryDTO(category models.Kategori) CategoryDTO {
	return CategoryDTO{
		ID: category.ID, Kode: category.Kode, Nama: category.Nama,
		UsiaPensiun: category.UsiaPensiun, Aktif: category.Aktif,
		CreatedAt: timeutil.InLocation(category.CreatedAt),
		UpdatedAt: timeutil.InLocation(category.UpdatedAt),
	}
}

func (api directoryAPI) listAuditLogs(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, domain.AuditLogSortFields(), "createdAt")
	aksi := strings.TrimSpace(r.URL.Query().Get("aksi"))
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	if err != nil || textLength(aksi) > listing.MaxSearchLength || textLength(search) > listing.MaxSearchLength {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{Aksi: aksi, Search: search})

	page, err := api.audits.List(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}

	// Pembacaan jejak audit sendiri merupakan operasi sensitif dan dicatat.
	if len(page.Items) > 0 {
		api.recordAuditAccess(r.Context(), principal, aksi, len(page.Items))
	}

	items := make([]AuditLogDTO, 0, len(page.Items))
	for _, entry := range page.Items {
		items = append(items, auditLogDTO(entry))
	}
	writeJSON(w, http.StatusOK, PageDTO[AuditLogDTO]{
		Items: items, Page: page.Page, PageSize: page.PageSize,
		TotalItems: page.TotalItems, TotalPages: page.TotalPages,
	})
}

// recordAuditAccess mencatat bahwa sebuah principal membaca jejak audit.
// Kegagalan pencatatan tidak menggagalkan pembacaan.
func (api directoryAPI) recordAuditAccess(ctx context.Context, principal auth.Principal, aksi string, count int) {
	note := fmt.Sprintf("%d entri", count)
	if aksi != "" {
		note = fmt.Sprintf("%s — %s", aksi, note)
	}
	var unitID *string
	if principal.HomebaseID != "" {
		unitID = &principal.HomebaseID
	}
	_ = api.audits.Record(ctx, audit.Entry{
		ActorID: principal.UserID,
		Action:  audit.ActionAuditAccessed,
		UnitID:  unitID,
		Note:    &note,
	})
}

// exportAuditLogs meng-stream jejak audit dalam cakupan principal sebagai
// berkas CSV. Sort dan pagination diabaikan: urutan selalu kronologis naik
// (createdAt ASC, id ASC) dan tidak ada batas jumlah baris. Filter sama
// dengan endpoint daftar (aksi, search). Ekspor yang berhasil dicatat sebagai
// aksi audit.exported setelah stream selesai.
func (api directoryAPI) exportAuditLogs(w http.ResponseWriter, r *http.Request) {
	aksi := strings.TrimSpace(r.URL.Query().Get("aksi"))
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	if textLength(aksi) > listing.MaxSearchLength || textLength(search) > listing.MaxSearchLength {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())

	filename := "audit-log-" + timeutil.InLocation(time.Now()).Format("20060102T150405") + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)

	// BOM UTF-8 agar Microsoft Excel membuka berkas dengan benar.
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return
	}

	writer := csv.NewWriter(w)
	if err := writer.Write(csvAuditHeader()); err != nil {
		return
	}
	writer.Flush()

	flush := func() {
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
	flush()

	count := 0
	err := api.audits.StreamAuditLogs(r.Context(), principal, listing.Options{Aksi: aksi, Search: search}, func(entry models.AuditLog) error {
		count++
		if err := writer.Write([]string{
			entry.ID,
			timeutil.InLocation(entry.CreatedAt).Format(time.RFC3339),
			strValue(entry.AktorId),
			entry.Aksi,
			strValue(entry.Sasaran),
			strValue(entry.UnitId),
			strValue(entry.Keterangan),
			strValue(entry.Ip),
		}); err != nil {
			return err
		}
		if count%100 == 0 {
			writer.Flush()
			flush()
		}
		return nil
	})
	if err != nil {
		return
	}
	writer.Flush()
	flush()

	// Catat ekspor setelah stream selesai; stream yang terputus di tengah
	// tidak dicatat sebagai ekspor (hanya diagnostics di log aplikasi).
	note := fmt.Sprintf("%d baris diekspor", count)
	if aksi != "" {
		note = fmt.Sprintf("%s — %d baris diekspor", aksi, count)
	}
	var unitID *string
	if principal.HomebaseID != "" {
		unitID = &principal.HomebaseID
	}
	_ = api.audits.Record(r.Context(), audit.Entry{
		ActorID: principal.UserID,
		Action:  audit.ActionAuditExported,
		UnitID:  unitID,
		Note:    &note,
	})
}

// csvAuditHeader mengembalikan nama kolom berkas ekspor jejak audit.
func csvAuditHeader() []string {
	return []string{"id", "createdAt", "aktorId", "aksi", "sasaranId", "unitId", "keterangan", "ip"}
}

// strValue mengembalikan isi pointer string, atau string kosong bila nil.
func strValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (api directoryAPI) listCategories(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, domain.CategorySortFields(), "kode")
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	if err != nil || textLength(search) > listing.MaxSearchLength {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}

	options := listOptionsFromQuery(query, listing.Options{Search: search})
	page, err := api.categories.List(r.Context(), options)
	if err != nil {
		writeAppError(w, err)
		return
	}

	items := make([]CategoryDTO, 0, len(page.Items))
	for _, category := range page.Items {
		items = append(items, categoryDTO(category))
	}
	writeJSON(w, http.StatusOK, PageDTO[CategoryDTO]{
		Items: items, Page: page.Page, PageSize: page.PageSize,
		TotalItems: page.TotalItems, TotalPages: page.TotalPages,
	})
}
