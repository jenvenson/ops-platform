package security

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/edy/ops-platform/internal/database"
	"github.com/edy/ops-platform/internal/models"
	"gorm.io/gorm"
)

type webScanPhase1Context struct {
	taskID uint
	runID  uint
}

func ensureWebScanPhase1Context(tx *gorm.DB, taskID uint) (*webScanPhase1Context, error) {
	support := getScanPhase1SchemaSupport(tx)
	if !support.foundationTablesReady() {
		return nil, nil
	}

	runID, err := loadCurrentRunIDForTask(tx, taskID)
	if err != nil || runID == 0 {
		return nil, err
	}

	return &webScanPhase1Context{taskID: taskID, runID: runID}, nil
}

func phase1WebTargetKey(kind string, rawURL string) string {
	return strings.Join([]string{
		firstNonEmpty(strings.TrimSpace(kind), "url"),
		sanitizeScanURL(rawURL),
	}, "|")
}

func phase1WebFindingKey(vuln models.SecurityVulnerability, targetURL string) string {
	identifier := firstNonEmpty(
		strings.TrimSpace(vuln.PrimaryCVEID),
		strings.TrimSpace(vuln.CVEID),
		strings.TrimSpace(vuln.TemplateID),
		strings.TrimSpace(vuln.Title),
	)
	return strings.Join([]string{
		firstNonEmpty(strings.TrimSpace(vuln.FindingSource), "legacy"),
		firstNonEmpty(normalizeScanHost(vuln.IP), strings.TrimSpace(vuln.IP)),
		strconv.Itoa(vuln.Port),
		identifier,
		firstNonEmpty(strings.TrimSpace(vuln.MatchMode), "unknown"),
		sanitizeScanURL(targetURL),
	}, "|")
}

func phase1WebTargetKind(kind string, rawURL string) string {
	switch strings.TrimSpace(kind) {
	case "url", "page", "api", "form", "auth":
		return strings.TrimSpace(kind)
	}

	targetURL := strings.ToLower(strings.TrimSpace(rawURL))
	switch {
	case strings.Contains(targetURL, "/auth/"), strings.Contains(targetURL, "/login"), strings.Contains(targetURL, "/signin"):
		return "auth"
	case strings.Contains(targetURL, "/api/"), strings.Contains(targetURL, "/base/"):
		return "api"
	default:
		return "page"
	}
}

func phase1WebTargetLookupKinds(rawURL string) []string {
	kinds := []string{phase1WebTargetKind("", rawURL), "page", "api", "form", "auth", "url"}
	seen := make(map[string]struct{}, len(kinds))
	result := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		kind = strings.TrimSpace(kind)
		if kind == "" {
			continue
		}
		if _, exists := seen[kind]; exists {
			continue
		}
		seen[kind] = struct{}{}
		result = append(result, kind)
	}
	return result
}

func findPhase1WebTargetByKind(tx *gorm.DB, ctx *webScanPhase1Context, rawURL string, kind string) (*models.SecurityScanTarget, error) {
	if tx == nil || ctx == nil {
		return nil, nil
	}

	var target models.SecurityScanTarget
	err := tx.Where("run_id = ? AND normalized_target = ?", ctx.runID, phase1WebTargetKey(kind, sanitizeScanURL(rawURL))).First(&target).Error
	if err == nil {
		return &target, nil
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return nil, err
}

func phase1WebEvidenceKind(source string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "browser":
		return "browser-discovery", "browser"
	default:
		return "http-discovery", "http"
	}
}

func phase1WebSourceEngine(item DiscoveredTarget) string {
	source := strings.ToLower(strings.TrimSpace(item.Source))
	switch source {
	case "browser":
		return "browser"
	case "entry", "html", "script", "/robots.txt", "/sitemap.xml":
		return "http"
	default:
		if strings.Contains(source, "browser") {
			return "browser"
		}
		return "http"
	}
}

func phase1WebAuthSnapshot(entryURL string, config *WebScanConfig) map[string]interface{} {
	if config == nil {
		return nil
	}
	authMode := strings.TrimSpace(config.AuthMode)
	if authMode == "" || strings.EqualFold(authMode, "none") {
		return nil
	}

	snapshot := map[string]interface{}{
		"auth_mode":      authMode,
		"auth_header":    strings.TrimSpace(config.AuthHeader),
		"has_credential": strings.TrimSpace(config.Credential) != "",
	}
	if loginURL := resolveLoginFormURL(entryURL, config); loginURL != "" {
		snapshot["login_url"] = loginURL
	}
	if value := strings.TrimSpace(config.LoginMethod); value != "" {
		snapshot["login_method"] = value
	}
	if value := strings.TrimSpace(config.LoginContentType); value != "" {
		snapshot["login_content_type"] = value
	}
	if value := strings.TrimSpace(config.Username); value != "" {
		snapshot["username"] = value
	}
	if value := strings.TrimSpace(config.UsernameField); value != "" {
		snapshot["username_field"] = value
	}
	if value := strings.TrimSpace(config.PasswordField); value != "" {
		snapshot["password_field"] = value
	}
	if value := strings.TrimSpace(config.TokenField); value != "" {
		snapshot["token_field"] = value
	}
	if config.AuthFlow != nil {
		snapshot["auth_flow"] = map[string]interface{}{
			"variables":       len(config.AuthFlow.Variables),
			"has_login":       config.AuthFlow.Login != nil,
			"has_signer":      config.AuthFlow.Signer != nil,
			"session_headers": len(config.AuthFlow.SessionHeaders),
		}
	}
	return snapshot
}

func phase1WebConfigSnapshot(inputTarget string, entryURLs []string, config *WebScanConfig) *string {
	discoveryMode := "http"
	scanProfile := "standard"
	discoveryMaxDepth := 1
	discoveryMaxURLs := 25
	options := []string{}
	authSnapshot := map[string]interface{}(nil)
	if config != nil {
		scanProfile = normalizeWebScanProfile(config.ScanProfile)
		if value := strings.TrimSpace(config.DiscoveryMode); value != "" {
			discoveryMode = value
		}
		if config.DiscoveryMaxDepth > 0 {
			discoveryMaxDepth = config.DiscoveryMaxDepth
		}
		if config.DiscoveryMaxURLs > 0 {
			discoveryMaxURLs = config.DiscoveryMaxURLs
		}
		options = append(options, config.Options...)
		if len(entryURLs) > 0 {
			authSnapshot = phase1WebAuthSnapshot(entryURLs[0], config)
		}
	}

	payload := map[string]interface{}{
		"scan_type":           "web",
		"scan_profile":        scanProfile,
		"target_type":         "url",
		"input_target":        inputTarget,
		"entry_urls":          entryURLs,
		"entry_count":         len(entryURLs),
		"options":             options,
		"discovery_mode":      discoveryMode,
		"discovery_max_depth": discoveryMaxDepth,
		"discovery_max_urls":  discoveryMaxURLs,
	}
	if authSnapshot != nil {
		payload["auth"] = authSnapshot
	}
	return phase1JSONString(payload)
}

func phase1WebTargetSnapshot(entryURLs []string, plan []DiscoveredTarget) *string {
	return phase1WebTargetSnapshotWithMeta(entryURLs, plan, len(plan), 0, 0, nil)
}

func phase1WebTargetSnapshotWithMeta(entryURLs []string, plan []DiscoveredTarget, discoveredCount int, skippedTargets int, ruleOnlyTargets int, warnings []map[string]interface{}) *string {
	kindCounts := map[string]int{"url": len(entryURLs)}
	for _, item := range plan {
		kind := phase1WebTargetKind(item.Kind, item.URL)
		kindCounts[kind]++
	}
	if discoveredCount < len(plan) {
		discoveredCount = len(plan)
	}

	payload := map[string]interface{}{
		"scan_type":                 "web",
		"entry_urls":                entryURLs,
		"entry_count":               len(entryURLs),
		"discovered_count":          discoveredCount,
		"verification_target_count": len(plan),
		"skipped_target_count":      skippedTargets,
		"rule_only_target_count":    ruleOnlyTargets,
		"kind_counts":               kindCounts,
	}
	if len(warnings) > 0 {
		payload["discovery_warnings"] = warnings
	}
	return phase1JSONString(payload)
}

func phase1WebDiscoveryWarning(entryURL string, requestedMode string, actualMode string, err error) map[string]interface{} {
	payload := map[string]interface{}{
		"entry_url":      sanitizeScanURL(entryURL),
		"requested_mode": firstNonEmpty(strings.TrimSpace(requestedMode), "http"),
		"effective_mode": firstNonEmpty(strings.TrimSpace(actualMode), "http"),
		"warning_code":   "browser_discovery_fallback",
	}
	if err != nil {
		payload["reason"] = phase1TrimText(strings.TrimSpace(err.Error()), 1024)
	}
	return payload
}

func phase1WebCompletionMessage(discoveredCount int, scannedTargets int, skippedTargets int, ruleOnlyTargets int, highRisk int, mediumRisk int, lowRisk int, warnings []map[string]interface{}) string {
	message := fmt.Sprintf("扫描完成，发现 %d 个入口，扫描 %d 个目标，命中 %d 个高危，%d 个中危，%d 个低危漏洞", discoveredCount, scannedTargets, highRisk, mediumRisk, lowRisk)
	if skippedTargets > 0 {
		message += fmt.Sprintf("，跳过 %d 个低优先级目标", skippedTargets)
	}
	if ruleOnlyTargets > 0 {
		message += fmt.Sprintf("，其中 %d 个低价值目标仅执行规则检测", ruleOnlyTargets)
	}
	if len(warnings) > 0 {
		message += fmt.Sprintf("；%d 个入口因 browser helper 不可达回退为 HTTP 发现", len(warnings))
	}
	return message
}

func phase1WebSummarySnapshot(entryURLs []string, discoveredCount int, scannedTargets int, skippedTargets int, ruleOnlyTargets int, requestedMode string, warnings []map[string]interface{}, highRisk int, mediumRisk int, lowRisk int, completedAt time.Time) *string {
	payload := map[string]interface{}{
		"scan_type":              "web",
		"entry_urls":             entryURLs,
		"entry_count":            len(entryURLs),
		"discovered_count":       discoveredCount,
		"scanned_targets":        scannedTargets,
		"skipped_target_count":   skippedTargets,
		"rule_only_target_count": ruleOnlyTargets,
		"discovery_mode":         firstNonEmpty(strings.TrimSpace(requestedMode), "http"),
		"browser_fallback_count": len(warnings),
		"high_risk":              highRisk,
		"medium_risk":            mediumRisk,
		"low_risk":               lowRisk,
		"completed_at":           completedAt,
	}
	if len(warnings) > 0 {
		payload["discovery_warnings"] = warnings
	}
	return phase1JSONString(payload)
}

func buildPhase1WebTarget(taskID uint, runID uint, rawURL string, kind string, parentTargetID *uint, discoverySource string, status string, metadata interface{}, startedAt *time.Time, completedAt *time.Time) (models.SecurityScanTarget, error) {
	sanitized := sanitizeScanURL(rawURL)
	parsed, err := url.Parse(sanitized)
	if err != nil {
		return models.SecurityScanTarget{}, err
	}

	targetKind := phase1WebTargetKind(kind, sanitized)
	host := normalizeScanHost(parsed.Hostname())
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	path := parsed.EscapedPath()
	if path == "" {
		path = "/"
	}
	if parsed.RawQuery != "" {
		path += "?" + parsed.RawQuery
	}

	var port *int
	portValue := inferPortFromScheme(scheme)
	if parsed.Port() != "" {
		if parsedPort, parseErr := strconv.Atoi(parsed.Port()); parseErr == nil {
			portValue = parsedPort
		}
	}
	if portValue > 0 {
		port = &portValue
	}

	return models.SecurityScanTarget{
		RunID:            runID,
		TaskID:           taskID,
		ParentTargetID:   parentTargetID,
		TargetKind:       targetKind,
		NormalizedTarget: phase1WebTargetKey(targetKind, sanitized),
		Host:             host,
		Port:             port,
		Scheme:           scheme,
		Path:             path,
		Status:           firstNonEmpty(status, "completed"),
		DiscoverySource:  firstNonEmpty(discoverySource, "http"),
		StartedAt:        startedAt,
		CompletedAt:      completedAt,
		MetadataJSON:     phase1JSONString(metadata),
	}, nil
}

func findPhase1WebTargetByURL(tx *gorm.DB, ctx *webScanPhase1Context, rawURL string) (*models.SecurityScanTarget, error) {
	if tx == nil || ctx == nil {
		return nil, nil
	}

	sanitized := sanitizeScanURL(rawURL)
	for _, kind := range phase1WebTargetLookupKinds(sanitized) {
		var target models.SecurityScanTarget
		err := tx.Where("run_id = ? AND normalized_target = ?", ctx.runID, phase1WebTargetKey(kind, sanitized)).First(&target).Error
		if err == nil {
			return &target, nil
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}
	return nil, nil
}

func ensurePhase1WebTarget(tx *gorm.DB, ctx *webScanPhase1Context, taskID uint, rawURL string, kind string, parentTargetID *uint, discoverySource string, metadata interface{}) (*models.SecurityScanTarget, error) {
	if ctx == nil {
		return nil, nil
	}

	kind = strings.TrimSpace(kind)
	if kind != "" {
		if existing, err := findPhase1WebTargetByKind(tx, ctx, rawURL, phase1WebTargetKind(kind, rawURL)); err != nil {
			return nil, err
		} else if existing != nil {
			return existing, nil
		}
	} else {
		if existing, err := findPhase1WebTargetByURL(tx, ctx, rawURL); err != nil {
			return nil, err
		} else if existing != nil {
			return existing, nil
		}
	}

	now := time.Now()
	target, err := buildPhase1WebTarget(taskID, ctx.runID, rawURL, kind, parentTargetID, discoverySource, "completed", metadata, &now, &now)
	if err != nil {
		return nil, err
	}
	return upsertPhase1Target(tx, target)
}

func recordWebDiscoveryPhase1(taskID uint, entryURL string, items []DiscoveredTarget, config *WebScanConfig) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		ctx, err := ensureWebScanPhase1Context(tx, taskID)
		if err != nil || ctx == nil {
			return err
		}

		entryMetadata := map[string]interface{}{
			"entry_url": sanitizeScanURL(entryURL),
		}
		if authSnapshot := phase1WebAuthSnapshot(entryURL, config); authSnapshot != nil {
			entryMetadata["auth"] = authSnapshot
		}

		entryTarget, err := ensurePhase1WebTarget(tx, ctx, taskID, entryURL, "url", nil, "manual", entryMetadata)
		if err != nil {
			return err
		}

		authSnapshot := phase1WebAuthSnapshot(entryURL, config)
		if authSnapshot != nil {
			authURL := firstNonEmpty(resolveLoginFormURL(entryURL, config), entryURL)
			authTarget, err := ensurePhase1WebTarget(tx, ctx, taskID, authURL, "auth", &entryTarget.ID, "auth", authSnapshot)
			if err != nil {
				return err
			}

			_, err = upsertPhase1Evidence(tx, models.SecurityScanEvidence{
				RunID:          ctx.runID,
				TaskID:         taskID,
				TargetID:       &authTarget.ID,
				EvidenceType:   "auth-login",
				SourceEngine:   "auth",
				Digest:         phase1Digest("auth-login", sanitizeScanURL(entryURL), sanitizeScanURL(authURL), strings.TrimSpace(config.AuthMode)),
				PayloadExcerpt: phase1TrimText(firstNonEmpty(strings.TrimSpace(config.AuthMode), authURL), 2048),
				MetadataJSON:   phase1JSONString(authSnapshot),
				RawJSON:        phase1JSONString(authSnapshot),
			})
			if err != nil {
				return err
			}
		}

		for _, item := range items {
			if strings.TrimSpace(item.URL) == "" {
				continue
			}
			sourceEngine := phase1WebSourceEngine(item)
			evidenceType, fallbackEngine := phase1WebEvidenceKind(sourceEngine)
			if sourceEngine == "" {
				sourceEngine = fallbackEngine
			}
			metadata := map[string]interface{}{
				"entry_url": sanitizeScanURL(entryURL),
				"url":       sanitizeScanURL(item.URL),
				"kind":      phase1WebTargetKind(item.Kind, item.URL),
				"source":    strings.TrimSpace(item.Source),
				"depth":     item.Depth,
			}
			target, err := ensurePhase1WebTarget(tx, ctx, taskID, item.URL, item.Kind, &entryTarget.ID, sourceEngine, metadata)
			if err != nil {
				return err
			}
			if target == nil {
				continue
			}

			raw := phase1JSONString(item)
			_, err = upsertPhase1Evidence(tx, models.SecurityScanEvidence{
				RunID:          ctx.runID,
				TaskID:         taskID,
				TargetID:       &target.ID,
				EvidenceType:   evidenceType,
				SourceEngine:   sourceEngine,
				Digest:         phase1Digest("web-discovery", sanitizeScanURL(entryURL), sanitizeScanURL(item.URL), phase1WebTargetKind(item.Kind, item.URL), strings.TrimSpace(item.Source), strconv.Itoa(item.Depth)),
				PayloadExcerpt: phase1TrimText(sanitizeScanURL(item.URL), 2048),
				MetadataJSON:   phase1JSONString(metadata),
				RawJSON:        raw,
			})
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func phase1WebResultTargetURL(vuln models.SecurityVulnerability, result NucleiResult) string {
	if value := strings.TrimSpace(vuln.VulnURL); value != "" {
		return sanitizeScanURL(value)
	}
	if value := strings.TrimSpace(result.URL); value != "" {
		return sanitizeScanURL(value)
	}
	if value := strings.TrimSpace(result.MatchedAt); strings.HasPrefix(strings.ToLower(value), "http://") || strings.HasPrefix(strings.ToLower(value), "https://") {
		return sanitizeScanURL(value)
	}

	host := firstNonEmpty(strings.TrimSpace(vuln.IP), strings.TrimSpace(result.IP), normalizeScanHost(result.Host))
	if host == "" {
		return ""
	}
	port := vuln.Port
	if port == 0 {
		port = result.Port
	}
	scheme := "http"
	if port == 443 || port == 8443 {
		scheme = "https"
	}
	if port > 0 && port != inferPortFromScheme(scheme) {
		return scheme + "://" + host + ":" + strconv.Itoa(port)
	}
	return scheme + "://" + host
}

func recordWebFindingOccurrence(tx *gorm.DB, ctx *webScanPhase1Context, vuln models.SecurityVulnerability, result NucleiResult) error {
	if ctx == nil {
		return nil
	}

	targetURL := phase1WebResultTargetURL(vuln, result)
	var target *models.SecurityScanTarget
	var err error
	if strings.TrimSpace(targetURL) != "" {
		target, err = ensurePhase1WebTarget(tx, ctx, ctx.taskID, targetURL, "", nil, inferFindingSource(&vuln), map[string]interface{}{
			"url":            targetURL,
			"finding_source": inferFindingSource(&vuln),
			"template_id":    strings.TrimSpace(result.TemplateID),
		})
		if err != nil {
			return err
		}
	}

	var targetID *uint
	if target != nil {
		targetID = &target.ID
	}

	source := firstNonEmpty(strings.TrimSpace(vuln.FindingSource), inferFindingSource(&vuln))
	evidenceType := "nuclei-result"
	sourceEngine := "nuclei"
	if source == "web-rule" {
		evidenceType = "rule-match"
		sourceEngine = "rule"
	}

	evidenceMeta := phase1JSONString(map[string]interface{}{
		"matched_at": result.MatchedAt,
		"url":        targetURL,
		"host":       result.Host,
		"port":       result.Port,
		"template":   result.TemplateID,
		"severity":   result.Severity,
		"cves":       result.CVEs,
		"tags":       result.Info.Tags,
	})
	evidence, err := upsertPhase1Evidence(tx, models.SecurityScanEvidence{
		RunID:           ctx.runID,
		TaskID:          ctx.taskID,
		TargetID:        targetID,
		EvidenceType:    evidenceType,
		SourceEngine:    sourceEngine,
		Digest:          phase1Digest(evidenceType, targetURL, result.TemplateID, result.MatchedAt, vuln.PrimaryCVEID, vuln.Title),
		RequestExcerpt:  phase1TrimText(result.Request, 8192),
		ResponseExcerpt: phase1TrimText(result.Response, 8192),
		PayloadExcerpt:  phase1TrimText(firstNonEmpty(targetURL, result.MatchedAt), 2048),
		MetadataJSON:    evidenceMeta,
		RawJSON:         phase1JSONString(result),
	})
	if err != nil {
		return err
	}

	now := time.Now()
	legacyID := vuln.ID
	return upsertPhase1Occurrence(tx, models.SecurityScanFindingOccurrence{
		RunID:                 ctx.runID,
		TaskID:                ctx.taskID,
		TargetID:              targetID,
		LegacyVulnerabilityID: &legacyID,
		FindingKey:            phase1WebFindingKey(vuln, targetURL),
		FindingFamily:         vuln.FindingFamily,
		FindingSource:         vuln.FindingSource,
		Severity:              vuln.Severity,
		Confidence:            vuln.Confidence,
		MatchMode:             vuln.MatchMode,
		PrimaryCVEID:          vuln.PrimaryCVEID,
		VulnDBID:              vuln.VulnDBID,
		Title:                 vuln.Title,
		Status:                firstNonEmpty(strings.TrimSpace(vuln.Status), "open"),
		VerificationStatus:    phase1VerificationStatus(vuln),
		EvidenceCount:         1,
		FirstSeenAt:           &now,
		LastSeenAt:            &now,
		MetadataJSON: phase1JSONString(map[string]interface{}{
			"asset_id":       vuln.AssetID,
			"template_id":    vuln.TemplateID,
			"scanner":        vuln.Scanner,
			"scan_method":    vuln.ScanMethod,
			"vuln_url":       targetURL,
			"evidence_id":    evidence.ID,
			"source_vuln":    vuln.SourceVulnID,
			"confirmed_id":   vuln.ConfirmedVulnID,
			"legacy_find_id": vuln.ID,
		}),
	})
}
