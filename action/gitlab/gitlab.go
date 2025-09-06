package gitlab

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/action/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// Define the struct for the inner "lines" object.
type Lines struct {
	Begin int `json:"begin"`
}

// Define the struct for the inner "location" object.
type Location struct {
	Path  string `json:"path"`
	Lines Lines  `json:"lines"`
}

// Define the struct for each JSON object in the array.
type Finding struct {
	Description string   `json:"description"`
	CheckName   string   `json:"check_name"`
	Fingerprint string   `json:"fingerprint"`
	Severity    string   `json:"severity"`
	Location    Location `json:"location"`
}

func WriteReleaseCheckToCodeQualityJSON(resp *v1pb.CheckReleaseResponse) error {
	var data []Finding
	var warningCount, errorCount int
	for _, result := range resp.Results {
		for _, advice := range result.Advices {
			switch advice.Status {
			case v1pb.Advice_WARNING:
				warningCount++
			case v1pb.Advice_ERROR:
				errorCount++
			default:
				continue
			}
		}
	}
	var riskDetail string
	switch resp.RiskLevel {
	case v1pb.CheckReleaseResponse_LOW:
		riskDetail = "🟢 Low"
	case v1pb.CheckReleaseResponse_MODERATE:
		riskDetail = "🟡 Moderate"
	case v1pb.CheckReleaseResponse_HIGH:
		riskDetail = "🔴 High"
	default:
		riskDetail = "⚪ None"
	}
	details := fmt.Sprintf(`Summary: • Total Affected Rows: %d
• Overall Risk Level: %s
• Advices Statistics: %d Error(s), %d Warning(s)`, resp.GetAffectedRows(), riskDetail, errorCount, warningCount)
	data = append(data, Finding{
		Description: details,
		CheckName:   "Summary",
		Fingerprint: "summary",
		Severity:    "info",
	})
	for _, result := range resp.Results {
		for _, advice := range result.Advices {
			var severity string
			// Valid values are info, minor, major, critical, or blocker.
			switch advice.Status {
			case v1pb.Advice_WARNING:
				severity = "minor"
			case v1pb.Advice_ERROR:
				severity = "critical"
			default:
				continue
			}
			data = append(data, Finding{
				Description: advice.Content,
				CheckName:   advice.Title,
				Fingerprint: fmt.Sprintf("%s#%d", result.File, advice.GetStartPosition().GetLine()),
				Severity:    severity,
				Location: Location{
					Path: result.File,
					Lines: Lines{
						Begin: common.ConvertLineToActionLine(int(advice.GetStartPosition().GetLine())),
					},
				},
			})
		}
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return errors.Wrapf(err, "error marshaling json")
	}
	return os.WriteFile("bytebase_codequality.json", jsonData, 0644)
}
