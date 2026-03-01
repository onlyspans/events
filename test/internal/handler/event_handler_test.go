package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/onlyspans/events/internal/dto"
	"github.com/onlyspans/events/internal/testutil"
)

func TestEventHandler_IngestEvent(t *testing.T) {
	testApp := testutil.NewAppBuilder(t).Build()

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "valid event",
			requestBody: testutil.NewEventIngestRequestBuilder().
				WithEntityName("test-project").
				Build(),
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body []byte) {
				var resp dto.SingleIngestResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.ID.String() == "" {
					t.Error("expected non-empty event ID")
				}
			},
		},
		{
			name: "missing required field",
			requestBody: dto.EventIngestRequest{
				UserID:     "test-user",
				EntityName: "test-category",
				// Missing Action
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse:  nil,
		},
		{
			name:           "invalid json",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := testutil.NewRequestBuilder(t, testApp).
				WithMethod(http.MethodPost).
				WithPath("/events/ingest")

			if str, ok := tt.requestBody.(string); ok {
				request.WithRawBody([]byte(str)).WithHeader("Content-Type", "application/json")
			} else {
				request.WithJSON(tt.requestBody)
			}

			resp := request.Do()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp.Body)
			}
		})
	}
}

func TestEventHandler_IngestEventsBatch(t *testing.T) {
	testApp := testutil.NewAppBuilder(t).Build()

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "valid batch",
			requestBody: dto.BatchIngestRequest{
				Events: []dto.EventIngestRequest{
					{
						EntityID:   "00000000-0000-0000-0000-000000000001",
						EntityName: "category1",
						UserID:     "user1",
						Action:     "action1",
					},
					{
						EntityID:   "00000000-0000-0000-0000-000000000002",
						EntityName: "category2",
						UserID:     "user2",
						Action:     "action2",
					},
				},
			},
			expectedStatus: http.StatusMultiStatus,
			checkResponse: func(t *testing.T, body []byte) {
				var resp dto.BatchIngestResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.SuccessCount != 2 {
					t.Errorf("expected 2 successes, got %d", resp.SuccessCount)
				}
				if resp.FailureCount != 0 {
					t.Errorf("expected 0 failures, got %d", resp.FailureCount)
				}
			},
		},
		{
			name: "partial success",
			requestBody: dto.BatchIngestRequest{
				Events: []dto.EventIngestRequest{
					{
						EntityID:   "00000000-0000-0000-0000-000000000001",
						EntityName: "category1",
						UserID:     "user1",
						Action:     "action1",
					},
					{
						EntityID:   "00000000-0000-0000-0000-000000000002",
						EntityName: "category2",
						UserID:     "user2",
						// Missing Action - validation error
					},
				},
			},
			expectedStatus: http.StatusMultiStatus,
			checkResponse: func(t *testing.T, body []byte) {
				var resp dto.BatchIngestResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.SuccessCount != 1 {
					t.Errorf("expected 1 success, got %d", resp.SuccessCount)
				}
				if resp.FailureCount != 1 {
					t.Errorf("expected 1 failure, got %d", resp.FailureCount)
				}
				if len(resp.Errors) != 1 {
					t.Errorf("expected 1 error detail, got %d", len(resp.Errors))
				}
			},
		},
		{
			name: "batch size exceeds limit",
			requestBody: func() dto.BatchIngestRequest {
				events := make([]dto.EventIngestRequest, 101)
				for i := range events {
					events[i] = dto.EventIngestRequest{
						EntityID:   "00000000-0000-0000-0000-000000000001",
						EntityName: "category",
						UserID:     "user",
						Action:     "action",
					}
				}
				return dto.BatchIngestRequest{Events: events}
			}(),
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name: "empty batch",
			requestBody: dto.BatchIngestRequest{
				Events: []dto.EventIngestRequest{},
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name:           "invalid json",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := testutil.NewRequestBuilder(t, testApp).
				WithMethod(http.MethodPost).
				WithPath("/events/ingest/batch")

			if str, ok := tt.requestBody.(string); ok {
				request.WithRawBody([]byte(str)).WithHeader("Content-Type", "application/json")
			} else {
				request.WithJSON(tt.requestBody)
			}

			resp := request.Do()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp.Body)
			}
		})
	}
}
