package data

import (
	"math"
	"strings"

	"github.com/mnabil1718/greenlight/internal/validator"
)

type Filters struct {
	Sort         string
	PageSize     int
	Page         int
	SortSafelist []string
}

type Metadata struct {
	CurrentPage  int `json:"current_page,omitempty"`
	PageSize     int `json:"page_size,omitempty"`
	FirstPage    int `json:"first_page,omitempty"`
	LastPage     int `json:"last_page,omitempty"`
	TotalRecords int `json:"total_records,omitempty"`
}

func ValidateFilters(validator *validator.Validator, filter *Filters) {
	validator.Check(filter.Page > 0, "page", "must be greater than zero")
	validator.Check(filter.Page <= 10_000_000, "page", "must be a maximum of 10 million")
	validator.Check(filter.PageSize > 0, "page_size", "must be greater than zero")
	validator.Check(filter.PageSize <= 100, "page_size", "must be a maximum of 100")
	validator.Check(validator.In(filter.Sort, filter.SortSafelist...), "sort", "invalid sort value")
}

func (filter Filters) sortColumn() string {
	for _, column := range filter.SortSafelist {
		if column == filter.Sort {
			return strings.TrimPrefix(filter.Sort, "-")
		}
	}

	panic("unsafe sort parameter: " + filter.Sort)
}

func (filter Filters) sortDirection() string {
	if strings.HasPrefix(filter.Sort, "-") {
		return "DESC"
	}

	return "ASC"
}

func (f Filters) limit() int {
	return f.PageSize
}
func (f Filters) offset() int {
	return (f.Page - 1) * f.PageSize
}

func calculateMetadata(totalRecords, page, pageSize int) Metadata {
	if totalRecords == 0 {
		return Metadata{} // id no records, return empty
	}
	return Metadata{
		CurrentPage:  page,
		PageSize:     pageSize,
		FirstPage:    1,
		LastPage:     int(math.Ceil(float64(totalRecords) / float64(pageSize))),
		TotalRecords: totalRecords,
	}
}
