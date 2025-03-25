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

	r := make([]byte, 0, len(s)+5)

	// Track if we've seen an uppercase letter
	inUpperSequence := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		if IsUpper(c) {
			// Handle the pattern of a single capital letter at the end, e.g., ItemsP -> itemsp
			// This matches legacy behavior where a single uppercase letter at the end doesn't get an underscore
			if i > 0 && i == len(s)-1 {
				// Just lowercase it without adding an underscore
				r = append(r, ToLower(c))
				continue
			}

			// Handle camelCase -> camel_case (lowercase to uppercase transition)
			if i > 0 && IsLower(s[i-1]) {
				r = append(r, '_')
				inUpperSequence = true
			} else if i > 0 && i+1 < len(s) && IsLower(s[i+1]) && inUpperSequence {
				// Handle acronyms like HTMLParser -> html_parser
				// Add underscore at acronym end (before the lowercase letter)
				r = append(r, '_')
			} else {
				// Start of an uppercase sequence
				inUpperSequence = true
			}

			r = append(r, ToLower(c))
		} else {
			r = append(r, c)
			inUpperSequence = false
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
