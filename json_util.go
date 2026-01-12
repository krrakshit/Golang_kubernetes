package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MarshalToPrettyJSON converts any data structure to pretty-printed JSON string
func MarshalToPrettyJSON(data interface{}) string {
	if data == nil {
		return "null"
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("<error marshaling: %v>", err)
	}

	return string(jsonBytes)
}

// IndentJSON adds indentation to each line of a JSON string
func IndentJSON(jsonStr string, indentLevel int) string {
	indent := strings.Repeat(" ", indentLevel)
	lines := strings.Split(jsonStr, "\n")
	
	indentedLines := make([]string, len(lines))
	for i, line := range lines {
		if line != "" {
			indentedLines[i] = indent + line
		} else {
			indentedLines[i] = ""
		}
	}
	
	return strings.Join(indentedLines, "\n")
}

// TruncateJSON truncates JSON string if too long
func TruncateJSON(jsonStr string, maxLines int) string {
	lines := strings.Split(jsonStr, "\n")
	
	if len(lines) <= maxLines {
		return jsonStr
	}
	
	truncated := strings.Join(lines[:maxLines], "\n")
	return truncated + fmt.Sprintf("\n      ... (%d more lines)", len(lines)-maxLines)
}

// CompareAndFormatJSON creates a formatted comparison of two JSON objects
func CompareAndFormatJSON(label string, oldData, newData interface{}) string {
	oldJSON := MarshalToPrettyJSON(oldData)
	newJSON := MarshalToPrettyJSON(newData)
	
	var result strings.Builder
	
	result.WriteString(fmt.Sprintf("      📝 %s changed\n\n", label))
	
	// Old value
	result.WriteString("         OLD:\n")
	result.WriteString(IndentJSON(oldJSON, 9))
	result.WriteString("\n\n")
	
	// New value
	result.WriteString("         NEW:\n")
	result.WriteString(IndentJSON(newJSON, 9))
	result.WriteString("\n")
	
	return result.String()
}

// FormatMapChange formats a map change with pretty JSON
func FormatMapChange(changeMap map[string]interface{}) string {
	if changeMap == nil {
		return ""
	}
	
	oldVal, hasOld := changeMap["old"]
	newVal, hasNew := changeMap["new"]
	
	if !hasOld && !hasNew {
		return ""
	}
	
	var result strings.Builder
	
	if hasOld {
		result.WriteString("\n         OLD:\n")
		oldJSON := MarshalToPrettyJSON(oldVal)
		result.WriteString(IndentJSON(oldJSON, 9))
		result.WriteString("\n")
	}
	
	if hasNew {
		result.WriteString("\n         NEW:\n")
		newJSON := MarshalToPrettyJSON(newVal)
		result.WriteString(IndentJSON(newJSON, 9))
		result.WriteString("\n")
	}
	
	return result.String()
}