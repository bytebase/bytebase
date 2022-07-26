package config

import "github.com/bytebase/bytebase/plugin/advisor"

// SQLReviewTemplateID is the template id for SQL review rules.
type SQLReviewTemplateID string

const (
	// TemplateForMySQLProd is the template id for mysql prod template
	TemplateForMySQLProd SQLReviewTemplateID = "bb.sql-review.mysql.prod"
	// TemplateForMySQLDev is the template id for mysql dev template
	TemplateForMySQLDev SQLReviewTemplateID = "bb.sql-review.mysql.dev"
)

// SQLReviewConfiguration is the API message for SQL review config.
type SQLReviewConfiguration struct {
	TemplateList []*SQLReviewTemplate `yaml:"templateList"`
}

// SQLReviewTemplate is the API message for SQL review template.
type SQLReviewTemplate struct {
	ID       SQLReviewTemplateID      `yaml:"id"`
	RuleList []*SQLReviewTemplateRule `yaml:"ruleList"`
}

// SQLReviewTemplateRule is the API message for SQL review rule template.
type SQLReviewTemplateRule struct {
	Type          advisor.SQLReviewRuleType  `yaml:"type"`
	Engine        string                     `yaml:"engine,omitempty"`
	Level         advisor.SQLReviewRuleLevel `yaml:"level,omitempty"`
	ComponentList []*TemplateRuleComponent   `yaml:"componentList,omitempty"`
}

// TemplateRuleComponent is the API message for component in rule template.
type TemplateRuleComponent struct {
	Key     string              `yaml:"key"`
	Payload TemplateRulePayload `yaml:"payload"`
}

// TemplateRulePayload is the API message for payload in rule template.
type TemplateRulePayload struct {
	Type    string      `yaml:"type"`
	Default interface{} `yaml:"default"`
}

// SQLReviewConfigUpdate is the API message for SQL review configuration update.
type SQLReviewConfigUpdate struct {
	Template SQLReviewTemplateID    `yaml:"template"`
	RuleList []*SQLReviewRuleUpdate `yaml:"ruleList"`
}

// SQLReviewRuleUpdate is the API message for SQL review rule update.
type SQLReviewRuleUpdate struct {
	Type    advisor.SQLReviewRuleType  `yaml:"type"`
	Level   advisor.SQLReviewRuleLevel `yaml:"level,omitempty"`
	Payload map[string]interface{}     `yaml:"payload"`
}
