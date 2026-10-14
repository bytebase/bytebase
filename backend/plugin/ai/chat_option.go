package ai

// ChatOption changes one Chat call.
type ChatOption func(*chatOptions)

type chatOptions struct {
	maxOutputTokens int
}

// WithMaxOutputTokens sets the output token limit of an adapter whose API
// requires one. Only Claude does; the OpenAI and Gemini adapters send no limit.
func WithMaxOutputTokens(tokens int) ChatOption {
	return func(o *chatOptions) {
		o.maxOutputTokens = tokens
	}
}

func newChatOptions(options []ChatOption) chatOptions {
	var o chatOptions
	for _, option := range options {
		option(&o)
	}
	return o
}
