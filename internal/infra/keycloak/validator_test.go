package keycloak

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

func makeTestKey(t *testing.T) (*rsa.PrivateKey, string, *httptest.Server) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	kid := "test-kid-1"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nBytes := privateKey.PublicKey.N.Bytes()
		eBytes := big.NewInt(int64(privateKey.PublicKey.E)).Bytes()
		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kid": kid,
					"kty": "RSA",
					"alg": "RS256",
					"use": "sig",
					"n":   base64.RawURLEncoding.EncodeToString(nBytes),
					"e":   base64.RawURLEncoding.EncodeToString(eBytes),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
	return privateKey, kid, srv
}

func signToken(t *testing.T, privateKey *rsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	return tokenString
}

func TestNewTokenValidator_NotNil(t *testing.T) {
	v := NewTokenValidator("http://localhost:8080", "fiapx", zap.NewNop())
	if v == nil {
		t.Fatal("expected non-nil validator")
	}
}

func TestValidate_ValidToken(t *testing.T) {
	privateKey, kid, srv := makeTestKey(t)
	defer srv.Close()

	v := NewTokenValidator(srv.URL, "test-realm", zap.NewNop())

	tokenString := signToken(t, privateKey, kid, jwt.MapClaims{
		"sub":   "user-123",
		"email": "user@test.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	})

	claims, err := v.Validate(tokenString)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("expected user-123, got %s", claims.UserID)
	}
	if claims.UserEmail != "user@test.com" {
		t.Errorf("expected user@test.com, got %s", claims.UserEmail)
	}
}

func TestValidate_MissingSub(t *testing.T) {
	privateKey, kid, srv := makeTestKey(t)
	defer srv.Close()

	v := NewTokenValidator(srv.URL, "test-realm", zap.NewNop())

	tokenString := signToken(t, privateKey, kid, jwt.MapClaims{
		"email": "user@test.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	})

	_, err := v.Validate(tokenString)
	if err == nil {
		t.Fatal("expected error for missing sub")
	}
}

func TestValidate_InvalidToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"keys": []interface{}{}})
	}))
	defer srv.Close()

	v := NewTokenValidator(srv.URL, "fiapx", zap.NewNop())
	claims, err := v.Validate("invalid.token.here")
	if err == nil {
		t.Fatal("expected error")
	}
	if claims != nil {
		t.Fatal("expected nil claims")
	}
}

func TestValidate_JWKSServerDown(t *testing.T) {
	v := NewTokenValidator("http://localhost:19999", "fiapx", zap.NewNop())
	claims, err := v.Validate("some.token.value")
	if err == nil {
		t.Fatal("expected error")
	}
	if claims != nil {
		t.Fatal("expected nil claims")
	}
}

func TestValidate_JWKSServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	v := NewTokenValidator(srv.URL, "fiapx", zap.NewNop())
	_, err := v.Validate("some.token.value")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_JWKSNonRSAKeys(t *testing.T) {
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

	v := NewTokenValidator(srv.URL, "fiapx", zap.NewNop())
	_, err := v.Validate("some.invalid.token")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_JWKSInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json {"))
	}))
	defer srv.Close()

	v := NewTokenValidator(srv.URL, "fiapx", zap.NewNop())
	_, err := v.Validate("some.token")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_UnknownKid(t *testing.T) {
	privateKey, _, srv := makeTestKey(t)
	defer srv.Close()

	v := NewTokenValidator(srv.URL, "test-realm", zap.NewNop())

	tokenString := signToken(t, privateKey, "unknown-kid", jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	_, err := v.Validate(tokenString)
	if err == nil {
		t.Fatal("expected error for unknown kid")
	}
}

func TestValidate_WrongSigningMethod(t *testing.T) {
	_, _, srv := makeTestKey(t)
	defer srv.Close()

	v := NewTokenValidator(srv.URL, "test-realm", zap.NewNop())

	// Sign with HMAC instead of RSA
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	token.Header["kid"] = "test-kid-1"
	tokenString, _ := token.SignedString([]byte("secret"))

	_, err := v.Validate(tokenString)
	if err == nil {
		t.Fatal("expected error for wrong signing method")
	}
}

func TestValidate_ExpiredToken(t *testing.T) {
	privateKey, kid, srv := makeTestKey(t)
	defer srv.Close()

	v := NewTokenValidator(srv.URL, "test-realm", zap.NewNop())

	tokenString := signToken(t, privateKey, kid, jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})

	_, err := v.Validate(tokenString)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestValidate_CachedKeys(t *testing.T) {
	privateKey, kid, srv := makeTestKey(t)
	defer srv.Close()

	v := NewTokenValidator(srv.URL, "test-realm", zap.NewNop())

	tokenString := signToken(t, privateKey, kid, jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	// First call fetches keys
	_, err := v.Validate(tokenString)
	if err != nil {
		t.Fatalf("first validate failed: %v", err)
	}

	// Second call uses cached keys
	claims, err := v.Validate(tokenString)
	if err != nil {
		t.Fatalf("second validate failed: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("expected user-123, got %s", claims.UserID)
	}
}

func TestValidate_JWKSURLFormat(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"keys": []interface{}{}})
	}))
	defer srv.Close()

	v := NewTokenValidator(srv.URL, "my-realm", zap.NewNop())
	_, _ = v.Validate("token")

	expected := "/realms/my-realm/protocol/openid-connect/certs"
	if capturedPath != expected {
		t.Errorf("expected path %s, got %s", expected, capturedPath)
	}
}

func TestValidate_InvalidKeyData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kid": "bad-key",
					"kty": "RSA",
					"alg": "RS256",
					"use": "sig",
					"n":   "!!!invalid-base64!!!",
					"e":   "AQAB",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
	defer srv.Close()

	v := NewTokenValidator(srv.URL, "test-realm", zap.NewNop())
	_, err := v.Validate("some.token")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseRSAPublicKey_Valid(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	nBytes := privateKey.PublicKey.N.Bytes()
	eBytes := big.NewInt(int64(privateKey.PublicKey.E)).Bytes()

	n := base64.RawURLEncoding.EncodeToString(nBytes)
	e := base64.RawURLEncoding.EncodeToString(eBytes)

	pub, err := parseRSAPublicKey(n, e)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if pub.N.Cmp(privateKey.PublicKey.N) != 0 {
		t.Error("modulus mismatch")
	}
}

func TestParseRSAPublicKey_InvalidModulus(t *testing.T) {
	_, err := parseRSAPublicKey("!!!invalid!!!", "AQAB")
	if err == nil {
		t.Fatal("expected error for invalid modulus")
	}
}

func TestParseRSAPublicKey_InvalidExponent(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	n := base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes())

	_, err := parseRSAPublicKey(n, "!!!invalid!!!")
	if err == nil {
		t.Fatal("expected error for invalid exponent")
	}
}
