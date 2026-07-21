package models

import (
   "time"
   "fmt"
)

type Product struct {
    ID          int64
    Name        string
    Slug        string
    Description string
    PriceCents  int64
    FilePath    string
    CreatedAt   time.Time
}


func (p Product) PriceDollars() string {
	return fmt.Sprintf("%.2f", float64(p.PriceCents)/100)
}
