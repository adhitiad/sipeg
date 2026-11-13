package app

// CategoryCacheKey mengekspor categoryCacheKey untuk pengujian.
import "simpeg/backend/internal/listing"

func CategoryCacheKey(options listing.Options) string {
	return categoryCacheKey(options)
}
