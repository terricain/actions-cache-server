package utils

import "strings"

func CleanStringSlice(parts []string) []string {
	result := make([]string, 0)
	for _, item := range parts {
		if cleaned := strings.Trim(item, " "); cleaned != "" {
			result = append(result, cleaned)
		}
	}
	return result
}
