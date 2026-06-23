//go:build integration

package concurrency_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/bootstrap"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/observability"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/platform/config"
)

const (
	throughputInitialStock  = 1000
	throughputOrderCount    = 1000
	throughputClientWorkers = 128
	benchmarkStock          = 5_000_000
)

type harness struct {
	server        *httptest.Server
	app           *bootstrap.App
	adminToken    string
	customerToken string
	productID     uint64
}

func newHarness(t testing.TB, initialStock int) *harness {
	t.Helper()

	os.Setenv("PAYMENT_FAILURE_RATE", "0")
	os.Setenv("RATE_LIMIT_ENABLED", "false")

	cfg, err := config.Load()
	require.NoError(t, err)
	cfg.Payment.FailureRate = 0
	cfg.RateLimit.Enabled = false

	log := observability.NewLogger("throughput-test", "error", false)
	metrics := observability.NewMetrics("throughput_test")

	app, err := bootstrap.New(cfg, log, metrics)
	require.NoError(t, err)

	server := httptest.NewServer(app.Router)

	h := &harness{server: server, app: app}

	suffix := uuid.NewString()
	h.register(t, "admin-"+suffix+"@example.com", "password123", "Admin", "admin")
	h.adminToken = h.login(t, "admin-"+suffix+"@example.com", "password123")

	h.productID = h.createProduct(t, "SKU-"+suffix, initialStock)

	h.register(t, "customer-"+suffix+"@example.com", "password123", "Customer", "customer")
	h.customerToken = h.login(t, "customer-"+suffix+"@example.com", "password123")

	require.Equal(t, initialStock, h.inventory(t), "seeded inventory not visible before load")

	return h
}

func (h *harness) close() {
	h.server.Close()
	h.app.Stop()
}

func (h *harness) do(t testing.TB, method, path, token string, body any, headers map[string]string) (int, []byte) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, h.server.URL+path, reader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := h.server.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, payload
}

func (h *harness) register(t testing.TB, email, password, fullName, role string) {
	t.Helper()
	status, payload := h.do(t, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"email":     email,
		"password":  password,
		"full_name": fullName,
		"role":      role,
	}, nil)
	require.Equalf(t, http.StatusCreated, status, "register failed: %s", payload)
}

func (h *harness) login(t testing.TB, email, password string) string {
	t.Helper()
	status, payload := h.do(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email":    email,
		"password": password,
	}, nil)
	require.Equalf(t, http.StatusOK, status, "login failed: %s", payload)

	var body struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal(payload, &body))
	require.NotEmpty(t, body.Token)
	return body.Token
}

func (h *harness) createProduct(t testing.TB, sku string, initialStock int) uint64 {
	t.Helper()
	status, payload := h.do(t, http.MethodPost, "/api/v1/products", h.adminToken, map[string]any{
		"sku":           sku,
		"name":          "Throughput Widget",
		"description":   "load test product",
		"price_amount":  1000,
		"currency":      "USD",
		"initial_stock": initialStock,
		"reorder_level": 0,
	}, nil)
	require.Equalf(t, http.StatusCreated, status, "create product failed: %s", payload)

	var body struct {
		ID uint64 `json:"id"`
	}
	require.NoError(t, json.Unmarshal(payload, &body))
	require.NotZero(t, body.ID)
	return body.ID
}

func (h *harness) placeOrder(t testing.TB, idempotencyKey string) (int, string) {
	t.Helper()
	status, payload := h.do(t, http.MethodPost, "/api/v1/orders", h.customerToken, map[string]any{
		"items": []map[string]any{
			{"product_id": h.productID, "quantity": 1},
		},
	}, map[string]string{"Idempotency-Key": idempotencyKey})

	if status != http.StatusCreated {
		return status, ""
	}
	var body struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(payload, &body))
	return status, body.Status
}

func (h *harness) inventory(t testing.TB) int {
	t.Helper()
	path := fmt.Sprintf("/api/v1/products/%d/inventory", h.productID)
	status, payload := h.do(t, http.MethodGet, path, h.customerToken, nil, nil)
	require.Equalf(t, http.StatusOK, status, "get inventory failed: %s", payload)

	var body struct {
		Available int `json:"available"`
	}
	require.NoError(t, json.Unmarshal(payload, &body))
	return body.Available
}

func TestThousandConcurrentOrders(t *testing.T) {
	h := newHarness(t, throughputInitialStock)
	defer h.close()

	var fulfilled int64
	var other int64

	jobs := make(chan int, throughputOrderCount)
	for i := 0; i < throughputOrderCount; i++ {
		jobs <- i
	}
	close(jobs)

	var wg sync.WaitGroup
	wg.Add(throughputClientWorkers)

	start := time.Now()
	for w := 0; w < throughputClientWorkers; w++ {
		go func() {
			defer wg.Done()
			for range jobs {
				status, orderStatus := h.placeOrder(t, uuid.NewString())
				if status == http.StatusCreated && orderStatus == "fulfilled" {
					atomic.AddInt64(&fulfilled, 1)
				} else {
					atomic.AddInt64(&other, 1)
				}
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	fulfilledCount := int(atomic.LoadInt64(&fulfilled))
	otherCount := int(atomic.LoadInt64(&other))

	require.Equal(t, throughputOrderCount, fulfilledCount+otherCount)

	available := h.inventory(t)

	require.GreaterOrEqual(t, available, 0, "oversell detected: available went negative")
	require.Equalf(t, throughputInitialStock-fulfilledCount, available,
		"inventory mismatch: stock=%d fulfilled=%d available=%d", throughputInitialStock, fulfilledCount, available)

	throughput := float64(throughputOrderCount) / elapsed.Seconds()
	t.Logf("placed %d orders in %s => %.0f orders/sec (fulfilled=%d other=%d available=%d)",
		throughputOrderCount, elapsed, throughput, fulfilledCount, otherCount, available)
}

func BenchmarkPlaceOrder(b *testing.B) {
	h := newHarness(b, benchmarkStock)
	defer h.close()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		status, orderStatus := h.placeOrder(b, uuid.NewString())
		if status != http.StatusCreated || orderStatus != "fulfilled" {
			b.Fatalf("order %d not fulfilled: status=%d orderStatus=%q", i, status, orderStatus)
		}
	}
}
