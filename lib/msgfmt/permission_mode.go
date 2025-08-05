package msgfmt

import (
	"regexp"
	"strings"
)

var permissionModeRegex = regexp.MustCompile(`(Bypassing Permissions|plan mode on|auto-accept edits on|Normal mode)`)

// ExtractPermissionMode extracts the permission mode from the terminal output
func ExtractPermissionMode(msg string) string {
	lines := strings.Split(msg, "\n")
	
	// Look for Permission Mode in the last few lines
	for i := len(lines) - 1; i >= max(len(lines)-5, 0); i-- {
		matches := permissionModeRegex.FindStringSubmatch(lines[i])
		if len(matches) > 0 {
			return matches[1]
		}
	}
	
	return ""
}