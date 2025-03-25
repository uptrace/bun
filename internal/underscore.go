package internal

import "strings"

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

	// Special handling for fields with existing underscores
	if strings.Contains(s, "_") {
		parts := strings.Split(s, "_")
		result := processWord(parts[0])

		for i := 1; i < len(parts); i++ {
			result += "__" + processWord(parts[i])
		}

		return result
	}

	return processWord(s)
}

// processWord handles a single word without underscores
func processWord(s string) string {
	if len(s) == 0 {
		return s
	}

	r := make([]byte, 0, len(s)+5)

	// Track uppercase sequences for acronym detection
	inUpperSequence := false

	// Track if we're at the beginning of a new word
	atWordStart := true

	for i := 0; i < len(s); i++ {
		c := s[i]

		if IsUpper(c) {
			// Handle single uppercase letter at the end (ItemsP -> itemsp)
			if i > 0 && i == len(s)-1 {
				r = append(r, ToLower(c))
				continue
			}

			// First letter of the string is always lowercase without underscore
			if atWordStart {
				r = append(r, ToLower(c))
				atWordStart = false
				inUpperSequence = true
				continue
			}

			// Handle camelCase transition (lowercase to uppercase)
			if i > 0 && !inUpperSequence {
				r = append(r, '_')
			}

			// Handle acronym transition (uppercase sequence to lowercase)
			if i+1 < len(s) && IsLower(s[i+1]) && inUpperSequence && i > 0 && i+1 < len(s) {
				// We're at the end of an acronym like "HTTP" in "HTTPRequest"
				// Only add underscore if this isn't immediately after another underscore
				if len(r) > 0 && r[len(r)-1] != '_' {
					r = append(r, '_')
				}
			}

			r = append(r, ToLower(c))
			inUpperSequence = true
		} else {
			r = append(r, c)
			inUpperSequence = false
			atWordStart = false
		}
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
