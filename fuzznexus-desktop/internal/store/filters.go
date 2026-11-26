package store

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"fuzznexus-desktop/internal/output"

	"modernc.org/sqlite"
)

var registerRegexOnce sync.Once

func init() {
	registerRegexOnce.Do(func() {
		// Register 'regexp' scalar function for SQLite: column REGEXP pattern -> regexp(pattern, column)
		_ = sqlite.RegisterDeterministicScalarFunction("regexp", 2, func(ctx *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			if len(args) != 2 {
				return false, nil
			}
			pattern, ok1 := args[0].(string)
			text, ok2 := args[1].(string)
			if !ok1 || !ok2 {
				return false, nil
			}
			// Safe capped compilation
			if len(pattern) > 256 {
				return false, errors.New("regex pattern exceeds 256 character safety limit")
			}
			re, err := regexp.Compile(pattern)
			if err != nil {
				return false, nil
			}
			return re.MatchString(text), nil
		})
	})
}

// FilterRule defines a single atomic criterion.
type FilterRule struct {
	ID       string `json:"id"`
	Field    string `json:"field"`    // status, length, words, lines, duration, url, host, payload
	Operator string `json:"operator"` // =, !=, <, <=, >, >=, between, in, contains, not_contains, regex
	Value    string `json:"value"`
	Value2   string `json:"value2,omitempty"` // For between operator
	Enabled  bool   `json:"enabled"`
	Layer    string `json:"layer"` // view, engine, both
}

// FilterGroup defines a composable tree of rules connected by AND or OR.
type FilterGroup struct {
	ID          string        `json:"id"`
	Conjunction string        `json:"conjunction"` // AND, OR
	Rules       []FilterRule  `json:"rules"`
	Groups      []FilterGroup `json:"groups,omitempty"` // Up to 2 levels of nesting
	Enabled     bool          `json:"enabled"`
}

// FilterPresetDTO represents a saved preset in SQLite.
type FilterPresetDTO struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
	FilterJSON string    `json:"filter_json"`
}

// Valid fields whitelist for SQL projection
var validFields = map[string]string{
	"status":   "status",
	"length":   "length",
	"words":    "words",
	"lines":    "lines",
	"duration": "duration_ms",
	"url":      "url",
	"host":     "host",
	"payload":  "inputs_json",
}

// IsFieldNumeric checks if field expects integer comparison.
func IsFieldNumeric(field string) bool {
	switch field {
	case "status", "length", "words", "lines", "duration":
		return true
	default:
		return false
	}
}

// BuildFilterSQL constructs a strictly parameterized SQL WHERE fragment and argument list.
func BuildFilterSQL(group FilterGroup) (string, []interface{}, error) {
	if !group.Enabled {
		return "", nil, nil
	}
	return buildGroupSQL(group, 1)
}

func buildGroupSQL(group FilterGroup, depth int) (string, []interface{}, error) {
	if depth > 3 {
		return "", nil, errors.New("maximum filter nesting depth (2 levels) exceeded")
	}

	conjunction := strings.ToUpper(strings.TrimSpace(group.Conjunction))
	if conjunction != "AND" && conjunction != "OR" {
		conjunction = "AND"
	}

	var clauses []string
	var args []interface{}

	for _, rule := range group.Rules {
		if !rule.Enabled {
			continue
		}
		clause, ruleArgs, err := buildRuleSQL(rule)
		if err != nil {
			return "", nil, err
		}
		if clause != "" {
			clauses = append(clauses, clause)
			args = append(args, ruleArgs...)
		}
	}

	for _, subGroup := range group.Groups {
		if !subGroup.Enabled {
			continue
		}
		subClause, subArgs, err := buildGroupSQL(subGroup, depth+1)
		if err != nil {
			return "", nil, err
		}
		if subClause != "" {
			clauses = append(clauses, subClause)
			args = append(args, subArgs...)
		}
	}

	if len(clauses) == 0 {
		return "", nil, nil
	}

	if len(clauses) == 1 {
		return clauses[0], args, nil
	}

	return fmt.Sprintf("(%s)", strings.Join(clauses, fmt.Sprintf(" %s ", conjunction))), args, nil
}

func buildRuleSQL(rule FilterRule) (string, []interface{}, error) {
	colName, ok := validFields[strings.ToLower(rule.Field)]
	if !ok {
		return "", nil, fmt.Errorf("invalid filter field: %s", rule.Field)
	}

	op := strings.ToLower(strings.TrimSpace(rule.Operator))
	val := strings.TrimSpace(rule.Value)

	if IsFieldNumeric(rule.Field) {
		numVal, err := strconv.ParseInt(val, 10, 64)
		if err != nil && op != "in" {
			return "", nil, fmt.Errorf("field %s requires integer value, got %q", rule.Field, val)
		}

		switch op {
		case "=":
			return fmt.Sprintf("%s = ?", colName), []interface{}{numVal}, nil
		case "!=":
			return fmt.Sprintf("%s != ?", colName), []interface{}{numVal}, nil
		case "<":
			return fmt.Sprintf("%s < ?", colName), []interface{}{numVal}, nil
		case "<=":
			return fmt.Sprintf("%s <= ?", colName), []interface{}{numVal}, nil
		case ">":
			return fmt.Sprintf("%s > ?", colName), []interface{}{numVal}, nil
		case ">=":
			return fmt.Sprintf("%s >= ?", colName), []interface{}{numVal}, nil
		case "between":
			numVal2, err := strconv.ParseInt(strings.TrimSpace(rule.Value2), 10, 64)
			if err != nil {
				return "", nil, fmt.Errorf("between operator requires second integer value, got %q", rule.Value2)
			}
			return fmt.Sprintf("%s BETWEEN ? AND ?", colName), []interface{}{numVal, numVal2}, nil
		case "in":
			parts := strings.Split(val, ",")
			var inArgs []interface{}
			var placeholders []string
			for _, p := range parts {
				pTrim := strings.TrimSpace(p)
				if pTrim == "" {
					continue
				}
				n, err := strconv.ParseInt(pTrim, 10, 64)
				if err != nil {
					return "", nil, fmt.Errorf("in operator expects comma-separated numbers, got %q", pTrim)
				}
				placeholders = append(placeholders, "?")
				inArgs = append(inArgs, n)
			}
			if len(placeholders) == 0 {
				return "", nil, nil
			}
			return fmt.Sprintf("%s IN (%s)", colName, strings.Join(placeholders, ", ")), inArgs, nil
		default:
			return "", nil, fmt.Errorf("unsupported operator %q for numeric field %s", op, rule.Field)
		}
	}

	// Text fields: url, host, payload (inputs_json)
	switch op {
	case "=":
		return fmt.Sprintf("%s = ?", colName), []interface{}{val}, nil
	case "!=":
		return fmt.Sprintf("%s != ?", colName), []interface{}{val}, nil
	case "contains":
		return fmt.Sprintf("%s LIKE ?", colName), []interface{}{"%" + val + "%"}, nil
	case "not_contains":
		return fmt.Sprintf("%s NOT LIKE ?", colName), []interface{}{"%" + val + "%"}, nil
	case "regex":
		if len(val) > 256 {
			return "", nil, errors.New("regex length exceeds 256 character safety limit")
		}
		if _, err := regexp.Compile(val); err != nil {
			return "", nil, fmt.Errorf("invalid regex %q: %w", val, err)
		}
		return fmt.Sprintf("%s REGEXP ?", colName), []interface{}{val}, nil
	default:
		return "", nil, fmt.Errorf("unsupported operator %q for text field %s", op, rule.Field)
	}
}

// EvaluateFilterGroup evaluates a filter group in memory against an output.ResultDTO.
func EvaluateFilterGroup(group FilterGroup, item *output.ResultDTO) bool {
	if !group.Enabled {
		return true
	}
	return evaluateGroupInternal(group, item, 1)
}

func evaluateGroupInternal(group FilterGroup, item *output.ResultDTO, depth int) bool {
	if depth > 3 {
		return true
	}

	conjunction := strings.ToUpper(strings.TrimSpace(group.Conjunction))
	isOR := (conjunction == "OR")

	hasActiveRules := false

	for _, rule := range group.Rules {
		if !rule.Enabled {
			continue
		}
		hasActiveRules = true
		matched := evaluateRuleInternal(rule, item)
		if isOR && matched {
			return true
		}
		if !isOR && !matched {
			return false
		}
	}

	for _, subGroup := range group.Groups {
		if !subGroup.Enabled {
			continue
		}
		hasActiveRules = true
		matched := evaluateGroupInternal(subGroup, item, depth+1)
		if isOR && matched {
			return true
		}
		if !isOR && !matched {
			return false
		}
	}

	if !hasActiveRules {
		return true
	}

	// For AND, if we reached here without returning false, it matched.
	// For OR, if we reached here without returning true, it did not match.
	return !isOR
}

func evaluateRuleInternal(rule FilterRule, item *output.ResultDTO) bool {
	field := strings.ToLower(strings.TrimSpace(rule.Field))
	op := strings.ToLower(strings.TrimSpace(rule.Operator))
	val := strings.TrimSpace(rule.Value)

	if IsFieldNumeric(field) {
		var itemVal int64
		switch field {
		case "status":
			itemVal = item.Status
		case "length":
			itemVal = item.Length
		case "words":
			itemVal = item.Words
		case "lines":
			itemVal = item.Lines
		case "duration":
			itemVal = item.DurationMs
		}

		numVal, err := strconv.ParseInt(val, 10, 64)
		if err != nil && op != "in" {
			return false
		}

		switch op {
		case "=":
			return itemVal == numVal
		case "!=":
			return itemVal != numVal
		case "<":
			return itemVal < numVal
		case "<=":
			return itemVal <= numVal
		case ">":
			return itemVal > numVal
		case ">=":
			return itemVal >= numVal
		case "between":
			numVal2, err := strconv.ParseInt(strings.TrimSpace(rule.Value2), 10, 64)
			if err != nil {
				return false
			}
			return itemVal >= numVal && itemVal <= numVal2
		case "in":
			parts := strings.Split(val, ",")
			for _, p := range parts {
				n, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64)
				if err == nil && itemVal == n {
					return true
				}
			}
			return false
		default:
			return false
		}
	}

	// Text fields
	var itemText string
	switch field {
	case "url":
		itemText = item.URL
	case "host":
		itemText = item.Host
	case "payload":
		if item.Inputs != nil {
			if fuzzVal, ok := item.Inputs["FUZZ"]; ok {
				itemText = fuzzVal
			} else {
				for _, v := range item.Inputs {
					itemText = v
					break
				}
			}
		}
	}

	switch op {
	case "=":
		return itemText == val
	case "!=":
		return itemText != val
	case "contains":
		return strings.Contains(strings.ToLower(itemText), strings.ToLower(val))
	case "not_contains":
		return !strings.Contains(strings.ToLower(itemText), strings.ToLower(val))
	case "regex":
		if len(val) > 256 {
			return false
		}
		re, err := regexp.Compile(val)
		if err != nil {
			return false
		}
		return re.MatchString(itemText)
	default:
		return false
	}
}
