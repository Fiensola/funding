package backpack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
	transport := &http.Transport{}

	httpClient := &http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,
	}

	return &Client{
		config:     config,
		httpClient: httpClient,
		logger:     logger,
	}
}

func (c *Client) Name() string {
	return "backpack"
}

type fundingResponse []struct {
	Rate   string `json:"fundingRate"`
	Symbol string `json:"symbol"`
	Price  string `json:"markPrice"`
}

func (c *Client) FetchFundingRates(ctx context.Context) ([]domain.FundingRate, error) {
	url := fmt.Sprintf("%s/v1/markPrices", c.config.BaseURL)

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

	for _, item := range fundingResp {
		rate, _ := strconv.ParseFloat(item.Rate, 64)
		price, _ := strconv.ParseFloat(item.Price, 64)
		symbolWords := strings.Split(item.Symbol, "_")
		rates = append(rates, domain.FundingRate{
			Exchange:  c.Name(),
			Symbol:    symbolWords[0],
			Rate:      rate,
			Price:     &price,
			Timestamp: now,
		})
	}

	c.logger.Info("fetched funding rates",
		zap.String("exchange", c.Name()),
		zap.Int("count", len(rates)),
	)

	return rates, nil
}
