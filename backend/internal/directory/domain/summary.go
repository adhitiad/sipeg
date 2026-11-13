package domain

// UnitCount menghitung jumlah pegawai pada satu unit organisasi.
// UnitID nil berarti pegawai tanpa unit penugasan.
type UnitCount struct {
	UnitID *string
	Count  int64
}

// EmployeeSummary merangkum jumlah pegawai dalam cakupan principal.
// Populasinya identik dengan daftar pegawai (seluruh baris dalam cakupan,
// termasuk yang nonaktif) sehingga ringkasan selalu konsisten dengan data
// yang terlihat di daftar.
type EmployeeSummary struct {
	Total    int64
	Active   int64
	Inactive int64
	ByUnit   []UnitCount
}
