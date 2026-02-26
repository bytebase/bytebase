package elasticsearch

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestClassifyMaskableAPI(t *testing.T) {
	type testCase struct {
		Description string      `yaml:"description"`
		Method      string      `yaml:"method"`
		URL         string      `yaml:"url"`
		WantAPI     MaskableAPI `yaml:"wantAPI"`
		WantIndex   string      `yaml:"wantIndex"`
	}

	a := require.New(t)
	data, err := os.Open("test-data/masking_classify_api.yaml")
	a.NoError(err)
	raw, err := io.ReadAll(data)
	a.NoError(err)
	a.NoError(data.Close())

	var cases []testCase
	a.NoError(yaml.Unmarshal(raw, &cases))

	for _, tc := range cases {
		t.Run(tc.Description, func(t *testing.T) {
			api, index := classifyMaskableAPI(tc.Method, tc.URL)
			require.Equal(t, tc.WantAPI, api, "API classification")
			require.Equal(t, tc.WantIndex, index, "index extraction")
		})
	}
}

func TestAnalyzeRequestBody(t *testing.T) {
	type testCase struct {
		Description        string           `yaml:"description"`
		Body               string           `yaml:"body"`
		WantBlocked        []BlockedFeature `yaml:"wantBlocked"`
		WantSourceDisabled bool             `yaml:"wantSourceDisabled"`
		WantSourceFields   []string         `yaml:"wantSourceFields"`
		WantFields         []string         `yaml:"wantFields"`
		WantHighlight      []string         `yaml:"wantHighlight"`
		WantSort           []string         `yaml:"wantSort"`
		WantInnerHits      bool             `yaml:"wantInnerHits"`
	}

	a := require.New(t)
	data, err := os.Open("test-data/masking_analyze_body.yaml")
	a.NoError(err)
	raw, err := io.ReadAll(data)
	a.NoError(err)
	a.NoError(data.Close())

	var cases []testCase
	a.NoError(yaml.Unmarshal(raw, &cases))

	for _, tc := range cases {
		t.Run(tc.Description, func(t *testing.T) {
			result := analyzeRequestBody(tc.Body)
			if tc.WantBlocked != nil {
				require.Equal(t, tc.WantBlocked, result.BlockedFeatures)
			} else {
				require.Empty(t, result.BlockedFeatures)
			}
			require.Equal(t, tc.WantSourceDisabled, result.SourceDisabled)
			if tc.WantSourceFields != nil {
				require.Equal(t, tc.WantSourceFields, result.SourceFields)
			}
			if tc.WantFields != nil {
				require.Equal(t, tc.WantFields, result.RequestedFields)
			}
			if tc.WantHighlight != nil {
				require.ElementsMatch(t, tc.WantHighlight, result.HighlightFields)
			}
			if tc.WantSort != nil {
				require.Equal(t, tc.WantSort, result.SortFields)
			}
			require.Equal(t, tc.WantInnerHits, result.HasInnerHits)
		})
	}
}

func TestAnalyzeRequest(t *testing.T) {
	type testCase struct {
		Description         string           `yaml:"description"`
		Method              string           `yaml:"method"`
		URL                 string           `yaml:"url"`
		Body                string           `yaml:"body"`
		WantAPI             MaskableAPI      `yaml:"wantAPI"`
		WantIndex           string           `yaml:"wantIndex"`
		WantBlocked         []BlockedFeature `yaml:"wantBlocked"`
		WantPredicateFields []string         `yaml:"wantPredicateFields"`
	}

	a := require.New(t)
	data, err := os.Open("test-data/masking_analyze_request.yaml")
	a.NoError(err)
	raw, err := io.ReadAll(data)
	a.NoError(err)
	a.NoError(data.Close())

	var cases []testCase
	a.NoError(yaml.Unmarshal(raw, &cases))

	for _, tc := range cases {
		t.Run(tc.Description, func(t *testing.T) {
			result := AnalyzeRequest(tc.Method, tc.URL, tc.Body)
			require.Equal(t, tc.WantAPI, result.API)
			if tc.WantIndex != "" {
				require.Equal(t, tc.WantIndex, result.Index)
			}
			if tc.WantBlocked != nil {
				require.Equal(t, tc.WantBlocked, result.BlockedFeatures)
			} else {
				require.Empty(t, result.BlockedFeatures)
			}
			if tc.WantPredicateFields != nil {
				require.ElementsMatch(t, tc.WantPredicateFields, result.PredicateFields)
			}
		})
	}
}

func TestExtractPredicateFields(t *testing.T) {
	type testCase struct {
		Description string   `yaml:"description"`
		Body        string   `yaml:"body"`
		WantFields  []string `yaml:"wantFields"`
	}

	a := require.New(t)
	data, err := os.Open("test-data/masking_predicate_fields.yaml")
	a.NoError(err)
	raw, err := io.ReadAll(data)
	a.NoError(err)
	a.NoError(data.Close())

	var cases []testCase
	a.NoError(yaml.Unmarshal(raw, &cases))

	for _, tc := range cases {
		t.Run(tc.Description, func(t *testing.T) {
			result := analyzeRequestBody(tc.Body)
			if len(tc.WantFields) > 0 {
				require.ElementsMatch(t, tc.WantFields, result.PredicateFields)
			} else {
				require.Empty(t, result.PredicateFields)
			}
		})
	}
}
