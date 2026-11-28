package stage

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"unicode"
)

// ParseSelectColumns extracts output column names/aliases from the outermost
// SELECT projection of a SQL query, handling CTEs, scalar subqueries, and
// nested FROM subqueries.
func ParseSelectColumns(query string) []string {
	q := strings.TrimSpace(query)
	q = strings.TrimRight(q, "; \t\r\n")
	if q == "" {
		return nil
	}

	selectIdx := findTopLevelKeyword(q, 0, "SELECT")
	if selectIdx == -1 {
		return nil
	}

	projStart := selectIdx + len("SELECT")
	projStart = skipWhitespace(q, projStart)
	if hasKeywordAt(q, projStart, "DISTINCT") {
		projStart = skipWhitespace(q, projStart+len("DISTINCT"))
	} else if hasKeywordAt(q, projStart, "ALL") {
		projStart = skipWhitespace(q, projStart+len("ALL"))
	}

	fromIdx := findTopLevelKeyword(q, projStart, "FROM")
	var proj string
	if fromIdx == -1 {
		proj = q[projStart:]
	} else {
		proj = q[projStart:fromIdx]
	}

	parts := splitTopLevelCommas(proj)
	if len(parts) == 1 && (strings.TrimSpace(parts[0]) == "*" || strings.HasSuffix(strings.TrimSpace(parts[0]), ".*")) {
		if fromIdx != -1 {
			afterFrom := skipWhitespace(q, fromIdx+len("FROM"))
			if afterFrom < len(q) && q[afterFrom] == '(' {
				if inner, ok := extractParenContent(q, afterFrom); ok {
					return ParseSelectColumns(inner)
				}
			}
		}
		return nil
	}

	cols := make([]string, 0, len(parts))
	for _, part := range parts {
		if col := extractProjectionName(part); col != "" {
			cols = append(cols, col)
		}
	}
	return cols
}

// IntrospectQueryColumns inspects output column names by executing the query
// with LIMIT 0. For sqlite3 file DSNs, it skips execution if the file does not
// exist yet so it never creates stray empty database files.
func IntrospectQueryColumns(ctx context.Context, driver, dsn, query string) ([]string, error) {
	if strings.TrimSpace(driver) == "" {
		driver = "sqlite3"
	}
	dsn = strings.TrimSpace(dsn)
	query = strings.TrimRight(strings.TrimSpace(query), "; \t\r\n")
	if dsn == "" || query == "" {
		return nil, nil
	}
	if driver == "sqlite3" && dsn != ":memory:" && !strings.HasPrefix(dsn, "file:") {
		if _, err := os.Stat(dsn); err != nil {
			return nil, err
		}
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, "SELECT * FROM ("+query+") LIMIT 0")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rows.Columns()
}

func skipWhitespace(s string, idx int) int {
	for idx < len(s) && unicode.IsSpace(rune(s[idx])) {
		idx++
	}
	return idx
}

func isWordBoundary(s string, idx int) bool {
	if idx < 0 || idx >= len(s) {
		return true
	}
	r := rune(s[idx])
	return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
}

func hasKeywordAt(s string, idx int, kw string) bool {
	if idx+len(kw) > len(s) {
		return false
	}
	if !strings.EqualFold(s[idx:idx+len(kw)], kw) {
		return false
	}
	return isWordBoundary(s, idx-1) && isWordBoundary(s, idx+len(kw))
}

func findTopLevelKeyword(s string, start int, kw string) int {
	depth := 0
	var quote byte
	for i := start; i < len(s); i++ {
		ch := s[i]
		if quote != 0 {
			if ch == quote {
				if i+1 < len(s) && s[i+1] == quote {
					i++
					continue
				}
				quote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"', '`':
			quote = ch
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 && hasKeywordAt(s, i, kw) {
				return i
			}
		}
	}
	return -1
}

func splitTopLevelCommas(s string) []string {
	var parts []string
	depth := 0
	var quote byte
	last := 0

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if quote != 0 {
			if ch == quote {
				if i+1 < len(s) && s[i+1] == quote {
					i++
					continue
				}
				quote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"', '`':
			quote = ch
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, s[last:i])
				last = i + 1
			}
		}
	}
	parts = append(parts, s[last:])
	return parts
}

func extractParenContent(s string, openIdx int) (string, bool) {
	depth := 0
	var quote byte
	for i := openIdx; i < len(s); i++ {
		ch := s[i]
		if quote != 0 {
			if ch == quote {
				if i+1 < len(s) && s[i+1] == quote {
					i++
					continue
				}
				quote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"', '`':
			quote = ch
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[openIdx+1 : i], true
			}
		}
	}
	return "", false
}

func extractProjectionName(expr string) string {
	expr = strings.TrimSpace(expr)
	if expr == "" || expr == "*" || strings.HasSuffix(expr, ".*") {
		return ""
	}

	// Check for explicit "AS <alias>" at depth 0.
	if asIdx := findTopLevelKeyword(expr, 0, "AS"); asIdx != -1 {
		alias := strings.TrimSpace(expr[asIdx+2:])
		return cleanIdentifier(alias)
	}

	// Check for implicit alias after closing paren or whitespace (e.g. "COUNT(*) total" or "u.name user_name").
	depth := 0
	var quote byte
	lastSpace := -1
	for i := 0; i < len(expr); i++ {
		ch := expr[i]
		if quote != 0 {
			if ch == quote {
				quote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"', '`':
			quote = ch
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 && unicode.IsSpace(rune(ch)) {
				lastSpace = i
			}
		}
	}

	if lastSpace != -1 {
		tail := strings.TrimSpace(expr[lastSpace+1:])
		if tail != "" && !strings.HasPrefix(tail, "(") {
			return cleanIdentifier(tail)
		}
	}

	// Strip table prefix (e.g. "u.name" -> "name") when not inside a function call.
	if dotIdx := strings.LastIndex(expr, "."); dotIdx != -1 && !strings.Contains(expr, "(") {
		return cleanIdentifier(expr[dotIdx+1:])
	}

	return cleanIdentifier(expr)
}

func cleanIdentifier(ident string) string {
	ident = strings.TrimSpace(ident)
	ident = strings.Trim(ident, "\"`[]")
	if ident == "" || ident == "*" {
		return ""
	}
	return ident
}
