package advisor

import (
	_ "embed"
	"encoding/json"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

//go:embed config/sql-review.sample.yaml
var sqlReviewSampleTemplateStr string

//go:embed config/sql-review.dev.yaml
var sqlReviewDevTemplateStr string

//go:embed config/sql-review.prod.yaml
var sqlReviewProdTemplateStr string

// SQLReviewTemplateData is the API message for SQL review rule template.
type SQLReviewTemplateData struct {
	ID       string               `yaml:"id"`
	RuleList []*SQLReviewRuleData `yaml:"ruleList"`
}

// SQLReviewRuleData is the API message for SQL review rule update.
type SQLReviewRuleData struct {
	Type    SQLReviewRuleType `yaml:"type"`
	Level   string            `yaml:"level,omitempty"`
	Comment string            `yaml:"comment"`
	Payload map[string]any    `yaml:"payload"`
}

// SQLReviewConfigOverride is the API message for SQL review configuration override.
type SQLReviewConfigOverride struct {
	Template string               `yaml:"template"`
	RuleList []*SQLReviewRuleData `yaml:"ruleList"`
}

// MergeSQLReviewRules will merge the input YML config into default template.
func MergeSQLReviewRules(override *SQLReviewConfigOverride) ([]*storepb.SQLReviewRule, error) {
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

	var res []*storepb.SQLReviewRule

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

func findTemplate(templateList []*SQLReviewTemplateData, id string) *SQLReviewTemplateData {
	for _, template := range templateList {
		if template.ID == id {
			return template
		}
	}
	return nil
}

func mergeRule(source *SQLReviewRuleData, override *SQLReviewRuleData) (*storepb.SQLReviewRule, error) {
	payload := source.Payload
	level := source.Level
	comment := source.Comment

	if override != nil {
		for key, val := range override.Payload {
			if _, ok := payload[key]; ok {
				payload[key] = val
			}
		}
		if override.Level == "ERROR" || override.Level == "WARNING" || override.Level == "DISABLED" {
			level = override.Level
		}
		comment = override.Comment
	}

	str, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	ruleLevelValue, ok := storepb.SQLReviewRuleLevel_value[level]
	if !ok {
		return nil, errors.Errorf("invalid rule level %q", level)
	}

	return &storepb.SQLReviewRule{
		Type:    string(source.Type),
		Level:   storepb.SQLReviewRuleLevel(ruleLevelValue),
		Comment: comment,
		Payload: string(str),
	}, nil
}
