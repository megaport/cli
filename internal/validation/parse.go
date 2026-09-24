package validation

import (
	"strconv"
)

// ParseInt converts a user-supplied string into an int. On failure it returns a
// friendly, typed *ValidationError naming the field and the offending value,
// instead of leaking strconv internals like `strconv.Atoi: parsing "x": invalid
// syntax` or relying on exit-code classification matching the message text.
func ParseInt(field, value string) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, NewValidationError(field, value, "is not a valid whole number")
	}
	return n, nil
}
