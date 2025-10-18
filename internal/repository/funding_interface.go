package repository

import (
	"context"

	"github.com/fiensola/funding/internal/domain"
)

type FundingRepository interface {
	CreateBatch(ctx context.Context, rates []domain.FundingRate) error
	GetLatest(ctx context.Context, filter domain.FundingRateFilter) ([]domain.FundingRate, error)
}
