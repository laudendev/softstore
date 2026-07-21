package models

import "testing"

func TestPriceDollars(t *testing.T) {
	cases := []struct {
		cents int64
		want  string
	}{
		{1999, "19.99"},
		{999, "9.99"},
		{1, "0.01"},
		{0, "0.00"},
		{10000, "100.00"},
		{5, "0.05"},
		{100, "1.00"},
	}

	for _, c := range cases {
		p := Product{PriceCents: c.cents}
		got := p.PriceDollars()
		if got != c.want {
			t.Errorf("PriceCents=%d: got %q, want %q", c.cents, got, c.want)
		}
	}
}
