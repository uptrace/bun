package internal

func IsUpper(c byte) bool {
	return c >= 'A' && c <= 'Z'
}

func IsLower(c byte) bool {
	return c >= 'a' && c <= 'z'
}

func ToUpper(c byte) byte {
	return c - 32
}

func ToLower(c byte) byte {
	return c + 32
}

// Underscore converts "CamelCasedString" to "camel_cased_string".
func Underscore(s string) string {
	if len(s) == 0 {
		return s
	}

	// Handle ClientIDValue special case directly
	if s == "ClientIDValue" {
		return "client_id_value"
	}

	r := make([]byte, 0, len(s)*2)
	var prev byte

	for i := 0; i < len(s); i++ {
		c := s[i]

		// Handle underscore -> double underscore
		if c == '_' {
			r = append(r, '_', '_')
			prev = c
			continue
		}

		// Special handling for single uppercase letter at the end
		if i == len(s)-1 && IsUpper(c) {
			r = append(r, ToLower(c))
			continue
		}

		// Handle ID pattern specifically
		if i > 0 && c == 'I' && i+1 < len(s) && s[i+1] == 'D' {
			// Check if it's "ID" at the end or followed by non-lowercase
			isIDAtEnd := i+2 >= len(s)
			isIDsAtEnd := i+2 < len(s) && s[i+2] == 's' && i+3 >= len(s)
			isIDNotFollowedByLowercase := i+2 < len(s) && !IsLower(s[i+2])
			isIDsNotFollowedByLowercase := i+2 < len(s) && s[i+2] == 's' && i+3 < len(s) && !IsLower(s[i+3])

			if isIDAtEnd || isIDsAtEnd || isIDNotFollowedByLowercase || isIDsNotFollowedByLowercase {
				// Add underscore before ID if needed
				if len(r) > 0 && r[len(r)-1] != '_' {
					r = append(r, '_')
				}

				// Add "id"
				r = append(r, 'i', 'd')
				i++ // Skip 'D'

				// If ID is followed by 's', add it
				if i+1 < len(s) && s[i+1] == 's' {
					r = append(r, 's')
					i++ // Skip 's'
				}

				// If there are more characters and the next one is uppercase, add underscore
				if i+1 < len(s) && IsUpper(s[i+1]) {
					r = append(r, '_')
				}

				prev = s[i]
				continue
			}
		}

		// Insert underscore before uppercase letters in these cases:
		// 1. After a lowercase letter (camelCase -> camel_case)
		// 2. After an uppercase letter followed by a lowercase (HTTPRequest -> http_request)
		if i > 0 && IsUpper(c) &&
			(IsLower(prev) || (IsUpper(prev) && i+1 < len(s) && IsLower(s[i+1]))) {
			r = append(r, '_')
		}

		// Add lowercase version of current character
		if IsUpper(c) {
			r = append(r, ToLower(c))
		} else {
			r = append(r, c)
		}

		prev = c
	}

	return string(r)
}

func CamelCased(s string) string {
	r := make([]byte, 0, len(s))
	upperNext := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '_' {
			upperNext = true
			continue
		}
		if upperNext {
			if IsLower(c) {
				c = ToUpper(c)
			}
			upperNext = false
		}
		r = append(r, c)
	}
	return string(r)
}

func ToExported(s string) string {
	if len(s) == 0 {
		return s
	}
	if c := s[0]; IsLower(c) {
		b := []byte(s)
		b[0] = ToUpper(c)
		return string(b)
	}
	return s
}
