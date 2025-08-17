package msgfmt

import (
	"regexp"
	"strings"
)

// ServiceSplitter separates terminal output into chat messages and service info
type ServiceSplitter struct {
	// Pattern for detecting Cooking Status line
	// Example: ✳ Zigzagging… (90s · ⚒ 1.1k tokens · esc to interrupt)
	cookingPattern *regexp.Regexp
}

// NewServiceSplitter creates a new service splitter
func NewServiceSplitter() *ServiceSplitter {
	// Pattern explanation:
	// . - any character (animated symbol)
	// \w+ - any word (Zigzagging, Cooking, etc.)
	// … - three dots or ellipsis
	// \(\d+s · . [\d.]+k? tokens · esc to interrupt\)
	// Note: · is Unicode U+00B7 (middle dot), not regular dot!
	pattern := `. \w+… \(\d+s · . [\d.]+k? tokens · esc to interrupt\)`
	
	return &ServiceSplitter{
		cookingPattern: regexp.MustCompile(pattern),
	}
}

// SplitOutput separates terminal output into chat content and service lines
// Returns: (chat content, service lines)
func (s *ServiceSplitter) SplitOutput(rawOutput string) (string, []string) {
	lines := strings.Split(rawOutput, "\n")
	
	// Find cooking status line
	cookingLineIndex := -1
	for i, line := range lines {
		if s.cookingPattern.MatchString(line) {
			cookingLineIndex = i
			break
		}
	}
	
	var chatLines []string
	var serviceLines []string
	
	if cookingLineIndex >= 0 {
		// Found cooking status - everything from it to the end is service info
		chatLines = lines[:cookingLineIndex]
		serviceLines = lines[cookingLineIndex:]
	} else {
		// No cooking status - take only last 2 lines as service info
		if len(lines) >= 2 {
			chatLines = lines[:len(lines)-2]
			serviceLines = lines[len(lines)-2:]
		} else {
			// Less than 2 lines - all are service lines
			serviceLines = lines
		}
	}
	
	// Join chat lines back into string
	chatContent := strings.Join(chatLines, "\n")
	
	// Remove empty lines from the beginning and end of chat content
	chatContent = strings.TrimSpace(chatContent)
	
	return chatContent, serviceLines
}

// GetServiceInfo formats service lines into a single string
func (s *ServiceSplitter) GetServiceInfo(serviceLines []string) string {
	// Filter out empty lines but keep the structure
	var filtered []string
	for _, line := range serviceLines {
		// Keep all non-empty lines
		if strings.TrimSpace(line) != "" {
			filtered = append(filtered, line)
		}
	}
	return strings.Join(filtered, "\n")
}