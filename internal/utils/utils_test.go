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

func TestSplitPath(t *testing.T) {
	tests := []struct {
		input        string
		pathToFolder string
		fileName     string
	}{
		{"/home/user/document.txt", "/home/user", "document.txt"},
		{"dir1/dir2/file.txt", "dir1/dir2", "file.txt"},
		{"file", "", "file"},
	}

	for _, test := range tests {
		pathToFolder, fileName := SplitPath(test.input)
		if pathToFolder != test.pathToFolder || fileName != test.fileName {
			t.Errorf("For %s, expected %s and %s, but got %s and %s", test.input, test.pathToFolder, test.fileName, pathToFolder, fileName)
		}
	}
}
