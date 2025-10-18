package domain

import (
	"time"

	"github.com/google/uuid"
)

type FundingRate struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	Exchange    string     `json:"exchange" db:"exchange"`
	Symbol      string     `json:"symbol" db:"symbol"`
	Price       *float64   `json:"price,omitempty" db:"price"`
	Rate        float64    `json:"rate" db:"rate"`
	Timestamp   time.Time  `json:"timestamp" db:"timestamp"`
	NextFunding *time.Time `json:"next_funding,omitempty" db:"next_funding"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

type FundingRateFilter struct {
	Exchange  *string
	Symbol    *string
	Limit     int
	Offset    int
	SortBy    string // rate, timestamp, symbol
	SortOrder string // asc, desc
}
