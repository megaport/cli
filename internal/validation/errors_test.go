package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("speed", 5000, "must be one of: [1000 10000]")
	assert.Equal(t, "speed", err.Field)
	assert.Equal(t, 5000, err.Value)
	assert.Equal(t, "must be one of: [1000 10000]", err.Reason)
}

func TestValidationErrorError(t *testing.T) {
	tests := []struct {
		name   string
		field  string
		value  any
		reason string
		want   string
	}{
		{"int value", "rate limit", 0, "must be positive", "Invalid rate limit: 0 - must be positive"},
		{"string value", "name", "x", "too short", "Invalid name: x - too short"},
		{"empty string value", "name", "", "cannot be empty", "Invalid name: \"\" - cannot be empty"},
		{"nil value", "peer ASN", nil, "is required", "Invalid peer ASN: <nil> - is required"},
		{"bool value", "enabled", true, "bad", "Invalid enabled: true - bad"},
		{"slice value", "terms", []int{1, 12}, "bad", "Invalid terms: [1 12] - bad"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidationError(tt.field, tt.value, tt.reason)
			assert.Equal(t, tt.want, err.Error())
		})
	}
}

func TestValidationErrorSatisfiesErrorInterface(t *testing.T) {
	var err error = NewValidationError("f", 1, "r")
	assert.EqualError(t, err, "Invalid f: 1 - r")
}
