package request

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetClientIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1, 10.0.0.1")
	ip := GetClientIP(req)
	assert.Equal(t, "192.168.1.1", ip)

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("X-Real-IP", "10.0.0.2")
	ip2 := GetClientIP(req2)
	assert.Equal(t, "10.0.0.2", ip2)

	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.RemoteAddr = "127.0.0.1:12345"
	ip3 := GetClientIP(req3)
	assert.Equal(t, "127.0.0.1", ip3)
}

func TestGetQueryString(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?name=satria", nil)
	assert.Equal(t, "satria", GetQueryString(req, "name", "default"))
	assert.Equal(t, "default", GetQueryString(req, "age", "default"))
}

func TestGetQueryInt(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?page=2&limit=abc", nil)
	assert.Equal(t, 2, GetQueryInt(req, "page", 1))
	assert.Equal(t, 10, GetQueryInt(req, "limit", 10))
	assert.Equal(t, 5, GetQueryInt(req, "offset", 5))
}

func TestGetQueryBool(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?active=true&admin=false&invalid=abc", nil)
	assert.Equal(t, true, GetQueryBool(req, "active", false))
	assert.Equal(t, false, GetQueryBool(req, "admin", true))
	assert.Equal(t, false, GetQueryBool(req, "invalid", false))
}

func TestGetBearerToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer my-secret-token")
	assert.Equal(t, "my-secret-token", GetBearerToken(req))

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("Authorization", "Basic user:pass")
	assert.Equal(t, "", GetBearerToken(req2))

	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	assert.Equal(t, "", GetBearerToken(req3))
}

type dummyPayload struct {
	Name string `json:"name" validate:"required"`
	Age  int    `json:"age" validate:"required,min=18"`
}

func TestBindJSON(t *testing.T) {
	body := bytes.NewBufferString(`{"name":"budi","age":25}`)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/json")

	var payload dummyPayload
	err := BindJSON(req, &payload)
	assert.NoError(t, err)
	assert.Equal(t, "budi", payload.Name)
	assert.Equal(t, 25, payload.Age)
}

func TestBindJSON_InvalidContentType(t *testing.T) {
	body := bytes.NewBufferString(`{"name":"budi","age":25}`)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "text/plain")

	var payload dummyPayload
	err := BindJSON(req, &payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported content type")
}

func TestBindAndValidate(t *testing.T) {
	body := bytes.NewBufferString(`{"name":"budi","age":15}`)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/json")

	var payload dummyPayload
	err := BindAndValidate(req, &payload)
	assert.Error(t, err) // validation should fail because age < 18
}
