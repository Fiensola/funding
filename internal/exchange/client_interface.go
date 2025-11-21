package exchange

import (
	"context"

	"github.com/fiensola/funding/internal/domain"
)

type Exchange interface {
	Name() string
	FetchFundingRates(ctx context.Context) ([]domain.FundingRate, error)
}

type Config struct {
	BaseURL  string
	Proxy    string
	IsActive bool
}
