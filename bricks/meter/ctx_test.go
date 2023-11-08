package meter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

func TestAttrsToCtx(t *testing.T) {
	// Test adding attributes to an empty context
	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("key1", "value1"),
		attribute.Int("key2", 42),
	}

	newCtx := AttrsToCtx(ctx, attrs)

	retrievedAttrs := AttrsFromCtx(newCtx)
	assert.Equal(t, 2, len(retrievedAttrs))
}

func TestAttrsToCtxWithExistingAttributes(t *testing.T) {
	// Test adding attributes to a context with existing attributes
	existingAttrs := []attribute.KeyValue{
		attribute.String("existingKey", "existingValue"),
	}
	ctx := context.WithValue(context.Background(), attrsKey, existingAttrs)

	newAttrs := []attribute.KeyValue{
		attribute.String("newKey", "newValue"),
	}
	newCtx := AttrsToCtx(ctx, newAttrs)

	retrievedAttrs := AttrsFromCtx(newCtx)
	assert.Equal(t, 2, len(retrievedAttrs))

	expectedKey := attribute.Key("existingKey")
	expectedValue := attribute.StringValue("existingValue")
	assert.Equal(t, expectedKey, retrievedAttrs[0].Key)
	assert.Equal(t, expectedValue, retrievedAttrs[0].Value)

	expectedKey = "newKey"
	expectedValue = attribute.StringValue("newValue")
	assert.Equal(t, expectedKey, retrievedAttrs[1].Key)
	assert.Equal(t, expectedValue, retrievedAttrs[1].Value)
}

func TestAttrsFromCtx(t *testing.T) {
	// Test retrieving attributes from a context with existing attributes
	existingAttrs := []attribute.KeyValue{
		attribute.String("existingKey", "existingValue"),
	}
	ctx := context.WithValue(context.Background(), attrsKey, existingAttrs)

	retrievedAttrs := AttrsFromCtx(ctx)
	assert.Equal(t, 1, len(retrievedAttrs))

	expectedKey := attribute.Key("existingKey")
	expectedValue := attribute.StringValue("existingValue")
	assert.Equal(t, expectedKey, retrievedAttrs[0].Key)
	assert.Equal(t, expectedValue, retrievedAttrs[0].Value)
}

func TestAttrsToCtxWithEmptyAttributes(t *testing.T) {
	// Test adding empty attributes to a context with existing attributes
	existingAttrs := []attribute.KeyValue{
		attribute.String("existingKey", "existingValue"),
	}
	ctx := context.WithValue(context.Background(), attrsKey, existingAttrs)

	newCtx := AttrsToCtx(ctx, nil)

	retrievedAttrs := AttrsFromCtx(newCtx)
	assert.Equal(t, 1, len(retrievedAttrs))
}

func TestFilterAttrsFromCtx(t *testing.T) {
	// Test filtering attributes from a context with existing attributes
	existingAttrs := []attribute.KeyValue{
		attribute.String("existingKey", "existingValue"),
		attribute.String("existingKey2", "existingValue2"),
	}
	ctx := context.WithValue(context.Background(), attrsKey, existingAttrs)

	filteredAttrs := FilterAttrsFromCtx(ctx, []string{"existingKey"})
	assert.Equal(t, 1, len(filteredAttrs))

	expectedKey := attribute.Key("existingKey")
	expectedValue := attribute.StringValue("existingValue")
	assert.Equal(t, expectedKey, filteredAttrs[0].Key)
	assert.Equal(t, expectedValue, filteredAttrs[0].Value)
}
