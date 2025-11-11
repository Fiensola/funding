package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/fiensola/funding/internal/domain"
	"github.com/fiensola/funding/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	tracker *service.TrackerService
	logger  *zap.Logger
}

func NewHandler(tracker *service.TrackerService, logger *zap.Logger) *Handler {
	return &Handler{
		tracker: tracker,
		logger:  logger,
	}
}

type Symbol struct {
	Exchanges map[string]float64   `json:"exchanges"`
	UpdatedAt map[string]time.Time `json:"updated_at"`
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	//cors
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	api := r.Group("/api/v1")
	{
		api.GET("/funding-rates", h.GetFundingRates)
	}
	r.Static("/assets", "./web/build/assets")
	r.StaticFile("/", "./web/build/index.html")
	r.NoRoute(func(c *gin.Context) {
		c.File("./web/build/index.html")
	})
}

func (h *Handler) GetFundingRates(c *gin.Context) {
	var filter domain.FundingRateFilter

	if exchange := c.Query("exchange"); exchange != "" {
		filter.Exchange = &exchange
	}

	if symbol := c.Query("symbol"); symbol != "" {
		filter.Symbol = &symbol
	}

	if limit := c.Query("limit"); limit != "" {
		if lim, err := strconv.Atoi(limit); err == nil {
			filter.Limit = lim
		}
	} else {
		filter.Limit = 0
	}

	if offset := c.Query("offset"); offset != "" {
		if off, err := strconv.Atoi(offset); err == nil {
			filter.Offset = off
		}
	}

	filter.SortBy = c.DefaultQuery("sort_by", "timestamp")
	filter.SortBy = c.DefaultQuery("sort_order", "desc")

	rates, err := h.tracker.GetLatestRates(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("failed to get funding rates", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	//format for frontend
	symbols := make(map[string]Symbol)
	for _, rate := range rates {
		if _, ok := symbols[rate.Symbol]; !ok {
			inner := Symbol{
				Exchanges: map[string]float64{
					rate.Exchange: rate.Rate * 100,
				},
				UpdatedAt: map[string]time.Time{
					rate.Exchange: rate.Timestamp,
				},
			}
			symbols[rate.Symbol] = inner
		} else {
			symbols[rate.Symbol].Exchanges[rate.Exchange] = rate.Rate * 100
			symbols[rate.Symbol].UpdatedAt[rate.Exchange] = rate.Timestamp
		}

	}

	c.JSON(http.StatusOK, gin.H{
		"data":  symbols,
		"count": len(symbols),
	})
}
