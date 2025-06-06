package api

import (
	_ "embed"

	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/base"
)

//go:embed plan.yaml
var planConfigStr string

// PlanLimit is the type for plan limits.
type PlanLimit int

const (
	// PlanLimitMaximumInstance is the key name for maximum number of instance for a plan.
	PlanLimitMaximumInstance = iota
	// PlanLimitMaximumUser is the key name for maximum number of user for a plan.
	PlanLimitMaximumUser
)

// PlanLimitValues is the plan limit value mapping.
var PlanLimitValues = map[PlanLimit]map[base.PlanType]int{
	PlanLimitMaximumInstance: {},
	PlanLimitMaximumUser:     {},
}

type planLimitConfig struct {
	Type                 base.PlanType `yaml:"type"`
	MaximumInstanceCount int           `yaml:"maximumInstanceCount"`
	MaximumSeatCount     int           `yaml:"maximumSeatCount"`
}

type planConfg struct {
	PlanList []*planLimitConfig `yaml:"planList"`
}

func init() {
	conf := &planConfg{}
	_ = yaml.Unmarshal([]byte(planConfigStr), conf)

	for _, limitConfig := range conf.PlanList {
		PlanLimitValues[PlanLimitMaximumInstance][limitConfig.Type] = limitConfig.MaximumInstanceCount
		PlanLimitValues[PlanLimitMaximumUser][limitConfig.Type] = limitConfig.MaximumSeatCount
	}
}
