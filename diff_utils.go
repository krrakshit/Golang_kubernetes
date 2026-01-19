package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
)

// DiffResult represents the result of a diff operation
type DiffResult struct {
	HasChanges bool
	Deltas     []string
	AsciiDiff  string
	JSONDiff   string
}

// FieldChange represents a single field change
type FieldChange struct {
	Type     string
	Path     string
	OldValue interface{}
	NewValue interface{}
}

// DiffJSON compares two JSON-serializable objects and returns the differences
func DiffJSON(old, new interface{}) (*DiffResult, error) {
	// Marshal to JSON
	oldJSON, err := json.Marshal(old)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal old object: %w", err)
	}

	newJSON, err := json.Marshal(new)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal new object: %w", err)
	}

	// Create differ
	differ := gojsondiff.New()

	// Compare
	diff, err := differ.Compare(oldJSON, newJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to compare JSON: %w", err)
	}

	// Check if there are changes
	if !diff.Modified() {
		return &DiffResult{HasChanges: false}, nil
	}

	// Get deltas (list of changes)
	deltas := diff.Deltas()
	deltaStrings := make([]string, 0, len(deltas))
	for _, delta := range deltas {
		deltaStrings = append(deltaStrings, fmt.Sprintf("%v", delta))
	}

	// Format as ASCII diff
	config := formatter.AsciiFormatterConfig{
		ShowArrayIndex: true,
		Coloring:       false,
	}

	// Unmarshal old JSON for formatter
	var oldData interface{}
	json.Unmarshal(oldJSON, &oldData)

	asciiFormatter := formatter.NewAsciiFormatter(oldData, config)
	asciiDiff, err := asciiFormatter.Format(diff)
	if err != nil {
		asciiDiff = "Error formatting diff"
	}

	// Format as JSON diff (for programmatic use)
	jsonFormatter := formatter.NewDeltaFormatter()
	jsonDiff, err := jsonFormatter.Format(diff)
	if err != nil {
		jsonDiff = "{}"
	}

	return &DiffResult{
		HasChanges: true,
		Deltas:     deltaStrings,
		AsciiDiff:  asciiDiff,
		JSONDiff:   jsonDiff,
	}, nil
}

// PrintDiff prints a formatted diff with context
func PrintDiff(label string, old, new interface{}) {
	result, err := DiffJSON(old, new)
	if err != nil {
		fmt.Printf("      ❌ Error comparing %s: %v\n", label, err)
		return
	}

	if !result.HasChanges {
		fmt.Printf("      ℹ️  No changes in %s\n", label)
		return
	}

	fmt.Printf("      📝 %s changed:\n\n", label)

	// Print the ASCII diff
	lines := strings.Split(result.AsciiDiff, "\n")
	for _, line := range lines {
		if line != "" {
			fmt.Printf("         %s\n", line)
		}
	}
	fmt.Println()
}

// LogChanges logs exact changes in a readable format
func LogChanges(old, new interface{}, label string) {
	oldJSON, err := json.Marshal(old)
	if err != nil {
		fmt.Printf("Error marshaling old: %v\n", err)
		return
	}

	newJSON, err := json.Marshal(new)
	if err != nil {
		fmt.Printf("Error marshaling new: %v\n", err)
		return
	}

	differ := gojsondiff.New()
	diff, err := differ.Compare(oldJSON, newJSON)
	if err != nil {
		fmt.Printf("Error comparing: %v\n", err)
		return
	}

	if !diff.Modified() {
		fmt.Printf("      ℹ️  No changes in %s\n", label)
		return
	}

	fmt.Printf("\n📋 Changes in %s:\n", label)
	deltas := diff.Deltas()
	logDeltasRecursive(deltas, "")
	fmt.Println()
}

// logDeltasRecursive recursively logs all deltas with their actual values
func logDeltasRecursive(deltas []gojsondiff.Delta, indent string) {
	for i, delta := range deltas {
		path := ""

		switch d := delta.(type) {
		case *gojsondiff.Added:
			if postDelta, ok := delta.(gojsondiff.PostDelta); ok {
				path = postDelta.PostPosition().String()
			}
			fmt.Printf("%s  [%d] ➕ ADDED: %s\n", indent, i+1, path)
			fmt.Printf("%s      Value: %s\n", indent, formatValueCompact(d.Value))

		case *gojsondiff.Deleted:
			if preDelta, ok := delta.(gojsondiff.PreDelta); ok {
				path = preDelta.PrePosition().String()
			}
			fmt.Printf("%s  [%d] ➖ DELETED: %s\n", indent, i+1, path)
			fmt.Printf("%s      Value: %s\n", indent, formatValueCompact(d.Value))

		case *gojsondiff.Modified:
			if postDelta, ok := delta.(gojsondiff.PostDelta); ok {
				path = postDelta.PostPosition().String()
			}
			fmt.Printf("%s  [%d] ✏️  MODIFIED: %s\n", indent, i+1, path)
			fmt.Printf("%s      OLD: %s\n", indent, formatValueCompact(d.OldValue))
			fmt.Printf("%s      NEW: %s\n", indent, formatValueCompact(d.NewValue))

		case *gojsondiff.TextDiff:
			if postDelta, ok := delta.(gojsondiff.PostDelta); ok {
				path = postDelta.PostPosition().String()
			}
			fmt.Printf("%s  [%d] ✏️  TEXT DIFF: %s\n", indent, i+1, path)
			fmt.Printf("%s      OLD: %s\n", indent, formatValueCompact(d.OldValue))
			fmt.Printf("%s      NEW: %s\n", indent, formatValueCompact(d.NewValue))
			fmt.Printf("%s      Diff: %s\n", indent, d.DiffString())

		case *gojsondiff.Object:
			if postDelta, ok := delta.(gojsondiff.PostDelta); ok {
				path = postDelta.PostPosition().String()
			}
			fmt.Printf("%s  [%d] 🔧 OBJECT: %s\n", indent, i+1, path)
			if len(d.Deltas) > 0 {
				logDeltasRecursive(d.Deltas, indent+"     ")
			}

		case *gojsondiff.Array:
			if postDelta, ok := delta.(gojsondiff.PostDelta); ok {
				path = postDelta.PostPosition().String()
			}
			fmt.Printf("%s  [%d] 📋 ARRAY: %s\n", indent, i+1, path)
			if len(d.Deltas) > 0 {
				logDeltasRecursive(d.Deltas, indent+"     ")
			}

		case *gojsondiff.Moved:
			fmt.Printf("%s  [%d] 🔄 MOVED\n", indent, i+1)
			fmt.Printf("%s      From: %v\n", indent, d.PrePosition())
			fmt.Printf("%s      To: %v\n", indent, d.PostPosition())

		default:
			fmt.Printf("%s  [%d] ❓ UNKNOWN (%T)\n", indent, i+1, delta)
		}
	}
}

// formatValueCompact formats values in a compact readable way
func formatValueCompact(val interface{}) string {
	if val == nil {
		return "<nil>"
	}

	switch v := val.(type) {
	case string:
		if len(v) > 100 {
			return fmt.Sprintf(`"%s..."`, v[:100])
		}
		return fmt.Sprintf(`"%s"`, v)
	case bool, float64, int:
		return fmt.Sprintf("%v", v)
	case map[string]interface{}, []interface{}:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		jsonStr := string(jsonBytes)
		if len(jsonStr) > 100 {
			return jsonStr[:100] + "..."
		}
		return jsonStr
	default:
		str := fmt.Sprintf("%v", val)
		if len(str) > 100 {
			return str[:100] + "..."
		}
		return str
	}
}

// GetFieldChanges extracts individual field changes with their paths
func GetFieldChanges(old, new interface{}) ([]FieldChange, error) {
	oldJSON, _ := json.Marshal(old)
	newJSON, _ := json.Marshal(new)

	differ := gojsondiff.New()
	diff, err := differ.Compare(oldJSON, newJSON)
	if err != nil {
		return nil, err
	}

	if !diff.Modified() {
		return nil, nil
	}

	changes := make([]FieldChange, 0)
	deltas := diff.Deltas()
	changes = extractChangesRecursive(deltas, changes)

	return changes, nil
}

// extractChangesRecursive recursively extracts all changes from deltas
func extractChangesRecursive(deltas []gojsondiff.Delta, changes []FieldChange) []FieldChange {
	for _, delta := range deltas {
		var change FieldChange

		// Get the path
		if postDelta, ok := delta.(gojsondiff.PostDelta); ok && postDelta.PostPosition() != nil {
			change.Path = postDelta.PostPosition().String()
		} else if preDelta, ok := delta.(gojsondiff.PreDelta); ok && preDelta.PrePosition() != nil {
			change.Path = preDelta.PrePosition().String()
		}

		// Determine the type and values based on delta type
		switch d := delta.(type) {
		case *gojsondiff.Object:
			// Recursively process object's nested deltas
			changes = extractChangesRecursive(d.Deltas, changes)
			continue

		case *gojsondiff.Array:
			// Recursively process array's nested deltas
			changes = extractChangesRecursive(d.Deltas, changes)
			continue

		case *gojsondiff.Added:
			change.Type = "ADDED"
			change.NewValue = d.Value

		case *gojsondiff.Deleted:
			change.Type = "REMOVED"
			change.OldValue = d.Value

		case *gojsondiff.Modified:
			change.Type = "MODIFIED"
			change.OldValue = d.OldValue
			change.NewValue = d.NewValue

		case *gojsondiff.TextDiff:
			change.Type = "MODIFIED"
			change.OldValue = d.OldValue
			change.NewValue = d.NewValue

		case *gojsondiff.Moved:
			change.Type = "MOVED"
			change.NewValue = fmt.Sprintf("moved from %v to %v", d.PrePosition(), d.PostPosition())

		default:
			// Unknown delta type
			continue
		}

		changes = append(changes, change)
	}

	return changes
}

// PrintFieldChanges prints individual field changes in a readable format
func PrintFieldChanges(changes []FieldChange) {
	if len(changes) == 0 {
		fmt.Println("      ℹ️  No changes detected")
		return
	}

	for _, change := range changes {
		switch change.Type {
		case "ADDED":
			fmt.Printf("      ➕ %s\n", change.Path)
			fmt.Printf("         Added: %v\n\n", formatValue(change.NewValue))

		case "REMOVED":
			fmt.Printf("      ➖ %s\n", change.Path)
			fmt.Printf("         Removed: %v\n\n", formatValue(change.OldValue))

		case "MODIFIED":
			fmt.Printf("      ✏️  %s\n", change.Path)
			fmt.Printf("         OLD: %v\n", formatValue(change.OldValue))
			fmt.Printf("         NEW: %v\n\n", formatValue(change.NewValue))

		case "MOVED":
			fmt.Printf("      🔄 %s\n", change.Path)
			fmt.Printf("         %v\n\n", change.NewValue)
		}
	}
}

// formatValue formats a value for display
func formatValue(val interface{}) string {
	if val == nil {
		return "<not set>"
	}

	// If it's a complex object, marshal to JSON
	switch v := val.(type) {
	case string:
		return fmt.Sprintf(`"%s"`, v)
	case map[string]interface{}, []interface{}:
		jsonBytes, err := json.MarshalIndent(v, "         ", "  ")
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return "\n         " + string(jsonBytes)
	default:
		return fmt.Sprintf("%v", val)
	}
}
