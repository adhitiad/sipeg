// Port persistensi baca Surat Keputusan.
package repository

import (
	"context"
	"errors"

	"simpeg/backend/internal/listing"

	"simpeg/backend/internal/sk/domain"
)

// Store adalah port persistensi SK (baca saja — penerbitan SK lahir dari alur
// kerja, bukan CRUD manual).
type Store interface {
	List(ctx context.Context, filter domain.Filter) (listing.Page[domain.Row], error)
	Get(ctx context.Context, id string) (domain.Row, error)
}

// ErrNotFound menandai SK yang tidak ditemukan.
var ErrNotFound = errors.New("surat keputusan tidak ditemukan")
