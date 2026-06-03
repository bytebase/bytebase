package oracle

import (
	"fmt"
	"strconv"
	"strings"
)

func normalizeDataTypeText(text string) string {
	if text == "" {
		return ""
	}

	normalized := strings.ToUpper(text)

	switch {
	case strings.HasPrefix(normalized, "VARCHAR2"):
		if strings.HasPrefix(normalized, "VARCHAR2(") && strings.HasSuffix(normalized, ")") {
			start := strings.Index(normalized, "(") + 1
			end := strings.LastIndex(normalized, ")")
			if start < end {
				sizeStr := normalized[start:end]
				if size := parseInt(sizeStr); size > 0 {
					return fmt.Sprintf("VARCHAR2(%d BYTE)", size)
				}
			}
		}
		return normalized
	case strings.HasPrefix(normalized, "CHAR("):
		if strings.HasSuffix(normalized, ")") {
			start := strings.Index(normalized, "(") + 1
			end := strings.LastIndex(normalized, ")")
			if start < end {
				sizeStr := normalized[start:end]
				if size := parseInt(sizeStr); size > 0 {
					return fmt.Sprintf("CHAR(%d BYTE)", size)
				}
			}
		}
		return normalized
	case strings.HasPrefix(normalized, "TIMESTAMP"):
		if strings.Contains(normalized, "WITH LOCAL TIME ZONE") || strings.Contains(normalized, "WITHLOCALTIMEZONE") {
			if normalized == "TIMESTAMP WITH LOCAL TIME ZONE" {
				return "TIMESTAMP(6) WITH LOCAL TIME ZONE"
			}
			if strings.Contains(normalized, "(") && strings.Contains(normalized, ")") {
				return strings.ReplaceAll(normalized, "WITHLOCALTIMEZONE", " WITH LOCAL TIME ZONE")
			}
			return "TIMESTAMP(6) WITH LOCAL TIME ZONE"
		}
		if strings.Contains(normalized, "WITH TIME ZONE") || strings.Contains(normalized, "WITHTIMEZONE") {
			if normalized == "TIMESTAMP WITH TIME ZONE" {
				return "TIMESTAMP(6) WITH TIME ZONE"
			}
			if strings.Contains(normalized, "(") && strings.Contains(normalized, ")") {
				return strings.ReplaceAll(normalized, "WITHTIMEZONE", " WITH TIME ZONE")
			}
			return "TIMESTAMP(6) WITH TIME ZONE"
		}
		if normalized == "TIMESTAMP" {
			return "TIMESTAMP(6)"
		}
		return normalized
	case strings.HasPrefix(normalized, "INTERVAL"):
		if strings.Contains(normalized, "YEAR TO MONTH") || strings.Contains(normalized, "YEARTOMONTH") {
			if normalized == "INTERVAL YEAR TO MONTH" {
				return "INTERVAL YEAR(2) TO MONTH"
			}
			if strings.Contains(normalized, "(") && strings.Contains(normalized, ")") {
				result := strings.ReplaceAll(normalized, "YEARTOMONTH", " YEAR TO MONTH")
				return strings.ReplaceAll(result, "INTERVALYEAR", "INTERVAL YEAR")
			}
			return "INTERVAL YEAR(2) TO MONTH"
		}
		if strings.Contains(normalized, "DAY TO SECOND") || strings.Contains(normalized, "DAYTOSECOND") {
			if normalized == "INTERVAL DAY TO SECOND" {
				return "INTERVAL DAY(2) TO SECOND(6)"
			}
			if strings.Contains(normalized, "(") && strings.Contains(normalized, ")") {
				result := strings.ReplaceAll(normalized, "DAYTOSECOND", " DAY TO SECOND")
				return strings.ReplaceAll(result, "INTERVALDAY", "INTERVAL DAY")
			}
			return "INTERVAL DAY(2) TO SECOND(6)"
		}
		return normalized
	case strings.HasPrefix(normalized, "LONGRAW"):
		return "LONG RAW"
	case strings.HasPrefix(normalized, "DOUBLE"):
		return "DOUBLE PRECISION"
	case normalized == "UROWID":
		return "UROWID(4000)"
	case normalized == "FLOAT":
		return "FLOAT(126)"
	case strings.HasPrefix(normalized, "NVARCHAR2"):
		if strings.HasPrefix(normalized, "NVARCHAR2(") && strings.HasSuffix(normalized, ")") {
			start := strings.Index(normalized, "(") + 1
			end := strings.LastIndex(normalized, ")")
			if start < end {
				sizeStr := normalized[start:end]
				if size := parseInt(sizeStr); size > 0 {
					if size == 2000 {
						return "NVARCHAR2(4000)"
					}
					if size <= 2000 {
						return fmt.Sprintf("NVARCHAR2(%d)", size*2)
					}
				}
			}
		}
		return normalized
	case strings.HasPrefix(normalized, "NCHAR"):
		if strings.HasPrefix(normalized, "NCHAR(") && strings.HasSuffix(normalized, ")") {
			start := strings.Index(normalized, "(") + 1
			end := strings.LastIndex(normalized, ")")
			if start < end {
				sizeStr := normalized[start:end]
				if size := parseInt(sizeStr); size > 0 {
					return fmt.Sprintf("NCHAR(%d)", size*2)
				}
			}
		}
		return normalized
	default:
		return normalized
	}
}

func parseInt(s string) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return 0
}
