package enterprise

import (
	_ "embed"
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

//go:embed plan.yaml
var planConfigStr string

var userLimitValues = map[v1pb.PlanType]int{}
var instanceLimitValues = map[v1pb.PlanType]int{}

// planFeatureMatrix maps plans to their available features
var planFeatureMatrix = make(map[v1pb.PlanType]map[v1pb.PlanFeature]bool)

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
		userLimitValues[plan.Type] = int(plan.MaximumSeatCount)
		instanceLimitValues[plan.Type] = int(plan.MaximumInstanceCount)

		planFeatureMatrix[plan.Type] = make(map[v1pb.PlanFeature]bool)
		for _, feature := range plan.Features {
			planFeatureMatrix[plan.Type][feature] = true
		}
	}
}
