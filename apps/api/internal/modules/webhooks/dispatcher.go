package webhooks

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Dispatcher interface {
	Dispatch(ctx context.Context, hook *Webhook, event string, action string, payload any) (*WebhookDelivery, error)
	DispatchAsync(hook *Webhook, event string, action string, payload any)
}

type httpDispatcher struct {
	store RepositoryStore
}

func NewDispatcher(store RepositoryStore) Dispatcher {
	return &httpDispatcher{store: store}
}

func (d *httpDispatcher) DispatchAsync(hook *Webhook, event string, action string, payload any) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, _ = d.Dispatch(ctx, hook, event, action, payload)
	}()
}

func (d *httpDispatcher) Dispatch(ctx context.Context, hook *Webhook, event string, action string, payload any) (*WebhookDelivery, error) {
	deliveryID := uuid.NewString()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal webhook payload: %w", err)
	}

	reqHeaders := map[string]string{
		"Content-Type":     hook.ContentType,
		"User-Agent":       "ForgeHub-Hookshot",
		"X-Forge-Delivery": deliveryID,
		"X-Forge-Event":    event,
	}
	if hook.Secret != "" {
		reqHeaders["X-Forge-Signature-256"] = SignPayload(hook.Secret, payloadBytes)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", hook.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		errMsg := err.Error()
		del := &WebhookDelivery{
			ID:             deliveryID,
			WebhookID:      hook.ID,
			Event:          event,
			Action:         action,
			RequestHeaders: reqHeaders,
			RequestPayload: payload,
			IsSuccess:      false,
			ErrorMessage:   &errMsg,
			DeliveredAt:    time.Now().UTC(),
		}
		_ = d.store.CreateDelivery(ctx, del)
		return del, err
	}

	for k, v := range reqHeaders {
		req.Header.Set(k, v)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	if !hook.SSLVerification {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	start := time.Now()
	res, err := client.Do(req)
	duration := time.Since(start).Milliseconds()

	delivery := &WebhookDelivery{
		ID:             deliveryID,
		WebhookID:      hook.ID,
		Event:          event,
		Action:         action,
		RequestHeaders: reqHeaders,
		RequestPayload: payload,
		DurationMs:     int(duration),
		DeliveredAt:    time.Now().UTC(),
	}

	if err != nil {
		errMsg := err.Error()
		delivery.IsSuccess = false
		delivery.ErrorMessage = &errMsg
		_ = d.store.CreateDelivery(ctx, delivery)
		return delivery, nil
	}
	defer res.Body.Close()

	delivery.ResponseStatusCode = &res.StatusCode
	delivery.IsSuccess = res.StatusCode >= 200 && res.StatusCode < 300

	resHeaders := make(map[string]string)
	for k := range res.Header {
		resHeaders[k] = res.Header.Get(k)
	}
	delivery.ResponseHeaders = resHeaders

	// Limit response body read to 4KB
	bodyBytes, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	bodyStr := string(bodyBytes)
	delivery.ResponseBody = &bodyStr

	_ = d.store.CreateDelivery(ctx, delivery)
	return delivery, nil
}
