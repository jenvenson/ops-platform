package license

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidKey      = errors.New("license key is invalid")
	ErrExpired         = errors.New("license key has expired")
	ErrMissingKey      = errors.New("license key is not configured")
	ErrInvalidFeatures = errors.New("license key has invalid features")
)

// License 表示解析后的授权信息
type License struct {
	Customer   string    `json:"customer"`
	ExpiresAt  time.Time `json:"expires_at"`
	Features   []string  `json:"features"`
	Valid      bool      `json:"valid"`
	ValidError string    `json:"valid_error,omitempty"`
}

// embeddedPublicKey 是编译进后端的 RSA 公钥，用于验证 License Key 签名
const embeddedPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAjHO59zrMpbwxpiGHsOnt
eOmwvYOT1d4KQXRng8aUebM+uWh4TM69GJ4wY14dimsz8AisORDKD+d2hHFttFcT
bt3GqjU1DZb7gDfLiBq1JGI/dAsXwUb1AsqSxUaMtB8Idj2cYWSTx3docoBk+ocf
jWy6Vsswi0oXP4cDWWa3adzycY+1/PMnEz9b+Bj9K8pvWYOr4eV5wBfD5YUOC5TE
Vinx8OHqQpbvQLryIyW4gmmC01cCohc8Iv53NemJg1lDAa2zXWmwRbufoPNe+M20
kpHUMnKhB9xDhkEx0N8EcG0e64QdGgNY9wKgOjNQE2yhxDp7uJDNV19n3VVmoPKc
mwIDAQAB
-----END PUBLIC KEY-----`

// Validate 验证 License Key 并返回授权信息
func Validate(key string) *License {
	if key == "" {
		return &License{Valid: false, ValidError: ErrMissingKey.Error()}
	}

	pubKey, err := parsePublicKey(embeddedPublicKey)
	if err != nil {
		return &License{Valid: false, ValidError: "failed to parse public key"}
	}

	token, err := jwt.Parse(key, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return pubKey, nil
	})

	if err != nil || !token.Valid {
		return &License{Valid: false, ValidError: ErrInvalidKey.Error()}
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return &License{Valid: false, ValidError: ErrInvalidKey.Error()}
	}

	customer, _ := claims["sub"].(string)
	if customer == "" {
		return &License{Valid: false, ValidError: "missing customer in license"}
	}

	exp, err := claims.GetExpirationTime()
	if err != nil || exp == nil {
		return &License{Valid: false, ValidError: "missing expiration in license"}
	}
	if exp.Before(time.Now()) {
		return &License{
			Customer:  customer,
			ExpiresAt: exp.Time,
			Valid:     false,
			ValidError: ErrExpired.Error(),
		}
	}

	features := parseFeatures(claims)

	return &License{
		Customer:  customer,
		ExpiresAt: exp.Time,
		Features:  features,
		Valid:     true,
	}
}

func parseFeatures(claims jwt.MapClaims) []string {
	raw, ok := claims["features"]
	if !ok {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func parsePublicKey(pemKey string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil {
		return nil, errors.New("failed to decode PEM public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}
	return rsaPub, nil
}
