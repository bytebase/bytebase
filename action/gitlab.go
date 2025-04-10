package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
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

func writeReleaseCheckToCodeQualityJSON(resp *v1pb.CheckReleaseResponse) error {
	var data []Finding
	for _, result := range resp.Results {
		for _, advice := range result.Advices {
			var severity string
			// Valid values are info, minor, major, critical, or blocker.
			switch advice.Status {
			case v1pb.Advice_WARNING:
				severity = "info"
			case v1pb.Advice_ERROR:
				severity = "critical"
			default:
				continue
			}
			data = append(data, Finding{
				Description: advice.Content,
				CheckName:   advice.Title,
				Fingerprint: fmt.Sprintf("%s#%d", result.File, advice.Line),
				Severity:    severity,
				Location: Location{
					Path: result.File,
					Lines: Lines{
						Begin: int(advice.Line),
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
