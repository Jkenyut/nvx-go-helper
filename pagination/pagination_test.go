package pagination

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFromInt(t *testing.T) {
	tests := []struct {
		name  string
		page  int
		limit int
		total int
		want  Pagination
	}{
		{
			name:  "valid input",
			page:  3,
			limit: 25,
			total: 1000,
			want: Pagination{
				Page:       3,
				Limit:      25,
				Total:      1000,
				TotalPages: 40,
				HasNext:    true,
				HasPrev:    true,
				NextPage:   4,
				PrevPage:   2,
			},
		},
		{
			name:  "default values (no default limit imposed)",
			page:  0,
			limit: 0,
			total: 50,
			want: Pagination{
				Page:       1,
				Limit:      0,
				Total:      50,
				TotalPages: 0,
				HasNext:    false,
				HasPrev:    false,
				NextPage:   0,
				PrevPage:   0,
			},
		},
		{
			name:  "limit 99999 → allowed",
			page:  1,
			limit: 99999,
			total: 10,
			want: Pagination{
				Page:       1,
				Limit:      99999,
				Total:      10,
				TotalPages: 1,
				HasNext:    false,
				HasPrev:    false,
				NextPage:   0,
				PrevPage:   0,
			},
		},
		{
			name:  "negative page → forced to 1",
			page:  -5,
			limit: 10,
			total: 100,
			want: Pagination{
				Page:       1,
				Limit:      10,
				Total:      100,
				TotalPages: 10,
				HasNext:    true,
				HasPrev:    false,
				NextPage:   2,
				PrevPage:   0,
			},
		},
		{
			name:  "ghost page out of bounds -> cap to max page",
			page:  100,
			limit: 10,
			total: 50,
			want: Pagination{
				Page:       5,
				Limit:      10,
				Total:      50,
				TotalPages: 5,
				HasNext:    false,
				HasPrev:    true,
				NextPage:   0,
				PrevPage:   4,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewFromInt(tt.page, tt.limit, tt.total)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOffset(t *testing.T) {
	tests := []struct {
		page, limit, expected int
	}{
		{1, 10, 0},
		{2, 10, 10},
		{5, 20, 80},
		{10, 100, 900},
	}

	for _, tt := range tests {
		p := Pagination{Page: tt.page, Limit: tt.limit}
		assert.Equal(t, tt.expected, p.Offset())
	}
}

func TestLinks(t *testing.T) {
	tests := []struct {
		name     string
		p        Pagination
		baseURL  string
		expected map[string]string // only check content, not order
		wantErr  bool
	}{
		{
			name: "both next and prev",
			p: Pagination{
				Page:       5,
				Limit:      20,
				Total:      1000,
				TotalPages: 50,
				HasNext:    true,
				HasPrev:    true,
				NextPage:   6,
				PrevPage:   4,
			},
			baseURL: "https://api.example.com/users",
			expected: map[string]string{
				"next": `<https://api.example.com/users?limit=20&page=6>; rel="next"`,
				"prev": `<https://api.example.com/users?limit=20&page=4>; rel="prev"`,
			},
		},
		{
			name: "only next",
			p: Pagination{
				Page:       1,
				Limit:      10,
				Total:      25,
				TotalPages: 3,
				HasNext:    true,
				HasPrev:    false,
				NextPage:   2,
				PrevPage:   0,
			},
			baseURL: "http://localhost:8080/v1/orders",
			expected: map[string]string{
				"next": `<http://localhost:8080/v1/orders?limit=10&page=2>; rel="next"`,
			},
		},
		{
			name:    "invalid URL",
			p:       Pagination{Page: 1, Limit: 10},
			baseURL: "%%invalid%%",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			links, err := tt.p.Links(tt.baseURL)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, links)
		})
	}
}

func BenchmarkNewFromInt(b *testing.B) {
	for b.Loop() {
		_ = NewFromInt(2, 25, 1000)
	}
}

func TestNewCursorFromInt(t *testing.T) {
	cp := NewCursorFromInt(15, "next_123", "prev_123", true)
	assert.Equal(t, 15, cp.Limit)
	assert.Equal(t, "next_123", cp.NextCursor)
	assert.Equal(t, "prev_123", cp.PrevCursor)
	assert.True(t, cp.HasNext)

	negative := NewCursorFromInt(-5, "", "", false)
	assert.Equal(t, 0, negative.Limit)
}
