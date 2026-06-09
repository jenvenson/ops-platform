package security

import (
	"fmt"
	"strings"

	"github.com/jenvenson/ops-platform/internal/models"
	"gorm.io/gorm"
)

const (
	hostManualConfirmedFindingSource = "host-manual-confirmed"
	manualReviewMatchMode            = "manual-review"
	manualReviewScanner              = "manual-review"
	manualReviewScanMethod           = "人工确认"
)

func isManualConfirmedFinding(vulnerability models.SecurityVulnerability) bool {
	return inferFindingSource(&vulnerability) == hostManualConfirmedFindingSource
}

func buildConfirmedVulnerabilityFromCandidate(candidate models.SecurityVulnerability) models.SecurityVulnerability {
	var firstTaskID *uint
	if candidate.FirstTaskID != nil {
		value := *candidate.FirstTaskID
		firstTaskID = &value
	}

	var lastTaskID *uint
	if candidate.LastTaskID != nil {
		value := *candidate.LastTaskID
		lastTaskID = &value
	}

	candidateID := candidate.ID
	matchedOn := strings.TrimSpace(candidate.MatchedOn)
	if matchedOn == "" {
		matchedOn = fmt.Sprintf("候选记录 #%d 经人工复核确认", candidate.ID)
	}

	return models.SecurityVulnerability{
		TaskID:        candidate.TaskID,
		FirstTaskID:   firstTaskID,
		LastTaskID:    lastTaskID,
		AssetID:       candidate.AssetID,
		IP:            candidate.IP,
		Port:          candidate.Port,
		Protocol:      candidate.Protocol,
		Severity:      candidate.Severity,
		CVSSScore:     candidate.CVSSScore,
		CVSSVector:    candidate.CVSSVector,
		CVEID:         candidate.CVEID,
		CNVDID:        candidate.CNVDID,
		CNNVDID:       candidate.CNNVDID,
		CNCVEID:       candidate.CNCVEID,
		Title:         candidate.Title,
		Description:   candidate.Description,
		VulnType:      candidate.VulnType,
		Solution:      candidate.Solution,
		MatchedOn:     matchedOn,
		ExploitPrereq: candidate.ExploitPrereq,
		Scanner:       manualReviewScanner,
		ScanMethod:    manualReviewScanMethod,
		VulnURL:       candidate.VulnURL,
		FindingSource: hostManualConfirmedFindingSource,
		FindingFamily: "vulnerability",
		Confidence:    "high",
		PrimaryCVEID:  candidate.PrimaryCVEID,
		VulnDBID:      candidate.VulnDBID,
		MatchMode:     manualReviewMatchMode,
		SourceVulnID:  &candidateID,
		Payload:       candidate.Payload,
		Request:       candidate.Request,
		Response:      candidate.Response,
		ReferenceURL:  candidate.ReferenceURL,
		Status:        candidate.Status,
		Priority:      candidate.Priority,
		FalsePositive: candidate.FalsePositive,
	}
}

func findConfirmedVulnerabilityForCandidate(tx *gorm.DB, candidate models.SecurityVulnerability) (*models.SecurityVulnerability, error) {
	if tx == nil {
		return nil, nil
	}

	var derived models.SecurityVulnerability
	switch {
	case candidate.ConfirmedVulnID != nil && *candidate.ConfirmedVulnID > 0:
		err := tx.Where("id = ? AND source_vuln_id = ? AND finding_source = ?", *candidate.ConfirmedVulnID, candidate.ID, hostManualConfirmedFindingSource).First(&derived).Error
		if err == nil {
			return &derived, nil
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	err := tx.Where("source_vuln_id = ? AND finding_source = ?", candidate.ID, hostManualConfirmedFindingSource).First(&derived).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &derived, nil
}

func upsertConfirmedVulnerabilityForCandidate(tx *gorm.DB, candidate *models.SecurityVulnerability) (*models.SecurityVulnerability, error) {
	if tx == nil || candidate == nil {
		return nil, nil
	}

	derivedPayload := buildConfirmedVulnerabilityFromCandidate(*candidate)
	existing, err := findConfirmedVulnerabilityForCandidate(tx, *candidate)
	if err != nil {
		return nil, err
	}

	if existing == nil {
		if err := tx.Create(&derivedPayload).Error; err != nil {
			return nil, err
		}
		candidate.ConfirmedVulnID = &derivedPayload.ID
		if err := tx.Model(&models.SecurityVulnerability{}).
			Where("id = ?", candidate.ID).
			Update("confirmed_vuln_id", derivedPayload.ID).Error; err != nil {
			return nil, err
		}
		return &derivedPayload, nil
	}

	derivedPayload.ID = existing.ID
	if err := tx.Model(&models.SecurityVulnerability{}).
		Where("id = ?", existing.ID).
		Updates(map[string]interface{}{
			"task_id":        derivedPayload.TaskID,
			"first_task_id":  derivedPayload.FirstTaskID,
			"last_task_id":   derivedPayload.LastTaskID,
			"asset_id":       derivedPayload.AssetID,
			"ip":             derivedPayload.IP,
			"port":           derivedPayload.Port,
			"protocol":       derivedPayload.Protocol,
			"severity":       derivedPayload.Severity,
			"cvss_score":     derivedPayload.CVSSScore,
			"cvss_vector":    derivedPayload.CVSSVector,
			"cve_id":         derivedPayload.CVEID,
			"cnvd_id":        derivedPayload.CNVDID,
			"cnnvd_id":       derivedPayload.CNNVDID,
			"cncve_id":       derivedPayload.CNCVEID,
			"title":          derivedPayload.Title,
			"description":    derivedPayload.Description,
			"vuln_type":      derivedPayload.VulnType,
			"solution":       derivedPayload.Solution,
			"matched_on":     derivedPayload.MatchedOn,
			"exploit_prereq": derivedPayload.ExploitPrereq,
			"scanner":        derivedPayload.Scanner,
			"scan_method":    derivedPayload.ScanMethod,
			"vuln_url":       derivedPayload.VulnURL,
			"finding_source": derivedPayload.FindingSource,
			"finding_family": derivedPayload.FindingFamily,
			"confidence":     derivedPayload.Confidence,
			"primary_cve_id": derivedPayload.PrimaryCVEID,
			"vuln_db_id":     derivedPayload.VulnDBID,
			"match_mode":     derivedPayload.MatchMode,
			"source_vuln_id": derivedPayload.SourceVulnID,
			"payload":        derivedPayload.Payload,
			"request":        derivedPayload.Request,
			"response":       derivedPayload.Response,
			"reference_url":  derivedPayload.ReferenceURL,
			"status":         derivedPayload.Status,
			"priority":       derivedPayload.Priority,
			"false_positive": derivedPayload.FalsePositive,
		}).Error; err != nil {
		return nil, err
	}

	candidate.ConfirmedVulnID = &existing.ID
	if err := tx.Model(&models.SecurityVulnerability{}).
		Where("id = ?", candidate.ID).
		Update("confirmed_vuln_id", existing.ID).Error; err != nil {
		return nil, err
	}

	existing = &derivedPayload
	return existing, nil
}

func removeConfirmedVulnerabilityForCandidate(tx *gorm.DB, candidate *models.SecurityVulnerability) error {
	if tx == nil || candidate == nil {
		return nil
	}

	var derivedIDs []uint
	if candidate.ConfirmedVulnID != nil && *candidate.ConfirmedVulnID > 0 {
		var confirmedIDs []uint
		err := tx.Model(&models.SecurityVulnerability{}).
			Where("id = ? AND source_vuln_id = ? AND finding_source = ?", *candidate.ConfirmedVulnID, candidate.ID, hostManualConfirmedFindingSource).
			Pluck("id", &confirmedIDs).Error
		if err != nil {
			return err
		}
		derivedIDs = append(derivedIDs, confirmedIDs...)
	}

	var linkedIDs []uint
	if err := tx.Model(&models.SecurityVulnerability{}).
		Where("source_vuln_id = ? AND finding_source = ?", candidate.ID, hostManualConfirmedFindingSource).
		Pluck("id", &linkedIDs).Error; err != nil {
		return err
	}
	derivedIDs = append(derivedIDs, linkedIDs...)
	derivedIDs = uniqueUintValues(derivedIDs)

	if len(derivedIDs) > 0 {
		if err := tx.Where("vuln_id IN ?", derivedIDs).Delete(&models.VulnTicket{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", derivedIDs).Delete(&models.SecurityVulnerability{}).Error; err != nil {
			return err
		}
	}

	candidate.ConfirmedVulnID = nil
	return tx.Model(&models.SecurityVulnerability{}).
		Where("id = ?", candidate.ID).
		Update("confirmed_vuln_id", nil).Error
}

func uniqueUintValues(values []uint) []uint {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[uint]struct{}, len(values))
	result := make([]uint, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
