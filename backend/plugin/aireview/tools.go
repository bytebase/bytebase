package aireview

import "context"

// Tools answers the model's tool calls for one target database.
type Tools interface {
	Definitions() []ToolDefinition
	// Call runs one tool. The returned text goes to the model, and it carries
	// what the model can act on, such as "object not found". A returned error
	// fails the review: a model that cannot get a fact raises no finding, so
	// feeding it an infrastructure failure as text would pass the change.
	Call(ctx context.Context, name string, arguments string) (string, error)
}

// ToolDefinition describes a tool to the model.
type ToolDefinition struct {
	Name        string
	Description string
	// ParametersSchema is a JSON Schema object. Gemini accepts an OpenAPI
	// subset only: type, properties, required, items, enum, and description.
	ParametersSchema string
}
