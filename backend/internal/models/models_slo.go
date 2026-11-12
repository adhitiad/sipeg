package models

import "time"

// SloMetric adalah agregasi metrik SLO untuk satu rute pada satu jendela
// waktu.
//
// Bucket latensi disimpan sebagai hitungan kumulatif (less-than-or-equal)
// sehingga persentil dapat dihitung ulang dari baris ini tanpa menyimpan
// setiap sampel. Satu baris mewakili satu kombinasi jendela waktu, metode,
// dan rute.
type SloMetric struct {
	BaseModel
	// WindowStart adalah awal jendela waktu agregasi dalam UTC.
	WindowStart time.Time `gorm:"column:windowStart;type:timestamp(3);uniqueIndex:slo_metric_window_route_key,priority:1;index:slo_metric_window_idx"`
	// WindowEnd adalah akhir jendela waktu agregasi dalam UTC.
	WindowEnd time.Time `gorm:"column:windowEnd;type:timestamp(3)"`
	// Method adalah metode HTTP yang diukur.
	Method string `gorm:"type:varchar(10);uniqueIndex:slo_metric_window_route_key,priority:2"`
	// Route adalah pola rute yang diukur, bukan path mentah.
	Route string `gorm:"type:varchar(200);uniqueIndex:slo_metric_window_route_key,priority:3"`
	// RequestCount adalah jumlah permintaan dalam jendela.
	RequestCount int64 `gorm:"column:requestCount"`
	// ErrorCount adalah jumlah permintaan yang berakhir dengan galat 5xx.
	ErrorCount int64 `gorm:"column:errorCount"`
	// LatencySumMs adalah jumlah latensi dalam milidetik.
	LatencySumMs int64 `gorm:"column:latencySumMs"`
	// LatencyMaxMs adalah latensi tertinggi dalam milidetik.
	LatencyMaxMs int64 `gorm:"column:latencyMaxMs"`
	// Bucket-bucket berikut menyimpan hitungan kumulatif permintaan dengan
	// latensi tidak lebih dari ambang (milidetik). Daftar ambang harus
	// selalu sama dengan domain.BucketBoundaries.
	BucketLe5    int64 `gorm:"column:bucketLe5"`
	BucketLe10   int64 `gorm:"column:bucketLe10"`
	BucketLe25   int64 `gorm:"column:bucketLe25"`
	BucketLe50   int64 `gorm:"column:bucketLe50"`
	BucketLe100  int64 `gorm:"column:bucketLe100"`
	BucketLe200  int64 `gorm:"column:bucketLe200"`
	BucketLe400  int64 `gorm:"column:bucketLe400"`
	BucketLe800  int64 `gorm:"column:bucketLe800"`
	BucketLe1200 int64 `gorm:"column:bucketLe1200"`
	BucketLe1600 int64 `gorm:"column:bucketLe1600"`
	BucketLe2400 int64 `gorm:"column:bucketLe2400"`
	BucketLe3200 int64 `gorm:"column:bucketLe3200"`
	BucketLe4800 int64 `gorm:"column:bucketLe4800"`
	BucketLe6400 int64 `gorm:"column:bucketLe6400"`
}

// BucketCounts mengembalikan hitungan kumulatif seluruh bucket teratas.
//
// Nilai returned berurutan dari ambang terkecil ke ambang terbesar.
func (m *SloMetric) BucketCounts() []int64 {
	return []int64{
		m.BucketLe5, m.BucketLe10, m.BucketLe25, m.BucketLe50, m.BucketLe100,
		m.BucketLe200, m.BucketLe400, m.BucketLe800, m.BucketLe1200, m.BucketLe1600,
		m.BucketLe2400, m.BucketLe3200, m.BucketLe4800, m.BucketLe6400,
	}
}

// SetBucketCounts menulis hitungan kumulatif seluruh bucket teratas.
//
// Bucket luapan tidak disimpan karena nilainya diturunkan dari jumlah
// permintaan dikurangi hitungan bucket terakhir.
func (m *SloMetric) SetBucketCounts(counts []int64) {
	targets := []*int64{
		&m.BucketLe5, &m.BucketLe10, &m.BucketLe25, &m.BucketLe50, &m.BucketLe100,
		&m.BucketLe200, &m.BucketLe400, &m.BucketLe800, &m.BucketLe1200, &m.BucketLe1600,
		&m.BucketLe2400, &m.BucketLe3200, &m.BucketLe4800, &m.BucketLe6400,
	}
	for index := range targets {
		if index < len(counts) {
			*targets[index] = counts[index]
		}
	}
}
