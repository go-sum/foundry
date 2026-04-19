package pager

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newRequest(query string) *http.Request {
	return httptest.NewRequest(http.MethodGet, "/?"+query, nil)
}

func TestNew(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		defaultPerPage int
		maxPerPage     int
		wantPage       int
		wantPerPage    int
	}{
		{
			name:           "defaults when no query params",
			query:          "",
			defaultPerPage: DefaultPerPage,
			maxPerPage:     MaxPerPage,
			wantPage:       1,
			wantPerPage:    DefaultPerPage,
		},
		{
			name:           "explicit page and per_page",
			query:          "page=3&per_page=10",
			defaultPerPage: DefaultPerPage,
			maxPerPage:     MaxPerPage,
			wantPage:       3,
			wantPerPage:    10,
		},
		{
			name:           "page clamped to 1 when zero",
			query:          "page=0",
			defaultPerPage: DefaultPerPage,
			maxPerPage:     MaxPerPage,
			wantPage:       1,
			wantPerPage:    DefaultPerPage,
		},
		{
			name:           "page clamped to 1 when negative",
			query:          "page=-5",
			defaultPerPage: DefaultPerPage,
			maxPerPage:     MaxPerPage,
			wantPage:       1,
			wantPerPage:    DefaultPerPage,
		},
		{
			name:           "invalid page falls back to 1",
			query:          "page=abc",
			defaultPerPage: DefaultPerPage,
			maxPerPage:     MaxPerPage,
			wantPage:       1,
			wantPerPage:    DefaultPerPage,
		},
		{
			name:           "per_page capped at maxPerPage",
			query:          "per_page=500",
			defaultPerPage: DefaultPerPage,
			maxPerPage:     MaxPerPage,
			wantPage:       1,
			wantPerPage:    MaxPerPage,
		},
		{
			name:           "per_page not capped when maxPerPage is 0",
			query:          "per_page=500",
			defaultPerPage: DefaultPerPage,
			maxPerPage:     0,
			wantPage:       1,
			wantPerPage:    500,
		},
		{
			name:           "invalid per_page falls back to default",
			query:          "per_page=bad",
			defaultPerPage: DefaultPerPage,
			maxPerPage:     MaxPerPage,
			wantPage:       1,
			wantPerPage:    DefaultPerPage,
		},
		{
			name:           "zero per_page falls back to default",
			query:          "per_page=0",
			defaultPerPage: DefaultPerPage,
			maxPerPage:     MaxPerPage,
			wantPage:       1,
			wantPerPage:    DefaultPerPage,
		},
		{
			name:           "custom defaultPerPage used when per_page absent",
			query:          "",
			defaultPerPage: 50,
			maxPerPage:     MaxPerPage,
			wantPage:       1,
			wantPerPage:    50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRequest(tt.query)
			got := New(r, tt.defaultPerPage, tt.maxPerPage)
			if got.Page != tt.wantPage {
				t.Errorf("Page = %d, want %d", got.Page, tt.wantPage)
			}
			if got.PerPage != tt.wantPerPage {
				t.Errorf("PerPage = %d, want %d", got.PerPage, tt.wantPerPage)
			}
		})
	}
}

func TestSetTotal(t *testing.T) {
	tests := []struct {
		name           string
		page           int
		perPage        int
		total          int
		wantTotalItems int
		wantTotalPages int
		wantPage       int
	}{
		{
			name:           "exact multiple",
			page:           1,
			perPage:        10,
			total:          30,
			wantTotalItems: 30,
			wantTotalPages: 3,
			wantPage:       1,
		},
		{
			name:           "remainder rounds up",
			page:           1,
			perPage:        10,
			total:          31,
			wantTotalItems: 31,
			wantTotalPages: 4,
			wantPage:       1,
		},
		{
			name:           "zero total",
			page:           1,
			perPage:        10,
			total:          0,
			wantTotalItems: 0,
			wantTotalPages: 0,
			wantPage:       1,
		},
		{
			name:           "page clamped when over total",
			page:           50,
			perPage:        10,
			total:          25,
			wantTotalItems: 25,
			wantTotalPages: 3,
			wantPage:       3,
		},
		{
			name:           "page not clamped when within total",
			page:           2,
			perPage:        10,
			total:          30,
			wantTotalItems: 30,
			wantTotalPages: 3,
			wantPage:       2,
		},
		{
			name:           "page stays at last page exactly",
			page:           3,
			perPage:        10,
			total:          30,
			wantTotalItems: 30,
			wantTotalPages: 3,
			wantPage:       3,
		},
		{
			name:           "zero perPage gives zero total pages",
			page:           1,
			perPage:        0,
			total:          100,
			wantTotalItems: 100,
			wantTotalPages: 0,
			wantPage:       1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Pager{Page: tt.page, PerPage: tt.perPage}
			p.SetTotal(tt.total)
			if p.TotalItems != tt.wantTotalItems {
				t.Errorf("TotalItems = %d, want %d", p.TotalItems, tt.wantTotalItems)
			}
			if p.TotalPages != tt.wantTotalPages {
				t.Errorf("TotalPages = %d, want %d", p.TotalPages, tt.wantTotalPages)
			}
			if p.Page != tt.wantPage {
				t.Errorf("Page = %d, want %d", p.Page, tt.wantPage)
			}
		})
	}
}

func TestOffset(t *testing.T) {
	tests := []struct {
		name    string
		page    int
		perPage int
		want    int
	}{
		{"page 1 always zero", 1, 20, 0},
		{"page 0 treated as first", 0, 20, 0},
		{"page 2", 2, 20, 20},
		{"page 3", 3, 20, 40},
		{"page 5 perPage 10", 5, 10, 40},
		{"page 1 perPage 100", 1, 100, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Pager{Page: tt.page, PerPage: tt.perPage}
			got := p.Offset()
			if got != tt.want {
				t.Errorf("Offset() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestLimit(t *testing.T) {
	p := Pager{PerPage: 25}
	if p.Limit() != 25 {
		t.Errorf("Limit() = %d, want 25", p.Limit())
	}
}

func TestIsFirst(t *testing.T) {
	tests := []struct {
		name string
		page int
		want bool
	}{
		{"page 1 is first", 1, true},
		{"page 0 is first", 0, true},
		{"page 2 is not first", 2, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Pager{Page: tt.page}
			if p.IsFirst() != tt.want {
				t.Errorf("IsFirst() = %v, want %v", p.IsFirst(), tt.want)
			}
		})
	}
}

func TestIsLast(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		totalPages int
		want       bool
	}{
		{"on last page", 5, 5, true},
		{"beyond last page", 6, 5, true},
		{"not on last page", 4, 5, false},
		{"page 1 of 1", 1, 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Pager{Page: tt.page, TotalPages: tt.totalPages}
			if p.IsLast() != tt.want {
				t.Errorf("IsLast() = %v, want %v", p.IsLast(), tt.want)
			}
		})
	}
}

func TestHasPages(t *testing.T) {
	tests := []struct {
		name       string
		totalPages int
		want       bool
	}{
		{"zero pages", 0, false},
		{"one page", 1, false},
		{"two pages", 2, true},
		{"many pages", 10, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Pager{TotalPages: tt.totalPages}
			if p.HasPages() != tt.want {
				t.Errorf("HasPages() = %v, want %v", p.HasPages(), tt.want)
			}
		})
	}
}

func TestPrevPage(t *testing.T) {
	tests := []struct {
		name string
		page int
		want int
	}{
		{"page 1 stays at 1", 1, 1},
		{"page 0 stays at 1", 0, 1},
		{"page 3 returns 2", 3, 2},
		{"page 5 returns 4", 5, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Pager{Page: tt.page}
			if p.PrevPage() != tt.want {
				t.Errorf("PrevPage() = %d, want %d", p.PrevPage(), tt.want)
			}
		})
	}
}

func TestNextPage(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		totalPages int
		want       int
	}{
		{"page 5 of 5 stays at 5", 5, 5, 5},
		{"beyond last page stays at total", 6, 5, 5},
		{"page 3 of 5 returns 4", 3, 5, 4},
		{"page 1 of 10 returns 2", 1, 10, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Pager{Page: tt.page, TotalPages: tt.totalPages}
			if p.NextPage() != tt.want {
				t.Errorf("NextPage() = %d, want %d", p.NextPage(), tt.want)
			}
		})
	}
}

func sliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestPageRange(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		totalPages int
		window     int
		want       []int
	}{
		{
			name:       "zero or one total pages returns empty",
			page:       1,
			totalPages: 1,
			window:     2,
			want:       []int{},
		},
		{
			name:       "zero total pages returns empty",
			page:       1,
			totalPages: 0,
			window:     2,
			want:       []int{},
		},
		{
			name:       "window=2 page=5 total=10 canonical example",
			page:       5,
			totalPages: 10,
			window:     2,
			want:       []int{1, -1, 3, 4, 5, 6, 7, -1, 10},
		},
		{
			name:       "window=1 page=5 total=10",
			page:       5,
			totalPages: 10,
			window:     1,
			want:       []int{1, -1, 4, 5, 6, -1, 10},
		},
		{
			name:       "near start page=2 total=10 window=2",
			page:       2,
			totalPages: 10,
			window:     2,
			want:       []int{1, 2, 3, 4, -1, 10},
		},
		{
			name:       "near end page=9 total=10 window=2",
			page:       9,
			totalPages: 10,
			window:     2,
			want:       []int{1, -1, 7, 8, 9, 10},
		},
		{
			name:       "page=1 total=10 window=2 no leading ellipsis",
			page:       1,
			totalPages: 10,
			window:     2,
			want:       []int{1, 2, 3, -1, 10},
		},
		{
			name:       "page=10 total=10 window=2 no trailing ellipsis",
			page:       10,
			totalPages: 10,
			window:     2,
			want:       []int{1, -1, 8, 9, 10},
		},
		{
			name:       "small total=3 window=2 all pages shown",
			page:       2,
			totalPages: 3,
			window:     2,
			want:       []int{1, 2, 3},
		},
		{
			name:       "small total=2 window=2 all pages shown",
			page:       1,
			totalPages: 2,
			window:     2,
			want:       []int{1, 2},
		},
		{
			name:       "window=0 small total<=5 returns all pages",
			page:       3,
			totalPages: 4,
			window:     0,
			want:       []int{1, 2, 3, 4},
		},
		{
			name:       "window=0 large total returns compact form",
			page:       7,
			totalPages: 20,
			window:     0,
			want:       []int{1, -1, 7, -1, 20},
		},
		{
			name:       "window negative treated same as zero small total",
			page:       2,
			totalPages: 3,
			window:     -1,
			want:       []int{1, 2, 3},
		},
		{
			name:       "window=2 page=3 total=10 gap of 1 before window — no ellipsis",
			page:       3,
			totalPages: 10,
			window:     2,
			want:       []int{1, 2, 3, 4, 5, -1, 10},
		},
		{
			name:       "window=2 page=8 total=10 gap of 1 after window — no ellipsis",
			page:       8,
			totalPages: 10,
			window:     2,
			want:       []int{1, -1, 6, 7, 8, 9, 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Pager{Page: tt.page, TotalPages: tt.totalPages}
			got := p.PageRange(tt.window)
			if !sliceEqual(got, tt.want) {
				t.Errorf("PageRange(%d) = %v, want %v", tt.window, got, tt.want)
			}
		})
	}
}
