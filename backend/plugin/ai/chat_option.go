package ai

// ChatOption changes one Chat call.
type ChatOption func(*chatOptions)

type chatOptions struct {
	maxOutputTokens int
}

// WithMaxOutputTokens asks for an output limit of up to tokens. Only the Claude
// adapter sends a limit, lowered to the model's documented maximum; the OpenAI
// and Gemini adapters send none.
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
