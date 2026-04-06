package service

import (
	"context"
	"encoding/json"
	"eth-sweeper/model"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type NewsService struct {
	httpClient *http.Client
	mu         sync.RWMutex
	cache      model.NewsResponse
}

func NewNewsService() *NewsService {
	return &NewsService{
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *NewsService) GetETHNews(ctx context.Context) model.NewsResponse {
	s.mu.RLock()
	cached := s.cache
	s.mu.RUnlock()
	if len(cached.Items) > 0 && time.Since(parseTimeOrZero(cached.CachedAt)) < 30*time.Minute {
		return cached
	}

	resp, err := s.fetchGDELT(ctx)
	if err != nil || len(resp.Items) == 0 {
		resp = demoNews()
	}
	s.mu.Lock()
	s.cache = resp
	s.mu.Unlock()
	return resp
}

func (s *NewsService) fetchGDELT(ctx context.Context) (model.NewsResponse, error) {
	q := url.Values{}
	q.Set("query", "ethereum OR ETH")
	q.Set("mode", "artlist")
	q.Set("format", "json")
	q.Set("maxrecords", "10")
	q.Set("sort", "hybridrel")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.gdeltproject.org/api/v2/doc/doc?"+q.Encode(), nil)
	if err != nil {
		return model.NewsResponse{}, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return model.NewsResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return model.NewsResponse{}, fmt.Errorf("gdelt status %d", resp.StatusCode)
	}

	var payload struct {
		Articles []struct {
			URL         string `json:"url"`
			Title       string `json:"title"`
			Seendate    string `json:"seendate"`
			SourceName  string `json:"sourceCountry"`
			Domain      string `json:"domain"`
			SocialImage string `json:"socialimage"`
		} `json:"articles"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return model.NewsResponse{}, err
	}

	items := make([]model.NewsItem, 0, len(payload.Articles))
	for _, article := range payload.Articles {
		if article.URL == "" || article.Title == "" {
			continue
		}
		source := article.Domain
		if source == "" {
			source = "GDELT"
		}
		items = append(items, model.NewsItem{
			ID:          stableID(article.URL),
			Title:       article.Title,
			URL:         article.URL,
			Source:      source,
			PublishedAt: gdeltTime(article.Seendate),
			Snippet:     "Public news link from GDELT. Full article content is not copied.",
		})
	}
	return model.NewsResponse{Items: items, Source: "gdelt_doc_api", CachedAt: nowISO()}, nil
}

func demoNews() model.NewsResponse {
	now := nowISO()
	return model.NewsResponse{
		Source:   "demo_fallback",
		CachedAt: now,
		Items: []model.NewsItem{
			{
				ID:          "demo-eth-market",
				Title:       "ETH market monitor is waiting for live news data",
				URL:         "https://www.gdeltproject.org/",
				Source:      "GDELT",
				PublishedAt: now,
				Snippet:     "GDELT will be used as the compliant free news source when the backend can reach the internet.",
			},
		},
	}
}

func gdeltTime(raw string) string {
	if raw == "" {
		return nowISO()
	}
	for _, layout := range []string{"20060102150405", "20060102T150405Z", time.RFC3339} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
	}
	return raw
}
