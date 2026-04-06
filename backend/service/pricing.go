package service

import (
	"context"
	"encoding/json"
	"eth-sweeper/model"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type PriceService struct {
	httpClient *http.Client
	mu         sync.RWMutex
	cache      map[string]model.PriceSeriesResponse
}

func NewPriceService() *PriceService {
	return &PriceService{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		cache:      map[string]model.PriceSeriesResponse{},
	}
}

func (s *PriceService) GetETHSeries(ctx context.Context, interval string) model.PriceSeriesResponse {
	if interval == "" {
		interval = "5m"
	}
	cacheKey := "eth:" + interval

	s.mu.RLock()
	cached, ok := s.cache[cacheKey]
	s.mu.RUnlock()
	if ok && time.Since(parseTimeOrZero(cached.CachedAt)) < 4*time.Minute {
		return cached
	}

	resp, err := s.fetchCoinGeckoMarketChart(ctx, interval)
	if err != nil || len(resp.Items) == 0 {
		resp = demoPriceSeries(interval)
	}

	s.mu.Lock()
	s.cache[cacheKey] = resp
	s.mu.Unlock()
	return resp
}

func (s *PriceService) fetchCoinGeckoMarketChart(ctx context.Context, interval string) (model.PriceSeriesResponse, error) {
	days := "1"
	if interval == "1w" {
		days = "90"
	} else if interval == "1d" {
		days = "30"
	}
	url := "https://api.coingecko.com/api/v3/coins/ethereum/market_chart?vs_currency=usd&days=" + days
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return model.PriceSeriesResponse{}, err
	}
	if key := os.Getenv("COINGECKO_API_KEY"); key != "" {
		req.Header.Set("x-cg-demo-api-key", key)
	}

	httpResp, err := s.httpClient.Do(req)
	if err != nil {
		return model.PriceSeriesResponse{}, err
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return model.PriceSeriesResponse{}, fmt.Errorf("coingecko status %d", httpResp.StatusCode)
	}

	var payload struct {
		Prices [][]float64 `json:"prices"`
	}
	if err := json.NewDecoder(httpResp.Body).Decode(&payload); err != nil {
		return model.PriceSeriesResponse{}, err
	}
	if len(payload.Prices) == 0 {
		return model.PriceSeriesResponse{}, fmt.Errorf("no price data")
	}

	items := compactPricePoints(payload.Prices, interval)
	return model.PriceSeriesResponse{
		Asset:    "ETH",
		Interval: interval,
		Items:    items,
		Source:   "coingecko_demo_api",
		CachedAt: nowISO(),
	}, nil
}

func compactPricePoints(prices [][]float64, interval string) []model.PricePoint {
	bucketSize := 5 * time.Minute
	if interval == "1d" {
		bucketSize = 24 * time.Hour
	} else if interval == "1w" {
		bucketSize = 7 * 24 * time.Hour
	}

	type bucket struct {
		open  float64
		high  float64
		low   float64
		close float64
		ts    time.Time
		set   bool
	}
	buckets := map[int64]*bucket{}
	keys := make([]int64, 0)
	for _, point := range prices {
		if len(point) < 2 {
			continue
		}
		ts := time.UnixMilli(int64(point[0])).UTC()
		key := ts.Truncate(bucketSize).Unix()
		b, ok := buckets[key]
		if !ok {
			b = &bucket{ts: time.Unix(key, 0).UTC()}
			buckets[key] = b
			keys = append(keys, key)
		}
		price := point[1]
		if !b.set {
			b.open = price
			b.high = price
			b.low = price
			b.close = price
			b.set = true
			continue
		}
		if price > b.high {
			b.high = price
		}
		if price < b.low {
			b.low = price
		}
		b.close = price
	}

	sortInt64(keys)
	items := make([]model.PricePoint, 0, len(keys))
	for _, key := range keys {
		b := buckets[key]
		if !b.set {
			continue
		}
		items = append(items, model.PricePoint{
			Timestamp: b.ts.Format(time.RFC3339),
			Open:      roundPrice(b.open),
			High:      roundPrice(b.high),
			Low:       roundPrice(b.low),
			Close:     roundPrice(b.close),
			Source:    "coingecko_demo_api",
		})
	}
	return items
}

func demoPriceSeries(interval string) model.PriceSeriesResponse {
	now := time.Now().UTC()
	step := 5 * time.Minute
	count := 48
	if interval == "1d" {
		step = 24 * time.Hour
		count = 30
	} else if interval == "1w" {
		step = 7 * 24 * time.Hour
		count = 12
	}

	items := make([]model.PricePoint, 0, count)
	base := 3200.0
	for i := count - 1; i >= 0; i-- {
		ts := now.Add(-time.Duration(i) * step)
		wave := float64((count-i)%9-4) * 8
		open := base + wave
		close := open + float64((count-i)%5-2)*5
		items = append(items, model.PricePoint{
			Timestamp: ts.Format(time.RFC3339),
			Open:      roundPrice(open),
			High:      roundPrice(maxFloat(open, close) + 14),
			Low:       roundPrice(minFloat(open, close) - 12),
			Close:     roundPrice(close),
			Source:    "demo_fallback",
		})
	}
	return model.PriceSeriesResponse{
		Asset:    "ETH",
		Interval: interval,
		Items:    items,
		Source:   "demo_fallback",
		CachedAt: nowISO(),
	}
}

func parseTimeOrZero(raw string) time.Time {
	t, _ := time.Parse(time.RFC3339, raw)
	return t
}

func roundPrice(v float64) float64 {
	out, _ := strconv.ParseFloat(strconv.FormatFloat(v, 'f', 2, 64), 64)
	return out
}

func sortInt64(values []int64) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j-1] > values[j]; j-- {
			values[j-1], values[j] = values[j], values[j-1]
		}
	}
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
