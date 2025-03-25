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
		}
	}
}
