package handlers

import (
	"math"
	"testing"
)

func TestPriceRounding(t *testing.T) {
	cases := []struct {
		dollars float64
		want    int64
	}{
		{19.99, 1999},
		{9.99, 999},
		{0.01, 1},
		{100.00, 10000},
		{5.55, 555},
	}

	for _, c := range cases {
		got := int64(math.Round(c.dollars * 100))
		if got != c.want {
			t.Errorf("dollars=%.2f: got %d cents, want %d cents", c.dollars, got, c.want)
		}
	}
}
