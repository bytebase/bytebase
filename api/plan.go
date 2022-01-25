package api

// FeatureType is the type of a feature.
type FeatureType string

const (
	// FeatureAdmin is the feature type for admin.
	FeatureAdmin FeatureType = "bb.admin"
	// FeatureDBAWorkflow is the feature type for DBA.
	FeatureDBAWorkflow FeatureType = "bb.dba-workflow"
	// FeatureDataSource is the feature type for data sources.
	FeatureDataSource FeatureType = "bb.data-source"
)

func (e FeatureType) String() string {
	switch e {
	case FeatureAdmin:
		return "bb.admin"
	case FeatureDBAWorkflow:
		return "bb.dba-workflow"
	case FeatureDataSource:
		return "bb.data-source"
	}
	return ""
}

// PlanType is the type for a plan.
type PlanType int

const (
	// FREE is the plan type for FREE.
	FREE PlanType = iota
	// TEAM is the plan type for TEAM.
	TEAM
	// ENTERPRISE is the plan type for ENTERPRISE.
	ENTERPRISE
)

// String returns the string format of plan type.
func (p PlanType) String() string {
	switch p {
	case FREE:
		return "FREE"
	case TEAM:
		return "TEAM"
	case ENTERPRISE:
		return "ENTERPRISE"
	}
	return ""
}

// FeatureMatrix is a map from the a particular feature to the respective enablement of a particular plan
var FeatureMatrix = map[FeatureType][3]bool{
	"bb.admin":        {false, true, true},
	"bb.dba-workflow": {false, false, true},
	"bb.data-source":  {false, false, false},
}

// Plan is the API message for a plan.
type Plan struct {
	Type PlanType `jsonapi:"attr,type"`
}

// PlanPatch is the API message for patching a plan.
type PlanPatch struct {
	Type PlanType `jsonapi:"attr,type"`
}
