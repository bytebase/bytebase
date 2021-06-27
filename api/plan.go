package api

type FeatureType string

const (
	FEATURE_ADMIN        FeatureType = "bb.admin"
	FEATURE_DBA_WORKFLOW FeatureType = "bb.dba-workflow"
	FEATURE_DATA_SOURCE  FeatureType = "bb.data-source"
)

func (e FeatureType) String() string {
	switch e {
	case FEATURE_ADMIN:
		return "bb.admin"
	case FEATURE_DBA_WORKFLOW:
		return "bb.dba-workflow"
	case FEATURE_DATA_SOURCE:
		return "bb.data-source"
	}
	return ""
}

type PlanType int

const (
	FREE PlanType = iota
	TEAM
	ENTERPRISE
)

// A map from the a particular feature to the respective enablement of a particular plan
var FEATURE_MATRIX = map[FeatureType][3]bool{
	"bb.admin":        {false, true, true},
	"bb.dba-workflow": {false, false, true},
	"bb.data-source":  {false, false, false},
}

type Plan struct {
	Type PlanType `jsonapi:"attr,type"`
}

type PlanPatch struct {
	Type PlanType `jsonapi:"attr,type"`
}
