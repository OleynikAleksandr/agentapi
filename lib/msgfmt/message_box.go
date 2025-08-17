package msgfmt

import (
	"strings"
)

// Usually something like
// ───────────────
// >
// ───────────────
// Used by Claude Code, Goose, and Aider.
// Returns the start index and end index of the message box (inclusive)
func findGreaterThanMessageBox(lines []string) (int, int) {
	// Look for the pattern from the end of the output
	for i := len(lines) - 1; i >= max(len(lines)-10, 0); i-- {
		// Check if this line contains only ">" (the input prompt)
		if strings.TrimSpace(lines[i]) == ">" {
			// Check if there's a border above and below
			if i > 0 && i < len(lines)-1 {
				above := lines[i-1]
				below := lines[i+1]
				// Both lines should contain the border pattern
				if strings.Contains(above, "───") && strings.Contains(below, "───") {
					// Found complete message box, return start and end indices
					return i - 1, i + 1
				}
			}
		}
	}
	return -1, -1
}

// Usually something like
// ───────────────
// |
// ───────────────
// Used by OpenAI Codex.
// Returns the start index and end index of the message box (inclusive)
func findGenericSlimMessageBox(lines []string) (int, int) {
	for i := len(lines) - 3; i >= max(len(lines)-10, 0); i-- {
		if strings.Contains(lines[i], "───────────────") &&
			(strings.Contains(lines[i+1], "|") || strings.Contains(lines[i+1], "│")) &&
			strings.Contains(lines[i+2], "───────────────") {
			// Found complete message box, return start and end indices
			return i, i + 2
		}
	}
	return -1, -1
}

func removeMessageBox(msg string) string {
	lines := strings.Split(msg, "\n")

	// Try to find Claude/Goose/Aider style message box
	startIdx, endIdx := findGreaterThanMessageBox(lines)
	
	// If not found, try Codex style
	if startIdx == -1 {
		startIdx, endIdx = findGenericSlimMessageBox(lines)
	}

	// If we found a message box, remove only those lines
	if startIdx != -1 && endIdx != -1 {
		// Create new slice without the message box lines
		newLines := []string{}
		
		// Add all lines before the message box
		if startIdx > 0 {
			newLines = append(newLines, lines[:startIdx]...)
		}
		
		// Add all lines after the message box
		if endIdx < len(lines)-1 {
			newLines = append(newLines, lines[endIdx+1:]...)
		}
		
		lines = newLines
	}

	return strings.Join(lines, "\n")
}

func removeCodexMessageBox(msg string) string {
	lines := strings.Split(msg, "\n")
	messageBoxEndIdx := -1
	messageBoxStartIdx := -1

	for i := len(lines) - 1; i >= 0; i-- {
		if messageBoxEndIdx == -1 {
			if strings.Contains(lines[i], "╰────────") && strings.Contains(lines[i], "───────╯") {
				messageBoxEndIdx = i
			}
		} else {
			// We reached the start of the message box (we don't want to show this line), also exit the loop
			if strings.Contains(lines[i], "╭") && strings.Contains(lines[i], "───────╮") {
				// We only want this to be i in case the top of the box is visible
				messageBoxStartIdx = i
				break
			}

			// We are in between the start and end of the message box, so remove the │ from the start and end of the line, let the trimEmptyLines handle the rest
			if strings.HasPrefix(lines[i], "│") {
				lines[i] = strings.TrimPrefix(lines[i], "│")
			}
			if strings.HasSuffix(lines[i], "│") {
				lines[i] = strings.TrimSuffix(lines[i], "│")
				lines[i] = strings.TrimRight(lines[i], " \t")
			}
		}
	}

	// If we didn't find messageBoxEndIdx, set it to the end of the lines
	if messageBoxEndIdx == -1 {
		messageBoxEndIdx = len(lines)
	}

	return strings.Join(lines[messageBoxStartIdx+1:messageBoxEndIdx], "\n")

}
