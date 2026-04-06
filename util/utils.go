package util

import (
	"fmt"
	"strconv"
	"strings"
)

func RemoveDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

func ContainsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func UnescapeString(s string) string {
	var result strings.Builder
	result.Grow(len(s))

	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			case 'r':
				result.WriteByte('\r')
			case '\\':
				result.WriteByte('\\')
			case '\'':
				result.WriteByte('\'')
			case '"':
				result.WriteByte('"')
			case '0':
				result.WriteByte('\x00')
			case 'a':
				result.WriteByte('\a')
			case 'b':
				result.WriteByte('\b')
			case 'f':
				result.WriteByte('\f')
			case 'v':
				result.WriteByte('\v')
			case 'x':
				if i+3 < len(s) {
					if val, err := strconv.ParseUint(s[i+2:i+4], 16, 8); err == nil {
						result.WriteByte(byte(val))
						i += 3
					} else {
						result.WriteByte('\\')
						result.WriteByte('x')
					}
				} else {
					result.WriteByte('\\')
					result.WriteByte('x')
				}
			case 'u':
				if i+5 < len(s) {
					if val, err := strconv.ParseUint(s[i+2:i+6], 16, 16); err == nil {
						result.WriteRune(rune(val))
						i += 5
					} else {
						result.WriteByte('\\')
						result.WriteByte('u')
					}
				} else {
					result.WriteByte('\\')
					result.WriteByte('u')
				}
			default:
				result.WriteByte('\\')
				result.WriteByte(s[i+1])
			}
			i++
		} else {
			result.WriteByte(s[i])
		}
	}

	return result.String()
}

func EscapeString(s string) string {
	var result strings.Builder
	result.Grow(len(s) * 2)

	for _, r := range s {
		switch r {
		case '\n':
			result.WriteString("\\n")
		case '\t':
			result.WriteString("\\t")
		case '\r':
			result.WriteString("\\r")
		case '\\':
			result.WriteString("\\\\")
		case '\'':
			result.WriteString("\\'")
		case '"':
			result.WriteString("\\\"")
		case '\x00':
			result.WriteString("\\0")
		case '\a':
			result.WriteString("\\a")
		case '\b':
			result.WriteString("\\b")
		case '\f':
			result.WriteString("\\f")
		case '\v':
			result.WriteString("\\v")
		default:
			if r < 32 || r == 127 {
				result.WriteString(fmt.Sprintf("\\x%02x", r))
			} else {
				result.WriteRune(r)
			}
		}
	}

	return result.String()
}

func logDebug(format string, args ...interface{}) {
	fmt.Printf("[DEBUG] "+format+"\n", args...)
}

func logInfo(format string, args ...interface{}) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}

func logError(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}
