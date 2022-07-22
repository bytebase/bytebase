package config

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/bytebase/bytebase/plugin/advisor"
	"gopkg.in/yaml.v3"
)

// TODO: remove the sqlReviewConfig.yaml in the frontend. Use the unified yml file in the advisor package.
//go:embed sql-check.yml
var sqlCheck string

// MergeSQLReviewRules will merge the input YML config into default template.
func MergeSQLReviewRules(ymlStr string) ([]*advisor.SQLReviewRule, error) {
	config := SQLReviewConfiguration{}

	if err := yaml.Unmarshal([]byte(sqlCheck), &config); err != nil {
		return nil, err
	}

	sqlReviewConfigUpdate := SQLReviewConfigUpdate{}
	if err := yaml.Unmarshal([]byte(ymlStr), &sqlReviewConfigUpdate); err != nil {
		return nil, err
	}

	template := findTemplate(config.TemplateList, sqlReviewConfigUpdate.Template)
	if template == nil {
		return nil, fmt.Errorf("cannot find template: %v", sqlReviewConfigUpdate.Template)
	}

	ruleUpdateMap := make(map[advisor.SQLReviewRuleType]*SQLReviewRuleUpdate)
	for _, rule := range sqlReviewConfigUpdate.RuleList {
		ruleUpdateMap[rule.Type] = rule
	}

	var res []*advisor.SQLReviewRule

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

func findTemplate(templateList []*SQLReviewTemplate, id SQLReviewTemplateID) *SQLReviewTemplate {
	for _, template := range templateList {
		if template.ID == id {
			return template
		}
	}
	return nil
}

func mergeRule(source *SQLReviewTemplateRule, update *SQLReviewRuleUpdate) (*advisor.SQLReviewRule, error) {
	payload := make(map[string]interface{})
	level := source.Level

	for _, component := range source.ComponentList {
		payload[component.Key] = component.Payload.Default
	}

	if update != nil {
		for key, val := range update.Payload {
			if _, ok := payload[key]; ok {
				payload[key] = val
			}
		}
		if update.Level == "ERROR" || update.Level == "WARNING" || update.Level == "DISABLED" {
			level = update.Level
		}
	}

	str, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &advisor.SQLReviewRule{
		Type:    source.Type,
		Level:   level,
		Payload: string(str),
	}, nil
}
