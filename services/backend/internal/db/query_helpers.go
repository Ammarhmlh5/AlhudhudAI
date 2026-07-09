package db

import (
	"fmt"
	"strings"
)

// QueryHelpers provides safe SQL query building utilities
// to prevent SQL injection vulnerabilities

// buildOrderByClause creates a safe ORDER BY clause with whitelist validation
// Requirements: 2.19 - SQL injection prevention
func BuildOrderByClause(sortField string, sortOrder string) (string, error) {
	// Whitelist of allowed sort fields
	allowedFields := map[string]bool{
		"created_at":    true,
		"updated_at":    true,
		"status":        true,
		"cost":          true,
		"phone_number":  true,
		"detected_at":   true,
		"issue_date":    true,
		"display_order": true,
		"priority":      true,
	}

	// Validate sort field
	if !allowedFields[sortField] {
		return "", fmt.Errorf("invalid sort field: %s", sortField)
	}

	// Validate sort order
	sortOrder = strings.ToUpper(sortOrder)
	if sortOrder != "ASC" && sortOrder != "DESC" {
		return "", fmt.Errorf("invalid sort order: %s", sortOrder)
	}

	// Safe to use in query after validation
	return fmt.Sprintf("ORDER BY %s %s", sortField, sortOrder), nil
}

// buildFilterQuery creates a safe WHERE clause with parameterized values
// Requirements: 2.19 - SQL injection prevention
func BuildFilterQuery(filters map[string]interface{}) (string, []interface{}, error) {
	// Whitelist of allowed filters with their SQL templates
	allowedFilters := map[string]string{
		"status":        "status = $",
		"phone_number":  "phone_number = $",
		"created_after": "created_at >= $",
		"created_before": "created_at <= $",
		"tenant_id":     "tenant_id = $",
		"user_id":       "user_id = $",
		"service_id":    "service_id = $",
		"is_active":     "is_active = $",
	}

	var conditions []string
	var args []interface{}
	paramIndex := 1

	for key, value := range filters {
		template, ok := allowedFilters[key]
		if !ok {
			return "", nil, fmt.Errorf("invalid filter: %s", key)
		}

		// Replace $ with actual parameter index
		condition := strings.Replace(template, "$", fmt.Sprintf("$%d", paramIndex), 1)
		conditions = append(conditions, condition)
		args = append(args, value)
		paramIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	return whereClause, args, nil
}

// ValidateTableName validates a table name against a whitelist
// Requirements: 2.19 - SQL injection prevention
func ValidateTableName(table string) error {
	// Whitelist of allowed tables for dynamic operations
	allowedTables := map[string]bool{
		"user_consents":           true,
		"notifications":           true,
		"messages":                true,
		"global_incoming_events":  true,
		"fraud_alerts":            true,
		"invoices":                true,
		"packages":                true,
		"accounting_transactions": true,
		"balance_history":         true,
		"subscriptions":           true,
	}

	if !allowedTables[table] {
		return fmt.Errorf("invalid table name: %s", table)
	}
	return nil
}

// ValidateColumnName validates a column name against a whitelist
// Requirements: 2.19 - SQL injection prevention
func ValidateColumnName(column string) error {
	// Whitelist of allowed column names for dynamic operations
	allowedColumns := map[string]bool{
		"id":            true,
		"created_at":    true,
		"updated_at":    true,
		"status":        true,
		"tenant_id":     true,
		"user_id":       true,
		"phone_number":  true,
		"cost":          true,
		"priority":      true,
		"detected_at":   true,
		"issue_date":    true,
		"display_order": true,
		"service_id":    true,
		"is_active":     true,
	}

	if !allowedColumns[column] {
		return fmt.Errorf("invalid column name: %s", column)
	}
	return nil
}

// SafeIdentifier validates and returns a safe SQL identifier (table or column name)
// Requirements: 2.19 - SQL injection prevention
func SafeIdentifier(identifier string) (string, error) {
	// Check for dangerous characters
	if strings.ContainsAny(identifier, "';\"\\-/*") {
		return "", fmt.Errorf("invalid identifier contains dangerous characters: %s", identifier)
	}

	// Check for SQL keywords
	sqlKeywords := []string{
		"SELECT", "INSERT", "UPDATE", "DELETE", "DROP", "CREATE",
		"ALTER", "TRUNCATE", "EXEC", "EXECUTE", "UNION", "OR", "AND",
	}
	upperIdent := strings.ToUpper(identifier)
	for _, keyword := range sqlKeywords {
		if strings.Contains(upperIdent, keyword) {
			return "", fmt.Errorf("identifier contains SQL keyword: %s", identifier)
		}
	}

	return identifier, nil
}
