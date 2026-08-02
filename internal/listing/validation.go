package listing

import (
	"fmt"
	"strings"
)

// ValidationError reports one or more invalid input fields. The Fields map is
// keyed by JSON field name so the HTTP layer can return it directly.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed: %d field(s)", len(e.Fields))
}

// TransitionError reports an illegal status change.
type TransitionError struct {
	From Status
	To   Status
}

func (e *TransitionError) Error() string {
	return fmt.Sprintf("cannot transition from %q to %q", e.From, e.To)
}

// normalizeCurrency trims, upper-cases, and defaults an empty currency to EUR.
func normalizeCurrency(c string) string {
	c = strings.ToUpper(strings.TrimSpace(c))
	if c == "" {
		return "EUR"
	}
	return c
}

// validateFields runs the shared field rules and returns a *ValidationError if
// anything is wrong, or nil otherwise.
func validateFields(title, currency, city, postalCode string, priceCents int64, surfaceM2, rooms int) error {
	v := validator{fields: map[string]string{}}
	v.required("title", title)
	v.maxLen("title", title, 200)
	v.positive("price_cents", priceCents)
	if len(currency) != 3 {
		v.add("currency", "must be a 3-letter code")
	}
	v.required("city", city)
	v.required("postal_code", postalCode)
	v.nonNegative("surface_m2", surfaceM2)
	v.nonNegative("rooms", rooms)
	return v.err()
}

type validator struct {
	fields map[string]string
}

func (v *validator) add(field, msg string) {
	if _, exists := v.fields[field]; !exists {
		v.fields[field] = msg
	}
}

func (v *validator) required(field, val string) {
	if strings.TrimSpace(val) == "" {
		v.add(field, "is required")
	}
}

func (v *validator) maxLen(field, val string, n int) {
	if len(val) > n {
		v.add(field, fmt.Sprintf("must be at most %d characters", n))
	}
}

func (v *validator) positive(field string, n int64) {
	if n <= 0 {
		v.add(field, "must be greater than 0")
	}
}

func (v *validator) nonNegative(field string, n int) {
	if n < 0 {
		v.add(field, "must be zero or greater")
	}
}

func (v *validator) err() error {
	if len(v.fields) == 0 {
		return nil
	}
	return &ValidationError{Fields: v.fields}
}
