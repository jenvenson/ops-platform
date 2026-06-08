package security

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

type RequestSigner interface {
	Name() string
	Apply(vars map[string]string, config *AuthSignerConfig) error
}

type sha256Signer struct{}

type md5Signer struct{}

func (s *md5Signer) Name() string {
	return "md5"
}

func (s *md5Signer) Apply(vars map[string]string, config *AuthSignerConfig) error {
	payload := renderAuthTemplate(config.Payload, vars)
	sum := md5.Sum([]byte(payload))
	vars["sign_payload"] = payload
	vars["signature"] = hex.EncodeToString(sum[:])
	return nil
}

func (s *sha256Signer) Name() string {
	return "sha256"
}

func (s *sha256Signer) Apply(vars map[string]string, config *AuthSignerConfig) error {
	payload := renderAuthTemplate(config.Payload, vars)
	sum := sha256.Sum256([]byte(payload))
	vars["sign_payload"] = payload
	vars["signature"] = hex.EncodeToString(sum[:])
	return nil
}

type hmacSHA256Signer struct{}

func (s *hmacSHA256Signer) Name() string {
	return "hmac-sha256"
}

func (s *hmacSHA256Signer) Apply(vars map[string]string, config *AuthSignerConfig) error {
	secret := renderAuthTemplate(config.Secret, vars)
	if strings.TrimSpace(secret) == "" {
		return fmt.Errorf("hmac-sha256 signer requires secret")
	}
	payload := renderAuthTemplate(config.Payload, vars)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	vars["sign_payload"] = payload
	vars["signature"] = hex.EncodeToString(mac.Sum(nil))
	return nil
}

func getRequestSigner(name string) RequestSigner {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "none":
		return nil
	case "sha256":
		return &sha256Signer{}
	case "md5":
		return &md5Signer{}
	case "hmac-sha256":
		return &hmacSHA256Signer{}
	default:
		return nil
	}
}

func applySignerVars(vars map[string]string, config *AuthSignerConfig) (bool, error) {
	if config == nil {
		return false, nil
	}
	signer := getRequestSigner(config.Name)
	if signer == nil {
		if strings.TrimSpace(config.Name) == "" || strings.EqualFold(strings.TrimSpace(config.Name), "none") {
			return false, nil
		}
		return false, fmt.Errorf("unsupported signer: %s", config.Name)
	}
	if err := signer.Apply(vars, config); err != nil {
		return false, err
	}
	return true, nil
}

func renderSignerHeaders(vars map[string]string, config *AuthSignerConfig) ([]AuthHeader, error) {
	if config == nil {
		return nil, nil
	}
	return renderAuthHeaders(config.Headers, vars), nil
}
