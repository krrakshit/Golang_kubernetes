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

	for _, delta := range deltas {
		var change FieldChange

		// Get the path
		if postDelta, ok := delta.(gojsondiff.PostDelta); ok && postDelta.PostPosition() != nil {
			change.Path = postDelta.PostPosition().String()
		} else if preDelta, ok := delta.(gojsondiff.PreDelta); ok && preDelta.PrePosition() != nil {
			change.Path = preDelta.PrePosition().String()
		}

		// Determine the type and values based on delta type
		switch delta.(type) {
		case *gojsondiff.Object:
			change.Type = "MODIFIED"
			// For objects, we'll just note they changed
			change.OldValue = "<object modified>"
			change.NewValue = "<object modified>"

		case *gojsondiff.Array:
			change.Type = "MODIFIED"
			change.OldValue = "<array modified>"
			change.NewValue = "<array modified>"

		case *gojsondiff.Added:
			d := delta.(*gojsondiff.Added)
			change.Type = "ADDED"
			change.NewValue = d.Value

		case *gojsondiff.Deleted:
			d := delta.(*gojsondiff.Deleted)
			change.Type = "REMOVED"
			change.OldValue = d.Value

		case *gojsondiff.Modified:
			d := delta.(*gojsondiff.Modified)
			change.Type = "MODIFIED"
			change.OldValue = d.OldValue
			change.NewValue = d.NewValue

		case *gojsondiff.TextDiff:
			change.Type = "MODIFIED"
			change.OldValue = "<text changed>"
			change.NewValue = "<text changed>"

		case *gojsondiff.Moved:
			d := delta.(*gojsondiff.Moved)
			change.Type = "MOVED"
			change.NewValue = fmt.Sprintf("moved from %v to %v", d.PrePosition(), d.PostPosition())

		default:
			// Unknown delta type
			continue
		}

		changes = append(changes, change)
	}

	return changes, nil
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
