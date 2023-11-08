package meter

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
)

type attrsCtxKey struct{}

var attrsKey = attrsCtxKey{}

// AttrsToCtx adds the provided attributes to the context.
// If the context already has attributes, the provided attributes are appended.
func AttrsToCtx(ctx context.Context, attrs []attribute.KeyValue) context.Context {
	existingAttrs := AttrsFromCtx(ctx)
	attrs = append(existingAttrs, attrs...)
	return context.WithValue(ctx, attrsKey, attrs)
}

// AttrsFromCtx returns the attributes from the context.
func AttrsFromCtx(ctx context.Context) []attribute.KeyValue {
	attrs, _ := ctx.Value(attrsKey).([]attribute.KeyValue)
	return attrs
}

// FilterAttrsFromCtx returns the attributes from the context that match the provided list of attribute names.
func FilterAttrsFromCtx(ctx context.Context, attrs []string) []attribute.KeyValue {
	attrsFromCtx := AttrsFromCtx(ctx)
	filteredAttrs := make([]attribute.KeyValue, 0, len(attrs))
	for _, attr := range attrsFromCtx {
		for _, attrName := range attrs {
			if attr.Key == attribute.Key(attrName) {
				filteredAttrs = append(filteredAttrs, attr)
			}
		}
	}
	return filteredAttrs
}
