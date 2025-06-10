package api

import (
	_ "embed"
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
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
var PlanLimitValues = map[PlanLimit]map[v1pb.PlanType]int{
	PlanLimitMaximumInstance: {},
	PlanLimitMaximumUser:     {},
}

// PlanFeatureMatrix maps plans to their available features
var PlanFeatureMatrix = make(map[v1pb.PlanType]map[v1pb.PlanLimitConfig_Feature]bool)

func init() {
	// First unmarshal YAML to a generic map, then convert to JSON for protojson
	var yamlData map[string]any
	if err := yaml.Unmarshal([]byte(planConfigStr), &yamlData); err != nil {
		panic("failed to unmarshal plan.yaml: " + err.Error())
	}

	// Convert YAML data to JSON bytes
	jsonBytes, err := json.Marshal(yamlData)
	if err != nil {
		panic("failed to convert plan.yaml to JSON: " + err.Error())
	}

	conf := &v1pb.PlanConfig{}
	//nolint:forbidigo
	if err := protojson.Unmarshal(jsonBytes, conf); err != nil {
		panic("failed to unmarshal plan config proto: " + err.Error())
	}

	for _, plan := range conf.Plans {
		PlanLimitValues[PlanLimitMaximumInstance][plan.Type] = int(plan.MaximumInstanceCount)
		PlanLimitValues[PlanLimitMaximumUser][plan.Type] = int(plan.MaximumSeatCount)

		// Initialize feature map for this plan
		PlanFeatureMatrix[plan.Type] = make(map[v1pb.PlanLimitConfig_Feature]bool)
		for _, feature := range plan.Features {
			PlanFeatureMatrix[plan.Type][feature] = true
		}
	}
}
