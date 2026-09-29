package project

import "strings"

func normalizeProtoName(name string) string {
	var result strings.Builder
	previousUnderscore := false
	for _, character := range strings.ToLower(name) {
		valid := character >= 'a' && character <= 'z' || character >= '0' && character <= '9'
		if valid {
			result.WriteRune(character)
			previousUnderscore = false
		} else if !previousUnderscore && result.Len() > 0 {
			result.WriteByte('_')
			previousUnderscore = true
		}
	}
	normalized := strings.Trim(result.String(), "_")
	if normalized == "" {
		normalized = "service"
	}
	if normalized[0] >= '0' && normalized[0] <= '9' {
		normalized = "service_" + normalized
	}
	return normalized
}

func normalizeProtoGoPackage(name string) string {
	return strings.ReplaceAll(normalizeProtoName(name), "_", "") + "v1"
}

func normalizeProtoServiceName(name string) string {
	parts := strings.Split(normalizeProtoName(name), "_")
	var result strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		result.WriteString(strings.ToUpper(part[:1]))
		result.WriteString(part[1:])
	}
	return result.String() + "Service"
}
