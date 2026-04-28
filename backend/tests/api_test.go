//go:build integration

package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/client"
	"github.com/xavierli/network-ticket/internal/config"
	"github.com/xavierli/network-ticket/internal/handler"
	"github.com/xavierli/network-ticket/internal/middleware"
	"github.com/xavierli/network-ticket/internal/nonce"
	"github.com/xavierli/network-ticket/internal/pkg"
	"github.com/xavierli/network-ticket/internal/repository"
	"github.com/xavierli/network-ticket/internal/service"
)

// ---------------------------------------------------------------------------
// Test infrastructure
// ---------------------------------------------------------------------------

// setupRouter builds a fully-wired Gin router matching the production route
// layout in cmd/server/main.go. Each test calls this to get a fresh router.
func setupRouter(db *sqlx.DB, logger *zap.Logger) *gin.Engine {
	gin.SetMode(gin.TestMode)

	// Repositories.
	ticketRepo := repository.NewTicketRepo(db)
	workflowStateRepo := repository.NewWorkflowStateRepo(db)
	alertSourceRepo := repository.NewAlertSourceRepo(db)
	alertRecordRepo := repository.NewAlertRecordRepo(db)
	clientRepo := repository.NewClientRepo(db)
	ticketLogRepo := repository.NewTicketLogRepo(db)
	auditLogRepo := repository.NewAuditLogRepo(db)
	userRepo := repository.NewUserRepo(db)

	// Nonce store (use DB-backed in integration tests).
	nonceStore := nonce.NewDBStore(db)

	// Services.
	ticketService := service.NewTicketService(
		ticketRepo, workflowStateRepo, ticketLogRepo, auditLogRepo, alertRecordRepo, logger,
	)
	alertService := service.NewAlertService(alertSourceRepo, ticketService, logger)
	authService := service.NewAuthService(userRepo, &config.JWTConfig{
		Secret:      "test-jwt-secret-integration",
		ExpireHours: 24,
	})
	clientService := service.NewClientService(clientRepo, logger)

	// Worker pool (start but tests don't depend on push delivery).
	retryCfg := client.RetryConfig{MaxAttempts: 3, BaseInterval: time.Second, MaxInterval: 5 * time.Second}
	workerPool := client.NewWorkerPool(2, retryCfg, ticketRepo, clientRepo, workflowStateRepo, logger)
	workerPool.Start()
	// NOTE: In a real TestMain you would call workerPool.Stop() in cleanup.

	// Handlers.
	authHandler := handler.NewAuthHandler(authService, logger)
	alertHandler := handler.NewAlertHandler(alertService, alertSourceRepo, logger)
	ticketHandler := handler.NewTicketHandler(ticketService, logger)
	clientHandler := handler.NewClientHandler(clientService, logger)
	callbackHandler := handler.NewCallbackHandler(ticketService, logger)
	adminHandler := handler.NewAdminHandler(auditLogRepo, logger)

	// Router mirrors cmd/server/main.go layout.
	r := gin.New()
	r.Use(gin.Recovery())

	api := r.Group("/api/v1")
	{
		api.POST("/alerts/webhook/:source_id", alertHandler.Webhook)

		alertsAuth := api.Group("")
		alertsAuth.Use(middleware.JWTAuth(authService))
		{
			alertsAuth.POST("/alerts", alertHandler.Create)
			alertsAuth.GET("/alert-sources", alertHandler.ListSources)
			alertsAuth.POST("/alert-sources", alertHandler.CreateSource)
			alertsAuth.PUT("/alert-sources/:id", alertHandler.UpdateSource)
			alertsAuth.DELETE("/alert-sources/:id", alertHandler.DeleteSource)
		}

		api.POST("/callback/authorization",
			middleware.HMACSignature(clientRepo),
			middleware.NonceCheck(nonceStore),
			callbackHandler.Handle,
		)

		tickets := api.Group("")
		tickets.Use(middleware.JWTAuth(authService))
		{
			tickets.GET("/tickets", ticketHandler.List)
			tickets.GET("/tickets/:id", ticketHandler.Get)
			tickets.PUT("/tickets/:id", ticketHandler.Update)
			tickets.POST("/tickets/:id/retry", ticketHandler.Retry)
			tickets.POST("/tickets/:id/cancel", ticketHandler.Cancel)
		}

		clients := api.Group("")
		clients.Use(middleware.JWTAuth(authService))
		{
			clients.GET("/clients", clientHandler.List)
		}
		clientsAdmin := api.Group("")
		clientsAdmin.Use(middleware.JWTAuth(authService), middleware.RequireAdmin())
		{
			clientsAdmin.POST("/clients", clientHandler.Create)
			clientsAdmin.PUT("/clients/:id", clientHandler.Update)
			clientsAdmin.DELETE("/clients/:id", clientHandler.Delete)
		}

		auditLogs := api.Group("")
		auditLogs.Use(middleware.JWTAuth(authService))
		{
			auditLogs.GET("/audit-logs", adminHandler.ListAuditLogs)
		}

		api.POST("/auth/login", authHandler.Login)
	}

	return r
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// connectDB opens a database connection using the NT_TEST_DSN environment
// variable. If it is not set the test is skipped with a clear message.
//
// Example:
//
//	NT_TEST_DSN="root:password@tcp(127.0.0.1:3306)/network_ticket_test?charset=utf8mb4&parseTime=true&loc=Local"
func connectDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("NT_TEST_DSN")
	if dsn == "" {
		t.Skip("NT_TEST_DSN not set, skipping integration test")
	}
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	if err := db.Ping(); err != nil {
		t.Fatalf("ping db: %v", err)
	}
	return db
}

// loginHelper performs a login request and returns the JWT token.
func loginHelper(t *testing.T, router *gin.Engine, username, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("login failed: status=%d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal login response: %v", err)
	}
	token, ok := resp["token"].(string)
	if !ok || token == "" {
		t.Fatal("login response missing token")
	}
	return token
}

// signRequest computes HMAC signature headers for callback endpoints.
// It mirrors what an external client would do before calling the callback API.
func signRequest(secret string, apiKey string, body []byte) http.Header {
	timestamp := time.Now().Unix()
	signature := pkg.SignHMAC(secret, timestamp, body)
	nonceVal := uuid.New().String()

	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("X-Api-Key", apiKey)
	h.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	h.Set("X-Signature", signature)
	h.Set("X-Nonce", nonceVal)
	return h
}

// authHeader returns HTTP headers including an Authorization Bearer token.
func authHeader(token string) http.Header {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	return h
}

// ---------------------------------------------------------------------------
// Integration Tests
// ---------------------------------------------------------------------------

// TestLogin tests POST /api/v1/auth/login
//
// Validates:
//   - A valid admin user can authenticate and receive a JWT token.
//   - The response body contains "token" and "user" fields.
//   - An incorrect password returns 401.
//   - A missing username returns 400.
//
// Prerequisites:
//   - MySQL running with migrations applied.
//   - A user row: username="admin", password=bcrypt("admin123"), role="admin".
func TestLogin(t *testing.T) {
	db := connectDB(t)
	defer db.Close()

	logger := zap.NewNop()
	router := setupRouter(db, logger)

	t.Run("success", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"username": "admin",
			"password": "admin123",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if _, ok := resp["token"]; !ok {
			t.Fatal("response missing 'token' field")
		}
		if user, ok := resp["user"].(map[string]interface{}); !ok {
			t.Fatal("response missing 'user' field")
		} else {
			if user["role"] != "admin" {
				t.Errorf("expected role=admin, got %v", user["role"])
			}
		}
	})

	t.Run("wrong_password", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"username": "admin",
			"password": "wrong-password",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("missing_fields", func(t *testing.T) {
		body := []byte(`{}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})
}

// TestAlertWebhook tests POST /api/v1/alerts/webhook/:source_id
//
// Validates the full alert-to-ticket pipeline:
//  1. Login to obtain JWT.
//  2. Create an alert source via POST /api/v1/alert-sources.
//  3. POST alert JSON to the webhook endpoint.
//  4. Assert response contains ticket_id, ticket_no, status="pending", is_new=true.
//  5. POST the same alert again (dedup) and verify is_new=false.
//
// Prerequisites:
//   - MySQL running with migrations applied.
//   - Admin user seeded.
func TestAlertWebhook(t *testing.T) {
	db := connectDB(t)
	defer db.Close()

	logger := zap.NewNop()
	router := setupRouter(db, logger)

	// Step 1: Login.
	token := loginHelper(t, router, "admin", "admin123")

	// Step 2: Create a "generic" alert source.
	sourceBody, _ := json.Marshal(map[string]interface{}{
		"name":             "integration-test-source",
		"type":             "generic",
		"status":           "active",
		"dedup_fields":     []string{"alertname", "instance"},
		"dedup_window_sec": 600,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-sources", bytes.NewReader(sourceBody))
	for k, v := range authHeader(token) {
		req.Header[k] = v
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create alert source: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var sourceResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &sourceResp)
	sourceID := int64(sourceResp["id"].(float64))

	// Step 3: POST alert to webhook.
	alertBody, _ := json.Marshal(map[string]interface{}{
		"alertname": "HighCPU",
		"instance":  "10.0.0.1:9090",
		"severity":  "critical",
		"message":   "CPU usage above 90%",
	})
	req = httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/v1/alerts/webhook/%d", sourceID),
		bytes.NewReader(alertBody),
	)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("webhook ingest: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var ingestResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &ingestResp)

	if ingestResp["status"] != "pending" {
		t.Errorf("expected status=pending, got %v", ingestResp["status"])
	}
	if ingestResp["is_new"] != true {
		t.Errorf("expected is_new=true, got %v", ingestResp["is_new"])
	}
	if _, ok := ingestResp["ticket_id"]; !ok {
		t.Error("response missing ticket_id")
	}
	if _, ok := ingestResp["ticket_no"]; !ok {
		t.Error("response missing ticket_no")
	}

	t.Logf("Created ticket: id=%v ticket_no=%v", ingestResp["ticket_id"], ingestResp["ticket_no"])

	// Step 5: Post same alert again — should dedup.
	req = httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/v1/alerts/webhook/%d", sourceID),
		bytes.NewReader(alertBody),
	)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("webhook dedup ingest: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &ingestResp)
	if ingestResp["is_new"] != false {
		t.Errorf("expected is_new=false on dedup, got %v", ingestResp["is_new"])
	}
}

// TestTicketList tests GET /api/v1/tickets
//
// Validates:
//   - Paginated ticket listing returns 200 with "items" array and "total" count.
//   - Query parameters (page, page_size, status) are respected.
//
// Prerequisites:
//   - At least one ticket exists (run TestAlertWebhook first, or seed data manually).
func TestTicketList(t *testing.T) {
	db := connectDB(t)
	defer db.Close()

	logger := zap.NewNop()
	router := setupRouter(db, logger)

	token := loginHelper(t, router, "admin", "admin123")

	t.Run("default_pagination", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets?page=1&page_size=10", nil)
		for k, v := range authHeader(token) {
			req.Header[k] = v
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if _, ok := resp["items"]; !ok {
			t.Error("response missing 'items' field")
		}
		if _, ok := resp["total"]; !ok {
			t.Error("response missing 'total' field")
		}
	})

	t.Run("filter_by_status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets?status=pending&page=1&page_size=10", nil)
		for k, v := range authHeader(token) {
			req.Header[k] = v
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if _, ok := resp["items"]; !ok {
			t.Error("response missing 'items' field")
		}
	})
}

// TestCallbackAuthorization tests POST /api/v1/callback/authorization
//
// Validates the end-to-end HMAC-signed callback flow:
//  1. Create a client with known API key and HMAC secret.
//  2. Create a ticket in "pending" state assigned to that client.
//  3. Build a callback body with action="authorize".
//  4. Sign the request with HMAC and send to the callback endpoint.
//  5. Assert the ticket transitions to "in_progress".
//
// Prerequisites:
//   - MySQL running with migrations applied.
//   - Admin user seeded.
func TestCallbackAuthorization(t *testing.T) {
	db := connectDB(t)
	defer db.Close()

	logger := zap.NewNop()
	router := setupRouter(db, logger)

	token := loginHelper(t, router, "admin", "admin123")

	// Step 1: Create a client via admin API.
	apiKey := "test-api-key-" + uuid.New().String()
	hmacSecret := "test-hmac-secret-" + uuid.New().String()
	callbackURL := "http://localhost:9999/callback"

	clientBody, _ := json.Marshal(map[string]interface{}{
		"name":         "test-client-auth",
		"api_endpoint": "http://localhost:9999/push",
		"api_key":      apiKey,
		"hmac_secret":  hmacSecret,
		"callback_url": callbackURL,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clients", bytes.NewReader(clientBody))
	for k, v := range authHeader(token) {
		req.Header[k] = v
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create client: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var clientResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &clientResp)
	clientID := int64(clientResp["id"].(float64))

	t.Logf("Created client: id=%d api_key=%s", clientID, apiKey)

	// Step 2: Create a ticket via the alert webhook and assign to client.
	sourceBody, _ := json.Marshal(map[string]interface{}{
		"name":   "callback-test-source",
		"type":   "generic",
		"status": "active",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/alert-sources", bytes.NewReader(sourceBody))
	for k, v := range authHeader(token) {
		req.Header[k] = v
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var sourceResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &sourceResp)
	sourceID := int64(sourceResp["id"].(float64))

	alertBody, _ := json.Marshal(map[string]interface{}{
		"alertname": "CallbackTest",
		"instance":  "10.0.0.99:9090",
		"severity":  "high",
		"message":   "Test alert for callback",
	})
	req = httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/v1/alerts/webhook/%d", sourceID),
		bytes.NewReader(alertBody),
	)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var ingestResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &ingestResp)
	ticketNo := ingestResp["ticket_no"].(string)
	ticketID := int64(ingestResp["ticket_id"].(float64))

	// Assign the ticket to the client we created (no dedicated API for assignment).
	_, err := db.Exec("UPDATE tickets SET client_id = ? WHERE id = ?", clientID, ticketID)
	if err != nil {
		t.Fatalf("assign client to ticket: %v", err)
	}

	// Step 3-4: Build and sign the callback request.
	cbBody, _ := json.Marshal(map[string]interface{}{
		"ticket_no":     ticketNo,
		"action":        "authorize",
		"operator":      "test-operator",
		"comment":       "Authorized via integration test",
		"authorized_at": time.Now().Format(time.RFC3339),
	})
	headers := signRequest(hmacSecret, apiKey, cbBody)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/callback/authorization", bytes.NewReader(cbBody))
	for k, vals := range headers {
		req.Header[k] = vals
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("callback authorize: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var cbResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &cbResp)
	if cbResp["status"] != "authorized" {
		t.Errorf("expected status=authorized, got %v", cbResp["status"])
	}
	if cbResp["ticket_no"] != ticketNo {
		t.Errorf("expected ticket_no=%s, got %v", ticketNo, cbResp["ticket_no"])
	}

	t.Logf("Ticket %s authorized successfully", ticketNo)

	// Verify ticket status in DB is now "in_progress".
	var status string
	err = db.QueryRow("SELECT status FROM tickets WHERE id = ?", ticketID).Scan(&status)
	if err != nil {
		t.Fatalf("query ticket status: %v", err)
	}
	if status != "in_progress" {
		t.Errorf("expected ticket status=in_progress, got %s", status)
	}
}

// TestCallbackReject tests the rejection flow via POST /api/v1/callback/authorization
//
// Validates:
//   - A pending ticket can be rejected via the callback endpoint.
//   - The ticket status transitions to "rejected".
//
// Prerequisites: same as TestCallbackAuthorization.
func TestCallbackReject(t *testing.T) {
	db := connectDB(t)
	defer db.Close()

	logger := zap.NewNop()
	router := setupRouter(db, logger)

	token := loginHelper(t, router, "admin", "admin123")

	// Create client.
	apiKey := "test-api-key-reject-" + uuid.New().String()
	hmacSecret := "test-hmac-secret-reject-" + uuid.New().String()

	clientBody, _ := json.Marshal(map[string]interface{}{
		"name":         "test-client-reject",
		"api_endpoint": "http://localhost:9999/push",
		"api_key":      apiKey,
		"hmac_secret":  hmacSecret,
		"callback_url": "http://localhost:9999/callback",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clients", bytes.NewReader(clientBody))
	for k, v := range authHeader(token) {
		req.Header[k] = v
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var clientResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &clientResp)
	clientID := int64(clientResp["id"].(float64))

	// Create alert source + ticket.
	sourceBody, _ := json.Marshal(map[string]interface{}{
		"name": "reject-test-source", "type": "generic", "status": "active",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/alert-sources", bytes.NewReader(sourceBody))
	for k, v := range authHeader(token) {
		req.Header[k] = v
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var sourceResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &sourceResp)
	sourceID := int64(sourceResp["id"].(float64))

	alertBody, _ := json.Marshal(map[string]interface{}{
		"alertname": "RejectTest",
		"instance":  "10.0.0.100:9090",
		"severity":  "warning",
		"message":   "Test alert for reject",
	})
	req = httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/v1/alerts/webhook/%d", sourceID),
		bytes.NewReader(alertBody),
	)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var ingestResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &ingestResp)
	ticketNo := ingestResp["ticket_no"].(string)
	ticketID := int64(ingestResp["ticket_id"].(float64))

	// Assign ticket to client.
	db.Exec("UPDATE tickets SET client_id = ? WHERE id = ?", clientID, ticketID)

	// Send reject callback.
	cbBody, _ := json.Marshal(map[string]interface{}{
		"ticket_no":     ticketNo,
		"action":        "reject",
		"operator":      "test-operator",
		"comment":       "Rejected via integration test",
		"authorized_at": time.Now().Format(time.RFC3339),
	})
	headers := signRequest(hmacSecret, apiKey, cbBody)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/callback/authorization", bytes.NewReader(cbBody))
	for k, vals := range headers {
		req.Header[k] = vals
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("callback reject: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var cbResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &cbResp)
	if cbResp["status"] != "rejected" {
		t.Errorf("expected status=rejected, got %v", cbResp["status"])
	}

	// Verify in DB.
	var status string
	err := db.QueryRow("SELECT status FROM tickets WHERE id = ?", ticketID).Scan(&status)
	if err != nil {
		t.Fatalf("query ticket status: %v", err)
	}
	if status != "rejected" {
		t.Errorf("expected ticket status=rejected, got %s", status)
	}
}

// TestStateTransitionValidation tests that invalid state transitions are rejected.
//
// Validates:
//   - Attempting to transition a "completed" ticket back to "pending" fails.
//   - The API returns an appropriate error response.
//
// Prerequisites:
//   - MySQL running with migrations applied.
//   - A ticket that has been completed (manually transitioned through the flow).
func TestStateTransitionValidation(t *testing.T) {
	db := connectDB(t)
	defer db.Close()

	logger := zap.NewNop()
	router := setupRouter(db, logger)

	token := loginHelper(t, router, "admin", "admin123")

	// Create a ticket via alert ingestion.
	sourceBody, _ := json.Marshal(map[string]interface{}{
		"name": "transition-test-source", "type": "generic", "status": "active",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-sources", bytes.NewReader(sourceBody))
	for k, v := range authHeader(token) {
		req.Header[k] = v
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var sourceResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &sourceResp)
	sourceID := int64(sourceResp["id"].(float64))

	alertBody, _ := json.Marshal(map[string]interface{}{
		"alertname": "TransitionTest",
		"instance":  "10.0.0.200:9090",
		"severity":  "info",
		"message":   "Test for state transition validation",
	})
	req = httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/v1/alerts/webhook/%d", sourceID),
		bytes.NewReader(alertBody),
	)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var ingestResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &ingestResp)
	ticketID := int64(ingestResp["ticket_id"].(float64))

	// Transition: pending -> in_progress (valid).
	updateBody, _ := json.Marshal(map[string]string{
		"status":   "in_progress",
		"operator": "test",
	})
	req = httptest.NewRequest(http.MethodPut,
		fmt.Sprintf("/api/v1/tickets/%d", ticketID),
		bytes.NewReader(updateBody),
	)
	for k, v := range authHeader(token) {
		req.Header[k] = v
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("pending->in_progress: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Transition: in_progress -> completed (valid).
	updateBody, _ = json.Marshal(map[string]string{
		"status":   "completed",
		"operator": "test",
	})
	req = httptest.NewRequest(http.MethodPut,
		fmt.Sprintf("/api/v1/tickets/%d", ticketID),
		bytes.NewReader(updateBody),
	)
	for k, v := range authHeader(token) {
		req.Header[k] = v
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("in_progress->completed: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Transition: completed -> pending (INVALID — should be rejected).
	updateBody, _ = json.Marshal(map[string]string{
		"status":   "pending",
		"operator": "test",
	})
	req = httptest.NewRequest(http.MethodPut,
		fmt.Sprintf("/api/v1/tickets/%d", ticketID),
		bytes.NewReader(updateBody),
	)
	for k, v := range authHeader(token) {
		req.Header[k] = v
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("completed->pending: expected 400, got %d (should reject invalid transition)", w.Code)
	}

	t.Logf("Invalid transition correctly rejected: status=%d body=%s", w.Code, w.Body.String())
}

// TestCallbackInvalidSignature tests that a callback with a wrong HMAC signature
// is rejected by the middleware.
func TestCallbackInvalidSignature(t *testing.T) {
	db := connectDB(t)
	defer db.Close()

	logger := zap.NewNop()
	router := setupRouter(db, logger)

	cbBody, _ := json.Marshal(map[string]interface{}{
		"ticket_no": "NT-000001",
		"action":    "authorize",
		"operator":  "attacker",
	})

	// Use completely wrong HMAC credentials.
	headers := signRequest("wrong-secret", "wrong-key", cbBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/callback/authorization", bytes.NewReader(cbBody))
	for k, vals := range headers {
		req.Header[k] = vals
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid signature, got %d", w.Code)
	}
}

// TestTicketGetSingle tests GET /api/v1/tickets/:id
//
// Validates:
//   - Fetching a single ticket by ID returns the ticket with workflow_states.
//   - A non-existent ID returns 404.
func TestTicketGetSingle(t *testing.T) {
	db := connectDB(t)
	defer db.Close()

	logger := zap.NewNop()
	router := setupRouter(db, logger)

	token := loginHelper(t, router, "admin", "admin123")

	// Create a ticket to fetch.
	sourceBody, _ := json.Marshal(map[string]interface{}{
		"name": "get-test-source", "type": "generic", "status": "active",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-sources", bytes.NewReader(sourceBody))
	for k, v := range authHeader(token) {
		req.Header[k] = v
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var sourceResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &sourceResp)
	sourceID := int64(sourceResp["id"].(float64))

	alertBody, _ := json.Marshal(map[string]interface{}{
		"alertname": "GetTest",
		"instance":  "10.0.0.50:9090",
		"severity":  "low",
		"message":   "Test alert for GET single",
	})
	req = httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/v1/alerts/webhook/%d", sourceID),
		bytes.NewReader(alertBody),
	)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var ingestResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &ingestResp)
	ticketID := int64(ingestResp["ticket_id"].(float64))

	// Fetch the ticket.
	req = httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/v1/tickets/%d", ticketID),
		nil,
	)
	for k, v := range authHeader(token) {
		req.Header[k] = v
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get ticket: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var getResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &getResp)
	if _, ok := getResp["ticket"]; !ok {
		t.Error("response missing 'ticket' field")
	}
	if _, ok := getResp["workflow_states"]; !ok {
		t.Error("response missing 'workflow_states' field")
	}

	// Non-existent ticket.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/tickets/99999999", nil)
	for k, v := range authHeader(token) {
		req.Header[k] = v
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent ticket, got %d", w.Code)
	}
}

// TestTicketCancel tests POST /api/v1/tickets/:id/cancel
//
// Validates:
//   - A pending ticket can be cancelled.
//   - The ticket status transitions to "cancelled".
func TestTicketCancel(t *testing.T) {
	db := connectDB(t)
	defer db.Close()

	logger := zap.NewNop()
	router := setupRouter(db, logger)

	token := loginHelper(t, router, "admin", "admin123")

	// Create a ticket.
	sourceBody, _ := json.Marshal(map[string]interface{}{
		"name": "cancel-test-source", "type": "generic", "status": "active",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-sources", bytes.NewReader(sourceBody))
	for k, v := range authHeader(token) {
		req.Header[k] = v
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var sourceResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &sourceResp)
	sourceID := int64(sourceResp["id"].(float64))

	alertBody, _ := json.Marshal(map[string]interface{}{
		"alertname": "CancelTest",
		"instance":  "10.0.0.55:9090",
		"severity":  "medium",
		"message":   "Test for cancel",
	})
	req = httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/v1/alerts/webhook/%d", sourceID),
		bytes.NewReader(alertBody),
	)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var ingestResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &ingestResp)
	ticketID := int64(ingestResp["ticket_id"].(float64))

	// Cancel the ticket.
	req = httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/v1/tickets/%d/cancel", ticketID),
		nil,
	)
	for k, v := range authHeader(token) {
		req.Header[k] = v
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("cancel ticket: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify in DB.
	var status string
	err := db.QueryRow("SELECT status FROM tickets WHERE id = ?", ticketID).Scan(&status)
	if err != nil {
		t.Fatalf("query ticket status: %v", err)
	}
	if status != "cancelled" {
		t.Errorf("expected status=cancelled, got %s", status)
	}
}

// TestUnauthenticatedAccess tests that endpoints requiring JWT return 401
// when no token is provided.
func TestUnauthenticatedAccess(t *testing.T) {
	db := connectDB(t)
	defer db.Close()

	logger := zap.NewNop()
	router := setupRouter(db, logger)

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/tickets"},
		{"GET", "/api/v1/tickets/1"},
		{"PUT", "/api/v1/tickets/1"},
		{"POST", "/api/v1/tickets/1/retry"},
		{"POST", "/api/v1/tickets/1/cancel"},
		{"GET", "/api/v1/clients"},
		{"POST", "/api/v1/clients"},
		{"GET", "/api/v1/alert-sources"},
		{"POST", "/api/v1/alert-sources"},
		{"GET", "/api/v1/audit-logs"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected 401 for %s %s without token, got %d", ep.method, ep.path, w.Code)
			}
		})
	}
}
