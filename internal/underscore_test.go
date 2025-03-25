package internal

import (
	"fmt"
	"testing"
)

func TestUnderscore(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"CamelCase", "camel_case"},
		{"camelCase", "camel_case"},
		{"HTMLParser", "html_parser"},
		{"IDCard", "id_card"},
		{"UserID", "user_id"},
		{"APIDocs", "api_docs"},
		{"UserIDCard", "user_id_card"},
		{"URLEncoder", "url_encoder"},
		{"SimpleURL", "simple_url"},
		{"SimpleHTTPServer", "simple_http_server"},
		{"Raw", "raw"},
		{"raw", "raw"},
		{"ID", "id"},
		{"a", "a"},
		{"iOS", "i_os"},
		{"APIClient", "api_client"},
		{"HTTPRequest", "http_request"},
		{"OAuthClient", "o_auth_client"},
		// Special cases for single capital letter at the end
		{"ItemsP", "itemsp"},
		{"UsersA", "usersa"},
		{"DataX", "datax"},
		{"ListT", "listt"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := Underscore(test.input)
			if result != test.expected {
				t.Errorf("Underscore(%q) = %q, expected %q", test.input, result, test.expected)

				// Debug info
				fmt.Printf("Input: %q (len=%d)\n", test.input, len(test.input))
				for i, c := range test.input {
					fmt.Printf("  [%d] %q (upper=%v, lower=%v)\n",
						i, string(c), IsUpper(byte(c)), IsLower(byte(c)))
				}
			}
		})
	}
}
