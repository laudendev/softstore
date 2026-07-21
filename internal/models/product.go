package models

import "time"

type Product struct {
    ID          int64
    Name        string
    Slug        string
    Description string
    PriceCents  int64
    FilePath    string
    CreatedAt   time.Time
}
