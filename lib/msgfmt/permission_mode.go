package msgfmt

import (
	"regexp"
	"strings"
)

var permissionModeRegex = regexp.MustCompile(`(Bypassing Permissions|plan mode on|auto-accept edits on|Normal mode)`)

// ExtractPermissionMode extracts the permission mode from the terminal output
func ExtractPermissionMode(msg string) string {
	lines := strings.Split(msg, "\n")
	
	// Look for Permission Mode in the last 10 lines
	for i := len(lines) - 1; i >= max(len(lines)-10, 0); i-- {
		line := lines[i]
		
		// First check for "Bypassing Permissions" which appears at the end of the line
		// Format: "? for shortcuts                    Bypassing Permissions"
		if strings.Contains(line, "Bypassing Permissions") {
			return "Bypassing Permissions"
		}
		
		// For other modes, check if they appear in the line
		// but only if "? for shortcuts" is present AND no permission mode at the end
		if strings.Contains(line, "? for shortcuts") {
			// If we have "? for shortcuts" but no special mode at the end,
			// it means we're in Normal mode
			if !strings.Contains(line, "plan mode on") && 
			   !strings.Contains(line, "auto-accept edits on") {
				return "Normal mode"
			}
		}
		
		// Check for other modes that appear differently
		if strings.Contains(line, "plan mode on") {
			return "plan mode on"
		}
		if strings.Contains(line, "auto-accept edits on") {
			return "auto-accept edits on"
		}
	}
	
	return ""
}