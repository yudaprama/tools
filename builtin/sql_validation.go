package builtin

import (
	"fmt"
	"regexp"
	"strings"
)

// sqlIdentRe validates SQL identifiers (connection names, table names, schema names, database names)
var sqlIdentRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// validateSQLIdent validates that a string is a safe SQL identifier
func validateSQLIdent(name string) error {
	if name == "" {
		return nil // Empty is OK for optional fields
	}
	if !sqlIdentRe.MatchString(name) {
		return fmt.Errorf("invalid identifier '%s': must start with letter/underscore and contain only alphanumeric/underscore", name)
	}
	return nil
}

// stripLeadingCTEs removes leading WITH clauses and returns the actual query command
// This prevents attacks like: WITH cte AS (SELECT 1) UPDATE users SET password = 'hacked'
func stripLeadingCTEs(query string) string {
	query = strings.TrimSpace(query)
	upper := strings.ToUpper(query)

	// If doesn't start with WITH, return as-is
	if !strings.HasPrefix(upper, "WITH") {
		return query
	}

	// Helper to check if byte is whitespace
	isSpace := func(b byte) bool {
		return b == ' ' || b == '\t' || b == '\n' || b == '\r'
	}

	// Skip past "WITH"
	pos := 4

	// Skip whitespace after WITH
	for pos < len(query) && isSpace(query[pos]) {
		pos++
	}

	// Check for optional RECURSIVE keyword
	if pos+9 <= len(upper) && upper[pos:pos+9] == "RECURSIVE" {
		pos += 9
		// Skip whitespace after RECURSIVE
		for pos < len(query) && isSpace(query[pos]) {
			pos++
		}
	}

	// Loop to consume all CTEs
	for pos < len(query) {
		// Skip whitespace
		for pos < len(query) && isSpace(query[pos]) {
			pos++
		}

		if pos >= len(query) {
			break
		}

		// Find " AS" (with whitespace before AS)
		asPos := -1
		for i := pos; i < len(upper)-2; i++ {
			if isSpace(upper[i]) {
				// Skip all whitespace
				j := i
				for j < len(upper) && isSpace(upper[j]) {
					j++
				}
				// Check if "AS" follows
				if j+2 <= len(upper) && upper[j:j+2] == "AS" {
					// Make sure AS is followed by whitespace or (
					if j+2 >= len(upper) || isSpace(upper[j+2]) || upper[j+2] == '(' {
						asPos = j + 2
						break
					}
				}
			}
		}

		if asPos == -1 {
			break
		}
		pos = asPos

		// Skip whitespace after AS
		for pos < len(query) && isSpace(query[pos]) {
			pos++
		}

		// Expect opening parenthesis
		if pos >= len(query) || query[pos] != '(' {
			break
		}

		// Find matching closing parenthesis (handle nesting)
		parenCount := 1
		pos++
		for pos < len(query) && parenCount > 0 {
			if query[pos] == '(' {
				parenCount++
			} else if query[pos] == ')' {
				parenCount--
			}
			pos++
		}

		// Skip whitespace after closing paren
		for pos < len(query) && isSpace(query[pos]) {
			pos++
		}

		// Check for comma (more CTEs) or end of CTEs
		if pos >= len(query) {
			break
		}

		if query[pos] == ',' {
			pos++ // Skip comma and continue to next CTE
			continue
		}

		// No comma means end of CTEs, return rest of query (preserving original case)
		return strings.TrimSpace(query[pos:])
	}

	return query
}
