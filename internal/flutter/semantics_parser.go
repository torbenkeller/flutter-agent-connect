package flutter

import (
	"regexp"
	"strconv"
	"strings"
)

// parseSemanticsText parses the text dump from debugDumpSemanticsTreeInTraversalOrder
// into a structured SemanticsNode tree.
func parseSemanticsText(text string) *SemanticsNode {
	root := &SemanticsNode{ID: -1, Label: "root"}

	lines := strings.Split(text, "\n")
	var nodes []*SemanticsNode
	nodeByID := make(map[int]*SemanticsNode)

	var currentNode *SemanticsNode

	for _, line := range lines {
		// Strip tree-drawing characters: │ ├─ └─ and whitespace
		trimmed := stripTreeChars(line)

		// New node: "SemanticsNode#5"
		if strings.HasPrefix(trimmed, "SemanticsNode#") {
			id := parseNodeID(trimmed)
			node := &SemanticsNode{ID: id}
			nodes = append(nodes, node)
			nodeByID[id] = node
			currentNode = node
			continue
		}

		if currentNode == nil {
			continue
		}

		// Parse properties of current node
		if strings.HasPrefix(trimmed, "Rect.fromLTRB(") {
			rect := parseRect(trimmed)
			if rect != nil {
				currentNode.Rect = rect
			}
		} else if strings.HasPrefix(trimmed, "[[") || (strings.HasPrefix(trimmed, "[") && strings.Contains(trimmed, ",0.0,")) {
			// Transform matrix row — extract translation offsets
			parseTransformRow(trimmed, currentNode)
		} else if strings.HasPrefix(trimmed, "label: ") {
			currentNode.Label = parseQuotedValue(trimmed, "label: ")
		} else if strings.HasPrefix(trimmed, "tooltip: ") {
			// tooltip acts as label for interaction purposes
			if currentNode.Label == "" {
				currentNode.Label = parseQuotedValue(trimmed, "tooltip: ")
			}
			currentNode.Hint = parseQuotedValue(trimmed, "tooltip: ")
		} else if strings.HasPrefix(trimmed, "value: ") {
			currentNode.Value = parseQuotedValue(trimmed, "value: ")
		} else if strings.HasPrefix(trimmed, "flags: ") {
			currentNode.Flags = parseCSV(trimmed, "flags: ")
		} else if strings.HasPrefix(trimmed, "actions: ") {
			currentNode.Actions = parseCSV(trimmed, "actions: ")
		} else if strings.HasPrefix(trimmed, "textDirection:") {
			// skip
		} else if strings.HasPrefix(trimmed, "sortKey:") {
			// skip
		} else if strings.Contains(trimmed, "merged up") || strings.Contains(trimmed, "merge boundary") {
			// Merge info - the child merges into parent
			// Actions/flags from merged children apply to the parent
		}
	}

	// Build parent-child relationships based on indentation
	// The text uses tree drawing characters (├─, └─, │) for structure
	// Simpler approach: just make all nodes children of root and rely on labels
	root.Children = nodes

	// Propagate actions from merged children to parents
	// Node #9 (actions: tap) is merged into Node #8 (tooltip: "Increment")
	for i := len(nodes) - 1; i >= 0; i-- {
		node := nodes[i]
		if len(node.Actions) > 0 && node.Label == "" {
			// This is likely a merged child - propagate actions to the previous node
			if i > 0 {
				parent := nodes[i-1]
				if len(parent.Actions) == 0 {
					parent.Actions = node.Actions
				}
				if len(parent.Flags) == 0 {
					parent.Flags = node.Flags
				}
			}
		}
	}

	return root
}

// parseNodeID extracts the numeric ID from "SemanticsNode#5"
func parseNodeID(s string) int {
	parts := strings.SplitN(s, "#", 2)
	if len(parts) != 2 {
		return 0
	}
	id, _ := strconv.Atoi(parts[1])
	return id
}

var rectRegex = regexp.MustCompile(`Rect\.fromLTRB\((-?[\d.]+),\s*(-?[\d.]+),\s*(-?[\d.]+),\s*(-?[\d.]+)\)`)

// parseRect extracts a Rect from "Rect.fromLTRB(0.0, 0.0, 56.0, 56.0)"
func parseRect(s string) *Rect {
	matches := rectRegex.FindStringSubmatch(s)
	if len(matches) != 5 {
		return nil
	}

	left, _ := strconv.ParseFloat(matches[1], 64)
	top, _ := strconv.ParseFloat(matches[2], 64)
	right, _ := strconv.ParseFloat(matches[3], 64)
	bottom, _ := strconv.ParseFloat(matches[4], 64)

	return &Rect{Left: left, Top: top, Right: right, Bottom: bottom}
}

var transformRowRegex = regexp.MustCompile(`\[(-?[\d.e+-]+),\s*(-?[\d.e+-]+),\s*(-?[\d.e+-]+),\s*(-?[\d.e+-]+)\]`)

// parseTransformRow extracts translation from a single matrix row.
// Transform rows look like: [[1.0,0.0,0.0,330.0]; or [0.0,1.0,0.0,768.0];
// The 4th element is the translation value (tx for row 0, ty for row 1).
func parseTransformRow(s string, node *SemanticsNode) {
	if node == nil || node.Rect == nil {
		return
	}

	matches := transformRowRegex.FindAllStringSubmatch(s, -1)
	for _, match := range matches {
		if len(match) != 5 {
			continue
		}

		// Check if this is a translation row (first element ~1.0, second ~0.0 or vice versa)
		v0, _ := strconv.ParseFloat(match[1], 64)
		v1, _ := strconv.ParseFloat(match[2], 64)
		translation, _ := strconv.ParseFloat(match[4], 64)

		if translation == 0 {
			continue
		}

		// Row [1,0,0,tx] → X translation
		if isClose(v0, 1.0) && isClose(v1, 0.0) {
			node.Rect.Left += translation
			node.Rect.Right += translation
		}
		// Row [0,1,0,ty] or [-eps,1,0,ty] → Y translation
		if isClose(v1, 1.0) && isClose(v0, 0.0) {
			node.Rect.Top += translation
			node.Rect.Bottom += translation
		}
	}
}

func isClose(a, b float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < 0.001
}

// parseQuotedValue extracts a quoted string value: `label: "Hello"` → `Hello`
func parseQuotedValue(s, prefix string) string {
	s = strings.TrimPrefix(s, prefix)
	s = strings.Trim(s, "\"")
	return s
}

// stripTreeChars removes tree-drawing characters and leading whitespace.
func stripTreeChars(s string) string {
	// Remove leading whitespace and tree chars: │ ├─ └─ ─
	result := strings.TrimLeft(s, " \t")
	for {
		changed := false
		for _, prefix := range []string{"│", "├─", "└─", "─", "│ ", "  "} {
			if strings.HasPrefix(result, prefix) {
				result = strings.TrimPrefix(result, prefix)
				changed = true
			}
		}
		result = strings.TrimLeft(result, " ")
		if !changed {
			break
		}
	}
	return result
}

// parseCSV extracts comma-separated values: `flags: isButton, isEnabled` → ["isButton", "isEnabled"]
func parseCSV(s, prefix string) []string {
	s = strings.TrimPrefix(s, prefix)
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
