package webhooks

import (
	"errors"
	"time"
)

var (
	ErrWebhookNotFound    = errors.New("webhook not found")
	ErrDeliveryNotFound   = errors.New("webhook delivery not found")
	ErrInvalidWebhookURL  = errors.New("invalid webhook URL: must begin with http:// or https://")
	ErrAccessDenied       = errors.New("access denied: admin permissions required to manage webhooks")
	ErrNoEventsSubscribed = errors.New("webhook must subscribe to at least one event")
)

const (
	EventPush         = "push"
	EventPullRequest  = "pull_request"
	EventIssues       = "issues"
	EventIssueComment = "issue_comment"
	EventPing         = "ping"
	EventAll          = "*"
)

type Webhook struct {
	ID              string    `json:"id"`
	RepositoryID    string    `json:"repository_id"`
	URL             string    `json:"url"`
	ContentType     string    `json:"content_type"`
	Secret          string    `json:"secret,omitempty"` // Omitted or masked in list
	HasSecret       bool      `json:"has_secret"`
	Events          []string  `json:"events"`
	IsActive        bool      `json:"is_active"`
	SSLVerification bool      `json:"ssl_verification"`
	LastStatus      *int      `json:"last_status,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type WebhookDelivery struct {
	ID                 string            `json:"id"`
	WebhookID          string            `json:"webhook_id"`
	Event              string            `json:"event"`
	Action             string            `json:"action,omitempty"`
	RequestHeaders     map[string]string `json:"request_headers"`
	RequestPayload     any               `json:"request_payload"`
	ResponseStatusCode *int              `json:"response_status_code,omitempty"`
	ResponseHeaders    map[string]string `json:"response_headers,omitempty"`
	ResponseBody       *string           `json:"response_body,omitempty"`
	DurationMs         int               `json:"duration_ms"`
	IsSuccess          bool              `json:"is_success"`
	ErrorMessage       *string           `json:"error_message,omitempty"`
	DeliveredAt        time.Time         `json:"delivered_at"`
}

type CreateWebhookRequest struct {
	URL             string   `json:"url"`
	ContentType     string   `json:"content_type"`
	Secret          string   `json:"secret,omitempty"`
	Events          []string `json:"events"`
	IsActive        *bool    `json:"is_active"`
	SSLVerification *bool    `json:"ssl_verification"`
}

type UpdateWebhookRequest struct {
	URL             *string  `json:"url,omitempty"`
	ContentType     *string  `json:"content_type,omitempty"`
	Secret          *string  `json:"secret,omitempty"`
	Events          []string `json:"events,omitempty"`
	IsActive        *bool    `json:"is_active,omitempty"`
	SSLVerification *bool    `json:"ssl_verification,omitempty"`
}

// PingPayload sends standard ForgeHub ping verification data
type PingPayload struct {
	Zen    string `json:"zen"`
	HookID string `json:"hook_id"`
	Hook   struct {
		Type   string   `json:"type"`
		ID     string   `json:"id"`
		Active bool     `json:"active"`
		Events []string `json:"events"`
	} `json:"hook"`
	Repository struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"repository"`
	Sender struct {
		Username string `json:"username"`
	} `json:"sender"`
}
