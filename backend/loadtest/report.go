package loadtest

import (
	"encoding/json"
	"os"
)

func writeReport(path string, results []Result, cfg Config) error {
	doc := struct {
		Config  Config   `json:"config"`
		Results []Result `json:"results"`
	}{cfg, results}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
