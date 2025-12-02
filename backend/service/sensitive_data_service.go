package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	v1 "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/service/sqlparse"
	"github.com/bytebase/bytebase/common/log"
)

// SensitiveDataService is the service for sensitive data detection and matching.
type SensitiveDataService interface {
	// DetectSensitiveData detects sensitive data in a database change
	DetectSensitiveData(ctx context.Context, change *DatabaseChange) ([]*SensitiveDataMatch, error)
	// GetMatchingSensitiveLevels finds sensitive levels that match a field
	GetMatchingSensitiveLevels(ctx context.Context, fieldName, fieldType string) ([]*v1.SensitiveLevel, error)
	// GetMaxSensitiveSeverity gets the maximum sensitive severity from matches
	GetMaxSensitiveSeverity(matches []*SensitiveDataMatch) v1.SensitiveLevel_Severity
}

// SensitiveDataMatch represents a matched sensitive data field in SQL.
type SensitiveDataMatch struct {
	FieldName       string
	FieldType       string
	SensitiveLevel  *v1.SensitiveLevel
	SQLStatement    string
	Database        string
	Schema          string
	Table           string
}

// sensitiveDataServiceImpl is the implementation of SensitiveDataService.
type sensitiveDataServiceImpl struct {
	sensitiveApprovalService v1.SensitiveApprovalServiceClient
}

// NewSensitiveDataService creates a new SensitiveDataService.
func NewSensitiveDataService(sensitiveApprovalService v1.SensitiveApprovalServiceClient) SensitiveDataService {
	return &sensitiveDataServiceImpl{
		sensitiveApprovalService: sensitiveApprovalService,
	}
}

// DetectSensitiveData detects sensitive data in a database change.
func (s *sensitiveDataServiceImpl) DetectSensitiveData(ctx context.Context, change *DatabaseChange) ([]*SensitiveDataMatch, error) {
	log.Infof("Detecting sensitive data in change %s: %s", change.ChangeID, change.SQL)

	// Parse SQL to get fields involved
	parsedSQL, err := sqlparse.Parse(change.SQL)
	if err != nil {
		log.Errorf("Failed to parse SQL: %v", err)
		return nil, err
	}

	// Extract tables and fields from parsed SQL
	tables, err := sqlparse.ExtractTables(parsedSQL)
	if err != nil {
		log.Errorf("Failed to extract tables from SQL: %v", err)
		return nil, err
	}

	var matches []*SensitiveDataMatch

	for _, table := range tables {
		fields, err := sqlparse.ExtractFieldsFromTable(parsedSQL, table)
		if err != nil {
			log.Errorf("Failed to extract fields from table %s: %v", table, err)
			continue
		}

		for _, field := range fields {
			// Get matching sensitive levels for this field
			sensitiveLevels, err := s.GetMatchingSensitiveLevels(ctx, field.Name, field.Type)
			if err != nil {
				log.Errorf("Failed to get sensitive levels for field %s: %v", field.Name, err)
				continue
			}

			for _, level := range sensitiveLevels {
				match := &SensitiveDataMatch{
					FieldName:      field.Name,
					FieldType:      field.Type,
					SensitiveLevel: level,
					SQLStatement:   change.SQL,
					Database:       change.Database,
					Schema:         change.Schema,
					Table:          change.Table,
				}
				matches = append(matches, match)
				log.Infof("Found sensitive data match: Field=%s, Type=%s, SensitiveLevel=%s", field.Name, field.Type, level.DisplayName)
			}
		}
	}

	return matches, nil
}

// GetMaxSensitiveSeverity gets the maximum sensitive severity from matches.
func (s *sensitiveDataServiceImpl) GetMaxSensitiveSeverity(matches []*SensitiveDataMatch) v1.SensitiveLevel_Severity {
	maxSeverity := v1.SensitiveLevel_SEVERITY_UNSPECIFIED

	for _, match := range matches {
		if match.SensitiveLevel == nil {
			continue
		}

		if match.SensitiveLevel.Severity > maxSeverity {
			maxSeverity = match.SensitiveLevel.Severity
		}
	}

	return maxSeverity
}

// GetMatchingSensitiveLevels finds sensitive levels that match a field.
func (s *sensitiveDataServiceImpl) GetMatchingSensitiveLevels(ctx context.Context, fieldName, fieldType string) ([]*v1.SensitiveLevel, error) {
	// Get all sensitive levels
	req := &v1.ListSensitiveLevelsRequest{}
	resp, err := s.sensitiveApprovalService.ListSensitiveLevels(ctx, req)
	if err != nil {
		log.Errorf("Failed to list sensitive levels: %v", err)
		return nil, err
	}

	var matchingLevels []*v1.SensitiveLevel

	for _, level := range resp.SensitiveLevels {
		// Check if any field match rule matches
		if s.matchesField(level, fieldName, fieldType) {
			matchingLevels = append(matchingLevels, level)
		}
	}

	return matchingLevels, nil
}

// matchesField checks if a sensitive level's rules match a field.
func (s *sensitiveDataServiceImpl) matchesField(level *v1.SensitiveLevel, fieldName, fieldType string) bool {
	// If no field match rules, consider it as not matching
	if len(level.FieldMatchRules) == 0 {
		return false
	}

	for _, rule := range level.FieldMatchRules {
		if s.matchesRule(rule, fieldName, fieldType) {
			return true
		}
	}

	return false
}

// matchesRule checks if a field match rule matches a field.
func (s *sensitiveDataServiceImpl) matchesRule(rule *v1.FieldMatchRule, fieldName, fieldType string) bool {
	// Check field name
	if rule.FieldNameRegex != "" {
		matched, err := regexp.MatchString(rule.FieldNameRegex, fieldName)
		if err != nil {
			log.Errorf("Failed to match field name regex %s: %v", rule.FieldNameRegex, err)
			return false
		}
		if !matched {
			return false
		}
	}

	// Check field type
	if rule.FieldType != "" {
		if rule.FieldType != fieldType {
			return false
		}
	}

	// Check exact value
	if rule.ExactValue != "" {
		// This is for data value matching, not field name/type
		// For field detection, this might not be applicable
		// But we can check if it's a fixed value field
	}

	// All conditions matched
	return true
}

// GetMaxSensitiveSeverity gets the maximum sensitive severity from matches.
func GetMaxSensitiveSeverity(matches []*SensitiveDataMatch) v1.SensitiveLevel_Severity {
	maxSeverity := v1.SensitiveLevel_SEVERITY_UNSPECIFIED

	for _, match := range matches {
		if match.SensitiveLevel == nil {
			continue
		}

		if match.SensitiveLevel.Severity > maxSeverity {
			maxSeverity = match.SensitiveLevel.Severity
		}
	}

	return maxSeverity
}

// NeedsApproval checks if a sensitive level requires approval.
func NeedsApproval(severity v1.SensitiveLevel_Severity) bool {
	return severity == v1.SensitiveLevel_SEVERITY_HIGH || severity == v1.SensitiveLevel_SEVERITY_MEDIUM
}
