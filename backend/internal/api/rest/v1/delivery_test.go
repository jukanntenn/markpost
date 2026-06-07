package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"markpost/internal/domain/delivery"
	"markpost/internal/service"
	delivery_svc "markpost/internal/service/delivery"

	"github.com/gin-gonic/gin"
)

func newTestChannel(overrides ...func(*delivery.Channel)) *delivery.Channel {
	ch := &delivery.Channel{
		ID:      1,
		UserID:  1,
		Kind:    delivery.ChannelKindFeishu,
		Name:    "Test Channel",
		Enabled: true,
		Configuration: delivery.ChannelConfiguration{
			"webhook_url":   "https://example.com/webhook",
			"card_link_url": "",
		},
		CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	for _, o := range overrides {
		o(ch)
	}
	return ch
}

type mockDeliveryService struct {
	channels map[int]*delivery.Channel
	nextID   int
	err      error
}

func newMockDeliveryService() *mockDeliveryService {
	return &mockDeliveryService{
		channels: make(map[int]*delivery.Channel),
		nextID:   1,
	}
}

func (m *mockDeliveryService) ListByUserID(_ context.Context, userID int) ([]delivery.Channel, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []delivery.Channel
	for _, ch := range m.channels {
		if ch.UserID == userID {
			result = append(result, *ch)
		}
	}
	return result, nil
}

func (m *mockDeliveryService) Create(_ context.Context, userID int, params delivery_svc.UpdateChannelParams) (*delivery.Channel, error) {
	if m.err != nil {
		return nil, m.err
	}

	var config delivery.ChannelConfiguration
	if len(params.Configuration) > 0 {
		_ = json.Unmarshal(params.Configuration, &config)
	}

	ch := &delivery.Channel{
		ID:            m.nextID,
		UserID:        userID,
		Kind:          delivery.ChannelKind(params.Kind),
		Name:          params.Name,
		Configuration: config,
		Keywords:      params.Keywords,
		Enabled:       true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	m.channels[ch.ID] = ch
	m.nextID++
	return ch, nil
}

func (m *mockDeliveryService) Update(_ context.Context, userID int, id int, params delivery_svc.UpdateChannelParams) (*delivery.Channel, error) {
	if m.err != nil {
		return nil, m.err
	}
	ch, ok := m.channels[id]
	if !ok || ch.UserID != userID {
		return nil, service.NewServiceError(service.ErrNotFound, "channel not found")
	}
	if params.Kind != "" {
		ch.Kind = delivery.ChannelKind(params.Kind)
	}
	if params.Name != "" {
		ch.Name = params.Name
	}
	if len(params.Configuration) > 0 {
		var config delivery.ChannelConfiguration
		_ = json.Unmarshal(params.Configuration, &config)
		ch.Configuration = config
	}
	if params.Keywords != "" {
		ch.Keywords = params.Keywords
	}
	if params.Enabled != nil {
		ch.Enabled = *params.Enabled
	}
	ch.UpdatedAt = time.Now()
	return ch, nil
}

func (m *mockDeliveryService) Delete(_ context.Context, userID int, id int) error {
	if m.err != nil {
		return m.err
	}
	ch, ok := m.channels[id]
	if !ok || ch.UserID != userID {
		return service.NewServiceError(service.ErrNotFound, "channel not found")
	}
	delete(m.channels, id)
	return nil
}

func TestParsePathID(t *testing.T) {
	tests := []struct {
		name      string
		param     string
		wantID    int
		wantOK    bool
		wantError bool
	}{
		{name: "valid positive", param: "42", wantID: 42, wantOK: true},
		{name: "valid zero", param: "0", wantID: 0, wantOK: true},
		{name: "negative number", param: "-1", wantID: -1, wantOK: true},
		{name: "non-numeric", param: "abc", wantOK: false, wantError: true},
		{name: "float-like", param: "3.14", wantOK: false, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newTestEngine()

			var gotID int
			var gotOK bool
			var w *httptest.ResponseRecorder

			router.GET("/test/:id", func(c *gin.Context) {
				gotID, gotOK = parsePathID(c)
			})

			req := httptest.NewRequest(http.MethodGet, "/test/"+tt.param, nil)
			w = httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if gotID != tt.wantID {
				t.Errorf("id = %d, want %d", gotID, tt.wantID)
			}
			if gotOK != tt.wantOK {
				t.Errorf("ok = %v, want %v", gotOK, tt.wantOK)
			}
			if tt.wantError {
				if w.Code != http.StatusBadRequest {
					t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
				}
				var body map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatalf("response body is not valid JSON: %v", err)
				}
				if body["code"] == nil {
					t.Error("expected error response with 'code' field")
				}
			}
		})
	}
}

func TestListDeliveryChannels_Success(t *testing.T) {
	mockSvc := newMockDeliveryService()
	mockSvc.channels[1] = newTestChannel()

	router := newTestEngine()
	router.GET("/channels", withTestUser(1), ListDeliveryChannels(mockSvc))

	req := httptest.NewRequest(http.MethodGet, "/channels", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp ChannelsListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(resp.Channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(resp.Channels))
	}
	if resp.Channels[0].Name != "Test Channel" {
		t.Errorf("expected name 'Test Channel', got %q", resp.Channels[0].Name)
	}
}

func TestListDeliveryChannels_NoUser(t *testing.T) {
	mockSvc := newMockDeliveryService()
	router := newTestEngine()
	router.GET("/channels", ListDeliveryChannels(mockSvc))

	req := httptest.NewRequest(http.MethodGet, "/channels", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestListDeliveryChannels_ServiceError(t *testing.T) {
	mockSvc := newMockDeliveryService()
	mockSvc.err = service.NewServiceError(service.ErrInternal, "db error")

	router := newTestEngine()
	router.GET("/channels", withTestUser(1), ListDeliveryChannels(mockSvc))

	req := httptest.NewRequest(http.MethodGet, "/channels", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestCreateDeliveryChannel_Success(t *testing.T) {
	mockSvc := newMockDeliveryService()
	router := newTestEngine()
	router.POST("/channels", withTestUser(1), CreateDeliveryChannel(mockSvc))

	config := delivery.ChannelConfiguration{
		"webhook_url":   "https://example.com/webhook",
		"card_link_url": "",
	}
	configJSON, _ := json.Marshal(config)
	body := CreateDeliveryChannelRequest{
		Kind:          "feishu",
		Name:          "My Channel",
		Configuration: configJSON,
		Keywords:      "alert,error",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/channels", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var resp SingleChannelResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Channel.Name != "My Channel" {
		t.Errorf("expected name 'My Channel', got %q", resp.Channel.Name)
	}
	if resp.Channel.Kind != delivery.ChannelKindFeishu {
		t.Errorf("expected kind 'feishu', got %q", resp.Channel.Kind)
	}
}

func TestCreateDeliveryChannel_NoUser(t *testing.T) {
	mockSvc := newMockDeliveryService()
	router := newTestEngine()
	router.POST("/channels", CreateDeliveryChannel(mockSvc))

	configJSON, _ := json.Marshal(map[string]string{"webhook_url": "https://example.com"})
	body := CreateDeliveryChannelRequest{
		Kind:          "feishu",
		Name:          "Test",
		Configuration: configJSON,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/channels", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestCreateDeliveryChannel_MissingRequiredFields(t *testing.T) {
	mockSvc := newMockDeliveryService()
	router := newTestEngine()
	router.POST("/channels", withTestUser(1), CreateDeliveryChannel(mockSvc))

	body := map[string]string{"name": "only name"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/channels", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var errResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp["code"] != "validation" {
		t.Errorf("expected error code 'validation', got %v", errResp["code"])
	}
}

func TestCreateDeliveryChannel_ServiceError(t *testing.T) {
	mockSvc := newMockDeliveryService()
	mockSvc.err = service.NewServiceError(service.ErrValidation, "unsupported channel kind")

	router := newTestEngine()
	router.POST("/channels", withTestUser(1), CreateDeliveryChannel(mockSvc))

	configJSON, _ := json.Marshal(map[string]string{"webhook_url": "https://example.com"})
	body := CreateDeliveryChannelRequest{
		Kind:          "feishu",
		Name:          "Test",
		Configuration: configJSON,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/channels", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUpdateDeliveryChannel_Success(t *testing.T) {
	mockSvc := newMockDeliveryService()
	mockSvc.channels[1] = newTestChannel(func(ch *delivery.Channel) { ch.Name = "Old Name" })

	router := newTestEngine()
	router.PUT("/channels/:id", withTestUser(1), UpdateDeliveryChannel(mockSvc))

	newName := "New Name"
	body := UpdateDeliveryChannelRequest{Name: &newName}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/channels/1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp SingleChannelResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Channel.Name != "New Name" {
		t.Errorf("expected name 'New Name', got %q", resp.Channel.Name)
	}
}

func TestUpdateDeliveryChannel_InvalidID(t *testing.T) {
	mockSvc := newMockDeliveryService()
	router := newTestEngine()
	router.PUT("/channels/:id", withTestUser(1), UpdateDeliveryChannel(mockSvc))

	body := UpdateDeliveryChannelRequest{}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/channels/abc", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUpdateDeliveryChannel_NotFound(t *testing.T) {
	mockSvc := newMockDeliveryService()

	router := newTestEngine()
	router.PUT("/channels/:id", withTestUser(1), UpdateDeliveryChannel(mockSvc))

	newName := "New Name"
	body := UpdateDeliveryChannelRequest{Name: &newName}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/channels/999", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestUpdateDeliveryChannel_NoUser(t *testing.T) {
	mockSvc := newMockDeliveryService()
	router := newTestEngine()
	router.PUT("/channels/:id", UpdateDeliveryChannel(mockSvc))

	body := UpdateDeliveryChannelRequest{}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/channels/1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestDeleteDeliveryChannel_Success(t *testing.T) {
	mockSvc := newMockDeliveryService()
	mockSvc.channels[1] = newTestChannel(func(ch *delivery.Channel) { ch.Name = "Channel" })

	router := newTestEngine()
	router.DELETE("/channels/:id", withTestUser(1), DeleteDeliveryChannel(mockSvc))

	req := httptest.NewRequest(http.MethodDelete, "/channels/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp MessageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Message == "" {
		t.Error("expected non-empty message in response")
	}
}

func TestDeleteDeliveryChannel_InvalidID(t *testing.T) {
	mockSvc := newMockDeliveryService()
	router := newTestEngine()
	router.DELETE("/channels/:id", withTestUser(1), DeleteDeliveryChannel(mockSvc))

	req := httptest.NewRequest(http.MethodDelete, "/channels/abc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestDeleteDeliveryChannel_NotFound(t *testing.T) {
	mockSvc := newMockDeliveryService()
	router := newTestEngine()
	router.DELETE("/channels/:id", withTestUser(1), DeleteDeliveryChannel(mockSvc))

	req := httptest.NewRequest(http.MethodDelete, "/channels/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDeleteDeliveryChannel_NoUser(t *testing.T) {
	mockSvc := newMockDeliveryService()
	router := newTestEngine()
	router.DELETE("/channels/:id", DeleteDeliveryChannel(mockSvc))

	req := httptest.NewRequest(http.MethodDelete, "/channels/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}
