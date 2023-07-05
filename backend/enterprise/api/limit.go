package api

import (
	_ "embed"

	"gopkg.in/yaml.v3"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

//go:embed plan.yaml
var planConfigStr string

// PlanLimit is the type for plan limits.
type PlanLimit int

const (
	// PlanLimitMaximumTask is the key name for maximum number of tasks for a plan.
	PlanLimitMaximumTask PlanLimit = iota
	// PlanLimitMaximumEnvironment is the key name for maximum number of environments for a plan.
	PlanLimitMaximumEnvironment
	// PlanLimitMaximumInstance is the key name for maximum number of instance for a plan.
	PlanLimitMaximumInstance
	// PlanLimitMaximumUser is the key name for maximum number of user for a plan.
	PlanLimitMaximumUser
)

// PlanLimitValues is the plan limit value mapping.
var PlanLimitValues = map[PlanLimit]map[api.PlanType]int64{
	PlanLimitMaximumTask:        {},
	PlanLimitMaximumEnvironment: {},
	PlanLimitMaximumInstance:    {},
	PlanLimitMaximumUser:        {},
}

type planLimitConfig struct {
	Type                    api.PlanType `yaml:"type"`
	MaximumInstanceCount    int64        `yaml:"maximumInstanceCount"`
	MaximumSeatCount        int64        `yaml:"maximumSeatCount"`
	MaximumTaskCount        int64        `yaml:"maximumTaskCount"`
	MaximumEnvironmentCount int64        `yaml:"maximumEnvironmentCount"`
}

type planConfg struct {
	PlanList []*planLimitConfig `yaml:"planList"`
}

func init() {
	conf := &planConfg{}
	_ = yaml.Unmarshal([]byte(planConfigStr), conf)

	for _, limitConfig := range conf.PlanList {
		PlanLimitValues[PlanLimitMaximumTask][limitConfig.Type] = limitConfig.MaximumTaskCount
		PlanLimitValues[PlanLimitMaximumEnvironment][limitConfig.Type] = limitConfig.MaximumEnvironmentCount
		PlanLimitValues[PlanLimitMaximumInstance][limitConfig.Type] = limitConfig.MaximumInstanceCount
		PlanLimitValues[PlanLimitMaximumUser][limitConfig.Type] = limitConfig.MaximumSeatCount
	}
}
