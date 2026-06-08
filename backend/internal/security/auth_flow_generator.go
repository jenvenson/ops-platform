package security

import (
	"fmt"
	"net/url"
	"strings"
)

type GenerateAuthFlowRequest struct {
	Preset        string       `json:"preset"`
	TargetURL     string       `json:"target_url"`
	LoginURL      string       `json:"login_url"`
	ContentType   string       `json:"content_type"`
	TokenPath     string       `json:"token_path"`
	SessionHeader string       `json:"session_header"`
	SessionPrefix string       `json:"session_prefix"`
	ExtraHeaders  []AuthHeader `json:"extra_headers"`
}

type GenerateAuthFlowResponse struct {
	Preset   string          `json:"preset"`
	AuthFlow *AuthFlowConfig `json:"auth_flow"`
	Preview  string          `json:"preview"`
}

func generateAuthFlow(req GenerateAuthFlowRequest) (*GenerateAuthFlowResponse, error) {
	preset := strings.ToLower(strings.TrimSpace(req.Preset))
	if preset == "" || preset == "auto" {
		preset = inferAuthFlowPreset(req.TargetURL)
	}

	var flow *AuthFlowConfig
	switch preset {
	case "web01-fscr":
		flow = buildWeb01AuthFlow()
	case "token-form":
		flow = buildGenericTokenAuthFlow(req, "form")
	case "token-json":
		flow = buildGenericTokenAuthFlow(req, "json")
	default:
		return nil, fmt.Errorf("unsupported auth flow preset: %s", preset)
	}

	preview, err := marshalAuthFlow(flow)
	if err != nil {
		return nil, err
	}

	return &GenerateAuthFlowResponse{
		Preset:   preset,
		AuthFlow: flow,
		Preview:  preview,
	}, nil
}

func inferAuthFlowPreset(targetURL string) string {
	lower := strings.ToLower(strings.TrimSpace(targetURL))
	if strings.Contains(lower, "/web_01") || strings.Contains(lower, "10.99.99.152") {
		return "web01-fscr"
	}
	return "token-json"
}

func buildGenericTokenAuthFlow(req GenerateAuthFlowRequest, fallbackContentType string) *AuthFlowConfig {
	contentType := strings.ToLower(strings.TrimSpace(req.ContentType))
	if contentType == "" {
		contentType = fallbackContentType
	}
	if contentType != "form" {
		contentType = "json"
	}

	loginURL := strings.TrimSpace(req.LoginURL)
	if loginURL == "" {
		loginURL = guessLoginURL(req.TargetURL)
	}

	tokenPath := strings.TrimSpace(req.TokenPath)
	if tokenPath == "" {
		tokenPath = "data.token"
	}

	sessionHeader := strings.TrimSpace(req.SessionHeader)
	if sessionHeader == "" {
		sessionHeader = "Authorization"
	}

	sessionPrefix := req.SessionPrefix
	if strings.EqualFold(sessionHeader, "Authorization") && sessionPrefix == "" {
		sessionPrefix = "Bearer "
	}

	variables := []AuthVariable{
		{Name: "lang", Value: "zh"},
	}

	headers := make([]AuthHeader, 0, len(req.ExtraHeaders)+1)
	if strings.TrimSpace(loginURL) != "" {
		headers = append(headers, req.ExtraHeaders...)
	}
	if len(headers) == 0 {
		headers = []AuthHeader{{Name: "language", Value: "${lang}"}}
	}

	sessionValue := "${token}"
	if sessionPrefix != "" {
		sessionValue = sessionPrefix + "${token}"
	}

	sessionHeaders := []AuthHeader{
		{Name: sessionHeader, Value: sessionValue},
	}
	extraSessionHeaders := filterOutHeader(req.ExtraHeaders, sessionHeader)
	if len(extraSessionHeaders) > 0 {
		sessionHeaders = append(sessionHeaders, extraSessionHeaders...)
	}

	return &AuthFlowConfig{
		Variables: variables,
		Login: &AuthRequestConfig{
			URL:         loginURL,
			Method:      "POST",
			ContentType: contentType,
			Headers:     headers,
			Body: map[string]string{
				"username": "${username}",
				"password": "${password}",
			},
			Extracts: []AuthExtract{
				{Name: "token", Source: "json", Path: tokenPath},
			},
		},
		SessionHeaders: sessionHeaders,
	}
}

func buildWeb01AuthFlow() *AuthFlowConfig {
	return &AuthFlowConfig{
		Variables: []AuthVariable{
			{Name: "tenant_key", Value: "web_01"},
			{Name: "lang", Value: "zh"},
			{Name: "client_basic", Value: "Basic {{base64:fscr-core-common:xsAQ0FNx7k}}"},
			{Name: "request_serial", Value: "{{uuid}}"},
			{Name: "request_ts", Value: "{{now_unix_ms}}"},
			{Name: "login_secret", Value: "5157F09EFDC096DE15EBE81A47057A7232F1B8E1"},
			{Name: "sign_source", Value: "{{upper:${request_serial}-${request_ts}-/auth/oauth2/token-POST-${login_secret}}}"},
			{Name: "sign_payload", Value: "{{base64:${sign_source}}}"},
			{Name: "login_signature", Value: "{{md5:${sign_payload}}}"},
			{Name: "encoded_password", Value: "{{base64:${request_serial}${password}${request_ts}}}"},
		},
		Login: &AuthRequestConfig{
			URL:         "http://10.99.99.152/auth/oauth2/token",
			Method:      "POST",
			ContentType: "form",
			Headers: []AuthHeader{
				{Name: "Authorization", Value: "${client_basic}"},
				{Name: "language", Value: "${lang}"},
				{Name: "tenantkey", Value: "${tenant_key}"},
				{Name: "fscr-request-serial", Value: "${request_serial}"},
				{Name: "fscr-timestamp", Value: "${request_ts}"},
				{Name: "fscr-signature", Value: "${login_signature}"},
			},
			Body: map[string]string{
				"grant_type": "password",
				"tenantKey":  "${tenant_key}",
				"username":   "${username}",
				"password":   "${encoded_password}",
			},
			Extracts: []AuthExtract{
				{Name: "token", Source: "json", Path: "data.accessToken.tokenValue"},
			},
		},
		SessionHeaders: []AuthHeader{
			{Name: "token", Value: "${token}"},
			{Name: "tenantkey", Value: "${tenant_key}"},
			{Name: "language", Value: "${lang}"},
		},
	}
}

func guessLoginURL(targetURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(targetURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	parsed.Path = "/api/login"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func filterOutHeader(headers []AuthHeader, headerName string) []AuthHeader {
	filtered := make([]AuthHeader, 0, len(headers))
	for _, header := range headers {
		if strings.EqualFold(strings.TrimSpace(header.Name), strings.TrimSpace(headerName)) {
			continue
		}
		filtered = append(filtered, header)
	}
	return filtered
}
