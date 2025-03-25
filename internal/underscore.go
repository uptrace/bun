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

	result := make([]byte, 0, len(s)*2)

	// Track positions that we should skip adding underscores for
	skipUnderscore := make(map[int]bool)

	// First scan for ID patterns to mark positions we should skip for underscores
	for i := 0; i < len(s); i++ {
		// Look for ID pattern
		if i+1 < len(s) && s[i] == 'I' && s[i+1] == 'D' {
			// Find what position comes after ID or IDs
			afterPos := i + 2 // After 'ID'
			if i+2 < len(s) && s[i+2] == 's' {
				afterPos = i + 3 // After 'IDs'
			}

			// If there's an uppercase after ID/IDs, mark it to skip adding underscore later
			if afterPos < len(s) && IsUpper(s[afterPos]) {
				skipUnderscore[afterPos] = true
			}
		}
	}

	// Now process the string
	for i := 0; i < len(s); i++ {
		c := s[i]

		// Handle underscore -> double underscore
		if c == '_' {
			result = append(result, '_', '_')
			continue
		}

		// Handle first character
		if i == 0 {
			if IsUpper(c) {
				result = append(result, ToLower(c))
			} else {
				result = append(result, c)
			}
			continue
		}

		// Handle single uppercase letter at the end
		if i == len(s)-1 && IsUpper(c) {
			result = append(result, ToLower(c))
			continue
		}

		// Handle ID pattern
		if c == 'I' && i+1 < len(s) && s[i+1] == 'D' {
			// Add underscore before ID if needed
			if i > 0 && isLetter(s[i-1]) {
				result = append(result, '_')
			}

			// Add "id"
			result = append(result, 'i', 'd')
			i++ // Skip 'D'

			// Handle plural 's'
			if i+1 < len(s) && s[i+1] == 's' {
				result = append(result, 's')
				i++ // Skip 's'
			}

			// Add underscore if followed by uppercase letter
			if i+1 < len(s) && IsUpper(s[i+1]) {
				result = append(result, '_')
			}

			continue
		}

		// Regular camelCase -> snake_case and acronym handling
		if IsUpper(c) {
			prev := s[i-1]

			// Skip adding underscore if this position was marked to skip
			if !skipUnderscore[i] && (IsLower(prev) || (IsUpper(prev) && i+1 < len(s) && IsLower(s[i+1]))) {
				result = append(result, '_')
			}

			result = append(result, ToLower(c))
		} else {
			result = append(result, c)
		}
	}

	return string(result)
}

// Helper to check if a byte is a letter (a-z, A-Z)
func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
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
