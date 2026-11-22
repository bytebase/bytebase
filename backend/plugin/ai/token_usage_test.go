package ai

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestChatTokenUsage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider storepb.AISetting_Provider
		endpoint string
		reply    string
		want     string
	}{
		{
			name:     "OpenAI chat completions includes cached input and reasoning output",
			provider: storepb.AISetting_OPEN_AI,
			reply:    `{"usage":{"total_tokens":125,"prompt_tokens":100,"prompt_tokens_details":{"cached_tokens":70},"completion_tokens":25,"completion_tokens_details":{"reasoning_tokens":15}}}`,
			want:     `{"totalTokens":125,"inputTokens":100,"outputTokens":25,"cachedInputTokens":70}`,
		},
		{
			name:     "OpenAI responses includes cached input and reasoning output",
			provider: storepb.AISetting_OPEN_AI,
			endpoint: "/responses",
			reply:    `{"usage":{"total_tokens":125,"input_tokens":100,"input_tokens_details":{"cached_tokens":70},"output_tokens":25,"output_tokens_details":{"reasoning_tokens":15}}}`,
			want:     `{"totalTokens":125,"inputTokens":100,"outputTokens":25,"cachedInputTokens":70}`,
		},
		{
			name:     "Azure uses the same usage contract",
			provider: storepb.AISetting_AZURE_OPENAI,
			endpoint: "/responses",
			reply:    `{"usage":{"total_tokens":125,"input_tokens":100,"input_tokens_details":{"cached_tokens":70},"output_tokens":25}}`,
			want:     `{"totalTokens":125,"inputTokens":100,"outputTokens":25,"cachedInputTokens":70}`,
		},
		{
			name:     "OpenAI preserves the provider total rather than recomputing",
			provider: storepb.AISetting_OPEN_AI,
			reply:    `{"usage":{"total_tokens":150,"prompt_tokens":100,"completion_tokens":25}}`,
			want:     `{"totalTokens":150,"inputTokens":100,"outputTokens":25}`,
		},
		{
			name:     "OpenAI total only leaves details unknown",
			provider: storepb.AISetting_OPEN_AI,
			reply:    `{"usage":{"total_tokens":125}}`,
			want:     `{"totalTokens":125}`,
		},
		{
			name:     "OpenAI partial details remain partial",
			provider: storepb.AISetting_OPEN_AI,
			reply:    `{"usage":{"total_tokens":125,"completion_tokens":25,"prompt_tokens_details":{}}}`,
			want:     `{"totalTokens":125,"outputTokens":25}`,
		},
		{
			name:     "OpenAI zero is reported rather than unknown",
			provider: storepb.AISetting_OPEN_AI,
			reply:    `{"usage":{"total_tokens":0,"prompt_tokens":0,"completion_tokens":0,"prompt_tokens_details":{"cached_tokens":0}}}`,
			want:     `{"inputTokens":0,"outputTokens":0,"cachedInputTokens":0}`,
		},
		{
			name:     "OpenAI responses zero is reported rather than unknown",
			provider: storepb.AISetting_OPEN_AI,
			endpoint: "/responses",
			reply:    `{"usage":{"total_tokens":0,"input_tokens":0,"output_tokens":0,"input_tokens_details":{"cached_tokens":0}}}`,
			want:     `{"inputTokens":0,"outputTokens":0,"cachedInputTokens":0}`,
		},
		{
			name:     "OpenAI missing total does not report an invented zero",
			provider: storepb.AISetting_OPEN_AI,
			reply:    `{"usage":{"prompt_tokens":100}}`,
		},
		{
			name:     "OpenAI absent usage stays unknown",
			provider: storepb.AISetting_OPEN_AI,
			reply:    `{}`,
		},
		{
			name:     "Claude input includes cache creation and cache reads",
			provider: storepb.AISetting_CLAUDE,
			reply:    `{"usage":{"input_tokens":20,"cache_creation_input_tokens":30,"cache_read_input_tokens":50,"output_tokens":25}}`,
			want:     `{"totalTokens":125,"inputTokens":100,"outputTokens":25,"cachedInputTokens":50}`,
		},
		{
			name:     "Claude without caching leaves cache count unknown",
			provider: storepb.AISetting_CLAUDE,
			reply:    `{"usage":{"input_tokens":100,"output_tokens":25}}`,
			want:     `{"totalTokens":125,"inputTokens":100,"outputTokens":25}`,
		},
		{
			name:     "Claude zero is reported rather than unknown",
			provider: storepb.AISetting_CLAUDE,
			reply:    `{"usage":{"input_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":0}}`,
			want:     `{"inputTokens":0,"outputTokens":0,"cachedInputTokens":0}`,
		},
		{
			name:     "Claude partial usage cannot establish total",
			provider: storepb.AISetting_CLAUDE,
			reply:    `{"usage":{"input_tokens":100}}`,
		},
		{
			name:     "Claude absent usage stays unknown",
			provider: storepb.AISetting_CLAUDE,
			reply:    `{}`,
		},
		{
			name:     "Gemini includes tool use prompt and reasoning output",
			provider: storepb.AISetting_GEMINI,
			reply:    `{"usageMetadata":{"totalTokenCount":137,"promptTokenCount":100,"cachedContentTokenCount":70,"toolUsePromptTokenCount":12,"candidatesTokenCount":10,"thoughtsTokenCount":15}}`,
			want:     `{"totalTokens":137,"inputTokens":112,"outputTokens":25,"cachedInputTokens":70}`,
		},
		{
			name:     "Gemini preserves the provider total rather than recomputing",
			provider: storepb.AISetting_GEMINI,
			reply:    `{"usageMetadata":{"totalTokenCount":150,"promptTokenCount":100,"candidatesTokenCount":25}}`,
			want:     `{"totalTokens":150,"inputTokens":100,"outputTokens":25}`,
		},
		{
			name:     "Gemini total only leaves details unknown",
			provider: storepb.AISetting_GEMINI,
			reply:    `{"usageMetadata":{"totalTokenCount":125}}`,
			want:     `{"totalTokens":125}`,
		},
		{
			name:     "Gemini partial details remain partial",
			provider: storepb.AISetting_GEMINI,
			reply:    `{"usageMetadata":{"totalTokenCount":125,"promptTokenCount":100}}`,
			want:     `{"totalTokens":125,"inputTokens":100}`,
		},
		{
			name:     "Gemini zero is reported rather than unknown",
			provider: storepb.AISetting_GEMINI,
			reply:    `{"usageMetadata":{"totalTokenCount":0,"promptTokenCount":0,"cachedContentTokenCount":0,"candidatesTokenCount":0}}`,
			want:     `{"inputTokens":0,"outputTokens":0,"cachedInputTokens":0}`,
		},
		{
			name:     "Gemini missing total does not report an invented zero",
			provider: storepb.AISetting_GEMINI,
			reply:    `{"usageMetadata":{"promptTokenCount":100}}`,
		},
		{
			name:     "Gemini absent usage stays unknown",
			provider: storepb.AISetting_GEMINI,
			reply:    `{}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			response, err := chatWithScriptedReply(t, tt.provider, tt.endpoint, tt.reply)
			require.NoError(t, err)
			if tt.want == "" {
				require.Nil(t, response.Usage)
				return
			}
			require.NotNil(t, response.Usage)
			usage, err := protojson.Marshal(response.Usage)
			require.NoError(t, err)
			require.JSONEq(t, tt.want, string(usage))
		})
	}
}
