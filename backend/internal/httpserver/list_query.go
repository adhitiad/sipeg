package httpserver

import (
	"net/http"
	"slices"
	"strings"

	"simpeg/backend/internal/listing"
)

// listQuery memuat parameter daftar yang sudah divalidasi.
type listQuery struct {
	Page     int
	PageSize int
	Sort     string
	Order    string
}

// parseListQuery memvalidasi parameter pagination, sort, dan order.
// Sort hanya diterima bila berada dalam allowlist resource; kolom SQL
// ditentukan di sisi store, bukan dari input mentah pengguna.
func parseListQuery(r *http.Request, allowedSort []string, defaultSort string) (listQuery, error) {
	page, pageSize, err := pagination(r)
	if err != nil {
		return listQuery{}, err
	}

	sortField := strings.TrimSpace(r.URL.Query().Get("sort"))
	if sortField == "" {
		sortField = defaultSort
	}
	if !slices.Contains(allowedSort, sortField) {
		return listQuery{}, errInvalidSort
	}

	order := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("order")))
	switch order {
	case "", "asc":
		order = "asc"
	case "desc":
		order = "desc"
	default:
		return listQuery{}, errInvalidOrder
	}

	return listQuery{Page: page, PageSize: pageSize, Sort: sortField, Order: order}, nil
}

// errInvalidSort dan errInvalidOrder menandai parameter daftar yang tidak dikenal.
var (
	errInvalidSort  = &listParamError{param: "sort"}
	errInvalidOrder = &listParamError{param: "order"}
)

type listParamError struct {
	param string
}

func (e *listParamError) Error() string {
	return "invalid " + e.param + " parameter"
}

// listOptionsFromQuery menyatukan parameter daftar dan filter resource.
func listOptionsFromQuery(query listQuery, options listing.Options) listing.Options {
	options.Sort = query.Sort
	options.Order = query.Order
	options.Page = query.Page
	options.PageSize = query.PageSize
	return options
}
