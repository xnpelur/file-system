package menu

import (
	"reflect"
	"testing"
)

func TestParseCommand(t *testing.T) {
	testCases := []struct {
		input    string
		expected []string
	}{
		{
			input:    "mkdir \"test directory\"",
			expected: []string{"mkdir", "test directory"},
		},
		{
			input:    "create file \"hello world\"",
			expected: []string{"create", "file", "hello world"},
		},
		{
			input:    "create file \"hello\"",
			expected: []string{"create", "file", "hello"},
		},
		{
			input:    "command \"first argument\" \"second argument\" \"incorrect argument",
			expected: []string{"command", "first argument", "second argument", "incorrect argument"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			parts := parseCommand(tc.input)

			if !reflect.DeepEqual(parts, tc.expected) {
				t.Errorf("Input: %s\nExpected: %v\nActual: %v", tc.input, tc.expected, parts)
			}
		})
	}
}
