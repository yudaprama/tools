package builtin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripLeadingCTEs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple SELECT",
			input:    "SELECT * FROM users",
			expected: "SELECT * FROM users",
		},
		{
			name:     "Single CTE with SELECT",
			input:    "WITH cte AS (SELECT 1) SELECT * FROM cte",
			expected: "SELECT * FROM cte",
		},
		{
			name:     "Single CTE with UPDATE (attack)",
			input:    "WITH cte AS (SELECT 1) UPDATE users SET password = 'hacked'",
			expected: "UPDATE users SET password = 'hacked'",
		},
		{
			name:     "Multiple CTEs with SELECT",
			input:    "WITH cte1 AS (SELECT 1), cte2 AS (SELECT 2) SELECT * FROM cte1",
			expected: "SELECT * FROM cte1",
		},
		{
			name:     "Multiple CTEs with DELETE (attack)",
			input:    "WITH cte1 AS (SELECT 1), cte2 AS (SELECT 2) DELETE FROM users",
			expected: "DELETE FROM users",
		},
		{
			name:     "Nested CTEs",
			input:    "WITH cte AS (SELECT * FROM (SELECT 1) AS sub) SELECT * FROM cte",
			expected: "SELECT * FROM cte",
		},
		{
			name:     "CTE with complex subquery",
			input:    "WITH RECURSIVE cte AS (SELECT 1 UNION ALL SELECT n+1 FROM cte WHERE n < 10) SELECT * FROM cte",
			expected: "SELECT * FROM cte",
		},
		{
			name:     "Whitespace variations",
			input:    "WITH  cte  AS  (  SELECT  1  )  SELECT * FROM cte",
			expected: "SELECT * FROM cte",
		},
		{
			name:     "Newlines in CTE",
			input:    "WITH cte AS (\n  SELECT 1\n)\nSELECT * FROM cte",
			expected: "SELECT * FROM cte",
		},
		{
			name:     "Case insensitive WITH",
			input:    "with cte as (select 1) select * from cte",
			expected: "select * from cte",
		},
		{
			name:     "Case insensitive WITH RECURSIVE",
			input:    "With Recursive cte As (Select 1) Select * From cte",
			expected: "Select * From cte",
		},
		{
			name:     "Mixed case with UPDATE attack",
			input:    "WiTh cte AS (SELECT 1) UpDaTe users SET x = 1",
			expected: "UpDaTe users SET x = 1",
		},
		{
			name:     "RECURSIVE with multiple CTEs",
			input:    "WITH RECURSIVE cte1 AS (SELECT 1), cte2 AS (SELECT 2) SELECT * FROM cte1",
			expected: "SELECT * FROM cte1",
		},
		{
			name:     "Tabs and mixed whitespace",
			input:    "WITH\t\tcte\tAS\t(\tSELECT\t1\t)\tSELECT * FROM cte",
			expected: "SELECT * FROM cte",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripLeadingCTEs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateSQLIdent(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "Valid identifier",
			input:     "my_table",
			wantError: false,
		},
		{
			name:      "Valid with underscore prefix",
			input:     "_private",
			wantError: false,
		},
		{
			name:      "Valid with numbers",
			input:     "table123",
			wantError: false,
		},
		{
			name:      "Empty string (allowed for optional fields)",
			input:     "",
			wantError: false,
		},
		{
			name:      "Invalid - starts with number",
			input:     "123table",
			wantError: true,
		},
		{
			name:      "Invalid - contains dash",
			input:     "my-table",
			wantError: true,
		},
		{
			name:      "Invalid - contains space",
			input:     "my table",
			wantError: true,
		},
		{
			name:      "Invalid - SQL injection attempt",
			input:     "users; DROP TABLE users--",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSQLIdent(tt.input)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
