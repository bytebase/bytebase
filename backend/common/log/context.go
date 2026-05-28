package log //nolint:revive // intentional package name

import (
	"context"
	"log/slog"
)

type contextAttrsKey struct{}

// WithAttrs returns a context carrying slog attributes for context-aware log calls.
func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	if len(attrs) == 0 {
		return ctx
	}

	existingAttrs := attrsFromContext(ctx)
	replacedKeys := make(map[string]struct{}, len(attrs))
	for _, attr := range attrs {
		replacedKeys[attr.Key] = struct{}{}
	}

	mergedAttrs := make([]slog.Attr, 0, len(existingAttrs)+len(attrs))
	for _, attr := range existingAttrs {
		if _, ok := replacedKeys[attr.Key]; ok {
			continue
		}
		mergedAttrs = append(mergedAttrs, attr)
	}
	mergedAttrs = append(mergedAttrs, attrs...)
	return context.WithValue(ctx, contextAttrsKey{}, mergedAttrs)
}

func attrsFromContext(ctx context.Context) []slog.Attr {
	if ctx == nil {
		return nil
	}
	attrs, ok := ctx.Value(contextAttrsKey{}).([]slog.Attr)
	if !ok {
		return nil
	}
	return attrs
}

type contextHandler struct {
	handler slog.Handler
}

// NewContextHandler wraps a slog handler so records include attributes carried by context.
func NewContextHandler(handler slog.Handler) slog.Handler {
	return &contextHandler{handler: handler}
}

func (h *contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *contextHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, attr := range attrsFromContext(ctx) {
		record.AddAttrs(attr)
	}
	return h.handler.Handle(ctx, record)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{handler: h.handler.WithGroup(name)}
}
