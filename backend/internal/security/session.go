// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package security

import (
	"fmt"
	"net/http"
	"strings"
)

type WebSession struct {
	Headers []AuthHeader
	Vars    map[string]string
}

func (s *WebSession) Apply(req *http.Request) {
	if s == nil || req == nil {
		return
	}
	for _, header := range s.Headers {
		name := strings.TrimSpace(header.Name)
		if name == "" {
			continue
		}
		req.Header.Set(name, header.Value)
	}
}

func BuildWebSession(targetURL string, config *WebScanConfig) (*WebSession, error) {
	if config == nil || strings.TrimSpace(config.AuthMode) == "" || strings.TrimSpace(config.AuthMode) == "none" {
		return &WebSession{}, nil
	}

	if config.AuthFlow != nil || strings.TrimSpace(config.AuthMode) == "advanced" {
		resolved, err := resolveAdvancedAuth(targetURL, config)
		if err != nil {
			return nil, err
		}
		return &WebSession{
			Headers: resolved.Headers,
			Vars:    resolved.Session,
		}, nil
	}

	authMode, credential, authHeader, err := resolveWebAuth(targetURL, config)
	if err != nil {
		return nil, err
	}

	session := &WebSession{
		Vars: map[string]string{},
	}
	if authMode == "" || authMode == "none" || credential == "" {
		return session, nil
	}

	if authMode == "multi-header" {
		for _, headerLine := range strings.Split(credential, "\n") {
			headerLine = strings.TrimSpace(headerLine)
			if headerLine == "" {
				continue
			}
			parts := strings.SplitN(headerLine, ":", 2)
			if len(parts) != 2 {
				continue
			}
			session.Headers = append(session.Headers, AuthHeader{
				Name:  strings.TrimSpace(parts[0]),
				Value: strings.TrimSpace(parts[1]),
			})
		}
		return session, nil
	}

	headerName := strings.TrimSpace(authHeader)
	headerValue := strings.TrimSpace(credential)

	switch authMode {
	case "cookie":
		if headerName == "" {
			headerName = "Cookie"
		}
	case "bearer":
		if headerName == "" {
			headerName = "Authorization"
		}
		if !strings.HasPrefix(strings.ToLower(headerValue), "bearer ") {
			headerValue = "Bearer " + headerValue
		}
	case "basic":
		if headerName == "" {
			headerName = "Authorization"
		}
		headerValue = "Basic " + headerValue
	case "header":
		if headerName == "" {
			headerName = "Authorization"
		}
	default:
		if headerName == "" {
			headerName = "Authorization"
		}
	}

	if headerName != "" && headerValue != "" {
		session.Headers = append(session.Headers, AuthHeader{
			Name:  headerName,
			Value: headerValue,
		})
	}
	return session, nil
}

func BuildAuthenticatedWebSession(targetURL string, config *WebScanConfig) (*WebSession, error) {
	if config == nil || strings.TrimSpace(config.AuthMode) == "" || strings.EqualFold(strings.TrimSpace(config.AuthMode), "none") {
		return nil, fmt.Errorf("web scan requires authenticated session")
	}

	session, err := BuildWebSession(targetURL, config)
	if err != nil {
		return nil, err
	}
	if session == nil || len(session.Headers) == 0 {
		return nil, fmt.Errorf("web scan requires resolved auth headers")
	}
	return session, nil
}