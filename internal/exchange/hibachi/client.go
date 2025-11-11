package hibachi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/fiensola/funding/internal/domain"
	"github.com/fiensola/funding/internal/exchange"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Client struct {
	config     exchange.Config
	httpClient *http.Client
	logger     *zap.Logger
}

type infoResponse struct {
	Contracts []struct {
		Pair   string `json:"symbol"`
		Symbol string `json:"underlyingSymbol"`
	} `json:"futureContracts"`
	Status string `json:"status"`
}

type symbolResponse struct {
	Price    string `json:"markPrice"`
	RateInfo struct {
		Rate string `json:"estimatedFundingRate"`
	} `json:"fundingRateEstimation"`
}

func NewClient(config exchange.Config, logger *zap.Logger) *Client {
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

func (c *Client) Name() string {
	return "hibachi"
}

func (c *Client) FetchFundingRates(ctx context.Context) ([]domain.FundingRate, error) {
	rates := make([]domain.FundingRate, 0)

	symbols, err := c.getSymbols(ctx)
	if err != nil {
		return nil, err
	}

	eg := errgroup.Group{}
	for symbol, pair := range symbols {
		eg.Go(func() error {
			url := fmt.Sprintf("%s/market/data/prices?symbol=%s", c.config.BaseURL, pair)

			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return fmt.Errorf("create request: %w", err)
			}

			req.Header.Set("Content-Type", "application/json")

			resp, err := c.httpClient.Do(req)
			if err != nil {
				return fmt.Errorf("execute request: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			}

			var symbolResponse symbolResponse
			if err := json.NewDecoder(resp.Body).Decode(&symbolResponse); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}

			now := time.Now()
			rate, _ := strconv.ParseFloat(symbolResponse.RateInfo.Rate, 64)
			rates = append(rates, domain.FundingRate{
				Exchange:  c.Name(),
				Symbol:    symbol,
				Rate:      rate,
				Timestamp: now,
			})

			return nil
		})
	}

	eg.Wait()

	c.logger.Info("fetched funding rates",
		zap.String("exchange", c.Name()),
		zap.Int("count", len(rates)),
	)

	return rates, nil
}

func (c *Client) getSymbols(ctx context.Context) (map[string]string, error) {
	infoUrl := fmt.Sprintf("%s/market/exchange-info", c.config.BaseURL)

	reqInfo, err := http.NewRequestWithContext(ctx, "GET", infoUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	reqInfo.Header.Set("Content-Type", "application/json")

	respInfo, err := c.httpClient.Do(reqInfo)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer respInfo.Body.Close()

	if respInfo.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", respInfo.StatusCode)
	}

	var infoResp infoResponse
	if err := json.NewDecoder(respInfo.Body).Decode(&infoResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if infoResp.Status != "NORMAL" {
		return nil, fmt.Errorf("unexpected info status: %s", infoResp.Status)
	}

	result := make(map[string]string)
	for _, contract := range infoResp.Contracts {
		result[contract.Symbol] = contract.Pair
	}

	return result, nil
}
