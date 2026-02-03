package util

import (
	"regexp"
	"strings"
)

// ExtractJsonFromText tries to find the largest JSON object/array in the text
func ExtractJsonFromText(text string) string {
	// 1. Try to find markdown code block first
	re := regexp.MustCompile("(?s)```(?:json)?(.*?)```")
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	
	// 2. Fallback: Find first '{' or '[' and last '}' or ']'
	startObj := strings.Index(text, "{")
	startArr := strings.Index(text, "[")
	
	start := -1
	if startObj != -1 && startArr != -1 {
		if startObj < startArr {
			start = startObj
		} else {
			start = startArr
		}
	} else if startObj != -1 {
		start = startObj
	} else {
		start = startArr
	}
	
	if start == -1 {
		return text // No JSON found, return raw text
	}
	
	endObj := strings.LastIndex(text, "}")
	endArr := strings.LastIndex(text, "]")
	
	end := -1
	if endObj != -1 && endArr != -1 {
		if endObj > endArr {
			end = endObj
		} else {
			end = endArr
		}
	} else if endObj != -1 {
		end = endObj
	} else {
		end = endArr
	}
	
	if end != -1 && end > start {
		return text[start : end+1]
	}
	
	return text
}
