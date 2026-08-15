package shared

import (
	"encoding/json"
	"mime/multipart"
	"strings"

	"github.com/lib/pq"
)

// MultipartValues returns the values submitted for a multipart text field,
// accepting both "field" and the "field[]" spelling some HTTP clients emit for
// repeated entries. The bool reports whether the field was present at all,
// which is what separates "leave it alone" from "clear it".
func MultipartValues(form *multipart.Form, field string) ([]string, bool) {
	if values, ok := form.Value[field]; ok {
		return values, true
	}
	if values, ok := form.Value[field+"[]"]; ok {
		return values, true
	}
	return nil, false
}

// ParseStringArrayField normalises a multipart field backed by a Postgres
// text[] column. Clients may send one value per entry, or a single
// JSON-encoded array; blank entries are dropped, so a lone empty value clears
// the list. The result is always non-nil so an empty list writes {} rather
// than NULL.
func ParseStringArrayField(values []string) (pq.StringArray, error) {
	if len(values) == 1 && strings.HasPrefix(strings.TrimSpace(values[0]), "[") {
		var decoded []string
		if err := json.Unmarshal([]byte(values[0]), &decoded); err != nil {
			return nil, err
		}
		values = decoded
	}

	parsed := make(pq.StringArray, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			parsed = append(parsed, trimmed)
		}
	}

	return parsed, nil
}
