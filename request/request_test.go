package request

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetClientIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("X-Forwarded-For", "192.168.1.1, 10.0.0.1")
	ip := GetClientIP(req)
	assert.Equal(t, "192.168.1.1", ip)

	req2 := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req2.Header.Set("X-Real-IP", "10.0.0.2")
	ip2 := GetClientIP(req2)
	assert.Equal(t, "10.0.0.2", ip2)

	req3 := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req3.RemoteAddr = "127.0.0.1:12345"
	ip3 := GetClientIP(req3)
	assert.Equal(t, "127.0.0.1", ip3)
}

func TestGetQueryString(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?name=satria", http.NoBody)
	assert.Equal(t, "satria", GetQueryString(req, "name", "default"))
	assert.Equal(t, "default", GetQueryString(req, "age", "default"))
}

func TestGetQueryInt(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?page=2&limit=abc", http.NoBody)
	assert.Equal(t, 2, GetQueryInt(req, "page", 1))
	assert.Equal(t, 10, GetQueryInt(req, "limit", 10))
	assert.Equal(t, 5, GetQueryInt(req, "offset", 5))
}

func TestGetQueryInt64(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?id=9223372036854775807&invalid=abc", http.NoBody)
	assert.Equal(t, int64(9223372036854775807), GetQueryInt64(req, "id", 0))
	assert.Equal(t, int64(100), GetQueryInt64(req, "invalid", 100))
	assert.Equal(t, int64(50), GetQueryInt64(req, "missing", 50))
}

func TestGetQueryBool(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?active=true&admin=false&invalid=abc", http.NoBody)
	assert.Equal(t, true, GetQueryBool(req, "active", false))
	assert.Equal(t, false, GetQueryBool(req, "admin", true))
	assert.Equal(t, false, GetQueryBool(req, "invalid", false))
}

func TestGetPathHelpers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/shield/admin/apps/app_123", http.NoBody)
	req.SetPathValue("id", "app_123")
	req.SetPathValue("num", "42")
	req.SetPathValue("big", "9223372036854775807")
	req.SetPathValue("flag", "true")
	testUUID := "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	req.SetPathValue("uuid", testUUID)

	// GetPathValue
	assert.Equal(t, "app_123", GetPathValue(req, "id"))
	assert.Equal(t, "app_123", GetPathValue(req, "{id}")) // test bracket trimming
	assert.Equal(t, "", GetPathValue(req, "nonexistent"))

	// GetPathString
	assert.Equal(t, "app_123", GetPathString(req, "id"))
	assert.Equal(t, "app_123", GetPathString(req, "{id}", "fallback"))
	assert.Equal(t, "fallback", GetPathString(req, "missing", "fallback"))
	assert.Equal(t, "", GetPathString(req, "missing"))

	// GetPathInt
	assert.Equal(t, 42, GetPathInt(req, "num"))
	assert.Equal(t, 42, GetPathInt(req, "{num}", 99))
	assert.Equal(t, 99, GetPathInt(req, "id", 99)) // "app_123" is not an int
	assert.Equal(t, 0, GetPathInt(req, "id"))      // default 0 without arg
	assert.Equal(t, 10, GetPathInt(req, "missing", 10))

	// GetPathInt64
	assert.Equal(t, int64(9223372036854775807), GetPathInt64(req, "big"))
	assert.Equal(t, int64(9223372036854775807), GetPathInt64(req, "{big}", 99))
	assert.Equal(t, int64(99), GetPathInt64(req, "id", 99))
	assert.Equal(t, int64(0), GetPathInt64(req, "id"))

	// GetPathBool
	assert.True(t, GetPathBool(req, "flag"))
	assert.True(t, GetPathBool(req, "{flag}", false))
	assert.False(t, GetPathBool(req, "id")) // invalid bool defaults to false
	assert.True(t, GetPathBool(req, "id", true))

	// GetPathUUID
	u, err := GetPathUUID(req, "uuid")
	assert.NoError(t, err)
	assert.Equal(t, testUUID, u.String())

	_, err = GetPathUUID(req, "id") // not a uuid
	assert.Error(t, err)

	_, err = GetPathUUID(req, "missing") // empty path value
	assert.Error(t, err)

	// Nil request safety
	assert.Equal(t, "", GetPathValue(nil, "id"))
	assert.Equal(t, "default", GetPathString(nil, "id", "default"))
	assert.Equal(t, 0, GetPathInt(nil, "id"))
	assert.Equal(t, int64(0), GetPathInt64(nil, "id"))
	assert.False(t, GetPathBool(nil, "id"))
	_, err = GetPathUUID(nil, "id")
	assert.Error(t, err)
}

func TestGetBearerToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("Authorization", "Bearer my-secret-token")
	assert.Equal(t, "my-secret-token", GetBearerToken(req))

	req2 := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req2.Header.Set("Authorization", "Basic user:pass")
	assert.Equal(t, "", GetBearerToken(req2))

	req3 := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
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

func TestGetQueryStringSlice(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?codes=APP1,%20APP2%20,,APP3", http.NoBody)
	assert.Equal(t, []string{"APP1", "APP2", "APP3"}, GetQueryStringSlice(req, "codes"))
	assert.Nil(t, GetQueryStringSlice(req, "nonexistent"))
}

func TestGetQueryIntSlice(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?ids=1,%202,%20invalid,%203,", http.NoBody)
	assert.Equal(t, []int{1, 2, 3}, GetQueryIntSlice(req, "ids"))
	assert.Nil(t, GetQueryIntSlice(req, "nonexistent"))
}

func TestGetQueryMapSlice(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?filter=status:active,pending&filter=role:admin&filter=invalid&filter=:val", http.NoBody)
	res := GetQueryMapSlice(req, "filter", ":")

	assert.ElementsMatch(t, []string{"active", "pending"}, res["status"])
	assert.Equal(t, []string{"admin"}, res["role"])
	assert.NotContains(t, res, "")
	assert.Empty(t, GetQueryMapSlice(req, "nonexistent", ":"))
}

func TestGetQueryMapIntSlice(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?group=status:1,2,abc&group=category:3,4&group=invalid", http.NoBody)
	res := GetQueryMapIntSlice(req, "group", ":")

	assert.Equal(t, []int{1, 2}, res["status"])
	assert.Equal(t, []int{3, 4}, res["category"])
	assert.Empty(t, GetQueryMapIntSlice(req, "nonexistent", ":"))
}

func TestGetQueryMapBoolSlice(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?flags=featureA:true,false,xyz&flags=featureB:1&flags=invalid", http.NoBody)
	res := GetQueryMapBoolSlice(req, "flags", ":")

	assert.Equal(t, []bool{true, false}, res["featureA"])
	assert.Equal(t, []bool{true}, res["featureB"])
	assert.Empty(t, GetQueryMapBoolSlice(req, "nonexistent", ":"))
}

func TestGetHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("X-Custom-Header", "custom-value")

	val, ok := GetHeader(req, "X-Custom-Header")
	assert.True(t, ok)
	assert.Equal(t, "custom-value", val)

	// Canonical case-insensitive lookup
	valLower, okLower := GetHeader(req, "x-custom-header")
	assert.True(t, okLower)
	assert.Equal(t, "custom-value", valLower)

	val, ok = GetHeader(req, "Non-Existent")
	assert.False(t, ok)
	assert.Empty(t, val)

	valNil, okNil := GetHeader(nil, "X-Header")
	assert.False(t, okNil)
	assert.Empty(t, valNil)
}

func TestGetBasicAuthUsernamePassword(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.SetBasicAuth("admin", "secret123")

	user, pass, ok := GetBasicAuthUsernamePassword(req)
	assert.True(t, ok)
	assert.Equal(t, "admin", user)
	assert.Equal(t, "secret123", pass)

	reqEmpty := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	_, _, ok = GetBasicAuthUsernamePassword(reqEmpty)
	assert.False(t, ok)
}

func TestGetClientIP_EdgeCases(t *testing.T) {
	// True-Client-IP header
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("True-Client-IP", "203.0.113.195")
	assert.Equal(t, "203.0.113.195", GetClientIP(req))

	// IPv6 with port
	reqIPv6 := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	reqIPv6.RemoteAddr = "[::1]:8080"
	assert.Equal(t, "::1", GetClientIP(reqIPv6))

	// IPv6 without port
	reqIPv6NoPort := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	reqIPv6NoPort.RemoteAddr = "2001:db8::1"
	assert.Equal(t, "2001:db8::1", GetClientIP(reqIPv6NoPort))

	// Invalid remote addr and no headers
	reqInvalid := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	reqInvalid.RemoteAddr = "invalid-ip-string"
	assert.Empty(t, GetClientIP(reqInvalid))
}

func TestGetQueryString_DoesNotDrainBody(t *testing.T) {
	body := bytes.NewBufferString(`{"name":"budi","age":25}`)
	req := httptest.NewRequest(http.MethodPost, "/?filter=active", body)
	req.Header.Set("Content-Type", "application/json")

	// Read query string first
	queryVal := GetQueryString(req, "filter", "")
	assert.Equal(t, "active", queryVal)

	// Body should NOT be drained; BindJSON must succeed
	var payload dummyPayload
	err := BindJSON(req, &payload)
	assert.NoError(t, err)
	assert.Equal(t, "budi", payload.Name)
}
