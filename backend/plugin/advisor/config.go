package advisor

import (
	_ "embed"
	"encoding/json"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

//go:embed config/sql-review.sample.yaml
var sqlReviewSampleTemplateStr string

//go:embed config/sql-review.dev.yaml
var sqlReviewDevTemplateStr string

//go:embed config/sql-review.prod.yaml
var sqlReviewProdTemplateStr string

// SQLReviewTemplateID is the template id for SQL review rules.
type SQLReviewTemplateID string

// SQLReviewTemplateData is the API message for SQL review rule template.
type SQLReviewTemplateData struct {
	ID       SQLReviewTemplateID  `yaml:"id"`
	RuleList []*SQLReviewRuleData `yaml:"ruleList"`
}

// SQLReviewRuleData is the API message for SQL review rule update.
type SQLReviewRuleData struct {
	Type        SQLReviewRuleType      `yaml:"type"`
	Level       SQLReviewRuleLevel     `yaml:"level,omitempty"`
	Description string                 `yaml:"description"`
	Payload     map[string]interface{} `yaml:"payload"`
}

// SQLReviewConfigOverride is the API message for SQL review configuration override.
type SQLReviewConfigOverride struct {
	Template SQLReviewTemplateID  `yaml:"template"`
	RuleList []*SQLReviewRuleData `yaml:"ruleList"`
}

// MergeSQLReviewRules will merge the input YML config into default template.
func MergeSQLReviewRules(override *SQLReviewConfigOverride) ([]*SQLReviewRule, error) {
	templateList, err := parseSQLReviewTemplateList()
	if err != nil {
		return nil, err
	}

	template := findTemplate(templateList, override.Template)
	if template == nil {
		return nil, errors.Errorf("cannot find the template: %v", override.Template)
	}

	ruleUpdateMap := make(map[SQLReviewRuleType]*SQLReviewRuleData)
	for _, rule := range override.RuleList {
		ruleUpdateMap[rule.Type] = rule
	}

	var res []*SQLReviewRule

	for _, ruleTemplate := range template.RuleList {
		ruleUpdate := ruleUpdateMap[ruleTemplate.Type]
		rule, err := mergeRule(ruleTemplate, ruleUpdate)
		if err != nil {
			return nil, err
		}
		res = append(res, rule)
	}

	return res, nil
}

func parseSQLReviewTemplateList() ([]*SQLReviewTemplateData, error) {
	sampleTemplate := &SQLReviewTemplateData{}
	prodTemplate := &SQLReviewTemplateData{}
	devTemplate := &SQLReviewTemplateData{}

	if err := yaml.Unmarshal([]byte(sqlReviewSampleTemplateStr), sampleTemplate); err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal([]byte(sqlReviewProdTemplateStr), prodTemplate); err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal([]byte(sqlReviewDevTemplateStr), devTemplate); err != nil {
		return nil, err
	}

	return []*SQLReviewTemplateData{
		prodTemplate,
		devTemplate,
	}, nil
}

func findTemplate(templateList []*SQLReviewTemplateData, id SQLReviewTemplateID) *SQLReviewTemplateData {
	for _, template := range templateList {
		if template.ID == id {
			return template
		}
	}
	return nil
}

func mergeRule(source *SQLReviewRuleData, override *SQLReviewRuleData) (*SQLReviewRule, error) {
	payload := source.Payload
	level := source.Level
	description := source.Description

	if override != nil {
		for key, val := range override.Payload {
			if _, ok := payload[key]; ok {
				payload[key] = val
			}
		}
		if override.Level == "ERROR" || override.Level == "WARNING" || override.Level == "DISABLED" {
			level = override.Level
		}
		description = override.Description
	}

	str, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &SQLReviewRule{
		Type:        source.Type,
		Level:       level,
		Description: description,
		Payload:     string(str),
	}, nil
}
