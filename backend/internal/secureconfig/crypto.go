// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package secureconfig

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
)

func EncryptString(plainText string) (string, error) {
	if plainText == "" {
		return "", nil
	}
	block, err := aes.NewCipher(loadKey())
	if err != nil {
		return "", fmt.Errorf("create cipher failed: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm failed: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce failed: %w", err)
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func DecryptString(cipherText string) (string, error) {
	if cipherText == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", fmt.Errorf("decode secret failed: %w", err)
	}
	block, err := aes.NewCipher(loadKey())
	if err != nil {
		return "", fmt.Errorf("create cipher failed: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm failed: %w", err)
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("invalid secret payload")
	}
	nonce := raw[:gcm.NonceSize()]
	encrypted := raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt secret failed: %w", err)
	}
	return string(plain), nil
}

func loadKey() []byte {
	secret := strings.TrimSpace(os.Getenv("CONFIG_CIPHER_KEY"))
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("JWT_SECRET"))
	}
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}