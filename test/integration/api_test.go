//go:build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/bootstrap"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/observability"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/platform/config"
)

type loginResponse struct {
	Token string `json:"token"`
	User  struct {
		ID   uint64 `json:"id"`
		Role string `json:"role"`
	} `json:"user"`
}

type productResponse struct {
	ID  uint64 `json:"id"`
	SKU string `json:"sku"`
}

type inventoryResponse struct {
	ProductID uint64 `json:"product_id"`
	Available int    `json:"available"`
	Reserved  int    `json:"reserved"`
}

type orderResponse struct {
	ID     uint64 `json:"id"`
	Status string `json:"status"`
}

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestOrderLifecycleIntegration(t *testing.T) {
	if os.Getenv("JWT_SECRET") == "" {
		os.Setenv("JWT_SECRET", "integration-test-secret-integration-test-secret")
	}
	os.Setenv("PAYMENT_FAILURE_RATE", "0")

	cfg, err := config.Load()
	require.NoError(t, err)

	log := zerolog.Nop()
	metrics := observability.NewMetrics("orderstest")

	app, err := bootstrap.New(cfg, log, metrics)
	require.NoError(t, err)
	defer app.Stop()

	ts := httptest.NewServer(app.Router)
	defer ts.Close()

	client := &apiClient{t: t, base: ts.URL, client: ts.Client()}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	pw := "pw-" + suffix
	adminEmail := "admin-" + suffix + "@example.com"
	customerEmail := "customer-" + suffix + "@example.com"
	sku := "SKU-" + suffix

	adminReg := client.do(t, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"email":     adminEmail,
		"password":  pw,
		"full_name": "Admin User",
		"role":      "admin",
	})
	require.Equal(t, http.StatusCreated, adminReg.status)

	adminToken := client.login(t, adminEmail, pw)

	var product productResponse
	createProduct := client.do(t, http.MethodPost, "/api/v1/products", adminToken, map[string]any{
		"sku":           sku,
		"name":          "Test Widget",
		"description":   "for integration",
		"price_amount":  1500,
		"currency":      "USD",
		"initial_stock": 5,
		"reorder_level": 1,
	})
	require.Equal(t, http.StatusCreated, createProduct.status, createProduct.bodyString())
	createProduct.decode(t, &product)
	require.NotZero(t, product.ID)

	customerReg := client.do(t, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"email":     customerEmail,
		"password":  pw,
		"full_name": "Customer User",
		"role":      "customer",
	})
	require.Equal(t, http.StatusCreated, customerReg.status)
	var customerUser struct {
		ID uint64 `json:"id"`
	}
	customerReg.decode(t, &customerUser)
	require.NotZero(t, customerUser.ID)

	customerToken := client.login(t, customerEmail, pw)

	var placed orderResponse
	place := client.do(t, http.MethodPost, "/api/v1/orders", customerToken, map[string]any{
		"items": []map[string]any{
			{"product_id": product.ID, "quantity": 2},
		},
	})
	require.Equal(t, http.StatusCreated, place.status, place.bodyString())
	place.decode(t, &placed)
	require.NotZero(t, placed.ID)
	require.Equal(t, "fulfilled", placed.Status)

	var inv inventoryResponse
	invResp := client.do(t, http.MethodGet, fmt.Sprintf("/api/v1/products/%d/inventory", product.ID), customerToken, nil)
	require.Equal(t, http.StatusOK, invResp.status, invResp.bodyString())
	invResp.decode(t, &inv)
	require.Equal(t, 3, inv.Available)

	var fetched orderResponse
	getResp := client.do(t, http.MethodGet, fmt.Sprintf("/api/v1/orders/%d", placed.ID), customerToken, nil)
	require.Equal(t, http.StatusOK, getResp.status, getResp.bodyString())
	getResp.decode(t, &fetched)
	require.Equal(t, placed.ID, fetched.ID)

	var second orderResponse
	placeSecond := client.do(t, http.MethodPost, "/api/v1/orders", customerToken, map[string]any{
		"items": []map[string]any{
			{"product_id": product.ID, "quantity": 1},
		},
	})
	require.Equal(t, http.StatusCreated, placeSecond.status, placeSecond.bodyString())
	placeSecond.decode(t, &second)
	require.Equal(t, "fulfilled", second.Status)

	cancelResp := client.do(t, http.MethodPut, fmt.Sprintf("/api/v1/orders/%d/cancel", second.ID), customerToken, nil)
	require.Equal(t, http.StatusConflict, cancelResp.status, cancelResp.bodyString())
	var cancelErr errorResponse
	cancelResp.decode(t, &cancelErr)
	require.Equal(t, "order.invalid_transition", cancelErr.Error.Code)

	noToken := client.do(t, http.MethodGet, "/api/v1/orders", "", nil)
	require.Equal(t, http.StatusUnauthorized, noToken.status)

	customerCreateProduct := client.do(t, http.MethodPost, "/api/v1/products", customerToken, map[string]any{
		"sku":          "SKU-forbidden-" + suffix,
		"name":         "Nope",
		"price_amount": 100,
		"currency":     "USD",
	})
	require.Equal(t, http.StatusForbidden, customerCreateProduct.status, customerCreateProduct.bodyString())

	outOfStock := client.do(t, http.MethodPost, "/api/v1/orders", customerToken, map[string]any{
		"items": []map[string]any{
			{"product_id": product.ID, "quantity": 999},
		},
	})
	require.Equal(t, http.StatusConflict, outOfStock.status, outOfStock.bodyString())
	var oosErr errorResponse
	outOfStock.decode(t, &oosErr)
	require.Equal(t, "order.out_of_stock", oosErr.Error.Code)

	idemKey := "idem-" + suffix
	var firstIdem orderResponse
	firstIdemResp := client.doWithHeaders(t, http.MethodPost, "/api/v1/orders", customerToken,
		map[string]string{"Idempotency-Key": idemKey},
		map[string]any{
			"items": []map[string]any{
				{"product_id": product.ID, "quantity": 1},
			},
		})
	require.Equal(t, http.StatusCreated, firstIdemResp.status, firstIdemResp.bodyString())
	firstIdemResp.decode(t, &firstIdem)

	var secondIdem orderResponse
	secondIdemResp := client.doWithHeaders(t, http.MethodPost, "/api/v1/orders", customerToken,
		map[string]string{"Idempotency-Key": idemKey},
		map[string]any{
			"items": []map[string]any{
				{"product_id": product.ID, "quantity": 1},
			},
		})
	require.Equal(t, http.StatusCreated, secondIdemResp.status, secondIdemResp.bodyString())
	secondIdemResp.decode(t, &secondIdem)

	require.Equal(t, firstIdem.ID, secondIdem.ID)

	listProducts := client.do(t, http.MethodGet, "/api/v1/products?page=1&size=20", "", nil)
	require.Equal(t, http.StatusOK, listProducts.status, listProducts.bodyString())

	getProduct := client.do(t, http.MethodGet, fmt.Sprintf("/api/v1/products/%d", product.ID), "", nil)
	require.Equal(t, http.StatusOK, getProduct.status, getProduct.bodyString())

	updateProduct := client.do(t, http.MethodPut, fmt.Sprintf("/api/v1/products/%d", product.ID), adminToken, map[string]any{
		"name":         "Updated Widget",
		"price_amount": 1800,
		"currency":     "USD",
		"active":       true,
	})
	require.Equal(t, http.StatusOK, updateProduct.status, updateProduct.bodyString())

	listOrders := client.do(t, http.MethodGet, "/api/v1/orders?page=1&size=20", customerToken, nil)
	require.Equal(t, http.StatusOK, listOrders.status, listOrders.bodyString())

	statusResp := client.do(t, http.MethodGet, fmt.Sprintf("/api/v1/orders/%d/status", placed.ID), customerToken, nil)
	require.Equal(t, http.StatusOK, statusResp.status, statusResp.bodyString())

	getUser := client.do(t, http.MethodGet, fmt.Sprintf("/api/v1/users/%d", customerUser.ID), customerToken, nil)
	require.Equal(t, http.StatusOK, getUser.status, getUser.bodyString())

	updateUser := client.do(t, http.MethodPut, fmt.Sprintf("/api/v1/users/%d", customerUser.ID), customerToken, map[string]any{
		"full_name": "Renamed Customer",
	})
	require.Equal(t, http.StatusOK, updateUser.status, updateUser.bodyString())

	adminOrders := client.do(t, http.MethodGet, "/api/v1/admin/orders?page=1&size=20", adminToken, nil)
	require.Equal(t, http.StatusOK, adminOrders.status, adminOrders.bodyString())

	dailyReport := client.do(t, http.MethodGet, "/api/v1/admin/reports/daily", adminToken, nil)
	require.Equal(t, http.StatusOK, dailyReport.status, dailyReport.bodyString())

	lowStock := client.do(t, http.MethodGet, "/api/v1/admin/inventory/low-stock", adminToken, nil)
	require.Equal(t, http.StatusOK, lowStock.status, lowStock.bodyString())
}

type apiClient struct {
	t      *testing.T
	base   string
	client *http.Client
}

type apiResponse struct {
	status int
	body   []byte
}

func (r apiResponse) decode(t *testing.T, target any) {
	require.NoError(t, json.Unmarshal(r.body, target), string(r.body))
}

func (r apiResponse) bodyString() string {
	return string(r.body)
}

func (a *apiClient) login(t *testing.T, email, password string) string {
	resp := a.do(t, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email":    email,
		"password": password,
	})
	require.Equal(t, http.StatusOK, resp.status, resp.bodyString())
	var out loginResponse
	resp.decode(t, &out)
	require.NotEmpty(t, out.Token)
	return out.Token
}

func (a *apiClient) do(t *testing.T, method, path, token string, payload any) apiResponse {
	return a.doWithHeaders(t, method, path, token, nil, payload)
}

func (a *apiClient) doWithHeaders(t *testing.T, method, path, token string, headers map[string]string, payload any) apiResponse {
	var body io.Reader
	if payload != nil {
		buf, err := json.Marshal(payload)
		require.NoError(t, err)
		body = bytes.NewReader(buf)
	}

	req, err := http.NewRequest(method, a.base+path, body)
	require.NoError(t, err)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := a.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return apiResponse{status: resp.StatusCode, body: raw}
}
