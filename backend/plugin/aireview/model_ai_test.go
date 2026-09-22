package aireview

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// geminiRequest is the part of the Gemini wire format the tests assert on.
type geminiRequest struct {
	SystemInstruction *struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"systemInstruction"`
	Contents []struct {
		Role  string `json:"role"`
		Parts []struct {
			Text             string `json:"text"`
			ThoughtSignature string `json:"thoughtSignature"`
			FunctionCall     *struct {
				Name string `json:"name"`
			} `json:"functionCall"`
			FunctionResponse *struct {
				Name     string         `json:"name"`
				Response map[string]any `json:"response"`
			} `json:"functionResponse"`
		} `json:"parts"`
	} `json:"contents"`
	Tools []struct {
		FunctionDeclarations []struct {
			Name string `json:"name"`
		} `json:"functionDeclarations"`
	} `json:"tools"`
}

// The scripted Model in reviewer_test.go stands in for ai.Chat, so this test
// holds the loop to the real Gemini adapter's wire format.
func TestSettingModelRoundTripsThroughGemini(t *testing.T) {
	t.Parallel()

	var requests []geminiRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request geminiRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		requests = append(requests, request)

		w.Header().Set("Content-Type", "application/json")
		reply := `{"candidates": [{"content": {"parts": [{"functionCall": {"name": "search", "args": {"text": "orders"}}, "thoughtSignature": "signature-1"}]}}], "usageMetadata": {"totalTokenCount": 40}}`
		if len(requests) == 2 {
			reply = `{"candidates": [{"content": {"parts": [{"text": "{\"findings\": []}"}]}}], "usageMetadata": {"totalTokenCount": 60}}`
		}
		_, err := w.Write([]byte(reply))
		require.NoError(t, err)
	}))
	defer server.Close()

	model := NewModel(&storepb.AISetting{Provider: storepb.AISetting_GEMINI, Endpoint: server.URL, Model: "gemini-3.5-flash", ApiKey: "test-key"})
	tools := &catalogTools{}
	result, err := NewReviewer(model).Review(context.Background(), &Request{Statement: "DROP TABLE orders;"}, tools)
	require.NoError(t, err)
	require.Empty(t, result.Findings)
	require.Equal(t, 2, result.Calls)
	require.Equal(t, 100, result.TotalTokens)

	require.Len(t, requests, 2)
	first := requests[0]
	require.NotNil(t, first.SystemInstruction)
	require.Contains(t, first.SystemInstruction.Parts[0].Text, "# Answer format")
	require.Len(t, first.Contents, 1)
	require.Contains(t, first.Contents[0].Parts[0].Text, "1\tDROP TABLE orders;")
	require.Len(t, first.Tools[0].FunctionDeclarations, 2)

	second := requests[1]
	require.Len(t, second.Contents, 3)
	require.Equal(t, "model", second.Contents[1].Role)
	require.Equal(t, "search", second.Contents[1].Parts[0].FunctionCall.Name)
	require.Equal(t, "signature-1", second.Contents[1].Parts[0].ThoughtSignature, "Gemini rejects a history that returns without the signature")
	require.Equal(t, "search", second.Contents[2].Parts[0].FunctionResponse.Name)
	// search returns a JSON array, which Gemini accepts only inside an object.
	require.Contains(t, second.Contents[2].Parts[0].FunctionResponse.Response, "result")
}

func TestSettingModelCorrectsAnEmptyGeminiReply(t *testing.T) {
	t.Parallel()

	var requests []geminiRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request geminiRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		requests = append(requests, request)

		w.Header().Set("Content-Type", "application/json")
		// Gemini returns a candidate without parts when it stops for safety or a malformed function call.
		reply := `{"candidates": [{"content": {}}]}`
		if len(requests) == 2 {
			reply = `{"candidates": [{"content": {"parts": [{"text": "{\"findings\": []}"}]}}]}`
		}
		_, err := w.Write([]byte(reply))
		require.NoError(t, err)
	}))
	defer server.Close()

	model := NewModel(&storepb.AISetting{Provider: storepb.AISetting_GEMINI, Endpoint: server.URL, Model: "gemini-3.5-flash", ApiKey: "test-key"})
	result, err := NewReviewer(model).Review(context.Background(), &Request{Statement: "DROP TABLE orders;"}, &catalogTools{})
	require.NoError(t, err)
	require.Equal(t, 2, result.Calls)

	require.Len(t, requests, 2)
	for _, content := range requests[1].Contents {
		require.Equal(t, "user", content.Role, "an empty model turn makes Gemini reject the request")
		require.NotEmpty(t, content.Parts)
	}
	last := requests[1].Contents[len(requests[1].Contents)-1]
	require.Contains(t, last.Parts[0].Text, "the reply is empty")
}

// TestReviewLiveGemini reviews a risky change with a real model. It needs a key
// and network access that CI does not have:
//
//	AIREVIEW_LIVE_GEMINI_API_KEY=... go test ./backend/plugin/aireview/ -run '^TestReviewLiveGemini$' -v -count=1
func TestReviewLiveGemini(t *testing.T) {
	apiKey := os.Getenv("AIREVIEW_LIVE_GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("AIREVIEW_LIVE_GEMINI_API_KEY is not set")
	}
	modelName := os.Getenv("AIREVIEW_LIVE_GEMINI_MODEL")
	if modelName == "" {
		modelName = "gemini-3.5-flash"
	}

	statement := strings.Join([]string{
		"-- Speed up the order history page.",
		"CREATE INDEX idx_orders_created_at ON orders (created_at);",
		"",
		"ALTER TABLE orders DROP COLUMN legacy_status;",
	}, "\n")
	request := &Request{
		WorkspacePolicy: "An index on a table with more than 1 million rows must be created with CONCURRENTLY.",
		Target: Target{
			Engine:      "POSTGRES",
			Version:     "16.2",
			Environment: "prod",
			Schemas:     []SchemaSummary{{Name: "public", ObjectCount: 2}},
		},
		Statement: statement,
	}
	model := NewModel(&storepb.AISetting{
		Provider: storepb.AISetting_GEMINI,
		Endpoint: "https://generativelanguage.googleapis.com/v1beta",
		Model:    modelName,
		ApiKey:   apiKey,
	})
	tools := &catalogTools{}

	result, err := NewReviewer(model).Review(context.Background(), request, tools)
	require.NoError(t, err)
	t.Logf("model calls: %d, tokens: %d, tool calls: %v", result.Calls, result.TotalTokens, tools.calls)
	for _, finding := range result.Findings {
		t.Logf("%s line %d: %s\n  rule: %s\n  evidence: %s\n  fix: %s", finding.Severity, finding.Line, finding.Title, finding.Rule, finding.Evidence, finding.Fix)
	}
	require.NotEmpty(t, tools.calls, "the model should look up the objects before it judges")
	require.NotEmpty(t, result.Findings)
}

// catalogTools is a canned two-object catalog: a large table and a view that reads it.
type catalogTools struct {
	calls []string
}

func (*catalogTools) Definitions() []*v1pb.AIChatToolDefinition {
	return []*v1pb.AIChatToolDefinition{
		{
			Name:             "search",
			Description:      "Find the objects whose name or definition contains the text, as a case insensitive substring. Use it to find the views, routines, and triggers that depend on a table.",
			ParametersSchema: `{"type": "object", "properties": {"text": {"type": "string"}}, "required": ["text"]}`,
		},
		{
			Name:             "read",
			Description:      "Return the definition and statistics of up to 20 objects by name. A table comes with its columns, indexes, constraints, triggers, row count, and size.",
			ParametersSchema: `{"type": "object", "properties": {"objects": {"type": "array", "items": {"type": "object", "properties": {"schema": {"type": "string"}, "name": {"type": "string"}}, "required": ["name"]}}}, "required": ["objects"]}`,
		},
	}
}

func (c *catalogTools) Call(_ context.Context, name string, arguments string) (string, error) {
	c.calls = append(c.calls, fmt.Sprintf("%s(%s)", name, arguments))
	definitions := map[string]string{
		"orders":         "CREATE TABLE public.orders (id bigint PRIMARY KEY, created_at timestamptz NOT NULL, legacy_status text, total numeric);\n-- rows: 52000000, size: 9 GB\n-- indexes: orders_pkey (id), 1.1 GB",
		"orders_summary": "CREATE VIEW public.orders_summary AS SELECT legacy_status, count(*) AS order_count FROM public.orders GROUP BY legacy_status;",
	}
	switch name {
	case "search":
		var args struct {
			Text string `json:"text"`
		}
		if message := decodeArguments(arguments, &args); message != "" {
			return message, nil
		}
		matches := []map[string]string{}
		for _, object := range []string{"orders", "orders_summary"} {
			if strings.Contains(strings.ToLower(object+" "+definitions[object]), strings.ToLower(args.Text)) {
				kind := "table"
				if object == "orders_summary" {
					kind = "view"
				}
				matches = append(matches, map[string]string{"kind": kind, "schema": "public", "name": object})
			}
		}
		out, err := json.Marshal(matches)
		return string(out), err
	case "read":
		var args struct {
			Objects []struct {
				Name string `json:"name"`
			} `json:"objects"`
		}
		if message := decodeArguments(arguments, &args); message != "" {
			return message, nil
		}
		var out []string
		for _, object := range args.Objects {
			definition, ok := definitions[object.Name]
			if !ok {
				definition = fmt.Sprintf("object %q not found", object.Name)
			}
			out = append(out, definition)
		}
		return strings.Join(out, "\n\n"), nil
	default:
		return "", errors.Errorf("unexpected tool %q", name)
	}
}

// decodeArguments returns the text that tells the model its arguments are
// invalid, or "" when they decode.
func decodeArguments(arguments string, target any) string {
	if err := json.Unmarshal([]byte(arguments), target); err != nil {
		return "invalid arguments: " + err.Error()
	}
	return ""
}
