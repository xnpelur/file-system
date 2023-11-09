package utils

import (
	"testing"
)

func TestChangeDirectoryPath(t *testing.T) {
	tests := []struct {
		currentPath  string
		cdArgument   string
		expectedPath string
	}{
		{"/home/user/documents", ".", "/home/user/documents"},
		{"/home/user/documents", "..", "/home/user"},
		{"/home/user/documents", "newfolder", "/home/user/documents/newfolder"},
		{"/root", "..", "/"},
		{"/", "..", "/"},
		{"/", "home", "/home"},
	}

	for _, test := range tests {
		result := ChangeDirectoryPath(test.currentPath, test.cdArgument)
		if result != test.expectedPath {
			t.Errorf("For %s + %s, expected %s, but got %s", test.currentPath, test.cdArgument, test.expectedPath, result)
		}
	}
}
