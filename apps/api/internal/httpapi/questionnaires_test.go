package httpapi

import (
	"math"
	"testing"
)

// FRM-01 paging: anything that isn't a page number is page 1, and a huge page number stops
// at the last page whose row offset Postgres accepts (it was a 500 before, found in S01).
func TestPageNumber(t *testing.T) {
	last := math.MaxInt32 / pageSize
	for in, want := range map[string]int{
		"": 1, "abc": 1, "-2": 1, "0": 1, "1": 1, "3": 3,
		"99999999999": last, "99999999999999999999999": last,
	} {
		if got := pageNumber(in); got != want {
			t.Errorf("pageNumber(%q) = %d, want %d", in, got, want)
		}
		if offset := (pageNumber(in) - 1) * pageSize; offset < 0 || offset > math.MaxInt32 {
			t.Errorf("pageNumber(%q) gives offset %d, outside int32", in, offset)
		}
	}
}
