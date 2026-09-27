package problem

import (
	"fmt"
	"slices"
	"strings"

	"github.com/isthatcentered/robert/internal/catalog"
)

// Format renders a command failure for stderr without exposing the JSON error shape.
func Format(err error) string {
	failure := AsError(err)
	context := failure.Context
	message := failure.Message
	if command, ok := context["command"].(string); ok && message == "unknown command" {
		message += " " + fmt.Sprintf("%q", command)
	}
	var output strings.Builder
	fmt.Fprintf(&output, "error: %s\n", message)

	known := map[string]bool{"command": true, "usage": true}
	add := func(key, label string) {
		known[key] = true
		value, exists := context[key]
		if !exists || value == nil {
			return
		}
		formatted := formatValue(value)
		if formatted != "" {
			formatted = strings.ReplaceAll(formatted, "\n", "\n  "+strings.Repeat(" ", 19))
			fmt.Fprintf(&output, "  %-18s %s\n", label+":", formatted)
		}
	}
	add("url", "Repository")
	add("reference", "Reference")
	add("matchingReferences", "Matches")
	pathLabel := "Checkout"
	if strings.HasPrefix(message, "failed to read configuration") || strings.HasPrefix(message, "failed to lock configuration") || strings.HasPrefix(message, "failed to unlock configuration") {
		pathLabel = "Config"
	}
	add("path", pathLabel)
	add("checkoutPath", "Checkout")
	add("installDir", "Checkout root")
	add("configPath", "Config")
	add("lockPath", "Lock file")
	add("gitCommand", "Git command")
	add("gitError", "Cause")
	add("cause", "Cause")
	add("cleanupError", "Cleanup error")
	if context["committed"] == true {
		fmt.Fprintln(&output, "  Catalogue saved:   yes")
	}
	known["committed"] = true
	known["hint"] = true
	for _, key := range sortedRemainingKeys(context, known) {
		add(key, key)
	}
	if hint, ok := context["hint"].(string); ok && hint != "" {
		fmt.Fprintf(&output, "  %-18s %s\n", "Hint:", sentence(hint))
	}
	if usage, ok := context["usage"].(string); ok {
		output.WriteByte('\n')
		fmt.Fprintf(&output, "Usage: %s\n", usage)
		command := "robert"
		parts := strings.Fields(usage)
		if len(parts) >= 2 && parts[0] == "robert" {
			command += " " + parts[1]
		}
		fmt.Fprintf(&output, "Run %q for details.\n", command+" --help")
	} else if message == "command is required" || strings.HasPrefix(message, "unknown command") {
		output.WriteString("\nRun \"robert --help\" for available commands.\n")
	}
	return output.String()
}

func formatValue(value any) string {
	switch v := value.(type) {
	case catalog.Reference:
		return v.Type + " " + v.Value
	case *catalog.Reference:
		if v == nil {
			return ""
		}
		return v.Type + " " + v.Value
	case []catalog.Reference:
		parts := make([]string, len(v))
		for i, ref := range v {
			parts[i] = formatValue(ref)
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprint(value)
	}
}

func sortedRemainingKeys(context map[string]any, known map[string]bool) []string {
	var keys []string
	for key := range context {
		if !known[key] {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	return keys
}

func sentence(value string) string {
	if value == "" {
		return value
	}
	value = strings.ToUpper(value[:1]) + value[1:]
	if strings.HasSuffix(value, ".") || strings.HasSuffix(value, "!") || strings.HasSuffix(value, "?") {
		return value
	}
	return value + "."
}
