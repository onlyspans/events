package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/onlyspans/events/internal/app"
	"github.com/onlyspans/events/internal/config"
)

// TestApp provides an HTTP test server around the real application.
type TestApp struct {
	App     *app.Application
	Server  *httptest.Server
	Client  *http.Client
	BaseURL string
}

// AppBuilder builds a TestApp using the real application wiring.
type AppBuilder struct {
	t      *testing.T
	cfg    *config.Config
	logger *slog.Logger
}

// NewAppBuilder returns a builder with sensible defaults.
func NewAppBuilder(t *testing.T) *AppBuilder {
	t.Helper()

	pg := SetupPostgres(t)
	cfg := &config.Config{
		Features: config.FeatureFlags{
			AutoMigrate: true,
		},
		Database: config.DatabaseConfig{
			DSN:               pg.DSN,
			MaxConns:          10,
			MinConns:          2,
			MaxConnLifetime:   5 * time.Minute,
			MaxConnIdleTime:   30 * time.Minute,
			HealthCheckPeriod: 60 * time.Second,
		},
		EventLog: config.EventLogConfig{
			RetentionPeriodDays: 90,
			MaxExportSize:       10000,
			RetentionCron:       "0 2 * * *",
		},
	}

	return &AppBuilder{
		t:      t,
		cfg:    cfg,
		logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}
}

func (b *AppBuilder) WithConfig(cfg *config.Config) *AppBuilder {
	b.cfg = cfg
	return b
}

func (b *AppBuilder) WithLogger(logger *slog.Logger) *AppBuilder {
	b.logger = logger
	return b
}

func (b *AppBuilder) Build() *TestApp {
	b.t.Helper()

	appInstance, err := app.New(b.cfg, b.logger)
	if err != nil {
		b.t.Fatalf("failed to build app: %v", err)
	}

	server := httptest.NewServer(appInstance.Handler())
	testApp := &TestApp{
		App:     appInstance,
		Server:  server,
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	b.t.Cleanup(func() {
		server.Close()
		_ = appInstance.Shutdown(context.Background())
	})

	return testApp
}

type RequestBuilder struct {
	t       *testing.T
	app     *TestApp
	method  string
	path    string
	body    []byte
	headers map[string]string
}

func NewRequestBuilder(t *testing.T, app *TestApp) *RequestBuilder {
	t.Helper()
	return &RequestBuilder{
		t:       t,
		app:     app,
		method:  http.MethodGet,
		headers: make(map[string]string),
	}
}

func (b *RequestBuilder) WithMethod(method string) *RequestBuilder {
	b.method = method
	return b
}

func (b *RequestBuilder) WithPath(path string) *RequestBuilder {
	b.path = path
	return b
}

func (b *RequestBuilder) WithJSON(payload interface{}) *RequestBuilder {
	b.t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		b.t.Fatalf("failed to marshal request body: %v", err)
	}
	b.body = data
	b.headers["Content-Type"] = "application/json"
	return b
}

func (b *RequestBuilder) WithRawBody(body []byte) *RequestBuilder {
	b.body = body
	return b
}

func (b *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	b.headers[key] = value
	return b
}

func (b *RequestBuilder) Do() *HTTPResponse {
	b.t.Helper()

	req, err := http.NewRequest(b.method, b.app.BaseURL+b.path, bytes.NewReader(b.body))
	if err != nil {
		b.t.Fatalf("failed to create request: %v", err)
	}
	for k, v := range b.headers {
		req.Header.Set(k, v)
	}

	resp, err := b.app.Client.Do(req)
	if err != nil {
		b.t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		b.t.Fatalf("failed to read response body: %v", err)
	}

	return &HTTPResponse{
		StatusCode: resp.StatusCode,
		Body:       body,
		Header:     resp.Header,
	}
}

type HTTPResponse struct {
	StatusCode int
	Body       []byte
	Header     http.Header
}
