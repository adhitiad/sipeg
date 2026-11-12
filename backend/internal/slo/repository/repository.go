// Package repository memuat port persistensi dan adapter GORM untuk metrik
// SLO.
//
// Metrik disimpan di PostgreSQL, bukan Redis, karena Redis pada repo ini
// diperuntukkan hanya untuk cache data referensi. Penyimpanan metrik harus
// bertahan lintas restart dan dapat ditelusuri saat insiden.
package repository

import (
	"context"
	"fmt"
	"time"

	"simpeg/backend/internal/models"
	"simpeg/backend/internal/slo/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository adalah port persistensi metrik SLO.
type Repository interface {
	// RecordBatch menyimpan agregasi metrik untuk sebuah jendela waktu.
	//
	// Penyimpanan memakaiUPSERT penuh: baris yang sudah ada pada kombinasi
	// jendela, metode, dan rute yang sama digantikan seluruh nilainya.
	// Karena itu flueks tetap idempoten dan tidak menggandakan hitungan.
	RecordBatch(ctx context.Context, window domain.Window, metrics []domain.Metric) error
	// ListWindow membaca seluruh metrik pada satu jendela waktu.
	ListWindow(ctx context.Context, window domain.Window) ([]domain.Metric, error)
	// Prune menghapus metrik yang lebih lama dari batas waktu yang diberikan
	// dan mengembalikan jumlah baris yang dihapus.
	Prune(ctx context.Context, before time.Time) (int64, error)
}

// GormStore adalah adapter persistensi metrik SLO berbasis GORM.
type GormStore struct {
	DB *gorm.DB
}

// NewGormStore membuat adapter persistensi metrik SLO.
func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{DB: db}
}

// RecordBatch menyimpan agregasi metrik untuk sebuah jendela waktu.
func (s GormStore) RecordBatch(ctx context.Context, window domain.Window, metrics []domain.Metric) error {
	if len(metrics) == 0 {
		return nil
	}
	rows := make([]models.SloMetric, 0, len(metrics))
	for _, metric := range metrics {
		rows = append(rows, toModel(window, metric))
	}
	err := s.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "windowStart"},
			{Name: "method"},
			{Name: "route"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"windowEnd",
			"requestCount",
			"errorCount",
			"latencySumMs",
			"latencyMaxMs",
			"bucketLe5",
			"bucketLe10",
			"bucketLe25",
			"bucketLe50",
			"bucketLe100",
			"bucketLe200",
			"bucketLe400",
			"bucketLe800",
			"bucketLe1200",
			"bucketLe1600",
			"bucketLe2400",
			"bucketLe3200",
			"bucketLe4800",
			"bucketLe6400",
			"updatedAt",
		}),
	}).Create(&rows).Error
	if err != nil {
		return fmt.Errorf("simpan metrik slo: %w", err)
	}
	return nil
}

// ListWindow membaca seluruh metrik pada satu jendela waktu.
func (s GormStore) ListWindow(ctx context.Context, window domain.Window) ([]domain.Metric, error) {
	var rows []models.SloMetric
	err := s.DB.WithContext(ctx).
		Where("\"windowStart\" >= ? AND \"windowStart\" < ?", window.Start, window.End).
		Order("\"windowStart\" ASC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("baca metrik slo: %w", err)
	}
	metrics := make([]domain.Metric, 0, len(rows))
	for _, row := range rows {
		metrics = append(metrics, fromModel(row))
	}
	return metrics, nil
}

// Prune menghapus metrik yang lebih lama dari batas waktu yang diberikan.
func (s GormStore) Prune(ctx context.Context, before time.Time) (int64, error) {
	result := s.DB.WithContext(ctx).Where("\"windowStart\" < ?", before).Delete(&models.SloMetric{})
	if result.Error != nil {
		return 0, fmt.Errorf("hapus metrik slo lama: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// toModel mengubah metrik domain menjadi baris tabel.
func toModel(window domain.Window, metric domain.Metric) models.SloMetric {
	row := models.SloMetric{
		WindowStart:  window.Start,
		WindowEnd:    window.End,
		Method:       metric.Method,
		Route:        metric.Route,
		RequestCount: metric.Requests,
		ErrorCount:   metric.Errors,
		LatencySumMs: metric.LatencySum.Milliseconds(),
		LatencyMaxMs: metric.LatencyMax.Milliseconds(),
	}
	buckets := row.BucketCounts()
	for index := range buckets {
		if index < len(metric.Buckets) {
			buckets[index] = metric.Buckets[index]
		}
	}
	row.SetBucketCounts(buckets)
	return row
}

// fromModel mengubah baris tabel menjadi metrik domain.
func fromModel(row models.SloMetric) domain.Metric {
	metric := domain.NewMetric(row.Route, row.Method)
	metric.Requests = row.RequestCount
	metric.Errors = row.ErrorCount
	metric.LatencySum = time.Duration(row.LatencySumMs) * time.Millisecond
	metric.LatencyMax = time.Duration(row.LatencyMaxMs) * time.Millisecond
	buckets := row.BucketCounts()
	for index := range buckets {
		if index < len(metric.Buckets) {
			metric.Buckets[index] = buckets[index]
		}
	}
	// Bucket luapan diturunkan: seluruh permintaan dikurangi yang sudah
	// tertangkap bucket 5000 milidetik.
	metric.Buckets[domain.BucketOverflow] = row.RequestCount - buckets[len(buckets)-1]
	if metric.Buckets[domain.BucketOverflow] < 0 {
		metric.Buckets[domain.BucketOverflow] = 0
	}
	return metric
}
