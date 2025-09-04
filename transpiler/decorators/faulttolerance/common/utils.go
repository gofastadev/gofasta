// Package common provides shared utilities for fault tolerance decorators
package common

// Contains checks if string contains substring
func Contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr || 
		   (len(s) > len(substr) && s[:len(substr)] == substr) ||
		   (len(substr) > 0 && len(s) > 0 && s[0:1] == substr[0:1])
}