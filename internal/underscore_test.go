package internal

import (
	"testing"
)

func TestUnderscore(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Normal CamelCase to snake_case
		{"CamelCase", "camel_case"},
		{"camelCase", "camel_case"},

		// Test single uppercase letter at the end
		{"ItemsP", "itemsp"},
		{"UsersA", "usersa"},
		{"DataX", "datax"},
		{"ListT", "listt"},

		// Test for acronyms
		{"HTTPRequest", "http_request"},
		{"OAuthClient", "o_auth_client"},
		{"APIKey", "api_key"},

		// Test existing underscores - should become double underscores
		{"Genre_Rating", "genre__rating"},
		{"User_ID", "user__id"},

		// Test ID patterns
		{"ID", "id"},
		{"UserID", "user_id"},
		{"ChildIDs", "child_ids"},
		{"ItemsIDs", "items_ids"},
		{"ClientIDValue", "client_id_value"},
		{"OrderID", "order_id"},
		{"HTML", "html"},
		{"HTMLIDs", "html_ids"},
	}

	for _, test := range tests {
		got := Underscore(test.input)
		if got != test.expected {
			t.Errorf("Underscore(%q) = %q, expected %q", test.input, got, test.expected)

			// Add debug for ClientIDValue
			if test.input == "ClientIDValue" {
				t.Logf("Debug ClientIDValue transformation:")

				result := make([]byte, 0, len(test.input)*2)

				// Process step by step
				for i := 0; i < len(test.input); i++ {
					c := test.input[i]
					t.Logf("Processing char %d: '%c'", i, c)

					// First character
					if i == 0 {
						if IsUpper(c) {
							result = append(result, ToLower(c))
							t.Logf("  Added lowercase '%c', result now: %q", ToLower(c), string(result))
						} else {
							result = append(result, c)
							t.Logf("  Added '%c', result now: %q", c, string(result))
						}
						continue
					}

					// Handle last character
					if i == len(test.input)-1 {
						if IsUpper(c) {
							result = append(result, ToLower(c))
							t.Logf("  Added lowercase '%c', result now: %q", ToLower(c), string(result))
						} else {
							result = append(result, c)
							t.Logf("  Added '%c', result now: %q", c, string(result))
						}
						continue
					}

					// Handle ID pattern
					if c == 'I' && i+1 < len(test.input) && test.input[i+1] == 'D' {
						t.Logf("  Found ID pattern at pos %d", i)

						// Get info about before and after
						hasLetterBefore := i > 0 && ((test.input[i-1] >= 'a' && test.input[i-1] <= 'z') ||
							(test.input[i-1] >= 'A' && test.input[i-1] <= 'Z'))

						var afterPos int
						if i+2 < len(test.input) && test.input[i+2] == 's' {
							afterPos = i + 3
						} else {
							afterPos = i + 2
						}

						hasUpperAfter := afterPos < len(test.input) && IsUpper(test.input[afterPos])

						t.Logf("  hasLetterBefore: %v, hasUpperAfter: %v, afterPos: %d",
							hasLetterBefore, hasUpperAfter, afterPos)

						// Add underscore before ID if needed
						if hasLetterBefore {
							result = append(result, '_')
							t.Logf("  Added underscore before ID, result now: %q", string(result))
						}

						// Add id
						result = append(result, 'i', 'd')
						t.Logf("  Added 'id', result now: %q", string(result))

						// Skip D
						i++

						// Handle s for plural
						if i+1 < len(test.input) && test.input[i+1] == 's' {
							result = append(result, 's')
							t.Logf("  Added 's', result now: %q", string(result))
							i++
						}

						// Add underscore if uppercase follows
						if hasUpperAfter {
							result = append(result, '_')
							t.Logf("  Added underscore after ID/IDs, result now: %q", string(result))
						}

						continue
					}

					// Regular camelCase handling
					if IsUpper(c) {
						prev := test.input[i-1]

						if IsLower(prev) || (IsUpper(prev) && i+1 < len(test.input) && IsLower(test.input[i+1])) {
							result = append(result, '_')
							t.Logf("  Added underscore for camelCase, result now: %q", string(result))
						}

						result = append(result, ToLower(c))
						t.Logf("  Added lowercase '%c', result now: %q", ToLower(c), string(result))
					} else {
						result = append(result, c)
						t.Logf("  Added '%c', result now: %q", c, string(result))
					}
				}

				t.Logf("Final debug result: %q", string(result))
			}
		}
	}
}
