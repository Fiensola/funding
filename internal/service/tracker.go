package service

import (
	"context"
	"sync"
	"time"

	"github.com/fiensola/funding/internal/domain"
	"github.com/fiensola/funding/internal/exchange"
	"github.com/fiensola/funding/internal/repository"
	"go.uber.org/zap"
)

type TrackerService struct {
	exchanges []exchange.Exchange
	repo      repository.FundingRepository
	logger    *zap.Logger
	interval  time.Duration
	stopCh    chan struct{}
}

func NewTrackerService(
	exchanges []exchange.Exchange,
	repo repository.FundingRepository,
	logger *zap.Logger,
	interval time.Duration,
) *TrackerService {
	return &TrackerService{
		exchanges: exchanges,
		repo:      repo,
		logger:    logger,
		interval:  interval,
		stopCh:    make(chan struct{}),
	}
}

func (s *TrackerService) Start(ctx context.Context) {
	s.logger.Info("starting funding tracker",
		zap.Duration("interval", s.interval),
		zap.Int("exchanges", len(s.exchanges)),
	)

	s.FetchAndStore(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	cleanupTicker := time.NewTicker(1 * time.Hour)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ticker.C:
			s.FetchAndStore(ctx)
		case <-s.stopCh:
			s.logger.Info("stopping funding tracker")
			return
		case <-ctx.Done():
			s.logger.Info("context canceled, stopping tracker")
			return
		}
	}
}

func (s *TrackerService) FetchAndStore(ctx context.Context) {
	wg := sync.WaitGroup{}
	ratesCh := make(chan []domain.FundingRate, len(s.exchanges))

	for _, ex := range s.exchanges {
		wg.Add(1)
		go func(exchange exchange.Exchange) {
			defer wg.Done()

			rates, err := exchange.FetchFundingRates(ctx)
			if err != nil {
				s.logger.Error("failed to fetch funding rates",
					zap.String("exchange", exchange.Name()),
					zap.Error(err),
				)
				return
			}

			ratesCh <- rates
		}(ex)
	}

	// close chan when go-s complete
	go func() {
		wg.Wait()
		close(ratesCh)
	}()

	var allRates []domain.FundingRate
	for rates := range ratesCh {
		allRates = append(allRates, rates...)
	}

	if len(allRates) == 0 {
		s.logger.Warn("no funding rates fetched")
		return
	}

	if err := s.repo.CreateBatch(ctx, allRates); err != nil {
		s.logger.Error("failed to store funding rates", zap.Error(err))
		return
	}

	s.logger.Info("succesfully update funding rates", zap.Int("total", len(allRates)))
}

func (s *TrackerService) Stop() {
	close(s.stopCh)
}

func (s *TrackerService) GetLatestRates(ctx context.Context, filter domain.FundingRateFilter) ([]domain.FundingRate, error) {
	return s.repo.GetLatest(ctx, filter)
}
