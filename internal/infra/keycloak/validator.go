package keycloak

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/fiapx/fiapx-video-service/internal/domain/port"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type jwksKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwksResponse struct {
	Keys []jwksKey `json:"keys"`
}

type TokenValidator struct {
	keycloakURL string
	realm       string
	logger      *zap.Logger

	mu          sync.RWMutex
	cachedKeys  map[string]interface{}
	cacheExpiry time.Time
}

func NewTokenValidator(keycloakURL, realm string, logger *zap.Logger) *TokenValidator {
	return &TokenValidator{
		keycloakURL: keycloakURL,
		realm:       realm,
		logger:      logger,
		cachedKeys:  make(map[string]interface{}),
	}
}

func (v *TokenValidator) jwksURL() string {
	return fmt.Sprintf("%s/realms/%s/protocol/openid-connect/certs", v.keycloakURL, v.realm)
}

func (v *TokenValidator) fetchJWKS(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL(), nil)
	if err != nil {
		return fmt.Errorf("create jwks request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks endpoint returned status %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("decode jwks: %w", err)
	}

	keys := make(map[string]interface{}, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pubKey, err := parseRSAPublicKey(k.N, k.E)
		if err != nil {
			v.logger.Warn("failed to parse JWK key", zap.String("kid", k.Kid), zap.Error(err))
			continue
		}
		keys[k.Kid] = pubKey
	}

	v.mu.Lock()
	v.cachedKeys = keys
	v.cacheExpiry = time.Now().Add(time.Hour)
	v.mu.Unlock()

	return nil
}

func (v *TokenValidator) getKeys(ctx context.Context) map[string]interface{} {
	v.mu.RLock()
	expired := time.Now().After(v.cacheExpiry)
	keys := v.cachedKeys
	v.mu.RUnlock()

	if expired || len(keys) == 0 {
		if err := v.fetchJWKS(ctx); err != nil {
			v.logger.Warn("failed to refresh JWKS, using cached keys", zap.Error(err))
		}
		v.mu.RLock()
		keys = v.cachedKeys
		v.mu.RUnlock()
	}

	return keys
}

func (v *TokenValidator) Validate(tokenString string) (*port.Claims, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	keys := v.getKeys(ctx)

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		kid, ok := t.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing kid in token header")
		}

		key, found := keys[kid]
		if !found {
			return nil, fmt.Errorf("unknown kid: %s", kid)
		}

		return key, nil
	}, jwt.WithValidMethods([]string{"RS256"}))

	if err != nil {
		return nil, fmt.Errorf("validate token: %w", err)
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	sub, _ := mapClaims["sub"].(string)
	email, _ := mapClaims["email"].(string)

	if sub == "" {
		return nil, fmt.Errorf("missing sub claim in token")
	}

	return &port.Claims{
		UserID:    sub,
		UserEmail: email,
	}, nil
}
