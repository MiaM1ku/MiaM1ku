package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// UserAgent 是上报请求的 User-Agent。
const UserAgent = "miam1ku-status/1.0"

// HTTPTimeout 是单次上报的超时时间。
const HTTPTimeout = 10 * time.Second

// Sender 负责把 Report 发送到 /api/report。
type Sender struct {
	Endpoint string
	Token    string
	Client   *http.Client
}

// NewSender 创建发送器。
func NewSender(cfg *Config) *Sender {
	return &Sender{
		Endpoint: cfg.Endpoint,
		Token:    cfg.Token,
		Client:   &http.Client{Timeout: HTTPTimeout},
	}
}

// Send 发送一次上报，非 2xx 响应视为失败。
func (s *Sender) Send(ctx context.Context, r Report) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.Endpoint, bytes.NewReader(r.JSON()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, bytes.TrimSpace(body))
	}
	return nil
}
