package security

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type AuthHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type AuthVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type AuthExtract struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Path   string `json:"path"`
	Header string `json:"header"`
}

type AuthRequestConfig struct {
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	ContentType string            `json:"content_type"`
	Headers     []AuthHeader      `json:"headers"`
	Body        map[string]string `json:"body"`
	Extracts    []AuthExtract     `json:"extracts"`
	TokenField  string            `json:"token_field"`
}

type AuthSignerConfig struct {
	Name    string       `json:"name"`
	Secret  string       `json:"secret"`
	Payload string       `json:"payload"`
	Headers []AuthHeader `json:"headers"`
}

type AuthFlowConfig struct {
	Variables      []AuthVariable     `json:"variables"`
	Login          *AuthRequestConfig `json:"login"`
	Signer         *AuthSignerConfig  `json:"signer"`
	SessionHeaders []AuthHeader       `json:"session_headers"`
}

type ResolvedAuth struct {
	Headers []AuthHeader
	Session map[string]string
}

var (
	authVarPattern    = regexp.MustCompile(`\$\{([a-zA-Z0-9_.-]+)\}`)
	authBase64Pattern = regexp.MustCompile(`\{\{base64:([^{}]*)\}\}`)
	authURLEncodeRe   = regexp.MustCompile(`\{\{urlencode:([^{}]*)\}\}`)
	authRandHexRe     = regexp.MustCompile(`\{\{rand_hex(?::(\d+))?\}\}`)
	authUUIDRe        = regexp.MustCompile(`\{\{uuid\}\}`)
	authUpperRe       = regexp.MustCompile(`\{\{upper:([^{}]*)\}\}`)
	authLowerRe       = regexp.MustCompile(`\{\{lower:([^{}]*)\}\}`)
	authMD5Re         = regexp.MustCompile(`\{\{md5:([^{}]*)\}\}`)
	authSHA256Re      = regexp.MustCompile(`\{\{sha256:([^{}]*)\}\}`)
)

func parseAuthFlow(raw json.RawMessage) (*AuthFlowConfig, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "" {
		return nil, nil
	}

	var cfg AuthFlowConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func marshalAuthFlow(cfg *AuthFlowConfig) (string, error) {
	if cfg == nil {
		return "", nil
	}
	encoded, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func buildBaseAuthVariables(targetURL string, config *WebScanConfig) map[string]string {
	now := time.Now()
	vars := map[string]string{
		"target_url":  sanitizeScanURL(targetURL),
		"now_unix":    strconv.FormatInt(now.Unix(), 10),
		"now_unix_ms": strconv.FormatInt(now.UnixMilli(), 10),
		"credential":  "",
		"username":    "",
		"password":    "",
		"login_url":   "",
		"auth_header": "",
		"token_field": "",
	}
	if config == nil {
		return vars
	}

	vars["credential"] = config.Credential
	vars["username"] = strings.TrimSpace(config.Username)
	vars["password"] = config.Password
	vars["login_url"] = resolveLoginFormURL(targetURL, config)
	vars["auth_header"] = strings.TrimSpace(config.AuthHeader)
	vars["token_field"] = strings.TrimSpace(config.TokenField)
	return vars
}

func renderAuthTemplate(input string, vars map[string]string) string {
	rendered := input
	for i := 0; i < 4; i++ {
		prev := rendered

		rendered = authVarPattern.ReplaceAllStringFunc(rendered, func(match string) string {
			parts := authVarPattern.FindStringSubmatch(match)
			if len(parts) != 2 {
				return ""
			}
			return vars[parts[1]]
		})

		rendered = strings.ReplaceAll(rendered, "{{now_unix}}", vars["now_unix"])
		rendered = strings.ReplaceAll(rendered, "{{now_unix_ms}}", vars["now_unix_ms"])

		rendered = authUUIDRe.ReplaceAllStringFunc(rendered, func(match string) string {
			return randomUUID()
		})

		rendered = authRandHexRe.ReplaceAllStringFunc(rendered, func(match string) string {
			parts := authRandHexRe.FindStringSubmatch(match)
			size := 16
			if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
				if parsed, err := strconv.Atoi(parts[1]); err == nil && parsed > 0 {
					size = parsed
				}
			}
			return randomHex(size)
		})

		rendered = authUpperRe.ReplaceAllStringFunc(rendered, func(match string) string {
			parts := authUpperRe.FindStringSubmatch(match)
			if len(parts) != 2 {
				return ""
			}
			return strings.ToUpper(parts[1])
		})

		rendered = authLowerRe.ReplaceAllStringFunc(rendered, func(match string) string {
			parts := authLowerRe.FindStringSubmatch(match)
			if len(parts) != 2 {
				return ""
			}
			return strings.ToLower(parts[1])
		})

		rendered = authBase64Pattern.ReplaceAllStringFunc(rendered, func(match string) string {
			parts := authBase64Pattern.FindStringSubmatch(match)
			if len(parts) != 2 {
				return ""
			}
			return base64.StdEncoding.EncodeToString([]byte(parts[1]))
		})

		rendered = authMD5Re.ReplaceAllStringFunc(rendered, func(match string) string {
			parts := authMD5Re.FindStringSubmatch(match)
			if len(parts) != 2 {
				return ""
			}
			sum := md5.Sum([]byte(parts[1]))
			return hex.EncodeToString(sum[:])
		})

		rendered = authSHA256Re.ReplaceAllStringFunc(rendered, func(match string) string {
			parts := authSHA256Re.FindStringSubmatch(match)
			if len(parts) != 2 {
				return ""
			}
			sum := sha256.Sum256([]byte(parts[1]))
			return hex.EncodeToString(sum[:])
		})

		rendered = authURLEncodeRe.ReplaceAllStringFunc(rendered, func(match string) string {
			parts := authURLEncodeRe.FindStringSubmatch(match)
			if len(parts) != 2 {
				return ""
			}
			return url.QueryEscape(parts[1])
		})

		if rendered == prev {
			break
		}
	}
	return rendered
}

func randomHex(size int) string {
	if size <= 0 {
		size = 16
	}
	buf := make([]byte, (size+1)/2)
	if _, err := rand.Read(buf); err != nil {
		return strings.Repeat("0", size)
	}
	return hex.EncodeToString(buf)[:size]
}

func randomUUID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "00000000-0000-0000-0000-000000000000"
	}

	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		buf[0:4],
		buf[4:6],
		buf[6:8],
		buf[8:10],
		buf[10:16],
	)
}

func applyAuthVariables(vars map[string]string, items []AuthVariable) {
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		vars[name] = renderAuthTemplate(item.Value, vars)
	}
}

func renderAuthHeaders(headers []AuthHeader, vars map[string]string) []AuthHeader {
	rendered := make([]AuthHeader, 0, len(headers))
	for _, header := range headers {
		name := strings.TrimSpace(renderAuthTemplate(header.Name, vars))
		if name == "" {
			continue
		}
		rendered = append(rendered, AuthHeader{
			Name:  name,
			Value: renderAuthTemplate(header.Value, vars),
		})
	}
	return rendered
}

func appendCookieSession(vars map[string]string, resp *http.Response) {
	if resp == nil {
		return
	}
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		return
	}

	parts := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		parts = append(parts, cookie.Name+"="+cookie.Value)
	}
	vars["cookie"] = strings.Join(parts, "; ")
}

func extractAuthResponseValues(resp *http.Response, body []byte, cfg *AuthRequestConfig, vars map[string]string) error {
	appendCookieSession(vars, resp)

	var parsed interface{}
	if len(body) > 0 {
		_ = json.Unmarshal(body, &parsed)
	}

	extracts := cfg.Extracts
	if tokenField := strings.TrimSpace(cfg.TokenField); tokenField != "" {
		extracts = append(extracts, AuthExtract{Name: "token", Source: "json", Path: tokenField})
	}

	for _, extract := range extracts {
		name := strings.TrimSpace(extract.Name)
		if name == "" {
			continue
		}

		source := strings.ToLower(strings.TrimSpace(extract.Source))
		if source == "" {
			source = "json"
		}

		switch source {
		case "header":
			headerName := strings.TrimSpace(extract.Header)
			if headerName == "" {
				headerName = strings.TrimSpace(extract.Path)
			}
			if headerName == "" || resp == nil {
				continue
			}
			if value := strings.TrimSpace(resp.Header.Get(headerName)); value != "" {
				vars[name] = value
			}
		default:
			if parsed == nil {
				continue
			}
			if value := extractTokenFromJSON(parsed, strings.TrimSpace(extract.Path)); value != "" {
				vars[name] = value
			}
		}
	}

	if _, exists := vars["token"]; !exists || vars["token"] == "" {
		if parsed != nil {
			tokenField := strings.TrimSpace(cfg.TokenField)
			if tokenField == "" {
				tokenField = "token"
			}
			if value := extractTokenFromJSON(parsed, tokenField); value != "" {
				vars["token"] = value
			}
		}
	}

	return nil
}

func executeAuthRequest(targetURL string, config *WebScanConfig, reqCfg *AuthRequestConfig, vars map[string]string) error {
	if reqCfg == nil {
		return nil
	}

	loginURL := strings.TrimSpace(renderAuthTemplate(reqCfg.URL, vars))
	if loginURL == "" {
		loginURL = resolveLoginFormURL(targetURL, config)
	}
	if loginURL == "" {
		return fmt.Errorf("login URL is required")
	}

	method := strings.ToUpper(strings.TrimSpace(reqCfg.Method))
	if method == "" {
		method = strings.ToUpper(strings.TrimSpace(config.LoginMethod))
	}
	if method == "" {
		method = http.MethodPost
	}

	contentType := strings.ToLower(strings.TrimSpace(reqCfg.ContentType))
	if contentType == "" {
		contentType = strings.ToLower(strings.TrimSpace(config.LoginContentType))
	}
	if contentType == "" {
		contentType = "form"
	}

	bodyValues := make(map[string]string, len(reqCfg.Body))
	for key, value := range reqCfg.Body {
		bodyValues[key] = renderAuthTemplate(value, vars)
	}
	if len(bodyValues) == 0 {
		usernameField := strings.TrimSpace(config.UsernameField)
		if usernameField == "" {
			usernameField = "username"
		}
		passwordField := strings.TrimSpace(config.PasswordField)
		if passwordField == "" {
			passwordField = "password"
		}
		bodyValues[usernameField] = vars["username"]
		bodyValues[passwordField] = vars["password"]
	}

	var body io.Reader
	requestContentType := "application/x-www-form-urlencoded"
	if contentType == "json" {
		payload, err := json.Marshal(bodyValues)
		if err != nil {
			return err
		}
		body = strings.NewReader(string(payload))
		requestContentType = "application/json"
	} else {
		form := url.Values{}
		for key, value := range bodyValues {
			form.Set(key, value)
		}
		body = strings.NewReader(form.Encode())
	}

	req, err := http.NewRequest(method, loginURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", requestContentType)
	req.Header.Set("User-Agent", "ops-platform-web-scan/1.0")

	for _, header := range renderAuthHeaders(reqCfg.Headers, vars) {
		req.Header.Set(header.Name, header.Value)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		bodySnippet := strings.TrimSpace(string(respBody))
		if len(bodySnippet) > 200 {
			bodySnippet = bodySnippet[:200]
		}
		if bodySnippet != "" {
			return fmt.Errorf("login request returned status %d: %s", resp.StatusCode, bodySnippet)
		}
		return fmt.Errorf("login request returned status %d", resp.StatusCode)
	}

	return extractAuthResponseValues(resp, respBody, reqCfg, vars)
}

func resolveAdvancedAuth(targetURL string, config *WebScanConfig) (*ResolvedAuth, error) {
	if config == nil || config.AuthFlow == nil {
		return &ResolvedAuth{}, nil
	}

	vars := buildBaseAuthVariables(targetURL, config)
	applyAuthVariables(vars, config.AuthFlow.Variables)

	if err := executeAuthRequest(targetURL, config, config.AuthFlow.Login, vars); err != nil {
		return nil, err
	}

	if _, err := applySignerVars(vars, config.AuthFlow.Signer); err != nil {
		return nil, err
	}

	headers := renderAuthHeaders(config.AuthFlow.SessionHeaders, vars)
	if signerHeaders, err := renderSignerHeaders(vars, config.AuthFlow.Signer); err != nil {
		return nil, err
	} else if len(signerHeaders) > 0 {
		headers = append(headers, signerHeaders...)
	}
	if len(headers) == 0 && strings.TrimSpace(vars["token"]) != "" {
		headerName := strings.TrimSpace(config.AuthHeader)
		if headerName == "" {
			headerName = "token"
		}
		headers = append(headers, AuthHeader{
			Name:  headerName,
			Value: vars["token"],
		})
	}

	return &ResolvedAuth{
		Headers: headers,
		Session: vars,
	}, nil
}
