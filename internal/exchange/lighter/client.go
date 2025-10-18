package lighter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/fiensola/funding/internal/domain"
	"github.com/fiensola/funding/internal/exchange"
	"go.uber.org/zap"
)

type Client struct {
	config     exchange.Config
	httpClient *http.Client
	logger     *zap.Logger
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
	return "lighter"
}

type fundingResponse struct {
	Data []struct {
		Rate     float64 `json:"rate"`
		Exchange string  `json:"exchange"`
		Symbol   string  `json:"symbol"`
	} `json:"funding_rates"`
	Code int `json:"code"`
}

func (c *Client) FetchFundingRates(ctx context.Context) ([]domain.FundingRate, error) {
	url := fmt.Sprintf("%s/v1/funding-rates", c.config.BaseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var fundingResp fundingResponse
	if err := json.NewDecoder(resp.Body).Decode(&fundingResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	now := time.Now()
	rates := make([]domain.FundingRate, 0)

	for _, item := range fundingResp.Data {
		if item.Exchange == c.Name() {
			rates = append(rates, domain.FundingRate{
				Exchange:  c.Name(),
				Symbol:    item.Symbol,
				Rate:      item.Rate,
				Timestamp: now,
			})
		}
	}

	c.logger.Info("fetched funding rates",
		zap.String("exchange", c.Name()),
		zap.Int("count", len(rates)),
	)

	return rates, nil
}
