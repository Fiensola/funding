package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/fiensola/funding/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type FundingRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewFundingRepository(db *pgxpool.Pool, logger *zap.Logger) *FundingRepository {
	return &FundingRepository{
		db:     db,
		logger: logger,
	}
}

func (f *FundingRepository) Create(ctx context.Context, rate domain.FundingRate) (uuid.UUID, error) {
	q := `
		INSERT INTO funding_rates (exchange, symbol, price, rate, timestamp, next_funding)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id uuid.UUID
	err := f.db.QueryRow(ctx, q,
		rate.Exchange,
		rate.Symbol,
		rate.Price,
		rate.Rate,
		rate.Timestamp,
		rate.NextFunding,
	).Scan(&id)

	if err != nil {
		return uuid.Nil, fmt.Errorf("insert funding rate: %w", err)
	}

	return id, nil
}

func (f *FundingRepository) CreateBatch(ctx context.Context, rates []domain.FundingRate) error {
	if len(rates) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	q := `
		INSERT INTO funding_rates (exchange, symbol, price, rate, timestamp, next_funding)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, rate := range rates {
		batch.Queue(q,
			rate.Exchange,
			rate.Symbol,
			rate.Price,
			rate.Rate,
			rate.Timestamp,
			rate.NextFunding,
		)
	}

	br := f.db.SendBatch(ctx, batch)
	defer br.Close()

	for i := range rates {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("batch insert at index %d: %w", i, err)
		}
	}

	return nil
}

func (f *FundingRepository) GetLatest(
	ctx context.Context,
	filter domain.FundingRateFilter,
) ([]domain.FundingRate, error) {
	q := `
		SELECT DISTINCT ON (exchange, symbol)
			id, exchange, symbol, price, rate, timestamp, next_funding, created_at
		FROM funding_rates
		WHERE (exchange='lighter' or exchange='extended')
	`

	args := []any{}
	argsCount := 1

	if filter.Exchange != nil {
		q += fmt.Sprintf(" AND exchange = $%d", argsCount)
		args = append(args, *filter.Exchange)
		argsCount++
	}

	if filter.Symbol != nil {
		q += fmt.Sprintf(" AND symbol = $%d", argsCount)
		args = append(args, *filter.Symbol)
		argsCount++
	}

	sortBy := "timestamp"
	sortOrder := "DESC"

	if filter.SortBy != "" {
		validSorts := map[string]bool{
			"rate":      true,
			"timestamp": true,
			"symbol":    true,
			"exchange":  true,
			"price":     true,
		}
		if validSorts[filter.SortBy] {
			sortBy = filter.SortBy
		}
	}

	if strings.ToUpper(filter.SortOrder) == "ASC" {
		sortOrder = "ASC"
	}

	q += " ORDER BY exchange, symbol, timestamp DESC"

	q = fmt.Sprintf(`
		SELECT * FROM (%s) as latest
		ORDER BY %s %s
	`, q, sortBy, sortOrder)

	if filter.Limit > 0 {
		q += fmt.Sprintf(" LIMIT $%d", argsCount)
		args = append(args, filter.Limit)
		argsCount++
	}

	if filter.Offset > 0 {
		q += fmt.Sprintf(" OFFSET $%d", argsCount)
		args = append(args, filter.Offset)
	}

	rows, err := f.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query funding rates: %w", err)
	}
	defer rows.Close()

	var rates []domain.FundingRate
	for rows.Next() {
		var rate domain.FundingRate
		err := rows.Scan(
			&rate.ID,
			&rate.Exchange,
			&rate.Symbol,
			&rate.Price,
			&rate.Rate,
			&rate.Timestamp,
			&rate.NextFunding,
			&rate.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		rates = append(rates, rate)
	}

	return rates, nil
}
