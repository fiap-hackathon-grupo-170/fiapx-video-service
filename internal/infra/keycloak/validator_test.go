package keycloak_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fiapx/fiapx-video-service/internal/infra/keycloak"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewTokenValidator_NotNil(t *testing.T) {
	v := keycloak.NewTokenValidator("http://localhost:8080", "fiapx", zap.NewNop())
	assert.NotNil(t, v)
}

func TestTokenValidator_Validate_InvalidToken(t *testing.T) {
	// JWKS server that returns empty keys
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"keys": []interface{}{}})
	}))
	defer srv.Close()

	v := keycloak.NewTokenValidator(srv.URL, "fiapx", zap.NewNop())
	claims, err := v.Validate("invalid.token.here")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestTokenValidator_Validate_JWKSServerDown(t *testing.T) {
	// Point to a server that won't respond
	v := keycloak.NewTokenValidator("http://localhost:19999", "fiapx", zap.NewNop())
	claims, err := v.Validate("some.token.value")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestTokenValidator_Validate_JWKSServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	v := keycloak.NewTokenValidator(srv.URL, "fiapx", zap.NewNop())
	claims, err := v.Validate("some.token.value")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestTokenValidator_Validate_JWKSNonRSAKeys(t *testing.T) {
	// Server returns non-RSA keys that should be skipped
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"keys": []map[string]interface{}{
				{"kid": "k1", "kty": "EC", "alg": "ES256", "use": "sig"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	v := keycloak.NewTokenValidator(srv.URL, "fiapx", zap.NewNop())
	claims, err := v.Validate("some.invalid.token")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestTokenValidator_Validate_JWKSInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json {"))
	}))
	defer srv.Close()

	v := keycloak.NewTokenValidator(srv.URL, "fiapx", zap.NewNop())
	claims, err := v.Validate("some.token")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestTokenValidator_JWKSURLFormat(t *testing.T) {
	// Verify the JWKS URL is constructed from keycloakURL + realm
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"keys": []interface{}{}})
	}))
	defer srv.Close()

	v := keycloak.NewTokenValidator(srv.URL, "my-realm", zap.NewNop())
	_, _ = v.Validate("token")

	require.Equal(t, "/realms/my-realm/protocol/openid-connect/certs", capturedPath)
}
