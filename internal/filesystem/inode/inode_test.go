package inode

import (
	"testing"
)

func TestGetTypeAndPermissionString(t *testing.T) {
	testCases := []struct {
		i        Inode
		expected string
	}{
		{
			i: Inode{
				TypeAndPermissions: 0b00111111,
			},
			expected: "drwxrwx",
		},
		{
			i: Inode{
				TypeAndPermissions: 0b10101101,
			},
			expected: "-r-xr-x",
		},
		{
			i: Inode{
				TypeAndPermissions: 0b10111000,
			},
			expected: "-rwx---",
		},
	}

	for _, testCase := range testCases {
		result := testCase.i.GetTypeAndPermissionString()
		if result != testCase.expected {
			t.Errorf("Expected: %s, Got: %s", testCase.expected, result)
		}
	}
}
