package extended

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
	return "extended"
}

type fundingResponse struct {
	Data []struct {
		Rate        float64 `json:"rate"`
		Symbol      string  `json:"assetName"`
		IsActive    bool    `json:"active"`
		MarketStats struct {
			Rate string `json:"fundingRate"`
		} `json:"marketStats"`
	} `json:"data"`
	Status string `json:"status"`
}

func (c *Client) FetchFundingRates(ctx context.Context) ([]domain.FundingRate, error) {
	url := fmt.Sprintf("%s/v1/info/markets", c.config.BaseURL)

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
		if item.IsActive {
			rate, _ := strconv.ParseFloat(item.MarketStats.Rate, 64)
			rates = append(rates, domain.FundingRate{
				Exchange:  c.Name(),
				Symbol:    item.Symbol,
				Rate:      rate,
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
