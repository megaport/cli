package validation

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInt(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     string
		want      int
		wantErr   bool
		errSubstr []string
	}{
		{name: "valid positive", field: "location ID", value: "123", want: 123},
		{name: "valid negative", field: "ASN", value: "-5", want: -5},
		{name: "valid zero", field: "VLAN ID", value: "0", want: 0},
		{
			name:    "non-numeric",
			field:   "location ID",
			value:   "bad-location",
			wantErr: true,
			// Friendly message: names the field and value, no strconv internals.
			errSubstr: []string{"location ID", "bad-location", "not a valid whole number"},
		},
		{
			name:      "empty string",
			field:     "employee ID",
			value:     "",
			wantErr:   true,
			errSubstr: []string{"employee ID", "not a valid whole number"},
		},
		{
			name:      "float string",
			field:     "term",
			value:     "1.5",
			wantErr:   true,
			errSubstr: []string{"term", "not a valid whole number"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInt(tt.field, tt.value)
			if tt.wantErr {
				assert.Error(t, err)
				for _, s := range tt.errSubstr {
					assert.Contains(t, err.Error(), s)
				}
				// Must not leak strconv internals to the user.
				assert.NotContains(t, err.Error(), "strconv")
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestParseIntReturnsTypedValidationError verifies ParseInt's failure is a
// typed *ValidationError, which is what actually drives the Usage exit code
// in classifyError (not a substring match against the message text).
func TestParseIntReturnsTypedValidationError(t *testing.T) {
	_, err := ParseInt("location ID", "abc")
	require.Error(t, err)
	var validationErr *ValidationError
	assert.True(t, errors.As(err, &validationErr), "expected a typed *ValidationError, got %T: %v", err, err)
}
