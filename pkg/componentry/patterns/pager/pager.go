// Package pager provides pagination calculation helpers.
package pager

import (
	"net/http"
	"strconv"
)

// DefaultPerPage is the page size used when the caller does not specify one.
const DefaultPerPage = 20

// MaxPerPage is the upper bound on per_page accepted from query params.
// Requests exceeding this are silently capped.
const MaxPerPage = 100

// Pager holds pagination state for a single page of results.
type Pager struct {
	Page       int
	PerPage    int
	TotalItems int
	TotalPages int
}

// New reads page and per_page query params from r.
// Page is clamped to >= 1. PerPage falls back to defaultPerPage when absent or invalid,
// and is capped at maxPerPage when maxPerPage > 0.
func New(r *http.Request, defaultPerPage, maxPerPage int) Pager {
	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	perPage := defaultPerPage
	if pp, err := strconv.Atoi(r.URL.Query().Get("per_page")); err == nil && pp > 0 {
		perPage = pp
	}
	if maxPerPage > 0 && perPage > maxPerPage {
		perPage = maxPerPage
	}
	return Pager{Page: page, PerPage: perPage}
}

// SetTotal updates TotalItems, computes TotalPages, and clamps Page to TotalPages.
// Clamping prevents users from staying on page 50 of a 3-page result set.
func (p *Pager) SetTotal(total int) {
	p.TotalItems = total
	if p.PerPage <= 0 {
		p.TotalPages = 0
		return
	}
	p.TotalPages = (total + p.PerPage - 1) / p.PerPage
	if p.TotalPages > 0 && p.Page > p.TotalPages {
		p.Page = p.TotalPages
	}
}

// Offset returns the SQL OFFSET value for the current page.
func (p *Pager) Offset() int {
	if p.Page <= 1 {
		return 0
	}
	return (p.Page - 1) * p.PerPage
}

// Limit is an alias for PerPage, SQL-friendly naming.
func (p *Pager) Limit() int { return p.PerPage }

func (p *Pager) IsFirst() bool { return p.Page <= 1 }

func (p *Pager) IsLast() bool { return p.Page >= p.TotalPages }

// HasPages returns true when TotalPages > 1 (i.e., pagination is needed).
func (p *Pager) HasPages() bool { return p.TotalPages > 1 }

func (p *Pager) PrevPage() int {
	if p.Page <= 1 {
		return 1
	}
	return p.Page - 1
}

func (p *Pager) NextPage() int {
	if p.Page >= p.TotalPages {
		return p.TotalPages
	}
	return p.Page + 1
}

// PageRange returns a slice of page numbers for visible pagination links.
// window is the number of pages to show on each side of the current page.
// A value of -1 in the result indicates an ellipsis should be shown.
// Example with window=2, current=5, total=10: [1, -1, 3, 4, 5, 6, 7, -1, 10]
func (p *Pager) PageRange(window int) []int {
	if p.TotalPages <= 1 {
		return []int{}
	}

	// Degenerate window: show compact form or full list for small totals.
	if window <= 0 {
		if p.TotalPages <= 5 {
			pages := make([]int, p.TotalPages)
			for i := range pages {
				pages[i] = i + 1
			}
			return pages
		}
		return []int{1, -1, p.Page, -1, p.TotalPages}
	}

	lo := p.Page - window
	hi := p.Page + window
	if lo < 1 {
		lo = 1
	}
	if hi > p.TotalPages {
		hi = p.TotalPages
	}

	var result []int

	// Leading section: always include page 1.
	result = append(result, 1)

	// Gap between 1 and lo.
	if lo > 2 {
		result = append(result, -1)
	} else if lo == 2 {
		// Adjacent — include without ellipsis.
		result = append(result, 2)
		lo = 3
	}

	// Window range (skip 1 if already added).
	start := lo
	if start <= 1 {
		start = 2
	}
	for i := start; i <= hi; i++ {
		result = append(result, i)
	}

	// Trailing section: always include last page if not already in window.
	if hi < p.TotalPages {
		if hi < p.TotalPages-1 {
			result = append(result, -1)
		}
		result = append(result, p.TotalPages)
	}

	return result
}
